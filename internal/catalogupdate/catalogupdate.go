package catalogupdate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	DefaultRepo     = "perplexityai/bumblebee"
	DefaultPath     = "threat_intel"
	revisionFile    = ".upstream-revision"
	pendingFileName = "catalog-update.json"
	userAgent       = "Bumblebee-CatalogCheck/0.1.1"
)

type Status struct {
	LocalSHA        string    `json:"local_sha"`
	RemoteSHA       string    `json:"remote_sha"`
	RemoteDate      string    `json:"remote_date,omitempty"`
	UpdateAvailable bool      `json:"update_available"`
	CheckedAt       time.Time `json:"checked_at"`
	Offline         bool      `json:"offline,omitempty"`
}

type Client struct {
	HTTP    *http.Client
	APIBase string
	Repo    string
	Path    string
}

func NewClient() *Client {
	return &Client{
		HTTP:    &http.Client{Timeout: 20 * time.Second},
		APIBase: "https://api.github.com",
		Repo:    DefaultRepo,
		Path:    DefaultPath,
	}
}

func RevisionPath(catalogDir string) string {
	return filepath.Join(catalogDir, revisionFile)
}

func ReadLocalSHA(catalogDir string) string {
	b, err := os.ReadFile(RevisionPath(catalogDir))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func SourceLabel() string {
	return DefaultRepo + " / " + DefaultPath
}

func LocalUpdatedAt(catalogDir string) string {
	var latest time.Time
	check := func(p string) {
		info, err := os.Stat(p)
		if err != nil {
			return
		}
		if info.ModTime().After(latest) {
			latest = info.ModTime()
		}
	}
	check(catalogDir)
	check(RevisionPath(catalogDir))
	if matches, err := filepath.Glob(filepath.Join(catalogDir, "*.json")); err == nil {
		for _, p := range matches {
			check(p)
		}
	}
	if latest.IsZero() {
		return "未知"
	}
	return latest.Local().Format("2006-01-02 15:04:05")
}

func WriteLocalSHA(catalogDir string, sha string) error {
	if err := os.MkdirAll(catalogDir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(RevisionPath(catalogDir), []byte(strings.TrimSpace(sha)+"\n"), 0o644)
}

func StateDir() string {
	base, err := os.UserCacheDir()
	if err != nil || base == "" {
		base = os.TempDir()
	}
	return filepath.Join(base, "Bumblebee")
}

func PendingPath() string {
	return filepath.Join(StateDir(), pendingFileName)
}

func WritePending(st Status) error {
	if err := os.MkdirAll(StateDir(), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(PendingPath(), b, 0o644)
}

func ReadPending() (Status, bool) {
	b, err := os.ReadFile(PendingPath())
	if err != nil {
		return Status{}, false
	}
	var st Status
	if json.Unmarshal(b, &st) != nil {
		return Status{}, false
	}
	return st, st.UpdateAvailable && st.RemoteSHA != ""
}

func ClearPending() {
	_ = os.Remove(PendingPath())
}

func (c *Client) Check(ctx context.Context, catalogDir string) (Status, error) {
	if c == nil {
		c = NewClient()
	}
	st := Status{LocalSHA: ReadLocalSHA(catalogDir), CheckedAt: time.Now().UTC()}
	sha, date, err := c.latestCommit(ctx)
	if err != nil {
		st.Offline = true
		return st, err
	}
	st.RemoteSHA = sha
	st.RemoteDate = date
	if st.LocalSHA == "" {
		if err := WriteLocalSHA(catalogDir, sha); err != nil {
			return st, err
		}
		st.LocalSHA = sha
		st.UpdateAvailable = false
		return st, nil
	}
	st.UpdateAvailable = !strings.EqualFold(st.LocalSHA, st.RemoteSHA)
	return st, nil
}

func (c *Client) Apply(ctx context.Context, catalogDir string, sha string) error {
	if c == nil {
		c = NewClient()
	}
	sha = strings.TrimSpace(sha)
	if sha == "" {
		return errors.New("missing upstream revision")
	}
	entries, err := c.listJSON(ctx, sha)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return errors.New("upstream threat_intel has no json catalogs")
	}
	tmp, err := os.MkdirTemp(filepath.Dir(catalogDir), "threat_intel-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	for _, e := range entries {
		body, err := c.get(ctx, e.DownloadURL)
		if err != nil {
			return fmt.Errorf("download %s: %w", e.Name, err)
		}
		if err := os.WriteFile(filepath.Join(tmp, e.Name), body, 0o644); err != nil {
			return err
		}
	}
	oldJSON, _ := filepath.Glob(filepath.Join(catalogDir, "*.json"))
	for _, p := range oldJSON {
		_ = os.Remove(p)
	}
	if err := os.MkdirAll(catalogDir, 0o755); err != nil {
		return err
	}
	for _, e := range entries {
		src := filepath.Join(tmp, e.Name)
		dst := filepath.Join(catalogDir, e.Name)
		if err := os.Rename(src, dst); err != nil {
			data, rerr := os.ReadFile(src)
			if rerr != nil {
				return err
			}
			if werr := os.WriteFile(dst, data, 0o644); werr != nil {
				return werr
			}
		}
	}
	if err := WriteLocalSHA(catalogDir, sha); err != nil {
		return err
	}
	ClearPending()
	return nil
}

func RunSilentCheck(ctx context.Context, catalogDir string) error {
	st, err := NewClient().Check(ctx, catalogDir)
	if err != nil {
		return nil
	}
	if st.UpdateAvailable {
		return WritePending(st)
	}
	ClearPending()
	return nil
}

type ghCommit struct {
	SHA    string `json:"sha"`
	Commit struct {
		Committer struct {
			Date string `json:"date"`
		} `json:"committer"`
	} `json:"commit"`
}

type ghContent struct {
	Name        string `json:"name"`
	DownloadURL string `json:"download_url"`
	Type        string `json:"type"`
}

func (c *Client) latestCommit(ctx context.Context) (string, string, error) {
	url := fmt.Sprintf("%s/repos/%s/commits?path=%s&per_page=1", strings.TrimRight(c.APIBase, "/"), c.Repo, c.Path)
	b, err := c.get(ctx, url)
	if err != nil {
		return "", "", err
	}
	var commits []ghCommit
	if err := json.Unmarshal(b, &commits); err != nil {
		return "", "", err
	}
	if len(commits) == 0 || commits[0].SHA == "" {
		return "", "", errors.New("no upstream commits for threat_intel")
	}
	return commits[0].SHA, commits[0].Commit.Committer.Date, nil
}

func (c *Client) listJSON(ctx context.Context, sha string) ([]ghContent, error) {
	url := fmt.Sprintf("%s/repos/%s/contents/%s?ref=%s", strings.TrimRight(c.APIBase, "/"), c.Repo, c.Path, sha)
	b, err := c.get(ctx, url)
	if err != nil {
		return nil, err
	}
	var entries []ghContent
	if err := json.Unmarshal(b, &entries); err != nil {
		return nil, err
	}
	var out []ghContent
	for _, e := range entries {
		if e.Type != "file" || !strings.HasSuffix(strings.ToLower(e.Name), ".json") || e.DownloadURL == "" {
			continue
		}
		out = append(out, e)
	}
	return out, nil
}

func (c *Client) get(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/vnd.github+json")
	res, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 32<<20))
	if err != nil {
		return nil, err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("github api %s: %s", res.Status, strings.TrimSpace(string(body)))
	}
	return body, nil
}
