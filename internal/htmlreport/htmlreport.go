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
	Mode           string
	GeneratedAt    string
	Status         string
	Duration       string
	Files          int
	Packages       int
	Findings       int
	Roots          []string
	Items          []Item
	CatalogSource  string
	CatalogPath    string
	CatalogSHA     string
	CatalogUpdated string
}

type Item struct {
	Severity    string
	Ecosystem   string
	Package     string
	CatalogName string
	CatalogID   string
	SourceFile  string
	Evidence    string
	Version     string
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
	dir := strings.TrimSpace(os.Getenv("BUMBLEBEE_REPORT_DIR"))
	if dir == "" {
		d, err := downloadsDir()
		if err != nil || strings.TrimSpace(d) == "" {
			home, herr := os.UserHomeDir()
			if herr != nil {
				if err != nil {
					return "", err
				}
				return "", herr
			}
			d = filepath.Join(home, "Downloads")
		}
		dir = d
	}
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

func statusClass(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case model.ScanStatusComplete:
		return "status-complete"
	case model.ScanStatusPartial:
		return "status-partial"
	case model.ScanStatusError:
		return "status-error"
	default:
		return "status-unknown"
	}
}

func statusLabel(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case model.ScanStatusComplete:
		return "完成 complete"
	case model.ScanStatusPartial:
		return "部分完成 partial"
	case model.ScanStatusError:
		return "失敗 error"
	default:
		return s
	}
}

func findingsTone(n int) string {
	if n > 0 {
		return "tone-alert"
	}
	return "tone-ok"
}

func catalogSHA(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "本機（無版本紀錄）"
	}
	if len(s) > 7 {
		return s[:7]
	}
	return s
}

var reportTmpl = template.Must(template.New("report").Funcs(template.FuncMap{
	"sevClass":     sevClass,
	"statusClass":  statusClass,
	"statusLabel":  statusLabel,
	"findingsTone": findingsTone,
	"catalogSHA":   catalogSHA,
}).Parse(reportHTML))

const reportHTML = `<!DOCTYPE html>
<html lang="zh-Hant" data-theme="dark">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Bumblebee 掃描報告</title>
<style>
:root, html[data-theme="dark"] {
  color-scheme: dark;
  --bg: #12100c;
  --card: #1c1916;
  --fg: #f5f0e8;
  --muted: #a8a29e;
  --line: #3f3a34;
  --hover: #292524;
  --btn: #292524;
  --green: #4ade80;
  --green-bg: #14532d;
  --red: #fb7185;
  --orange: #fbbf24;
  --orange-bg: #713f12;
}
html[data-theme="light"] {
  color-scheme: light;
  --bg: #faf7f2;
  --card: #ffffff;
  --fg: #1c1917;
  --muted: #78716c;
  --line: #e7e0d5;
  --hover: #f5f0e8;
  --btn: #eeead2;
  --green: #16803c;
  --green-bg: #dcfce7;
  --red: #dc2626;
  --orange: #b45309;
  --orange-bg: #fef3c7;
}
* { box-sizing: border-box; }
body {
  font-family: "Segoe UI", "Noto Sans TC", sans-serif;
  margin: 0;
  background: var(--bg);
  color: var(--fg);
}
header { padding: 36px 32px 20px; }
.header-row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}
h1 {
  margin: 0 0 10px;
  font-size: 34px;
  font-weight: 700;
  letter-spacing: -.02em;
}
h1 span.accent { color: #d97706; }
.theme-btn {
  flex: 0 0 auto;
  border: 1px solid var(--line);
  background: var(--btn);
  color: var(--fg);
  border-radius: 10px;
  padding: 8px 14px;
  font-size: 14px;
  cursor: pointer;
}
.theme-btn:hover { background: var(--hover); }
header p { margin: 0; max-width: 740px; line-height: 1.55; color: var(--muted); }
main { padding: 8px 32px 48px; }
.meta { display: grid; grid-template-columns: repeat(auto-fit, minmax(150px, 1fr)); gap: 14px; margin-bottom: 22px; }
.card {
  background: var(--card);
  border: 1px solid var(--line);
  border-radius: 10px;
  padding: 16px 16px 14px;
}
.k { font-size: 12px; color: var(--muted); letter-spacing: .08em; text-transform: uppercase; }
.v { font-size: 26px; font-weight: 700; margin-top: 8px; font-variant-numeric: tabular-nums; }
.v.tone-ok { color: var(--green); }
.v.tone-alert { color: var(--red); }
.status-badge {
  margin-top: 10px;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 6px 12px 6px 10px;
  border-radius: 999px;
  font-weight: 700;
  font-size: 14px;
  border: 1px solid transparent;
}
.status-badge .dot { width: 8px; height: 8px; border-radius: 50%; background: currentColor; }
.status-complete { background: var(--green-bg); color: var(--green); }
.status-partial { background: var(--orange-bg); color: var(--orange); }
.status-error { background: #7f1d1d; color: #fecaca; }
html[data-theme="light"] .status-error { background: #fee2e2; color: #b91c1c; }
.status-unknown { background: var(--hover); color: var(--muted); }
.banner {
  border-radius: 10px;
  padding: 14px 16px;
  margin-bottom: 22px;
  border: 1px solid var(--line);
  background: var(--card);
}
.banner.banner-ok { color: var(--green); }
.banner.banner-alert { color: var(--orange); }
h2 { font-size: 18px; margin: 22px 0 10px; }
.roots { margin: 0 0 20px; padding-left: 18px; color: var(--muted); }
.table-wrap {
  border: 1px solid var(--line);
  border-radius: 10px;
  overflow: auto;
  background: var(--card);
}
table { width: 100%; border-collapse: collapse; }
th, td { text-align: left; padding: 12px 14px; vertical-align: top; font-size: 14px; }
th { color: var(--muted); font-weight: 600; border-bottom: 1px solid var(--line); background: var(--bg); }
tbody tr:hover { background: var(--hover); }
tr + tr td { border-top: 1px solid var(--line); }
.sev { display: inline-block; padding: 3px 9px; border-radius: 999px; font-size: 12px; font-weight: 700; }
.sev-critical { background: #7f1d1d; color: #fecaca; }
.sev-high { background: #7c2d12; color: #fed7aa; }
.sev-medium { background: #713f12; color: #fde68a; }
.sev-low { background: #0c4a6e; color: #bae6fd; }
.sev-info { background: var(--hover); color: var(--muted); }
html[data-theme="light"] .sev-critical { background: #fee2e2; color: #991b1b; }
html[data-theme="light"] .sev-high { background: #ffedd5; color: #9a3412; }
html[data-theme="light"] .sev-medium { background: #fef3c7; color: #92400e; }
html[data-theme="light"] .sev-low { background: #e0f2fe; color: #075985; }
.path { font-family: ui-monospace, Consolas, monospace; font-size: 12px; word-break: break-all; color: var(--muted); }
footer {
  margin-top: 28px;
  color: var(--muted);
  font-size: 13px;
  line-height: 1.7;
  border-top: 1px solid var(--line);
  padding-top: 16px;
}
footer .path { margin-top: 2px; }
</style>
</head>
<body>
<header>
  <div class="header-row">
    <h1>Bumblebee <span class="accent">掃描報告</span></h1>
    <button type="button" class="theme-btn" id="theme-toggle">淺色主題</button>
  </div>
  <p>僅比對已知曝光清單中的精確 (ecosystem, 套件, 版本)。這不是 CVE 掃描，也不會讀取或執行原始碼。</p>
</header>
<main>
  <div class="meta">
    <div class="card"><div class="k">模式</div><div class="v">{{.Mode}}</div></div>
    <div class="card">
      <div class="k">狀態</div>
      <div class="status-badge {{statusClass .Status}}"><span class="dot"></span><span>{{statusLabel .Status}}</span></div>
    </div>
    <div class="card"><div class="k">曝光項目</div><div class="v {{findingsTone .Findings}}" data-count="{{.Findings}}">0</div></div>
    <div class="card"><div class="k">套件紀錄</div><div class="v" data-count="{{.Packages}}">0</div></div>
    <div class="card"><div class="k">檢查檔案</div><div class="v" data-count="{{.Files}}">0</div></div>
    <div class="card"><div class="k">耗時</div><div class="v">{{.Duration}}</div></div>
  </div>
  {{if .Findings}}
  <div class="banner banner-alert">發現 {{.Findings}} 筆與已知曝光清單相符的套件。請依來源路徑檢查後處理。</div>
  {{else}}
  <div class="banner banner-ok">這次掃描沒有比對到已知曝光清單中的套件版本。</div>
  {{end}}
  <h2>掃描範圍</h2>
  <ul class="roots">
    {{range .Roots}}<li class="path">{{.}}</li>{{else}}<li>（無）</li>{{end}}
  </ul>
  <h2>曝光比對結果</h2>
  {{if .Items}}
  <div class="table-wrap">
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
  </div>
  {{else}}
  <p>沒有曝光項目。</p>
  {{end}}
  <footer>
    <div>產生時間：{{.GeneratedAt}}</div>
    <div>資料庫來源：{{if .CatalogSource}}{{.CatalogSource}}{{else}}本機曝光清單{{end}}</div>
    {{if .CatalogPath}}<div>本機清單路徑：<span class="path">{{.CatalogPath}}</span></div>{{end}}
    <div>清單版本：{{catalogSHA .CatalogSHA}}</div>
    <div>清單更新時間：{{if .CatalogUpdated}}{{.CatalogUpdated}}{{else}}未知{{end}}</div>
  </footer>
</main>
<script>
(function () {
  var root = document.documentElement;
  var btn = document.getElementById("theme-toggle");
  var key = "bumblebee-report-theme";
  function label(theme) { return theme === "light" ? "深色主題" : "淺色主題"; }
  function apply(theme) {
    if (theme !== "light") theme = "dark";
    root.setAttribute("data-theme", theme);
    if (btn) btn.textContent = label(theme);
  }
  try { apply(localStorage.getItem(key) || "dark"); } catch (e) { apply("dark"); }
  if (btn) {
    btn.addEventListener("click", function () {
      var next = root.getAttribute("data-theme") === "light" ? "dark" : "light";
      try { localStorage.setItem(key, next); } catch (e) {}
      apply(next);
    });
  }
  var nodes = document.querySelectorAll("[data-count]");
  nodes.forEach(function (n) {
    var end = parseInt(n.getAttribute("data-count"), 10) || 0;
    var t0 = performance.now();
    var dur = 900;
    function frame(t) {
      var p = Math.min(1, (t - t0) / dur);
      var eased = 1 - Math.pow(1 - p, 3);
      n.textContent = String(Math.round(end * eased));
      if (p < 1) requestAnimationFrame(frame);
    }
    requestAnimationFrame(frame);
  });
})();
</script>
</body>
</html>
`
