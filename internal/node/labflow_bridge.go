package node

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
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

func (n *NodoAlset) labflowRequireAuth() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("LABFLOW_REQUIRE_AUTH")))
	return v == "1" || v == "true" || v == "yes"
}

func (n *NodoAlset) labflowPrincipal(r *http.Request) (labflow.Principal, int, string) {
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		tok := strings.TrimSpace(auth[7:])
		token, err := n.validarToken(tok)
		if err != nil {
			return labflow.Principal{}, 401, err.Error()
		}
		p := labflow.Principal{
			AgentID: token.AgentID,
			Roles:   labflow.NormalizeRoles(token.Roles),
			OrgID:   r.Header.Get("X-Lab-Org"),
		}
		if p.OrgID == "" {
			p.OrgID = r.URL.Query().Get("org_id")
		}
		return p, 0, ""
	}
	if n.labflowRequireAuth() {
		return labflow.Principal{}, 401, "authorization required (Bearer token)"
	}
	roles := r.Header.Get("X-Lab-Role")
	if roles == "" {
		roles = labflow.RoleTechnician
	}
	org := r.Header.Get("X-Lab-Org")
	if org == "" {
		org = "lab-default"
	}
	agent := r.Header.Get("X-Lab-Actor")
	if agent == "" {
		agent = "dev-user"
	}
	return labflow.Principal{
		AgentID: agent,
		Roles:   labflow.NormalizeRoles(strings.Split(roles, ",")),
		OrgID:   org,
	}, 0, ""
}


func (n *NodoAlset) registerLabFlow(extra map[string]http.HandlerFunc) {
	extra["/api/labflow/samples"] = n.handleLabflowSamples
	extra["/api/labflow/samples/"] = n.handleLabflowSampleByID
	extra["/api/labflow/verify/"] = n.handleLabflowVerify
	extra["/api/labflow/root"] = n.handleLabflowRoot
	extra["/api/labflow/stats"] = n.handleLabflowStats
	extra["/api/labflow/qr/"] = n.handleLabflowQR
	extra["/api/labflow/auth/token"] = n.handleLabflowAuthToken
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
	p, code, msg := n.labflowPrincipal(r)
	if code != 0 {
		writeJSON(w, code, map[string]string{"error": msg})
		return
	}
	svc := n.labflowService()
	switch r.Method {
	case http.MethodGet:
		list, err := svc.List()
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": err.Error()})
			return
		}
		filtered := make([]labflow.Sample, 0, len(list))
		for i := range list {
			if p.CanViewSample(&list[i]) {
				filtered = append(filtered, list[i])
			}
		}
		writeJSON(w, 200, map[string]interface{}{
			"samples":  filtered,
			"root_cid": svc.RootCID(),
			"count":    len(filtered),
			"actor":    p.AgentID,
			"roles":    p.Roles,
		})
	case http.MethodPost:
		if !p.CanCreateSample() {
			writeJSON(w, 403, map[string]string{"error": "forbidden: role cannot create samples"})
			return
		}
		var in labflow.CreateInput
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			writeJSON(w, 400, map[string]string{"error": "invalid json"})
			return
		}
		if in.OrgID == "" {
			in.OrgID = p.OrgID
		}
		if in.Actor == "" {
			in.Actor = p.AgentID
		}
		// non-admin cannot create outside their org
		if !p.IsAdmin() && p.OrgID != "" && in.OrgID != p.OrgID {
			writeJSON(w, 403, map[string]string{"error": "forbidden: org mismatch"})
			return
		}
		res, err := svc.Create(in)
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": err.Error()})
			return
		}
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
			p, code, msg := n.labflowPrincipal(r)
			if code != 0 {
				writeJSON(w, code, map[string]string{"error": msg})
				return
			}
			sample, _, errS := svc.Get(id)
			if errS != nil {
				writeJSON(w, 404, map[string]string{"error": errS.Error()})
				return
			}
			if !p.CanViewSample(sample) {
				writeJSON(w, 403, map[string]string{"error": "forbidden"})
				return
			}
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
		p, code, msg := n.labflowPrincipal(r)
		if code != 0 {
			writeJSON(w, code, map[string]string{"error": msg})
			return
		}
		var in labflow.TransitionInput
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			writeJSON(w, 400, map[string]string{"error": "invalid json"})
			return
		}
		if !p.CanTransition(in.ToStatus) {
			writeJSON(w, 403, map[string]string{"error": "forbidden: role cannot transition to " + string(in.ToStatus)})
			return
		}
		existing, _, errGet := svc.Get(id)
		if errGet != nil {
			writeJSON(w, 404, map[string]string{"error": errGet.Error()})
			return
		}
		if !p.CanViewSample(existing) {
			writeJSON(w, 403, map[string]string{"error": "forbidden: sample outside your scope"})
			return
		}
		if in.Actor == "" {
			in.Actor = p.AgentID
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
	p, code, msg := n.labflowPrincipal(r)
	if code != 0 {
		writeJSON(w, code, map[string]string{"error": msg})
		return
	}
	sample, sampleCID, err := svc.Get(id)
	if err != nil {
		writeJSON(w, 404, map[string]string{"error": err.Error()})
		return
	}
	if !p.CanViewSample(sample) {
		writeJSON(w, 403, map[string]string{"error": "forbidden"})
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

func (n *NodoAlset) handleLabflowAuthToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	var req struct {
		AgentID       string   `json:"agent_id"`
		Roles         []string `json:"roles"`
		OrgID         string   `json:"org_id"`
		DurationHours int      `json:"duration_hours"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid json"})
		return
	}
	if req.AgentID == "" {
		req.AgentID = "lab-user"
	}
	if len(req.Roles) == 0 {
		req.Roles = []string{labflow.RoleTechnician}
	}
	if req.DurationHours <= 0 {
		req.DurationHours = 24
	}
	n.mu.Lock()
	if n.agentes == nil {
		n.agentes = make(map[string]*Agente)
	}
	if _, ok := n.agentes[req.AgentID]; !ok {
		n.agentes[req.AgentID] = &Agente{ID: req.AgentID, UltimaActual: 0, BalanceUTXO: 0}
	}
	n.mu.Unlock()
	token, err := n.generarTokenAlset(req.AgentID, labflow.NormalizeRoles(req.Roles), req.DurationHours)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, 201, map[string]interface{}{
		"token":      token.Token,
		"agent_id":   token.AgentID,
		"roles":      token.Roles,
		"expires_at": token.ExpiresAt,
		"org_id":     req.OrgID,
		"usage":      "Authorization: Bearer <token>",
		"note":       "Pass X-Lab-Org for org scope when calling LabFlow APIs",
	})
}
