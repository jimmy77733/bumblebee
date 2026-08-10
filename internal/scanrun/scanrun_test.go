package scanrun

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/perplexityai/bumblebee/internal/model"
)

func TestSpecifiedDirRootsRejectsDriveRoot(t *testing.T) {
	target := "/"
	if runtime.GOOS == "windows" {
		target = `C:\`
	}
	_, err := SpecifiedDirRoots(target)
	if err == nil {
		t.Fatal("expected drive root to be rejected")
	}
}

func TestSpecifiedDirRootsNormalizesFyneWindowsPath(t *testing.T) {
	dir := t.TempDir()
	fynePath := "/" + strings.ReplaceAll(dir, `\`, "/")
	if runtime.GOOS != "windows" {
		t.Skip("fyne windows path shape")
	}
	roots, err := SpecifiedDirRoots(fynePath)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Clean(roots[0].Path) != filepath.Clean(dir) {
		t.Fatalf("got %q want %q", roots[0].Path, dir)
	}
}

func TestSpecifiedDirRootsAcceptsTemp(t *testing.T) {
	dir := t.TempDir()
	roots, err := SpecifiedDirRoots(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(roots) != 1 || filepath.Clean(roots[0].Path) != filepath.Clean(dir) {
		t.Fatalf("roots = %+v", roots)
	}
}

func TestRunProducesHTMLReport(t *testing.T) {
	root := t.TempDir()
	proj := filepath.Join(root, "proj")
	if err := os.MkdirAll(proj, 0o755); err != nil {
		t.Fatal(err)
	}
	lock := `{
  "name": "demo",
  "lockfileVersion": 3,
  "packages": {
    "": {"name": "demo", "version": "1.0.0"},
    "node_modules/bumblebee-selftest-evil": {"version": "0.0.0"}
  }
}`
	if err := os.WriteFile(filepath.Join(proj, "package-lock.json"), []byte(lock), 0o644); err != nil {
		t.Fatal(err)
	}
	catalogDir := filepath.Join(root, "threat_intel")
	if err := os.MkdirAll(catalogDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cat := `{
  "schema_version": "0.2.0",
  "entries": [
    {"id": "selftest-evil", "name": "selftest", "ecosystem": "npm", "package": "bumblebee-selftest-evil", "versions": ["0.0.0"], "severity": "critical"}
  ]
}`
	if err := os.WriteFile(filepath.Join(catalogDir, "demo.json"), []byte(cat), 0o644); err != nil {
		t.Fatal(err)
	}
	r, err := SpecifiedDirRoots(proj)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("BUMBLEBEE_REPORT_DIR", t.TempDir())
	oc := Run(context.Background(), Options{
		Mode:       "指定目錄",
		Profile:    model.ProfileDeep,
		Roots:      r,
		CatalogDir: catalogDir,
	})
	if oc.Err != nil {
		t.Fatal(oc.Err)
	}
	if oc.Report == "" {
		t.Fatal("missing report path")
	}
	b, err := os.ReadFile(oc.Report)
	if err != nil {
		t.Fatal(err)
	}
	html := string(b)
	if !strings.Contains(html, "bumblebee-selftest-evil") {
		t.Fatalf("report missing finding: %s", html)
	}
	if !strings.Contains(html, "官方 GitHub：perplexityai/bumblebee / threat_intel") {
		t.Fatal("missing catalog source")
	}
	if !strings.Contains(html, catalogDir) {
		t.Fatal("missing catalog path")
	}
	if len(oc.Findings) == 0 {
		t.Fatal("expected at least one finding")
	}
}
