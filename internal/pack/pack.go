// Package pack downloads vibe-time's sound pack at runtime: a manifest
// lists every sound with its size and SHA-256, files land in the user's
// config directory once fetched, and callers either wait on a specific
// sound (Ensure) or let the background prefetch quietly fill the cache.
//
// Sources are tried in order — gitee first (reachable from mainland
// China without a mirror), GitHub raw as the fallback — and every
// download is verified against the manifest's hash before it is kept.
// Callbacks always run on the app's goroutine through cx.Post, so
// handlers may touch signals and the audio engine freely.
package pack

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/nsgo-dev/nsgo/core"

	"vibe-time/internal/catalog"
)

// Sources are base URLs serving the pack repository: manifest.json,
// labels.json and the files' Paths live under each. gitee's raw endpoint
// first (no mirror needed in mainland China), GitHub raw behind it.
// VIBE_PACK_SOURCES (comma-separated) overrides the list, for mirrors and
// local testing.
var Sources = defaultSources()

func defaultSources() []string {
	var out []string
	for _, part := range strings.Split(os.Getenv("VIBE_PACK_SOURCES"), ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	if len(out) > 0 {
		return out
	}
	return []string{
		"https://gitee.com/limengwe/vibe-time-sound/raw/master",
		"https://raw.githubusercontent.com/limengwei/vibe-time-sound/master",
	}
}

// File is one sound in the manifest.
type File struct {
	ID    string `json:"id"` // "category/file-base", the app's sound ID
	Path  string `json:"path"`
	Size  int64  `json:"size"`
	SHA56 string `json:"sha256"`
}

// Manifest is the pack's index.
type Manifest struct {
	Version int    `json:"version"`
	Files   []File `json:"files"`
	// Labels and Categories come from the repository's labels.json (fetched
	// alongside the manifest, cached with it): base name → Chinese label,
	// category ID → title and emoji. Both may be empty when it is missing.
	Labels     map[string]string                  `json:"labels,omitempty"`
	Categories map[string]catalog.CategoryMeta    `json:"categories,omitempty"`
}

// ByID returns the manifest entry for a sound ID.
func (m *Manifest) ByID(id string) (File, bool) {
	for _, f := range m.Files {
		if f.ID == id {
			return f, true
		}
	}
	return File{}, false
}

// Pack is the runtime downloader and local cache.
type Pack struct {
	cx  *core.Scope
	dir string // <user config>/vibe-time/pack

	mu      sync.Mutex
	man     *Manifest
	pending map[string][]func(bool) // waiters for a download in flight
}

// Open prepares the cache directory and loads the last fetched manifest,
// so sounds already downloaded work offline before (or without) a refresh.
func Open(cx *core.Scope) (*Pack, error) {
	cfg, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	p := &Pack{
		cx:      cx,
		dir:     filepath.Join(cfg, "vibe-time", "pack"),
		pending: map[string][]func(bool){},
	}
	if err := os.MkdirAll(p.soundsDir(), 0o755); err != nil {
		return nil, err
	}
	if m, err := p.loadCachedManifest(); err == nil {
		p.man = m
	}
	return p, nil
}

// Start fetches the manifest (falling back to the cached copy offline),
// reports it on the app's goroutine, prunes files the pack dropped, then
// prefetches whatever is missing in the background.
func (p *Pack) Start(onManifest func(*Manifest)) {
	go func() {
		ctx := p.cx.Context()
		m, err := p.fetchManifest(ctx)
		if err != nil {
			log.Printf("[pack] manifest 拉取失败（使用本地缓存）: %v", err)
			p.mu.Lock()
			m = p.man
			p.mu.Unlock()
			if m == nil {
				return
			}
		} else {
			data, _ := json.MarshalIndent(m, "", "  ")
			if err := os.WriteFile(p.manifestPath(), data, 0o644); err != nil {
				log.Printf("[pack] manifest 缓存写入失败: %v", err)
			}
			p.mu.Lock()
			p.man = m
			p.mu.Unlock()
			p.prune(m)
		}
		mm := m
		p.cx.Post(func() { onManifest(mm) })
		p.prefetch(ctx, m)
	}()
}

// Cached reports whether a sound's bytes are on disk.
func (p *Pack) Cached(id string) bool {
	p.mu.Lock()
	path := p.pathOfLocked(id)
	p.mu.Unlock()
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}

// Pending reports whether a download of the sound is in flight.
func (p *Pack) Pending(id string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	_, busy := p.pending[id]
	return busy
}

// Read returns a cached sound's bytes, or an error when not downloaded.
func (p *Pack) Read(id string) ([]byte, error) {
	p.mu.Lock()
	path := p.pathOfLocked(id)
	p.mu.Unlock()
	if path == "" {
		return nil, fmt.Errorf("pack: %s not in manifest", id)
	}
	return os.ReadFile(path)
}

// Ensure downloads a sound if needed and reports ok on the app's
// goroutine; concurrent calls for the same sound share one download.
func (p *Pack) Ensure(id string, done func(ok bool)) {
	if p.Cached(id) {
		ok := true
		p.cx.Post(func() { done(ok) })
		return
	}
	p.mu.Lock()
	entry, inManifest := File{}, false
	if p.man != nil {
		entry, inManifest = p.man.ByID(id)
	}
	if !inManifest {
		p.mu.Unlock()
		p.cx.Post(func() { done(false) })
		return
	}
	if waiters, busy := p.pending[id]; busy {
		p.pending[id] = append(waiters, done)
		p.mu.Unlock()
		return
	}
	p.pending[id] = []func(bool){done}
	p.mu.Unlock()

	go func() {
		err := p.download(p.cx.Context(), entry)
		ok := err == nil
		if err != nil {
			log.Printf("[pack] 下载 %s 失败: %v", id, err)
		}
		p.mu.Lock()
		waiters := p.pending[id]
		delete(p.pending, id)
		p.mu.Unlock()
		p.cx.Post(func() {
			for _, w := range waiters {
				w(ok)
			}
		})
	}()
}

// soundsDir is where the pack's files land, mirroring the repo layout.
func (p *Pack) soundsDir() string { return filepath.Join(p.dir, "sounds") }

// manifestPath is the cached copy of the last fetched manifest.
func (p *Pack) manifestPath() string { return filepath.Join(p.dir, "manifest.json") }

// pathOfLocked maps a sound ID to its cache path, "" when unknown.
func (p *Pack) pathOfLocked(id string) string {
	if p.man == nil {
		return ""
	}
	f, ok := p.man.ByID(id)
	if !ok {
		return ""
	}
	return filepath.Join(p.dir, filepath.FromSlash(f.Path))
}

// fetchManifest pulls manifest.json — and labels.json when it answers —
// from the first source that serves them.
func (p *Pack) fetchManifest(ctx context.Context) (*Manifest, error) {
	var lastErr error
	for _, base := range Sources {
		cctx, cancel := context.WithTimeout(ctx, 15*time.Second)
		data, err := p.fetch(cctx, "manifest.json")
		if err != nil {
			cancel()
			lastErr = err
			continue
		}
		var m Manifest
		if err := json.Unmarshal(data, &m); err != nil {
			cancel()
			lastErr = fmt.Errorf("%s: %w", base, err)
			continue
		}
		if len(m.Files) == 0 {
			cancel()
			lastErr = fmt.Errorf("%s: empty manifest", base)
			continue
		}
		if ldata, err := p.fetch(cctx, "labels.json"); err == nil {
			var lf struct {
				Categories map[string]catalog.CategoryMeta `json:"categories"`
				Labels     map[string]string               `json:"labels"`
			}
			if err := json.Unmarshal(ldata, &lf); err == nil {
				m.Categories, m.Labels = lf.Categories, lf.Labels
			}
		}
		cancel()
		return &m, nil
	}
	if lastErr == nil {
		lastErr = errors.New("pack: no sources")
	}
	return nil, lastErr
}

// prefetch quietly downloads every missing file, one at a time; a file
// the user's Ensure already fetched is skipped by the time we reach it.
func (p *Pack) prefetch(ctx context.Context, m *Manifest) {
	for _, f := range m.Files {
		select {
		case <-ctx.Done():
			return
		default:
		}
		if p.Cached(f.ID) || p.Pending(f.ID) {
			continue
		}
		if err := p.download(ctx, f); err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("[pack] 预取 %s 失败: %v", f.ID, err)
		}
	}
	log.Printf("[pack] 声音包预取完成（version %d，%d 个文件）", m.Version, len(m.Files))
}

// fetch GETs rel (relative to a source) from the first source that
// answers, returning its bytes.
func (p *Pack) fetch(ctx context.Context, rel string) ([]byte, error) {
	var lastErr error
	for _, base := range Sources {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(base, "/")+"/"+rel, nil)
		if err != nil {
			return nil, err
		}
		resp, err := p.client().Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		data, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = err
			continue
		}
		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("%s: HTTP %d", base, resp.StatusCode)
			continue
		}
		return data, nil
	}
	return nil, lastErr
}

// download fetches one file, verifies its hash and moves it into place.
func (p *Pack) download(ctx context.Context, f File) error {
	cctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	data, err := p.fetch(cctx, f.Path)
	if err != nil {
		return err
	}
	if int64(len(data)) != f.Size {
		return fmt.Errorf("size %d, want %d", len(data), f.Size)
	}
	sum := sha256.Sum256(data)
	if hex.EncodeToString(sum[:]) != f.SHA56 {
		return errors.New("sha256 mismatch")
	}
	dst := filepath.Join(p.dir, filepath.FromSlash(f.Path))
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	tmp := dst + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, dst)
}

// prune removes cached files (and manifest entries' leftovers) the pack
// no longer lists.
func (p *Pack) prune(m *Manifest) {
	keep := map[string]bool{}
	for _, f := range m.Files {
		keep[f.Path] = true
	}
	root := p.soundsDir()
	var paths []string
	filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && path != p.manifestPath() {
			paths = append(paths, path)
		}
		return nil
	})
	n := 0
	sort.Sort(sort.Reverse(sort.StringSlice(paths)))
	for _, path := range paths {
		rel, _ := filepath.Rel(p.dir, path)
		if keep[filepath.ToSlash(rel)] || strings.HasSuffix(path, ".tmp") {
			if strings.HasSuffix(path, ".tmp") {
				os.Remove(path)
				n++
			}
			continue
		}
		if os.Remove(path) == nil {
			n++
		}
	}
	if n > 0 {
		log.Printf("[pack] 清理 %d 个已下架的缓存文件", n)
	}
}

// loadCachedManifest reads the manifest saved by an earlier run.
func (p *Pack) loadCachedManifest() (*Manifest, error) {
	data, err := os.ReadFile(p.manifestPath())
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// client is the shared HTTP client (lazily built, like http.DefaultClient
// but with our own timeout policy per request).
var client struct {
	once sync.Once
	c    *http.Client
}

func (p *Pack) client() *http.Client {
	client.once.Do(func() {
		client.c = &http.Client{} // timeouts come per request via contexts
	})
	return client.c
}
