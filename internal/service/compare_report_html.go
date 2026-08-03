package service

import (
	"bytes"
	"html/template"
	"time"
)

// CompareReportDocData is the view model for the printable Compare receipt —
// the migration analog of the restore-drill report: a self-contained document
// stating what was checked, what matched, and the verdict, suitable for
// print-to-PDF and for sharing with stakeholders before cut-over.
type CompareReportDocData struct {
	TenantName  string
	Source      string // "stripe" | "chargebee" | "revenuecat"
	Ready       bool
	GeneratedAt time.Time
	Report      CompareReport
}

var compareDocTmpl = template.Must(template.New("compare-doc").Parse(`<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<title>Migration Compare Report — {{.Source}}</title>
<style>
  body { font-family: -apple-system, Segoe UI, Helvetica, Arial, sans-serif; color: #1c1917; margin: 40px auto; max-width: 820px; padding: 0 24px; }
  h1 { font-size: 22px; margin-bottom: 4px; }
  .meta { color: #57534e; font-size: 13px; margin-bottom: 24px; }
  .verdict { border-radius: 8px; padding: 14px 18px; font-weight: 600; margin-bottom: 28px; }
  .ready { background: #ecfdf5; color: #047857; border: 1px solid #a7f3d0; }
  .notready { background: #fef2f2; color: #b91c1c; border: 1px solid #fecaca; }
  table { border-collapse: collapse; width: 100%; margin-bottom: 28px; font-size: 14px; }
  th { text-align: left; font-size: 11px; text-transform: uppercase; letter-spacing: 0.04em; color: #78716c; border-bottom: 2px solid #e7e5e4; padding: 8px 10px; }
  td { border-bottom: 1px solid #f5f5f4; padding: 8px 10px; }
  .num { text-align: right; font-variant-numeric: tabular-nums; }
  .mono { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 12px; }
  .section { font-size: 15px; font-weight: 600; margin: 24px 0 8px; }
  .method { color: #57534e; font-size: 12px; line-height: 1.6; border-top: 1px solid #e7e5e4; padding-top: 16px; margin-top: 36px; }
  @media print { body { margin: 0 auto; } }
</style>
</head>
<body>
  <h1>Migration Compare Report</h1>
  <div class="meta">
    {{.TenantName}} &middot; source: <strong>{{.Source}}</strong> &middot; generated {{.GeneratedAt.Format "Jan 2, 2006 15:04 MST"}}
  </div>

  {{if .Ready}}
  <div class="verdict ready">READY — every importable record matched; no fidelity or continuity issues. Safe to cut over.</div>
  {{else}}
  <div class="verdict notready">NOT READY — {{len .Report.Issues}} issue(s) require attention before cut-over.</div>
  {{end}}

  <div class="section">Coverage</div>
  <table>
    <tr><th>Record kind</th><th class="num">In export</th><th class="num">Matched</th><th class="num">Missing</th></tr>
    <tr><td>Customers</td><td class="num">{{.Report.Customers.Source}}</td><td class="num">{{.Report.Customers.Matched}}</td><td class="num">{{.Report.Customers.Missing}}</td></tr>
    <tr><td>Plans</td><td class="num">{{.Report.Plans.Source}}</td><td class="num">{{.Report.Plans.Matched}}</td><td class="num">{{.Report.Plans.Missing}}</td></tr>
    <tr><td>Subscriptions</td><td class="num">{{.Report.Subscriptions.Source}}</td><td class="num">{{.Report.Subscriptions.Matched}}</td><td class="num">{{.Report.Subscriptions.Missing}}</td></tr>
  </table>

  {{if .Report.Issues}}
  <div class="section">Issues</div>
  <table>
    <tr><th>Kind</th><th>Record</th><th>Field</th><th>Source value</th><th>Recurso value</th></tr>
    {{range .Report.Issues}}
    <tr>
      <td>{{.Kind}}</td>
      <td class="mono">{{.ExternalID}}</td>
      <td>{{.Field}}</td>
      <td class="mono">{{.Source}}</td>
      <td class="mono">{{.Recurso}}</td>
    </tr>
    {{end}}
  </table>
  {{end}}

  <div class="method">
    <strong>Method.</strong> This report is a read-only diff of the uploaded {{.Source}}
    export against live Recurso data at generation time. Three properties are
    checked per record: <em>coverage</em> (every importable source record exists in
    Recurso), <em>fidelity</em> (plan amount, currency and interval exact; customer
    identity intact), and <em>continuity</em> (a subscription whose period end
    drifted more than one hour from the source is flagged as a double-billing /
    billing-gap risk). READY means zero issues. Nothing was written by this check.
  </div>
</body>
</html>`))

// RenderCompareReportHTML renders the printable receipt.
func RenderCompareReportHTML(data CompareReportDocData) (string, error) {
	var buf bytes.Buffer
	if err := compareDocTmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
