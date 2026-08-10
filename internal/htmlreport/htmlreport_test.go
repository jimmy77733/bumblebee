package htmlreport

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/perplexityai/bumblebee/internal/model"
)

func TestWriteFileEscapesAndIncludesFinding(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "report.html")
	data := FromScan("指定目錄", "complete", 12*time.Millisecond, 3, 1, []string{`C:\proj`}, []model.Finding{
		{
			Severity:    "critical",
			Ecosystem:   "npm",
			PackageName: "<evil>",
			Version:     "1.0.0",
			CatalogName: "test catalog",
			CatalogID:   "test-id",
			SourceFile:  `C:\proj\package-lock.json`,
			Evidence:    "exact name+version match",
		},
	})
	if err := WriteFile(path, data); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	html := string(b)
	if strings.Contains(html, "<evil>") {
		t.Fatal("package name was not HTML-escaped")
	}
	if !strings.Contains(html, "&lt;evil&gt;") {
		t.Fatal("expected escaped package name")
	}
	if !strings.Contains(html, "critical") || !strings.Contains(html, "指定目錄") {
		t.Fatal("missing expected report content")
	}
	if !strings.Contains(html, "status-complete") {
		t.Fatal("complete status missing semantic class")
	}
	if !strings.Contains(html, "tone-alert") {
		t.Fatal("nonzero findings should use alert tone")
	}
	if strings.Contains(html, "https://") || strings.Contains(html, "http://") {
		t.Fatal("report must stay offline without external URLs")
	}
	if !strings.Contains(html, "theme-toggle") || !strings.Contains(html, "淺色主題") {
		t.Fatal("missing theme toggle")
	}
	if strings.Contains(html, "radial-gradient") || strings.Contains(html, "linear-gradient(180deg") {
		t.Fatal("cards should use flat solid backgrounds")
	}
	if !strings.Contains(html, "資料庫來源") || !strings.Contains(html, "清單更新時間") {
		t.Fatal("missing catalog footer")
	}
}

func TestZeroFindingsUsesOkTone(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ok.html")
	data := FromScan("智能掃描", model.ScanStatusComplete, time.Millisecond, 10, 4, nil, nil)
	if err := WriteFile(path, data); err != nil {
		t.Fatal(err)
	}
	html := string(mustRead(t, path))
	if !strings.Contains(html, "status-complete") || !strings.Contains(html, "tone-ok") {
		t.Fatal("zero findings complete report should be green")
	}
	if !strings.Contains(html, "banner-ok") {
		t.Fatal("missing success banner")
	}
}

func TestDefaultPathHonorsOverride(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("BUMBLEBEE_REPORT_DIR", dir)
	p, err := DefaultPath()
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(p) != dir {
		t.Fatalf("got %q want dir %q", p, dir)
	}
	if !strings.HasSuffix(p, ".html") {
		t.Fatalf("expected html file: %s", p)
	}
}

func TestStatusClasses(t *testing.T) {
	if statusClass(model.ScanStatusComplete) != "status-complete" {
		t.Fatal(statusClass(model.ScanStatusComplete))
	}
	if statusClass(model.ScanStatusPartial) != "status-partial" {
		t.Fatal(statusClass(model.ScanStatusPartial))
	}
	if statusClass(model.ScanStatusError) != "status-error" {
		t.Fatal(statusClass(model.ScanStatusError))
	}
	if findingsTone(0) != "tone-ok" || findingsTone(2) != "tone-alert" {
		t.Fatal("findings tone mismatch")
	}
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
