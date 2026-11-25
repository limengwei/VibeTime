// Package pack downloads vibe-time's sound pack at runtime: a manifest
// lists every sound with its size and SHA-256, files land in the user's
// config directory once fetched, and callers either wait on a specific
// sound (Ensure) or let the background prefetch quietly fill the cache.
//
// Every download is verified against the manifest's hash before it is
// kept, and a state table of those hashes lets later runs tell a changed
// file from a cached one — repository updates flow in silently. Sources
// are raced, all at once: the built-in mirrors plus any raw base URLs the
// repository hands down in sources.json (swappable without a client
// release), and, per file, a direct URL from labels.json's attribution.
// Callbacks always run on the app's goroutine through post, so handlers
// may touch signals and the audio engine freely.
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
// which manifest.json, labels.json and the core files' Paths live (the
// repository's master branch); cloudRaw is the cloud branch's raw base,
// where cloud-manifest.json and the on-demand tier's files live — the
// branch split keeps the master zip a pure core download. archive, when
// set, serves the whole master branch as a zip (gitee's
// repository/archive endpoint, GitHub's codeload), which a cold start
// fetches in one request.
type source struct {
	raw      string
	cloudRaw string
	archive  string
}

// Sources are the places the pack is served from. gitee first (no mirror
// needed in mainland China), GitHub behind it. VIBE_PACK_SOURCES
// (comma-separated) overrides the raw URLs, for mirrors and local testing;
// such a source serves both branches from one root and has no archive,
// falling back to file-by-file.
var Sources = defaultSources()

func defaultSources() []source {
	if env := os.Getenv("VIBE_PACK_SOURCES"); env != "" {
		var out []source
		for _, part := range strings.Split(env, ",") {
			if p := strings.TrimSpace(part); p != "" {
				out = append(out, source{raw: p, cloudRaw: p})
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
	cloudRaw := "https://raw.githubusercontent.com/" + repo + "/cloud"
	zip := "https://github.com/" + repo + "/archive/refs/heads/master.zip"
	mirror := func(base string) source {
		return source{raw: base + "/" + raw, cloudRaw: base + "/" + cloudRaw, archive: base + "/" + zip}
	}
	return []source{
		mirror("https://gh-proxy.com"),
		mirror("https://ghfast.top"),
		mirror("https://ghproxy.net"),
		mirror("https://github.akams.cn"),
		mirror("https://gh.llkk.cc"),
		mirror("https://hub.gitmirror.com"),
		mirror("https://ghproxy.homeboyc.cn"),
		{raw: raw, cloudRaw: cloudRaw, archive: "https://codeload.github.com/" + repo + "/zip/refs/heads/master"},
		{raw: "https://gitee.com/limengwe/vibe-time-sound/raw/master", cloudRaw: "https://gitee.com/limengwe/vibe-time-sound/raw/cloud"},
	}
}

// File is one sound in the manifest.
type File struct {
	ID     string `json:"id"` // "category/file-base", the app's sound ID
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
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
	// Attribution is labels.json's optional per-base origin/license/url
	// metadata; URL is raced as a direct download candidate (see fetch).
	Attribution map[string]catalog.Attribution `json:"attribution,omitempty"`
	// Sources comes from the repository's optional sources.json: raw base
	// URLs the repository hands down, so mirrors can be swapped without a
	// client release. They join every fetch's race after the built-ins.
	Sources []string `json:"sources,omitempty"`
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

// cloudFile is one on-demand sound in the repository's cloud-manifest.json:
// its metadata is self-contained — the Chinese title and category inline,
// plus duration and where the bytes came from. Only the list is fetched;
// the bytes download when a user actually picks the sound.
type cloudFile struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Category   string `json:"category"`
	Path       string `json:"path"`
	DurationMs int64  `json:"durationMs"`
	Size       int64  `json:"size"`
	SHA256     string `json:"sha256"`
	License    string `json:"license,omitempty"`
	Origin     string `json:"origin,omitempty"`
}

// cloudManifest is the repository's cloud-manifest.json, the on-demand
// tier alongside the core library's manifest.json.
type cloudManifest struct {
	Version    int                             `json:"version"`
	Categories map[string]catalog.CategoryMeta `json:"categories"`
	Files      []cloudFile                     `json:"files"`
}

// convertCloud folds a cloud manifest into the internal Manifest shape so
// every existing mechanism — download, verify, state, Ensure — serves the
// cloud tier unchanged. Titles become Labels; license and origin an
// Attribution without a URL (origin is a page, not a direct link).
func convertCloud(c *cloudManifest) *Manifest {
	m := &Manifest{
		Version:     c.Version,
		Files:       make([]File, 0, len(c.Files)),
		Labels:      map[string]string{},
		Categories:  c.Categories,
		Attribution: map[string]catalog.Attribution{},
	}
	for _, f := range c.Files {
		m.Files = append(m.Files, File{ID: f.ID, Path: f.Path, Size: f.Size, SHA256: f.SHA256})
		base := f.ID
		if i := strings.LastIndex(f.ID, "/"); i >= 0 {
			base = f.ID[i+1:]
		}
		if f.Title != "" {
			m.Labels[base] = f.Title
		}
		if f.License != "" || f.Origin != "" {
			m.Attribution[base] = catalog.Attribution{Origin: f.Origin, License: f.License}
		}
	}
	return m
}

// Pack is the runtime downloader and local cache.
type Pack struct {
	dir string // <user config>/vibe-time/pack

	// post lands callbacks on the app's goroutine; ctxFn yields the
	// context downloads respect. Open wires both from the UI scope;
	// tests inject their own.
	post  func(func())
	ctxFn func() context.Context

	mu       sync.Mutex
	man      *Manifest               // the core library: prefetched, zip-cold-started
	cloud    *Manifest               // the on-demand tier, converted from cloud-manifest.json
	pending  map[string][]func(bool) // waiters for a download in flight
	state    map[string]string       // sound ID → sha256 of the bytes on disk
	extraRaw []string                // raw sources the repository handed down
}

// Open prepares the cache directory and loads the last fetched manifest,
// so sounds already downloaded work offline before (or without) a refresh.
func Open(cx *core.Scope) (*Pack, error) {
	cfg, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	return openAt(filepath.Join(cfg, "vibe-time", "pack"), cx.Post, cx.Context)
}

// openAt builds the pack rooted at dir: post runs callbacks on the app's
// goroutine, ctxFn yields the app's context. Tests inject a synchronous
// post and context.Background.
func openAt(dir string, post func(func()), ctxFn func() context.Context) (*Pack, error) {
	p := &Pack{
		dir:     dir,
		post:    post,
		ctxFn:   ctxFn,
		pending: map[string][]func(bool){},
		state:   map[string]string{},
	}
	if err := os.MkdirAll(p.soundsDir(), 0o755); err != nil {
		return nil, err
	}
	p.loadState()
	if m, err := p.loadCachedManifest(); err == nil {
		p.man = m
		p.extraRaw = m.Sources
	}
	if c, err := p.readManifest(p.cloudManifestPath()); err == nil {
		p.cloud = c
	}
	return p, nil
}

// Start fetches both manifests (falling back to the cached copies
// offline), reports them together on the app's goroutine, prunes files
// the pack dropped, then prefetches whatever is missing in the
// background — unless prefetch is false, when the welcome page's Download
// owns the first fill instead. The prefetch and the whole-repository zip
// cover the core library only; cloud sounds stay on-demand.
//
// cloud is nil when the repository has no cloud library (or none could be
// fetched and none is cached) — that is logged, not a failure. With no
// core manifest to fall back to either (a first launch offline), onFail
// runs instead of onSync; calling Start again retries.
func (p *Pack) Start(onSync func(main, cloud *Manifest), prefetch bool, onFail func()) {
	go func() {
		ctx := p.ctxFn()
		m, cloud, err := p.fetchManifest(ctx)
		if err != nil {
			log.Printf("[pack] manifest 拉取失败（使用本地缓存）: %v", err)
			p.mu.Lock()
			m = p.man
			cloud = p.cloud
			p.mu.Unlock()
			if m == nil {
				if onFail != nil {
					p.post(onFail)
				}
				return
			}
		} else {
			if err := p.cacheManifest(p.manifestPath(), m); err != nil {
				log.Printf("[pack] manifest 缓存写入失败: %v", err)
			}
			if cloud != nil {
				if err := p.cacheManifest(p.cloudManifestPath(), cloud); err != nil {
					log.Printf("[pack] 云清单缓存写入失败: %v", err)
				}
			}
			p.mu.Lock()
			p.man = m
			p.cloud = cloud
			p.extraRaw = m.Sources
			p.mu.Unlock()
			p.prune(m, cloud)
		}
		mm, cm := m, cloud
		p.post(func() { onSync(mm, cm) })
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
		if !p.fresh(f) {
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
		if p.fresh(f) {
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

// ReadyCount reports how many of the current manifest's files are on disk
// with the manifest's current bytes (0 with no manifest — a fresh install).
func (p *Pack) ReadyCount() int {
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

// Stats reports the core library's file count and total bytes, for the
// welcome page's copy (the cloud tier stays on-demand and uncounted).
func (p *Pack) Stats() (files int, bytes int64) {
	p.mu.Lock()
	m := p.man
	p.mu.Unlock()
	if m == nil {
		return 0, 0
	}
	for _, f := range m.Files {
		bytes += f.Size
	}
	return len(m.Files), bytes
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

// fresh reports whether the bytes on disk are the manifest's current
// bytes for f. The state table answers from its recorded hash; a cache
// from before the table existed (or a damaged table) hashes the file
// once and backfills the entry, so old caches never re-download.
func (p *Pack) fresh(f File) bool {
	path := filepath.Join(p.dir, filepath.FromSlash(f.Path))
	st, err := os.Stat(path)
	if err != nil || st.Size() != f.Size {
		return false
	}
	p.mu.Lock()
	h, ok := p.state[f.ID]
	p.mu.Unlock()
	if ok {
		return h == f.SHA256
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	sum := sha256.Sum256(data)
	if hex.EncodeToString(sum[:]) != f.SHA256 {
		return false
	}
	p.mu.Lock()
	p.state[f.ID] = f.SHA256
	p.saveStateLocked()
	p.mu.Unlock()
	return true
}

// statePath is the id → sha256 table of the bytes on disk.
func (p *Pack) statePath() string { return filepath.Join(p.dir, "state.json") }

// loadState reads the state table; a missing or damaged file just means
// the hashes get re-derived (see fresh).
func (p *Pack) loadState() {
	data, err := os.ReadFile(p.statePath())
	if err != nil {
		return
	}
	var s map[string]string
	if err := json.Unmarshal(data, &s); err == nil && s != nil {
		p.state = s
	}
}

// saveStateLocked persists the state table atomically; mu is held.
func (p *Pack) saveStateLocked() {
	data, err := json.MarshalIndent(p.state, "", "  ")
	if err != nil {
		return
	}
	tmp := p.statePath() + ".tmp"
	if os.WriteFile(tmp, data, 0o644) == nil {
		_ = os.Rename(tmp, p.statePath())
	}
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
	p.mu.Lock()
	entry, inManifest := p.fileOfLocked(id)
	p.mu.Unlock()
	if inManifest && p.fresh(entry) {
		ok := true
		p.post(func() { done(ok) })
		return
	}
	if !inManifest {
		p.post(func() { done(false) })
		return
	}
	p.mu.Lock()
	if waiters, busy := p.pending[id]; busy {
		p.pending[id] = append(waiters, done)
		p.mu.Unlock()
		return
	}
	p.pending[id] = []func(bool){done}
	p.mu.Unlock()

	go func() {
		err := p.download(p.ctxFn(), entry)
		ok := err == nil
		if err != nil {
			log.Printf("[pack] 下载 %s 失败: %v", id, err)
		}
		p.mu.Lock()
		waiters := p.pending[id]
		delete(p.pending, id)
		p.mu.Unlock()
		p.post(func() {
			for _, w := range waiters {
				w(ok)
			}
		})
	}()
}

// soundsDir is where the pack's files land, mirroring the repo layout.
func (p *Pack) soundsDir() string { return filepath.Join(p.dir, "sounds") }

// cloudDir is where the on-demand tier's files land.
func (p *Pack) cloudDir() string { return filepath.Join(p.dir, "cloud") }

// manifestPath is the cached copy of the last fetched manifest.
func (p *Pack) manifestPath() string { return filepath.Join(p.dir, "manifest.json") }

// cloudManifestPath is the cached copy of the last fetched cloud manifest
// (stored already converted, so a reload needs no cloud schema).
func (p *Pack) cloudManifestPath() string { return filepath.Join(p.dir, "cloud-manifest.json") }

// pathOfLocked maps a sound ID to its cache path, "" when unknown.
func (p *Pack) pathOfLocked(id string) string {
	f, ok := p.fileOfLocked(id)
	if !ok {
		return ""
	}
	return filepath.Join(p.dir, filepath.FromSlash(f.Path))
}

// fileOfLocked finds a sound's entry across both manifests, the core
// library first — a core entry shadows a cloud duplicate of the same ID.
// mu is held.
func (p *Pack) fileOfLocked(id string) (File, bool) {
	if p.man != nil {
		if f, ok := p.man.ByID(id); ok {
			return f, true
		}
	}
	if p.cloud != nil {
		if f, ok := p.cloud.ByID(id); ok {
			return f, true
		}
	}
	return File{}, false
}

// fetchManifest pulls manifest.json — labels.json, sources.json and the
// cloud library's cloud-manifest.json when they answer. fetch races every
// source, so one round is enough. The cloud manifest is optional and
// converted on the way in; a missing one yields cloud == nil, no error.
func (p *Pack) fetchManifest(ctx context.Context) (m *Manifest, cloud *Manifest, err error) {
	data, err := p.fetch(ctx, "manifest.json", nil, false)
	if err != nil {
		return nil, nil, err
	}
	var mm Manifest
	if err := json.Unmarshal(data, &mm); err != nil {
		return nil, nil, fmt.Errorf("manifest: %w", err)
	}
	if len(mm.Files) == 0 {
		return nil, nil, errors.New("manifest: no files")
	}
	m = &mm
	if ldata, err := p.fetch(ctx, "labels.json", nil, false); err == nil {
		var lf struct {
			Categories  map[string]catalog.CategoryMeta `json:"categories"`
			Labels      map[string]string               `json:"labels"`
			Attribution map[string]catalog.Attribution  `json:"attribution"`
		}
		if err := json.Unmarshal(ldata, &lf); err == nil {
			m.Categories, m.Labels, m.Attribution = lf.Categories, lf.Labels, lf.Attribution
		}
	}
	// sources.json is optional: the repository hands down extra raw
	// sources here, swappable without a client release.
	if sdata, err := p.fetch(ctx, "sources.json", nil, false); err == nil {
		var sf struct {
			Sources []string `json:"sources"`
		}
		if err := json.Unmarshal(sdata, &sf); err == nil && len(sf.Sources) > 0 {
			m.Sources = sf.Sources
		}
	}
	if cdata, cerr := p.fetch(ctx, "cloud-manifest.json", nil, true); cerr == nil {
		var cm cloudManifest
		if err := json.Unmarshal(cdata, &cm); err == nil && len(cm.Files) > 0 {
			cloud = convertCloud(&cm)
		}
	}
	return m, cloud, nil
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
		if p.fresh(f) || p.Pending(f.ID) {
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

// cachedCount counts how many of the manifest's files are cached with
// their current bytes.
func (p *Pack) cachedCount(m *Manifest) int {
	n := 0
	for _, f := range m.Files {
		if p.fresh(f) {
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

// fetch GETs rel — relative to every source and to any raw base URLs the
// repository handed down — racing them alongside direct URLs (a file's
// attribution link): the first healthy answer wins, and cancelling the
// context it returns with stops the stragglers. A response that is not a
// 200, or is an HTML page (a mirror's error or risk-control answer),
// loses its heat. A cloud fetch (the on-demand tier's manifest and files)
// races the sources' cloud-branch raw bases instead; the repo-delivered
// sources serve master.
func (p *Pack) fetch(ctx context.Context, rel string, direct []string, cloud bool) ([]byte, error) {
	cctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	var bases []string
	if cloud {
		for _, src := range Sources {
			if src.cloudRaw != "" {
				bases = append(bases, src.cloudRaw)
			}
		}
	} else {
		p.mu.Lock()
		bases = append(bases, p.extraRaw...)
		p.mu.Unlock()
		for _, src := range Sources {
			bases = append(bases, src.raw)
		}
	}
	urls := make([]string, 0, len(bases)+len(direct))
	for _, b := range bases {
		urls = append(urls, strings.TrimRight(b, "/")+"/"+rel)
	}
	urls = append(urls, direct...)

	type result struct {
		data []byte
		err  error
	}
	ch := make(chan result, len(urls))
	for _, url := range urls {
		go func(url string) {
			req, err := http.NewRequestWithContext(cctx, http.MethodGet, url, nil)
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
				ch <- result{nil, fmt.Errorf("%s: HTTP %d", url, code)}
			case strings.Contains(ct, "text/html"):
				ch <- result{nil, fmt.Errorf("%s: HTML answer, not a file", url)}
			default:
				ch <- result{data, nil}
			}
		}(url)
	}
	var lastErr error = errors.New("pack: no source answered")
	for range urls {
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
// A direct URL from the file's attribution rides along in the race.
func (p *Pack) download(ctx context.Context, f File) error {
	cctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	var direct []string
	p.mu.Lock()
	for _, m := range []*Manifest{p.man, p.cloud} {
		if m == nil || m.Attribution == nil {
			continue
		}
		if _, base, ok := strings.Cut(f.ID, "/"); ok {
			if a := m.Attribution[base]; a.URL != "" {
				direct = append(direct, a.URL)
			}
		}
	}
	p.mu.Unlock()
	// Cloud files live on the repository's cloud branch; core files on
	// master. The path tells them apart.
	data, err := p.fetch(cctx, f.Path, direct, strings.HasPrefix(f.Path, "cloud/"))
	if err != nil {
		return err
	}
	return p.store(f, data)
}

// store verifies a file's bytes against the manifest and moves them into
// place, atomically, recording the hash in the state table.
func (p *Pack) store(f File, data []byte) error {
	if int64(len(data)) != f.Size {
		return fmt.Errorf("%s: size %d, want %d", f.ID, len(data), f.Size)
	}
	sum := sha256.Sum256(data)
	if hex.EncodeToString(sum[:]) != f.SHA256 {
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
	if err := os.Rename(tmp, dst); err != nil {
		return err
	}
	p.mu.Lock()
	p.state[f.ID] = f.SHA256
	p.saveStateLocked()
	p.mu.Unlock()
	return nil
}

// prune removes cached files (and state entries) the pack no longer lists
// in either manifest.
func (p *Pack) prune(m, cloud *Manifest) {
	keep := map[string]bool{}
	keepID := map[string]bool{}
	for _, f := range m.Files {
		keep[f.Path] = true
		keepID[f.ID] = true
	}
	if cloud != nil {
		for _, f := range cloud.Files {
			keep[f.Path] = true
			keepID[f.ID] = true
		}
	}
	n := 0
	for _, root := range []string{p.soundsDir(), p.cloudDir()} {
		var paths []string
		filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err == nil && !d.IsDir() {
				paths = append(paths, path)
			}
			return nil
		})
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
	}
	// State entries follow their files out.
	p.mu.Lock()
	dirty := false
	for id := range p.state {
		if !keepID[id] {
			delete(p.state, id)
			dirty = true
		}
	}
	if dirty {
		p.saveStateLocked()
	}
	p.mu.Unlock()
	if n > 0 {
		log.Printf("[pack] 清理 %d 个已下架的缓存文件", n)
	}
}

// loadCachedManifest reads the core manifest saved by an earlier run.
func (p *Pack) loadCachedManifest() (*Manifest, error) {
	return p.readManifest(p.manifestPath())
}

// readManifest parses a cached manifest copy; the cloud cache holds the
// converted internal shape, so the same reader serves both.
func (p *Pack) readManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// cacheManifest writes a manifest copy for offline starts.
func (p *Pack) cacheManifest(path string, m *Manifest) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
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
