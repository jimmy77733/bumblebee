package htmlreport

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/perplexityai/bumblebee/internal/model"
)

type Data struct {
	Mode        string
	GeneratedAt string
	Status      string
	Duration    string
	Files       int
	Packages    int
	Findings    int
	Roots       []string
	Items       []Item
}

type Item struct {
	Severity    string
	Ecosystem   string
	Package     string
	Version     string
	CatalogName string
	CatalogID   string
	SourceFile  string
	Evidence    string
}

func FromScan(mode, status string, duration time.Duration, files, packages int, roots []string, findings []model.Finding) Data {
	items := make([]Item, 0, len(findings))
	for _, f := range findings {
		items = append(items, Item{
			Severity:    f.Severity,
			Ecosystem:   f.Ecosystem,
			Package:     f.PackageName,
			Version:     f.Version,
			CatalogName: f.CatalogName,
			CatalogID:   f.CatalogID,
			SourceFile:  f.SourceFile,
			Evidence:    f.Evidence,
		})
	}
	return Data{
		Mode:        mode,
		GeneratedAt: time.Now().Format("2006-01-02 15:04:05"),
		Status:      status,
		Duration:    duration.Round(time.Millisecond).String(),
		Files:       files,
		Packages:    packages,
		Findings:    len(findings),
		Roots:       roots,
		Items:       items,
	}
}

func WriteFile(path string, data Data) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return reportTmpl.Execute(f, data)
}

func DefaultPath() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil || base == "" {
		base = os.TempDir()
	}
	dir := filepath.Join(base, "Bumblebee", "reports")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	name := fmt.Sprintf("bumblebee-report-%s.html", time.Now().Format("20060102-150405"))
	return filepath.Join(dir, name), nil
}

func sevClass(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "critical":
		return "sev-critical"
	case "high":
		return "sev-high"
	case "medium":
		return "sev-medium"
	case "low":
		return "sev-low"
	default:
		return "sev-info"
	}
}

var reportTmpl = template.Must(template.New("report").Funcs(template.FuncMap{
	"sevClass": sevClass,
}).Parse(reportHTML))

const reportHTML = `<!DOCTYPE html>
<html lang="zh-Hant">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Bumblebee 掃描報告</title>
<style>
:root { color-scheme: light; }
body { font-family: "Segoe UI", "Noto Sans TC", sans-serif; margin: 0; background: #f4f1ea; color: #1c1917; }
header { background: #1c1917; color: #fafaf9; padding: 28px 32px 24px; }
header h1 { margin: 0 0 8px; font-size: 28px; font-weight: 650; }
header p { margin: 0; opacity: .82; max-width: 720px; line-height: 1.5; }
main { padding: 24px 32px 48px; }
.meta { display: grid; grid-template-columns: repeat(auto-fit, minmax(140px, 1fr)); gap: 12px; margin-bottom: 24px; }
.card { background: #fff; border: 1px solid #e7e5e4; border-radius: 10px; padding: 14px 16px; }
.card .k { font-size: 12px; color: #78716c; text-transform: uppercase; letter-spacing: .04em; }
.card .v { font-size: 22px; font-weight: 650; margin-top: 4px; }
.alert { background: #fff7ed; border: 1px solid #fdba74; color: #9a3412; padding: 12px 14px; border-radius: 8px; margin-bottom: 20px; }
.ok { background: #f0fdf4; border: 1px solid #86efac; color: #166534; padding: 12px 14px; border-radius: 8px; margin-bottom: 20px; }
table { width: 100%; border-collapse: collapse; background: #fff; border-radius: 10px; overflow: hidden; border: 1px solid #e7e5e4; }
th, td { text-align: left; padding: 10px 12px; vertical-align: top; font-size: 14px; }
th { background: #fafaf9; color: #57534e; font-weight: 600; border-bottom: 1px solid #e7e5e4; }
tr + tr td { border-top: 1px solid #f5f5f4; }
.sev { display: inline-block; padding: 2px 8px; border-radius: 999px; font-size: 12px; font-weight: 650; }
.sev-critical { background: #fee2e2; color: #991b1b; }
.sev-high { background: #ffedd5; color: #9a3412; }
.sev-medium { background: #fef9c3; color: #854d0e; }
.sev-low { background: #e0f2fe; color: #075985; }
.sev-info { background: #f5f5f4; color: #44403c; }
.path { font-family: ui-monospace, Consolas, monospace; font-size: 12px; word-break: break-all; color: #44403c; }
.roots { margin: 0 0 20px; padding-left: 18px; color: #57534e; }
footer { margin-top: 28px; color: #78716c; font-size: 13px; }
</style>
</head>
<body>
<header>
  <h1>Bumblebee 掃描報告</h1>
  <p>僅比對已知曝光清單中的精確 (ecosystem, 套件, 版本)。這不是 CVE 掃描，也不會讀取或執行原始碼。</p>
</header>
<main>
  <div class="meta">
    <div class="card"><div class="k">模式</div><div class="v">{{.Mode}}</div></div>
    <div class="card"><div class="k">狀態</div><div class="v">{{.Status}}</div></div>
    <div class="card"><div class="k">曝光項目</div><div class="v">{{.Findings}}</div></div>
    <div class="card"><div class="k">套件紀錄</div><div class="v">{{.Packages}}</div></div>
    <div class="card"><div class="k">檢查檔案</div><div class="v">{{.Files}}</div></div>
    <div class="card"><div class="k">耗時</div><div class="v">{{.Duration}}</div></div>
  </div>
  {{if .Findings}}
  <div class="alert">發現 {{.Findings}} 筆與已知曝光清單相符的套件。請依來源路徑檢查後處理。</div>
  {{else}}
  <div class="ok">這次掃描沒有比對到已知曝光清單中的套件版本。</div>
  {{end}}
  <h2>掃描範圍</h2>
  <ul class="roots">
    {{range .Roots}}<li class="path">{{.}}</li>{{else}}<li>（無）</li>{{end}}
  </ul>
  <h2>曝光比對結果</h2>
  {{if .Items}}
  <table>
    <thead>
      <tr>
        <th>嚴重度</th>
        <th>生態系</th>
        <th>套件</th>
        <th>版本</th>
        <th>清單項目</th>
        <th>來源</th>
      </tr>
    </thead>
    <tbody>
      {{range .Items}}
      <tr>
        <td><span class="sev {{sevClass .Severity}}">{{if .Severity}}{{.Severity}}{{else}}info{{end}}</span></td>
        <td>{{.Ecosystem}}</td>
        <td>{{.Package}}</td>
        <td>{{.Version}}</td>
        <td>{{.CatalogName}}<div class="path">{{.CatalogID}}</div></td>
        <td><div class="path">{{.SourceFile}}</div><div>{{.Evidence}}</div></td>
      </tr>
      {{end}}
    </tbody>
  </table>
  {{else}}
  <p>沒有曝光項目。</p>
  {{end}}
  <footer>產生時間：{{.GeneratedAt}}</footer>
</main>
</body>
</html>
`
