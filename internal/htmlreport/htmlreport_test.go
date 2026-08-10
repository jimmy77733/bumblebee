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
}
