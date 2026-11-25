package pack

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"vibe-time/internal/catalog"
)

func syncPost(f func())      { f() }
func bgCtx() context.Context { return context.Background() }

// fakeRepo is a mutable in-memory pack repository served over HTTP; tests
// flip its contents between Start calls to simulate repository updates.
type fakeRepo struct {
	mu    sync.Mutex
	path  map[string][]byte
	html  map[string]bool          // entries served as text/html (risk-control pages)
	delay map[string]time.Duration // per-path artificial latency
	hits  map[string]int
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{path: map[string][]byte{}, html: map[string]bool{}, hits: map[string]int{}}
}

func (r *fakeRepo) set(rel, data string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.path[rel] = []byte(data)
}

// setHTML serves an entry as a text/html page, the shape a mirror under
// risk control answers with.
func (r *fakeRepo) setHTML(rel, data string) {
	r.set(rel, data)
	r.mu.Lock()
	defer r.mu.Unlock()
	r.html[rel] = true
}

func (r *fakeRepo) del(rel string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.path, rel)
}

func (r *fakeRepo) hitsFor(rel string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.hits[rel]
}

// setSlow delays one path's answers, simulating a lagging network.
func (r *fakeRepo) setSlow(rel string, d time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.delay == nil {
		r.delay = map[string]time.Duration{}
	}
	r.delay[rel] = d
}

func (r *fakeRepo) serve(w http.ResponseWriter, req *http.Request) {
	rel := strings.TrimPrefix(req.URL.Path, "/")
	r.mu.Lock()
	if d, slow := r.delay[rel]; slow && d > 0 {
		r.mu.Unlock()
		time.Sleep(d)
		r.mu.Lock()
	}
	data, ok := r.path[rel]
	isHTML := r.html[rel]
	r.hits[rel]++
	r.mu.Unlock()
	if !ok {
		http.NotFound(w, req)
		return
	}
	if isHTML {
		w.Header().Set("Content-Type", "text/html")
	} else {
		w.Header().Set("Content-Type", "application/octet-stream")
	}
	w.Write(data)
}

func serveRepo(t *testing.T, r *fakeRepo) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(r.serve))
	t.Cleanup(srv.Close)
	return srv
}

// newPack points a Pack at t.TempDir(), replacing the package's Sources
// with the given raw base URLs for the test's duration.
func newPack(t *testing.T, sources ...string) *Pack {
	t.Helper()
	old := Sources
	Sources = make([]source, len(sources))
	for i, s := range sources {
		Sources[i] = source{raw: s, cloudRaw: s}
	}
	t.Cleanup(func() { Sources = old })
	p, err := openAt(filepath.Join(t.TempDir(), "pack"), syncPost, bgCtx)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func entry(id, data string) File {
	sum := sha256.Sum256([]byte(data))
	return File{ID: id, Path: "sounds/" + id + ".m4a", Size: int64(len(data)), SHA256: hex.EncodeToString(sum[:])}
}

func manifestJSON(t *testing.T, version int, files []File) string {
	t.Helper()
	data, err := json.Marshal(Manifest{Version: version, Files: files})
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// cloudEntry pairs a cloud sound's File with its cloud-manifest metadata.
type cloudEntry struct {
	File
	title, license, origin string
}

// cloudFileEntry builds a cloud-tier File: its path lives under cloud/,
// on the repository's cloud branch.
func cloudFileEntry(id, data string) File {
	sum := sha256.Sum256([]byte(data))
	return File{ID: id, Path: "cloud/" + id + ".mp3", Size: int64(len(data)), SHA256: hex.EncodeToString(sum[:])}
}

// cloudJSON marshals a cloud-manifest.json wire document.
func cloudJSON(t *testing.T, version int, entries []cloudEntry) string {
	t.Helper()
	cm := cloudManifest{Version: version, Categories: map[string]catalog.CategoryMeta{"cat": {Title: "云端", Emoji: "☁️"}}}
	for _, e := range entries {
		cat := e.ID
		if i := strings.LastIndex(e.ID, "/"); i >= 0 {
			cat = e.ID[:i]
		}
		cm.Files = append(cm.Files, cloudFile{
			ID: e.ID, Title: e.title, Category: cat,
			Path: e.Path, Size: e.Size, SHA256: e.SHA256,
			License: e.license, Origin: e.origin,
		})
	}
	data, err := json.Marshal(cm)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func waitFor(t *testing.T, cond func() bool, msg string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal(msg)
}

func readState(t *testing.T, dir string) map[string]string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, "state.json"))
	if err != nil {
		t.Fatalf("state.json: %v", err)
	}
	var s map[string]string
	if err := json.Unmarshal(data, &s); err != nil {
		t.Fatalf("state.json: %v", err)
	}
	return s
}

func diskPath(dir string, f File) string { return filepath.Join(dir, filepath.FromSlash(f.Path)) }

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// TestUpdatePropagation walks a v1 → v2 repository update through a full
// Start: changed bytes re-download, additions land, retirements leave the
// disk and the state table, and a renamed label arrives with the manifest.
func TestUpdatePropagation(t *testing.T) {
	repo := newFakeRepo()
	srv := serveRepo(t, repo)
	p := newPack(t, srv.URL)

	a1, b := entry("cat/a", "A-version-1"), entry("cat/b", "B")
	repo.set("manifest.json", manifestJSON(t, 1, []File{a1, b}))
	repo.set(a1.Path, "A-version-1")
	repo.set(b.Path, "B")
	repo.set("labels.json", `{"labels":{"a":"甲","b":"乙"}}`)

	manifests := make(chan *Manifest, 2)
	p.Start(func(m, _ *Manifest) { manifests <- m }, true, nil)
	if m := <-manifests; len(m.Files) != 2 || m.Labels["a"] != "甲" {
		t.Fatalf("v1 manifest = %d files, labels %v", len(m.Files), m.Labels)
	}
	waitFor(t, func() bool { return p.fresh(a1) && p.fresh(b) }, "v1 files downloaded")

	if data, _ := os.ReadFile(diskPath(p.dir, a1)); string(data) != "A-version-1" {
		t.Fatalf("a's bytes = %q", data)
	}
	state := readState(t, p.dir)
	if state["cat/a"] != a1.SHA256 || state["cat/b"] != b.SHA256 {
		t.Fatalf("state table = %v", state)
	}

	// v2: a's bytes change, b retires, c joins, a's label is renamed.
	a2, c := entry("cat/a", "A-version-2"), entry("cat/c", "C")
	repo.set("manifest.json", manifestJSON(t, 2, []File{a2, c}))
	repo.set(a2.Path, "A-version-2")
	repo.set(c.Path, "C")
	repo.del(b.Path)
	repo.set("labels.json", `{"labels":{"a":"甲改名","c":"丙"}}`)

	p.Start(func(m, _ *Manifest) {}, true, nil)
	// The cached v1 delivers first; the v2 refresh follows.
	waitFor(t, func() bool {
		p.mu.Lock()
		defer p.mu.Unlock()
		return p.man != nil && p.man.Labels["a"] == "甲改名"
	}, "v2 manifest with the renamed label arrived")
	waitFor(t, func() bool {
		return p.fresh(a2) && p.fresh(c) && !fileExists(diskPath(p.dir, b))
	}, "v2 propagated: re-download, addition, retirement")

	if data, _ := os.ReadFile(diskPath(p.dir, a2)); string(data) != "A-version-2" {
		t.Fatalf("a's bytes = %q, want v2", data)
	}
	state = readState(t, p.dir)
	if state["cat/a"] != a2.SHA256 || state["cat/c"] != c.SHA256 {
		t.Fatalf("state table after v2 = %v", state)
	}
	if _, ok := state["cat/b"]; ok {
		t.Fatal("state table kept the retired sound")
	}
}

// TestStateMigrationWithoutRedownload covers a cache from before the state
// table existed (or with a damaged one): the local file is hashed once,
// the entry backfills, and nothing re-downloads.
func TestStateMigrationWithoutRedownload(t *testing.T) {
	repo := newFakeRepo()
	srv := serveRepo(t, repo)
	p := newPack(t, srv.URL)

	a := entry("cat/a", "A")
	repo.set("manifest.json", manifestJSON(t, 1, []File{a}))
	repo.set(a.Path, "A")
	p.Start(func(m, _ *Manifest) {}, true, nil)
	waitFor(t, func() bool { return p.fresh(a) }, "v1 downloaded")

	if err := os.Remove(filepath.Join(p.dir, "state.json")); err != nil {
		t.Fatal(err)
	}
	p2, err := openAt(p.dir, syncPost, bgCtx)
	if err != nil {
		t.Fatal(err)
	}
	before := repo.hitsFor(a.Path)
	p2.Start(func(m, _ *Manifest) {}, true, nil)
	waitFor(t, func() bool { return p2.fresh(a) }, "local file validated by hash")

	if after := repo.hitsFor(a.Path); after != before {
		t.Fatalf("sound re-downloaded: %d → %d requests", before, after)
	}
	if state := readState(t, p.dir); state["cat/a"] != a.SHA256 {
		t.Fatalf("state table not backfilled: %v", state)
	}
}

// TestRepoDeliveredSources covers sources.json: the repository hands down
// a second raw base URL, and the file the built-in source cannot serve
// arrives through it.
func TestRepoDeliveredSources(t *testing.T) {
	main, files := newFakeRepo(), newFakeRepo()
	mainSrv, filesSrv := serveRepo(t, main), serveRepo(t, files)

	a := entry("cat/a", "A")
	main.set("manifest.json", manifestJSON(t, 1, []File{a}))
	main.set("sources.json", `{"sources":["`+filesSrv.URL+`"]}`)
	files.set(a.Path, "A")

	p := newPack(t, mainSrv.URL)
	p.Start(func(m, _ *Manifest) {}, true, nil)
	waitFor(t, func() bool { return p.fresh(a) }, "file fetched via the delivered source")

	if files.hitsFor(a.Path) == 0 {
		t.Fatal("the delivered source never served the file")
	}
	p.mu.Lock()
	extra := append([]string{}, p.extraRaw...)
	p.mu.Unlock()
	if len(extra) != 1 || extra[0] != filesSrv.URL {
		t.Fatalf("repo-delivered sources = %v", extra)
	}
}

// TestColdStartArchive covers the whole-repository zip a cold start takes:
// files stored out of the archive land with their hashes in the state
// table, and no per-file request is ever made.
func TestColdStartArchive(t *testing.T) {
	repo := newFakeRepo()
	srv := serveRepo(t, repo)
	a, b := entry("cat/a", "A"), entry("cat/b", "BB")
	repo.set("manifest.json", manifestJSON(t, 1, []File{a, b}))

	// The zip mirrors the repository layout under "<repo>-<branch>/".
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, e := range []struct{ name, data string }{
		{"vibe-time-sound-master/manifest.json", manifestJSON(t, 1, []File{a, b})},
		{"vibe-time-sound-master/sounds/cat/a.m4a", "A"},
		{"vibe-time-sound-master/sounds/cat/b.m4a", "BB"},
		{"vibe-time-sound-master/cloud/cat/c.mp3", "CLOUD"}, // never extracted: not in the core manifest
	} {
		w, err := zw.Create(e.name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(e.data)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	repo.set("archive.zip", buf.String())

	old := Sources
	Sources = []source{{raw: srv.URL, archive: srv.URL + "/archive.zip"}}
	t.Cleanup(func() { Sources = old })

	p, err := openAt(filepath.Join(t.TempDir(), "pack"), syncPost, bgCtx)
	if err != nil {
		t.Fatal(err)
	}
	p.Start(func(m, _ *Manifest) {}, true, nil)
	waitFor(t, func() bool { return p.fresh(a) && p.fresh(b) }, "cold start filled from the archive")

	if n := repo.hitsFor(a.Path); n != 0 {
		t.Fatalf("per-file request made although the zip covered it: %d", n)
	}
	if fileExists(p.cloudDir()) {
		t.Fatal("the cloud tier was extracted from the core zip")
	}
	state := readState(t, p.dir)
	if state["cat/a"] != a.SHA256 || state["cat/b"] != b.SHA256 {
		t.Fatalf("state table after the archive = %v", state)
	}
}

// TestAttributionDirectURL covers labels.json's attribution: a direct URL
// serves the bytes the repository itself cannot, and an attribution URL
// answering with HTML loses its heat so the race falls back.
func TestAttributionDirectURL(t *testing.T) {
	repo, direct := newFakeRepo(), newFakeRepo()
	repoSrv, directSrv := serveRepo(t, repo), serveRepo(t, direct)

	a := entry("cat/a", "A")
	repo.set("manifest.json", manifestJSON(t, 1, []File{a}))
	repo.set("labels.json", `{"attribution":{"a":{"origin":"https://example.org","license":"MIT","url":"`+directSrv.URL+`/sounds/cat/a.m4a"}}}`)
	direct.set(a.Path, "A")

	p := newPack(t, repoSrv.URL)
	p.Start(func(m, _ *Manifest) {}, true, nil)
	waitFor(t, func() bool { return p.fresh(a) }, "file fetched via the attribution URL")
	p.mu.Lock()
	att := p.man.Attribution["a"]
	p.mu.Unlock()
	if att.Origin != "https://example.org" || att.License != "MIT" {
		t.Fatalf("attribution parsed = %+v", att)
	}

	// The attribution URL turns into an HTML risk-control page; the
	// repository's own source then wins the race for the new sound.
	b := entry("cat/b", "B")
	repo.set("manifest.json", manifestJSON(t, 2, []File{a, b}))
	repo.set(b.Path, "B")
	repo.set("labels.json", `{"attribution":{"b":{"url":"`+directSrv.URL+`/blocked"}}}`)
	direct.del(a.Path)
	direct.setHTML("blocked", "<html>blocked</html>")
	p.Start(func(m, _ *Manifest) {}, true, nil)
	waitFor(t, func() bool { return p.fresh(b) }, "HTML attribution fell back to the repository source")
}

// TestWarmStartShowsCacheBeforeNetwork pins the user-visible promise: a
// warm start's cached library delivers while the network is still
// chewing on the manifest.
func TestWarmStartShowsCacheBeforeNetwork(t *testing.T) {
	repo := newFakeRepo()
	srv := serveRepo(t, repo)
	p := newPack(t, srv.URL)

	a := entry("cat/a", "A")
	repo.set("manifest.json", manifestJSON(t, 1, []File{a}))
	repo.set(a.Path, "A")
	p.Start(func(m, _ *Manifest) {}, true, nil)
	waitFor(t, func() bool { return p.fresh(a) }, "v1 downloaded")

	// A restart into a network that answers slowly.
	repo.setSlow("manifest.json", 1500*time.Millisecond)
	p2, err := openAt(p.dir, syncPost, bgCtx)
	if err != nil {
		t.Fatal(err)
	}
	delivered := make(chan *Manifest, 2)
	start := time.Now()
	p2.Start(func(m, _ *Manifest) { delivered <- m }, true, nil)
	select {
	case m := <-delivered:
		if len(m.Files) != 1 {
			t.Fatalf("cached manifest wrong: %d files", len(m.Files))
		}
		if since := time.Since(start); since > 500*time.Millisecond {
			t.Fatalf("cached manifest delivered only after %v — it raced the network", since)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("cached manifest never delivered")
	}
}

// TestCloudManifestConvertAndCache covers the cloud tier's arrival: the
// wire document converts into the internal manifest shape (titles as
// labels, license/origin as attribution), the copy is cached, and nothing
// downloads until asked. It arrives after the core delivery, in the
// background.
func TestCloudManifestConvertAndCache(t *testing.T) {
	repo := newFakeRepo()
	srv := serveRepo(t, repo)
	p := newPack(t, srv.URL)

	a := entry("cat/a", "A")
	ce := cloudEntry{File: cloudFileEntry("cat/c", "CCC"), title: "丙", license: "CC-BY", origin: "https://example.org/set"}
	repo.set("manifest.json", manifestJSON(t, 1, []File{a}))
	repo.set(a.Path, "A")
	repo.set("cloud-manifest.json", cloudJSON(t, 1, []cloudEntry{ce}))
	repo.set(ce.Path, "CCC")

	p.Start(func(m, _ *Manifest) {}, true, nil)
	waitFor(t, func() bool { return p.fresh(a) }, "core file downloaded")
	waitFor(t, func() bool {
		p.mu.Lock()
		defer p.mu.Unlock()
		return p.cloud != nil
	}, "cloud manifest arrived in the background")

	p.mu.Lock()
	cloud := p.cloud
	p.mu.Unlock()
	if len(cloud.Files) != 1 {
		t.Fatalf("cloud manifest = %+v", cloud)
	}
	if cloud.Labels["c"] != "丙" {
		t.Fatalf("title not converted to a label: %v", cloud.Labels)
	}
	if cloud.Attribution["c"].License != "CC-BY" || cloud.Attribution["c"].Origin != "https://example.org/set" {
		t.Fatalf("attribution not converted: %v", cloud.Attribution)
	}

	// The cloud tier stays on the shelf: bytes only when asked.
	if fileExists(diskPath(p.dir, ce.File)) {
		t.Fatal("cloud sound downloaded although the tier is on-demand")
	}
	if _, err := p.readManifest(p.cloudManifestPath()); err != nil {
		t.Fatalf("cloud manifest not cached: %v", err)
	}
}

// TestCloudOnDemandOnly covers the tier's promise: a full prefetch pass
// never touches a cloud sound, and Ensure fetches it on the spot.
func TestCloudOnDemandOnly(t *testing.T) {
	repo := newFakeRepo()
	srv := serveRepo(t, repo)
	p := newPack(t, srv.URL)

	a := entry("cat/a", "A")
	ce := cloudEntry{File: cloudFileEntry("cat/c", "CCC"), title: "丙"}
	repo.set("manifest.json", manifestJSON(t, 1, []File{a}))
	repo.set(a.Path, "A")
	repo.set("cloud-manifest.json", cloudJSON(t, 1, []cloudEntry{ce}))
	repo.set(ce.Path, "CCC")

	p.Start(func(m, _ *Manifest) {}, true, nil)
	waitFor(t, func() bool { return p.fresh(a) }, "core file downloaded")
	waitFor(t, func() bool {
		p.mu.Lock()
		defer p.mu.Unlock()
		return p.cloud != nil
	}, "cloud manifest arrived")

	if n := repo.hitsFor(ce.Path); n != 0 {
		t.Fatalf("cloud sound requested during prefetch: %d hits", n)
	}
	if p.Cached(ce.ID) {
		t.Fatal("cloud sound cached without a request")
	}

	done := make(chan bool, 1)
	p.Ensure(ce.ID, func(ok bool) { done <- ok })
	if ok := <-done; !ok {
		t.Fatal("Ensure failed for the cloud sound")
	}
	if !p.fresh(ce.File) {
		t.Fatal("cloud sound not stored")
	}
	if n := repo.hitsFor(ce.Path); n == 0 {
		t.Fatal("cloud sound never fetched")
	}
	if state := readState(t, p.dir); state[ce.ID] != ce.SHA256 {
		t.Fatalf("state entry for the cloud sound missing: %v", state)
	}
}

// TestCloudPrune covers retirement: a cloud sound that leaves the cloud
// manifest loses both its file and its state entry.
func TestCloudPrune(t *testing.T) {
	repo := newFakeRepo()
	srv := serveRepo(t, repo)
	p := newPack(t, srv.URL)

	a := entry("cat/a", "A")
	ce := cloudEntry{File: cloudFileEntry("cat/c", "CCC"), title: "丙"}
	next := cloudEntry{File: cloudFileEntry("cat/d", "DDD"), title: "丁"}
	repo.set("manifest.json", manifestJSON(t, 1, []File{a}))
	repo.set(a.Path, "A")
	repo.set("cloud-manifest.json", cloudJSON(t, 1, []cloudEntry{ce}))
	repo.set(ce.Path, "CCC")

	p.Start(func(m, _ *Manifest) {}, true, nil)
	waitFor(t, func() bool { return p.fresh(a) }, "core file downloaded")
	waitFor(t, func() bool {
		p.mu.Lock()
		defer p.mu.Unlock()
		return p.cloud != nil
	}, "cloud manifest arrived")
	done := make(chan bool, 1)
	p.Ensure(ce.ID, func(ok bool) { done <- ok })
	if ok := <-done; !ok {
		t.Fatal("Ensure failed for the cloud sound")
	}

	// v2 swaps the cloud entry out.
	repo.set("cloud-manifest.json", cloudJSON(t, 2, []cloudEntry{next}))
	repo.set(next.Path, "DDD")
	p.Start(func(m, _ *Manifest) {}, true, nil)
	// The state table's atomic rewrite trails the file removal by a
	// moment; wait for both.
	waitFor(t, func() bool {
		if fileExists(diskPath(p.dir, ce.File)) {
			return false
		}
		data, err := os.ReadFile(filepath.Join(p.dir, "state.json"))
		return err == nil && !strings.Contains(string(data), ce.ID)
	}, "retired cloud file and state entry removed")

	if _, ok := readState(t, p.dir)[ce.ID]; ok {
		t.Fatal("state kept the retired cloud sound")
	}
}

// TestCloudBranchSources covers the repository's branch split: the core
// library is served from master, the on-demand tier from the cloud
// branch, and no request crosses the line.
func TestCloudBranchSources(t *testing.T) {
	master, cloudRepo := newFakeRepo(), newFakeRepo()
	masterSrv, cloudSrv := serveRepo(t, master), serveRepo(t, cloudRepo)

	a := entry("cat/a", "A")
	ce := cloudEntry{File: cloudFileEntry("cat/c", "CCC"), title: "丙"}
	master.set("manifest.json", manifestJSON(t, 1, []File{a}))
	master.set(a.Path, "A")
	cloudRepo.set("cloud-manifest.json", cloudJSON(t, 1, []cloudEntry{ce}))
	cloudRepo.set(ce.Path, "CCC")

	old := Sources
	Sources = []source{{raw: masterSrv.URL, cloudRaw: cloudSrv.URL}}
	t.Cleanup(func() { Sources = old })

	p, err := openAt(filepath.Join(t.TempDir(), "pack"), syncPost, bgCtx)
	if err != nil {
		t.Fatal(err)
	}
	clouds := make(chan *Manifest, 4)
	p.Start(func(_, cloud *Manifest) { clouds <- cloud }, true, nil)
	waitFor(t, func() bool { return p.fresh(a) }, "core file downloaded from master")
	waitFor(t, func() bool {
		p.mu.Lock()
		defer p.mu.Unlock()
		return p.cloud != nil
	}, "cloud manifest fetched from the cloud branch")
	if master.hitsFor("cloud-manifest.json") != 0 {
		t.Fatal("master was asked for the cloud manifest")
	}

	done := make(chan bool, 1)
	p.Ensure(ce.ID, func(ok bool) { done <- ok })
	if ok := <-done; !ok {
		t.Fatal("Ensure failed for the cloud sound")
	}
	if cloudRepo.hitsFor(ce.Path) == 0 {
		t.Fatal("cloud file not fetched from the cloud branch")
	}
	if master.hitsFor(ce.Path) != 0 {
		t.Fatal("master was asked for a cloud file")
	}
}

// TestCloudCacheServedWhileFetchFails covers the first-paint promise: a
// cached cloud manifest goes out with the core delivery even when the
// cloud branch cannot be reached, and the failed refresh keeps the cache.
func TestCloudCacheServedWhileFetchFails(t *testing.T) {
	repo := newFakeRepo()
	srv := serveRepo(t, repo)
	p := newPack(t, srv.URL)

	a := entry("cat/a", "A")
	ce := cloudEntry{File: cloudFileEntry("cat/c", "CCC"), title: "丙"}
	repo.set("manifest.json", manifestJSON(t, 1, []File{a}))
	repo.set(a.Path, "A")
	repo.set("cloud-manifest.json", cloudJSON(t, 1, []cloudEntry{ce}))
	repo.set(ce.Path, "CCC")

	p.Start(func(m, _ *Manifest) {}, true, nil)
	waitFor(t, func() bool {
		p.mu.Lock()
		defer p.mu.Unlock()
		return p.cloud != nil
	}, "cloud manifest cached")

	// The cloud branch goes dark; a restart still hands the cache out
	// with the first (core) delivery, and keeps it after the failure.
	repo.del("cloud-manifest.json")
	p2, err := openAt(p.dir, syncPost, bgCtx)
	if err != nil {
		t.Fatal(err)
	}
	clouds := make(chan *Manifest, 4)
	p2.Start(func(_, cloud *Manifest) { clouds <- cloud }, true, nil)
	if cloud := <-clouds; cloud == nil || len(cloud.Files) != 1 {
		t.Fatalf("cached cloud manifest not delivered first: %v", cloud)
	}
	// The prefetch finishing means the failed cloud fetch returned first.
	waitFor(t, func() bool { return p2.fresh(a) }, "core file downloaded")
	p2.mu.Lock()
	still := p2.cloud
	p2.mu.Unlock()
	if still == nil || len(still.Files) != 1 {
		t.Fatal("a failed cloud refresh dropped the cached manifest")
	}
}

// TestCloudOfflineRestore covers an offline restart: both manifests come
// back from the local copies, before any request can answer.
func TestCloudOfflineRestore(t *testing.T) {
	repo := newFakeRepo()
	srv := serveRepo(t, repo)
	p := newPack(t, srv.URL)

	a := entry("cat/a", "A")
	ce := cloudEntry{File: cloudFileEntry("cat/c", "CCC"), title: "丙"}
	repo.set("manifest.json", manifestJSON(t, 1, []File{a}))
	repo.set(a.Path, "A")
	repo.set("cloud-manifest.json", cloudJSON(t, 1, []cloudEntry{ce}))
	repo.set(ce.Path, "CCC")

	p.Start(func(m, _ *Manifest) {}, true, nil)
	waitFor(t, func() bool { return p.fresh(a) }, "core file downloaded")

	// The repository goes dark; a restart still delivers both manifests.
	repo.del("manifest.json")
	repo.del("cloud-manifest.json")
	repo.del(a.Path)
	repo.del(ce.Path)
	p2, err := openAt(p.dir, syncPost, bgCtx)
	if err != nil {
		t.Fatal(err)
	}
	synced := make(chan *Manifest, 4)
	p2.Start(func(m, cloud *Manifest) { synced <- m; synced <- cloud }, true, nil)
	m, cloud := <-synced, <-synced
	if m == nil || cloud == nil || len(cloud.Files) != 1 {
		t.Fatalf("offline manifests = %v, %v", m, cloud)
	}
}
