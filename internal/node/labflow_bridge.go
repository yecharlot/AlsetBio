package node

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"

	qrcode "github.com/skip2/go-qrcode"

	"redalset/internal/labflow"
)

// nodeBlockBackend adapts NodoAlset IPFS/blockstore to labflow.BlockBackend.
type nodeBlockBackend struct {
	n *NodoAlset
}

func (b *nodeBlockBackend) Put(data []byte) (string, error) {
	return b.n.GenerarCID(data)
}

func (b *nodeBlockBackend) Get(cid string) ([]byte, error) {
	return b.n.BuscarContenidoPorCID(cid)
}

var (
	labflowOnce sync.Once
	labflowSvc  *labflow.Service
)

func (n *NodoAlset) labflowService() *labflow.Service {
	labflowOnce.Do(func() {
		store := labflow.NewStore(&nodeBlockBackend{n: n}, "alset_data")
		labflowSvc = labflow.NewService(store)
	})
	return labflowSvc
}

func (n *NodoAlset) registerLabFlow(extra map[string]http.HandlerFunc) {
	extra["/api/labflow/samples"] = n.handleLabflowSamples
	extra["/api/labflow/samples/"] = n.handleLabflowSampleByID
	extra["/api/labflow/verify/"] = n.handleLabflowVerify
	extra["/api/labflow/root"] = n.handleLabflowRoot
	extra["/api/labflow/stats"] = n.handleLabflowStats
	extra["/api/labflow/qr/"] = n.handleLabflowQR
	extra["/verify/"] = n.handleLabflowVerifyPage
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func (n *NodoAlset) handleLabflowRoot(w http.ResponseWriter, r *http.Request) {
	svc := n.labflowService()
	writeJSON(w, 200, map[string]interface{}{
		"root_cid": svc.RootCID(),
		"storage":  "ipfs-blocks",
		"note":     "LabFlow index and samples are content-addressed CIDs in the node blockstore",
	})
}

func (n *NodoAlset) handleLabflowSamples(w http.ResponseWriter, r *http.Request) {
	svc := n.labflowService()
	switch r.Method {
	case http.MethodGet:
		list, err := svc.List()
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]interface{}{
			"samples":  list,
			"root_cid": svc.RootCID(),
			"count":    len(list),
		})
	case http.MethodPost:
		var in labflow.CreateInput
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			writeJSON(w, 400, map[string]string{"error": "invalid json"})
			return
		}
		res, err := svc.Create(in)
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": err.Error()})
			return
		}
		// optional node persistence of blocks already done via GenerarCID (disk + memory)
		n.PersistirLocamente()
		writeJSON(w, 201, res)
	default:
		http.Error(w, "method not allowed", 405)
	}
}

func (n *NodoAlset) handleLabflowSampleByID(w http.ResponseWriter, r *http.Request) {
	svc := n.labflowService()
	path := strings.TrimPrefix(r.URL.Path, "/api/labflow/samples/")
	path = strings.Trim(path, "/")
	if path == "" {
		http.NotFound(w, r)
		return
	}
	parts := strings.Split(path, "/")
	id := parts[0]

	// /api/labflow/samples/:id/events
	if len(parts) >= 2 && parts[1] == "events" {
		if r.Method == http.MethodGet {
			evs, err := svc.Events(id)
			if err != nil {
				writeJSON(w, 500, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, 200, map[string]interface{}{"sample_id": id, "events": evs})
			return
		}
		http.Error(w, "method not allowed", 405)
		return
	}

	// /api/labflow/samples/:id/transition
	if len(parts) >= 2 && parts[1] == "transition" {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", 405)
			return
		}
		var in labflow.TransitionInput
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			writeJSON(w, 400, map[string]string{"error": "invalid json"})
			return
		}
		sample, sampleCID, rootCID, err := svc.Transition(id, in)
		if err != nil {
			writeJSON(w, 400, map[string]string{"error": err.Error()})
			return
		}
		n.PersistirLocamente()
		writeJSON(w, 200, map[string]interface{}{
			"sample":     sample,
			"sample_cid": sampleCID,
			"root_cid":   rootCID,
		})
		return
	}

	// /api/labflow/samples/:id
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", 405)
		return
	}
	sample, sampleCID, err := svc.Get(id)
	if err != nil {
		writeJSON(w, 404, map[string]string{"error": err.Error()})
		return
	}
	evs, _ := svc.Events(id)
	writeJSON(w, 200, map[string]interface{}{
		"sample":     sample,
		"sample_cid": sampleCID,
		"events":     evs,
		"root_cid":   sample.RootCID,
	})
}

func (n *NodoAlset) handleLabflowVerify(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/labflow/verify/")
	id = strings.Trim(id, "/")
	if id == "" {
		writeJSON(w, 400, map[string]string{"error": "missing id"})
		return
	}
	view, err := n.labflowService().Verify(id)
	if err != nil {
		writeJSON(w, 404, view)
		return
	}
	writeJSON(w, 200, view)
}

func (n *NodoAlset) handleLabflowVerifyPage(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/verify/")
	id = strings.Trim(id, "/")
	view, err := n.labflowService().Verify(id)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err != nil || view == nil || !view.Verified {
		w.WriteHeader(404)
		_, _ = w.Write([]byte(`<!DOCTYPE html><html><body style="font-family:system-ui;background:#0b0c10;color:#eee;padding:2rem">
<h1>Sample not found</h1><p>Integrity: NOT_FOUND</p></body></html>`))
		return
	}
	html := `<!DOCTYPE html><html lang="es"><head><meta charset="utf-8"/><meta name="viewport" content="width=device-width,initial-scale=1"/>
<title>Verify ` + view.ExternalID + `</title>
<style>body{font-family:system-ui;background:#0b0c10;color:#e8e9ef;display:flex;min-height:100vh;align-items:center;justify-content:center;margin:0}
.card{background:#161822;border-radius:16px;padding:2rem;max-width:420px;border:1px solid rgba(255,255,255,.08)}
.ok{color:#25d366;font-weight:700}.muted{color:#8b90a0;font-size:.9rem}code{font-size:.75rem;word-break:break-all;color:#f4b400}</style></head>
<body><div class="card"><div class="ok">✓ VERIFIED</div>
<h1 style="margin:.5rem 0">` + view.ExternalID + `</h1>
<p>Status: <strong>` + string(view.Status) + `</strong></p>
<p class="muted">Created: ` + view.Created + `</p>
<p>Integrity: <span class="ok">` + view.Integrity + `</span></p>
<p>Certificate: <span class="ok">` + view.Certificate + `</span></p>
<p class="muted">Sample CID</p><code>` + view.SampleCID + `</code>
<p class="muted" style="margin-top:1rem">LabFlow root CID</p><code>` + view.RootCID + `</code>
</div></body></html>`
	_, _ = w.Write([]byte(html))
}

func (n *NodoAlset) handleLabflowStats(w http.ResponseWriter, r *http.Request) {
	st, err := n.labflowService().Stats()
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, 200, st)
}

func (n *NodoAlset) handleLabflowQR(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/labflow/qr/")
	id = strings.Trim(id, "/")
	if id == "" {
		http.Error(w, "missing id", 400)
		return
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if xf := r.Header.Get("X-Forwarded-Proto"); xf != "" {
		scheme = xf
	}
	verifyURL := fmt.Sprintf("%s://%s/verify/%s", scheme, r.Host, id)
	png, err := qrcode.Encode(verifyURL, qrcode.Medium, 256)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(png)
}
