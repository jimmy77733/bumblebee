package catalogupdate

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestCheckAndApply(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "old.json"), []byte(`{"schema_version":"0.2.0","entries":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := WriteLocalSHA(dir, "aaa111"); err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/repos/perplexityai/bumblebee/commits", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{"sha": "bbb222", "commit": map[string]any{"committer": map[string]any{"date": "2026-08-01T00:00:00Z"}}},
		})
	})
	mux.HandleFunc("/repos/perplexityai/bumblebee/contents/threat_intel", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{"name": "new.json", "type": "file", "download_url": "http://" + r.Host + "/dl/new.json"},
			{"name": "README.md", "type": "file", "download_url": "http://" + r.Host + "/dl/README.md"},
		})
	})
	mux.HandleFunc("/dl/new.json", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"schema_version":"0.2.0","entries":[{"id":"x","ecosystem":"npm","package":"demo","versions":["1.0.0"]}]}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := &Client{HTTP: srv.Client(), APIBase: srv.URL, Repo: DefaultRepo, Path: DefaultPath}
	st, err := c.Check(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if !st.UpdateAvailable || st.RemoteSHA != "bbb222" {
		t.Fatalf("status = %+v", st)
	}
	if err := c.Apply(context.Background(), dir, st.RemoteSHA); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "old.json")); !os.IsNotExist(err) {
		t.Fatal("old catalog should be removed")
	}
	if _, err := os.Stat(filepath.Join(dir, "new.json")); err != nil {
		t.Fatal(err)
	}
	if got := ReadLocalSHA(dir); got != "bbb222" {
		t.Fatalf("local sha = %q", got)
	}
}

func TestCheckBootstrapsMissingRevision(t *testing.T) {
	dir := t.TempDir()
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/perplexityai/bumblebee/commits", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{"sha": "ccc333", "commit": map[string]any{"committer": map[string]any{"date": "2026-08-01T00:00:00Z"}}},
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	c := &Client{HTTP: srv.Client(), APIBase: srv.URL, Repo: DefaultRepo, Path: DefaultPath}
	st, err := c.Check(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if st.UpdateAvailable || st.LocalSHA != "ccc333" {
		t.Fatalf("status = %+v", st)
	}
}
