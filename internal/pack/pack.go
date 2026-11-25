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
	"archive/zip"
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

// source is one place the pack is served from: raw is the base URL under
// which manifest.json, labels.json and the files' Paths live; archive, when
// set, serves the whole repository as a zip (gitee's repository/archive
// endpoint, GitHub's codeload), which a cold start fetches in one request.
type source struct {
	raw     string
	archive string
}

// Sources are the places the pack is served from. gitee first (no mirror
// needed in mainland China), GitHub behind it. VIBE_PACK_SOURCES
// (comma-separated) overrides the raw URLs, for mirrors and local testing;
// such sources have no archive and fall back to file-by-file.
var Sources = defaultSources()

func defaultSources() []source {
	if env := os.Getenv("VIBE_PACK_SOURCES"); env != "" {
		var out []source
		for _, part := range strings.Split(env, ",") {
			if p := strings.TrimSpace(part); p != "" {
				out = append(out, source{raw: p})
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	// Mirror availability swings by region and by the day, so fetch races
	// every source at once (see fetch) and the list carries them all:
	// whichever answers first wins. Archives only ride on sources whose
	// URL shape is known to serve them; gitee's archive endpoint sits
	// behind risk-control captcha pages and stays off.
	const repo = "limengwei/vibe-time-sound"
	raw := "https://raw.githubusercontent.com/" + repo + "/master"
	zip := "https://github.com/" + repo + "/archive/refs/heads/master.zip"
	mirror := func(base string) source {
		return source{raw: base + "/" + raw, archive: base + "/" + zip}
	}
	return []source{
		mirror("https://gh-proxy.com"),
		mirror("https://ghfast.top"),
		mirror("https://ghproxy.net"),
		mirror("https://github.akams.cn"),
		mirror("https://gh.llkk.cc"),
		mirror("https://hub.gitmirror.com"),
		mirror("https://ghproxy.homeboyc.cn"),
		{raw: raw, archive: "https://codeload.github.com/" + repo + "/zip/refs/heads/master"},
		{raw: "https://gitee.com/limengwe/vibe-time-sound/raw/master"},
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
	Labels     map[string]string               `json:"labels,omitempty"`
	Categories map[string]catalog.CategoryMeta `json:"categories,omitempty"`
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
// prefetches whatever is missing in the background — unless prefetch is
// false, when the welcome page's Download owns the first fill instead.
func (p *Pack) Start(onManifest func(*Manifest), prefetch bool) {
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
		if prefetch {
			p.prefetch(ctx, m)
		}
	}()
}

// Download fetches the whole pack now and blocks until every file of the
// current manifest is on disk. Progress reports a phase — "下载" (the
// archive's bytes), "入库" (files verified and stored out of the archive),
// "补齐" (files fetched one by one, the count) — each with a done/total
// pair; total may be 0 while unknown. The welcome page calls it from a
// plain goroutine — onProgress runs there too.
func (p *Pack) Download(ctx context.Context, onProgress func(phase string, done, total int64)) error {
	p.mu.Lock()
	m := p.man
	p.mu.Unlock()
	if m == nil {
		return errors.New("pack: manifest 未知，无法下载")
	}
	missing := 0
	for _, f := range m.Files {
		if !p.Cached(f.ID) {
			missing++
		}
	}
	if missing == 0 {
		return nil
	}
	// The whole-repository zip covers everything in one request; failures
	// fall through to the per-file loop below. Chunked transfers carry no
	// Content-Length, but the pack's files are already compressed, so the
	// manifest's total stands in for the archive's size.
	var totalBytes int64
	for _, f := range m.Files {
		totalBytes += f.Size
	}
	onProgress("下载", 0, 0)
	if err := p.fetchArchive(ctx, m,
		func(done, total int64) {
			if total <= 0 {
				total = totalBytes
			}
			onProgress("下载", done, total)
		},
		func(done, total int) { onProgress("入库", int64(done), int64(total)) },
	); err != nil {
		log.Printf("[pack] 整库 zip 失败，改为逐个文件: %v", err)
	}
	left := 0
	for i, f := range m.Files {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if p.Cached(f.ID) {
			continue
		}
		onProgress("补齐", int64(i+1), int64(len(m.Files)))
		if err := p.download(ctx, f); err != nil {
			log.Printf("[pack] 下载 %s 失败: %v", f.ID, err)
			left++
		}
	}
	if left > 0 {
		return fmt.Errorf("%d 个文件下载失败，请重试", left)
	}
	return nil
}

// CachedCount reports how many of the current manifest's files are on
// disk (0 with no manifest — a fresh install).
func (p *Pack) CachedCount() int {
	p.mu.Lock()
	m := p.man
	p.mu.Unlock()
	if m == nil {
		return 0
	}
	return p.cachedCount(m)
}

// Dir is the pack's cache directory, for display.
func (p *Pack) Dir() string { return p.dir }

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

// fetchManifest pulls manifest.json — and labels.json when it answers.
// fetch races every source, so one round is enough.
func (p *Pack) fetchManifest(ctx context.Context) (*Manifest, error) {
	data, err := p.fetch(ctx, "manifest.json")
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("manifest: %w", err)
	}
	if len(m.Files) == 0 {
		return nil, errors.New("manifest: no files")
	}
	if ldata, err := p.fetch(ctx, "labels.json"); err == nil {
		var lf struct {
			Categories map[string]catalog.CategoryMeta `json:"categories"`
			Labels     map[string]string               `json:"labels"`
		}
		if err := json.Unmarshal(ldata, &lf); err == nil {
			m.Categories, m.Labels = lf.Categories, lf.Labels
		}
	}
	return &m, nil
}

// prefetch quietly fills the cache. A cold start (nothing cached) takes
// the whole-repository zip in one request — gitee's archive streams at
// several MB/s where its raw endpoint answers under one — while updates
// and stragglers go file by file, so one changed sound never re-downloads
// the pack. A file the user's Ensure already fetched is skipped.
func (p *Pack) prefetch(ctx context.Context, m *Manifest) {
	if p.cachedCount(m) == 0 {
		if err := p.fetchArchive(ctx, m, nil, nil); err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("[pack] 整库 zip 下载失败，回退逐个文件: %v", err)
		} else {
			log.Printf("[pack] 声音包整库下载完成（version %d，%d 个文件）", m.Version, len(m.Files))
		}
	}
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

// cachedCount counts how many of the manifest's files are on disk.
func (p *Pack) cachedCount(m *Manifest) int {
	n := 0
	for _, f := range m.Files {
		if p.Cached(f.ID) {
			n++
		}
	}
	return n
}

// fetchArchive downloads a whole-repository zip, trying every source that
// serves one, and stores each file it lists against the manifest's hash;
// the zip itself lands in a temp file, never memory. onBytes follows the
// download and onFiles the extraction, both on the caller's goroutine.
func (p *Pack) fetchArchive(ctx context.Context, m *Manifest, onBytes func(done, total int64), onFiles func(done, total int)) error {
	var lastErr error = errors.New("pack: no archive source")
	for _, src := range Sources {
		if src.archive == "" {
			continue
		}
		err := p.fetchArchiveFrom(ctx, src.archive, m, onBytes, onFiles)
		if err == nil {
			return nil
		}
		if ctx.Err() != nil {
			return err
		}
		lastErr = err
	}
	return lastErr
}

// fetchArchiveFrom fetches one archive URL.
func (p *Pack) fetchArchiveFrom(ctx context.Context, url string, m *Manifest, onBytes func(done, total int64), onFiles func(done, total int)) error {
	cctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	req, err := http.NewRequestWithContext(cctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := p.client().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("archive: HTTP %d", resp.StatusCode)
	}
	// A mirror or host under risk control answers with an HTML page where
	// the zip should be; fail that source cleanly instead of parsing it.
	if ct := resp.Header.Get("Content-Type"); strings.Contains(ct, "text/html") {
		return fmt.Errorf("archive: got %s, not a zip (risk control?)", ct)
	}
	zipPath := filepath.Join(p.dir, "archive.zip.tmp")
	out, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	var body io.Reader = resp.Body
	if onBytes != nil {
		total := resp.ContentLength
		cr := &countingReader{r: resp.Body, on: func(n int64) { onBytes(n, total) }}
		body = cr
	}
	if _, err := io.Copy(out, body); err != nil {
		out.Close()
		os.Remove(zipPath)
		return err
	}
	if err := out.Close(); err != nil {
		os.Remove(zipPath)
		return err
	}
	defer os.Remove(zipPath)

	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer zr.Close()
	byPath := make(map[string]File, len(m.Files))
	for _, f := range m.Files {
		byPath[f.Path] = f
	}
	stored := 0
	for _, e := range zr.File {
		if e.FileInfo().IsDir() {
			continue
		}
		// Zip entries sit under a "<repo>-<branch>/" prefix; the pack's
		// Paths follow it.
		rel := e.Name
		if i := strings.Index(rel, "/"); i >= 0 {
			rel = rel[i+1:]
		}
		f, listed := byPath[filepath.ToSlash(rel)]
		if !listed {
			continue
		}
		rc, err := e.Open()
		if err != nil {
			return err
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return err
		}
		if err := p.store(f, data); err != nil {
			return err
		}
		stored++
		if onFiles != nil {
			onFiles(stored, len(m.Files))
		}
	}
	if stored == 0 {
		return errors.New("archive: no manifest files inside")
	}
	return nil
}

// fetch GETs rel (relative to every source) and races them: the first
// healthy answer wins, and cancelling the context it returns with stops
// the stragglers. A response that is not a 200, or is an HTML page (a
// mirror's error or risk-control answer), loses its heat.
func (p *Pack) fetch(ctx context.Context, rel string) ([]byte, error) {
	cctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	type result struct {
		data []byte
		err  error
	}
	ch := make(chan result, len(Sources))
	for _, src := range Sources {
		go func(raw string) {
			req, err := http.NewRequestWithContext(cctx, http.MethodGet, strings.TrimRight(raw, "/")+"/"+rel, nil)
			if err != nil {
				ch <- result{nil, err}
				return
			}
			resp, err := p.client().Do(req)
			if err != nil {
				ch <- result{nil, err}
				return
			}
			data, err := io.ReadAll(resp.Body)
			ct := resp.Header.Get("Content-Type")
			code := resp.StatusCode
			resp.Body.Close()
			switch {
			case err != nil:
				ch <- result{nil, err}
			case code != http.StatusOK:
				ch <- result{nil, fmt.Errorf("%s: HTTP %d", raw, code)}
			case strings.Contains(ct, "text/html"):
				ch <- result{nil, fmt.Errorf("%s: HTML answer, not a file", raw)}
			default:
				ch <- result{data, nil}
			}
		}(src.raw)
	}
	var lastErr error = errors.New("pack: no source answered")
	for range Sources {
		if r := <-ch; r.err == nil {
			// Returning cancels the context, closing the losing requests.
			return r.data, nil
		} else if r.err != context.Canceled {
			lastErr = r.err
		}
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
	return p.store(f, data)
}

// store verifies a file's bytes against the manifest and moves them into
// place, atomically.
func (p *Pack) store(f File, data []byte) error {
	if int64(len(data)) != f.Size {
		return fmt.Errorf("%s: size %d, want %d", f.ID, len(data), f.Size)
	}
	sum := sha256.Sum256(data)
	if hex.EncodeToString(sum[:]) != f.SHA56 {
		return fmt.Errorf("%s: sha256 mismatch", f.ID)
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

// countingReader wraps a reader, reporting the running total of bytes
// read through it.
type countingReader struct {
	r  io.Reader
	n  int64
	on func(n int64)
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += int64(n)
	c.on(c.n)
	return n, err
}
