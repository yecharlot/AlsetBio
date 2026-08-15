package labflow

import (
	"encoding/csv"
	"fmt"
	"io"
	"strings"
	"time"
)

// WriteSamplesCSV exports samples to CSV.
func WriteSamplesCSV(w io.Writer, samples []Sample) error {
	cw := csv.NewWriter(w)
	_ = cw.Write([]string{
		"id", "external_id", "type", "workflow_id", "status", "org_id", "client_id",
		"location", "owner", "created_at", "evidence_cid", "root_cid",
	})
	for _, s := range samples {
		_ = cw.Write([]string{
			s.ID, s.ExternalID, s.Type, s.WorkflowID, string(s.Status), s.OrgID, s.ClientID,
			s.CurrentLocation, s.CurrentOwner, s.CreatedAt.Format(time.RFC3339), s.EvidenceCID, s.RootCID,
		})
	}
	cw.Flush()
	return cw.Error()
}

// WriteEventsCSV exports custody events.
func WriteEventsCSV(w io.Writer, events []CustodyEvent) error {
	cw := csv.NewWriter(w)
	_ = cw.Write([]string{
		"id", "sample_id", "type", "actor", "timestamp", "previous_state", "new_state", "evidence_ref",
	})
	for _, e := range events {
		_ = cw.Write([]string{
			e.ID, e.SampleID, string(e.Type), e.Actor, e.Timestamp.Format(time.RFC3339),
			string(e.PrevStatus), string(e.NewStatus), e.EvidenceRef,
		})
	}
	cw.Flush()
	return cw.Error()
}

// BuildCustodyReportHTML returns a printable HTML report for one sample.
func BuildCustodyReportHTML(sample *Sample, events []CustodyEvent, verifyURL string) string {
	var b strings.Builder
	b.WriteString(`<!DOCTYPE html><html lang="es"><head><meta charset="utf-8"/>
<title>Custody Report ` + sample.ExternalID + `</title>
<style>
body{font-family:system-ui,sans-serif;max-width:720px;margin:2rem auto;color:#0f172a;line-height:1.45}
h1{font-size:1.35rem;margin:0 0 .25rem} .muted{color:#64748b;font-size:.9rem}
table{width:100%;border-collapse:collapse;margin-top:1.25rem;font-size:.88rem}
th,td{border:1px solid #e2e8f0;padding:.45rem .55rem;text-align:left}
th{background:#f8fafc} .ok{color:#059669;font-weight:700}
@media print{button{display:none}}
</style></head><body>
<button onclick="window.print()">Print / PDF</button>
<h1>Chain of Custody Report</h1>
<p class="muted">AlsetBio LabFlow · generated ` + time.Now().UTC().Format(time.RFC3339) + `</p>
<p><strong>Sample:</strong> ` + sample.ExternalID + `<br/>
<strong>Status:</strong> ` + string(sample.Status) + `<br/>
<strong>Workflow:</strong> ` + sample.WorkflowID + `<br/>
<strong>Org:</strong> ` + sample.OrgID + `<br/>
<strong>Evidence CID:</strong> <code>` + sample.EvidenceCID + `</code><br/>
<strong>Root CID:</strong> <code>` + sample.RootCID + `</code></p>`)
	if verifyURL != "" {
		b.WriteString(`<p><strong>Verify:</strong> ` + verifyURL + `</p>`)
	}
	b.WriteString(`<p class="ok">INTEGRITY: content-addressed custody events</p>
<table><thead><tr><th>When</th><th>Event</th><th>From</th><th>To</th><th>Actor</th></tr></thead><tbody>`)
	for _, e := range events {
		b.WriteString(fmt.Sprintf("<tr><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td></tr>",
			e.Timestamp.Format(time.RFC3339), e.Type, e.PrevStatus, e.NewStatus, e.Actor))
	}
	b.WriteString(`</tbody></table>
<p class="muted">Not a medical device. Operational laboratory workflow record only.</p>
</body></html>`)
	return b.String()
}
