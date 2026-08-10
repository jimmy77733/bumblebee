package scanrun

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/perplexityai/bumblebee/internal/endpoint"
	"github.com/perplexityai/bumblebee/internal/exposure"
	"github.com/perplexityai/bumblebee/internal/htmlreport"
	"github.com/perplexityai/bumblebee/internal/model"
	"github.com/perplexityai/bumblebee/internal/output"
	"github.com/perplexityai/bumblebee/internal/roots"
	"github.com/perplexityai/bumblebee/internal/scanner"
)

type Options struct {
	Mode       string
	Profile    string
	Roots      []scanner.Root
	CatalogDir string
	OnProgress func(scanner.Progress)
}

type Outcome struct {
	Mode     string
	Status   string
	Result   scanner.Result
	Findings []model.Finding
	Roots    []scanner.Root
	Report   string
	Err      error
}

func ResolveCatalogDir() (string, error) {
	var candidates []string
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "threat_intel"))
	}
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(cwd, "threat_intel"))
	}
	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && info.IsDir() {
			return c, nil
		}
	}
	return "", fmt.Errorf("找不到 threat_intel 目錄（請與 Bumblebee.exe 放在同一層）")
}

func SpecifiedDirRoots(path string) ([]scanner.Root, error) {
	path = normalizeOSPath(path)
	if path == "" {
		return nil, fmt.Errorf("未選擇目錄")
	}
	if roots.IsDriveRoot(path) {
		return nil, fmt.Errorf("拒絕掃描整個磁碟根目錄 %q", path)
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("不是目錄：%s", path)
	}
	return []scanner.Root{{Path: path, Kind: roots.Classify(path, model.ProfileDeep)}}, nil
}

func SmartRoots() ([]scanner.Root, []string, error) {
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		home = strings.TrimSpace(os.Getenv("USERPROFILE"))
	}
	if home == "" {
		return nil, nil, fmt.Errorf("無法取得使用者目錄")
	}
	if roots.IsDriveRoot(home) {
		return nil, nil, fmt.Errorf("使用者目錄看起來像磁碟根目錄，已中止")
	}
	var out []scanner.Root
	var notes []string
	baseline, bnotes, berr := roots.Resolve(model.ProfileBaseline, nil, roots.Options{})
	if berr == nil {
		out = append(out, baseline...)
		notes = append(notes, bnotes...)
	}
	out = append(out, scanner.Root{Path: home, Kind: model.RootKindDeepHome})
	return dedupeRoots(out), notes, nil
}

func Run(ctx context.Context, opts Options) Outcome {
	oc := Outcome{Mode: opts.Mode, Roots: opts.Roots, Status: model.ScanStatusError}
	if len(opts.Roots) == 0 {
		oc.Err = fmt.Errorf("沒有可掃描的目錄")
		return oc
	}
	catalogDir := opts.CatalogDir
	if catalogDir == "" {
		var err error
		catalogDir, err = ResolveCatalogDir()
		if err != nil {
			oc.Err = err
			return oc
		}
	}
	catalog, err := exposure.Load(catalogDir, 64<<20)
	if err != nil {
		oc.Err = err
		return oc
	}

	var mu sync.Mutex
	var findings []model.Finding
	runID := newRunID()
	emitter := output.New(io.Discard, io.Discard, runID)
	scanStart := time.Now().UTC()
	base := model.Record{
		RecordType:     model.RecordTypePackage,
		SchemaVersion:  model.SchemaVersion,
		ScannerName:    model.ScannerName,
		ScannerVersion: scannerVersion(),
		RunID:          runID,
		ScanTime:       scanStart.Format(time.RFC3339Nano),
		Endpoint:       endpoint.Current(""),
		Profile:        opts.Profile,
	}
	res, runErr := scanner.Run(ctx, scanner.Config{
		Profile:      opts.Profile,
		Roots:        opts.Roots,
		Catalog:      catalog,
		FindingsOnly: false,
		MaxFileSize:  5 << 20,
		Concurrency:  4,
		BaseRecord:   base,
		Emitter:      emitter,
		OnProgress:   opts.OnProgress,
		OnFinding: func(f model.Finding) {
			mu.Lock()
			findings = append(findings, f)
			mu.Unlock()
		},
	})
	oc.Result = res
	oc.Findings = findings
	oc.Err = runErr
	oc.Status = model.ScanStatusComplete
	if runErr != nil && res.FilesConsidered > 0 {
		oc.Status = model.ScanStatusPartial
	} else if runErr != nil {
		oc.Status = model.ScanStatusError
	}

	rootPaths := make([]string, 0, len(opts.Roots))
	for _, r := range opts.Roots {
		rootPaths = append(rootPaths, r.Path)
	}
	reportPath, err := htmlreport.DefaultPath()
	if err != nil {
		oc.Err = err
		return oc
	}
	data := htmlreport.FromScan(opts.Mode, oc.Status, res.Duration, res.FilesConsidered, res.RecordsEmitted, rootPaths, findings)
	if err := htmlreport.WriteFile(reportPath, data); err != nil {
		oc.Err = err
		return oc
	}
	oc.Report = reportPath
	return oc
}

func normalizeOSPath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return p
	}
	if len(p) >= 3 && (p[0] == '/' || p[0] == '\\') && p[2] == ':' {
		p = p[1:]
	}
	return filepath.Clean(filepath.FromSlash(p))
}

func dedupeRoots(in []scanner.Root) []scanner.Root {
	seen := map[string]struct{}{}
	var out []scanner.Root
	for _, r := range in {
		key := filepath.Clean(r.Path)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, r)
	}
	return out
}

func newRunID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func scannerVersion() string {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return "0.1.1"
	}
	v := strings.TrimSpace(bi.Main.Version)
	if v == "" || v == "(devel)" {
		return "0.1.1"
	}
	return v
}
