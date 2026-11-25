// Command packgen builds the vibe-time sound pack repository: it mirrors
// a source sound library (<category>/<file>.m4a) into <repo>/sounds/,
// prunes pack files the source no longer has, and writes <repo>/manifest.json
// with every file's size and SHA-256.
//
// The manifest is what the app fetches at runtime; the client downloads
// files per the manifest and verifies them against the hash. Run it again
// whenever the sound library changes:
//
//	go run ./cmd/packgen -src assets/audio -repo ../vibe-time-assets
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"vibe-time/internal/catalog"
)

// manifestFile is the pack's index, fetched by the app at startup.
type manifestFile struct {
	ID    string `json:"id"`    // "category/file-base", the app's sound ID
	Path  string `json:"path"`  // path inside the repo, "sounds/..."
	Size  int64  `json:"size"`  // bytes
	SHA56 string `json:"sha256"`
}

type manifest struct {
	Version int           `json:"version"` // bumped when contents change
	Files   []manifestFile `json:"files"`
}

func main() {
	src := flag.String("src", "assets/audio", "source sound library: directories of <category>/<file>")
	repo := flag.String("repo", "", "pack repository directory (creates <repo>/sounds and <repo>/manifest.json)")
	version := flag.Int("version", 0, "manifest version; 0 bumps from the previous when contents changed")
	flag.Parse()
	if *src == "" || *repo == "" {
		flag.Usage()
		os.Exit(2)
	}

	// Scan the source into manifest entries, in a stable order.
	var files []manifestFile
	err := filepath.WalkDir(*src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".mp3" && ext != ".wav" && ext != ".m4a" && ext != ".ogg" {
			return nil
		}
		rel, err := filepath.Rel(*src, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		base := strings.TrimSuffix(rel, filepath.Ext(rel))
		sum, size, err := hashFile(path)
		if err != nil {
			return err
		}
		files = append(files, manifestFile{ID: base, Path: "sounds/" + rel, Size: size, SHA56: sum})
		return nil
	})
	if err != nil {
		fatal(err)
	}
	if len(files) == 0 {
		fatal(fmt.Errorf("no sound files under %s", *src))
	}
	sort.Slice(files, func(i, j int) bool { return files[i].ID < files[j].ID })

	// Mirror the files into <repo>/sounds, and collect what that mirror
	// holds so stale copies can go.
	soundsDir := filepath.Join(*repo, "sounds")
	if err := os.MkdirAll(soundsDir, 0o755); err != nil {
		fatal(err)
	}
	// keep is keyed by path relative to soundsDir, as pruneDir walks it.
	keep := map[string]bool{}
	var total int64
	for _, f := range files {
		keep[strings.TrimPrefix(f.Path, "sounds/")] = true
		total += f.Size
		dst := filepath.Join(*repo, filepath.FromSlash(f.Path))
		if err := copyIfChanged(filepath.Join(*src, filepath.FromSlash(strings.TrimPrefix(f.Path, "sounds/"))), dst, f.SHA56); err != nil {
			fatal(err)
		}
	}
	pruned := pruneDir(soundsDir, keep)

	// Version: the flag's, else keep the old one when nothing changed and
	// bump it when something did.
	next := *version
	if next == 0 {
		next = 1
		if old, err := readManifestVersion(filepath.Join(*repo, "manifest.json")); err == nil {
			next = old
			if pruned > 0 || manifestChanged(files, *repo) {
				next = old + 1
			}
		}
	}

	out, err := json.MarshalIndent(manifest{Version: next, Files: files}, "", "  ")
	if err != nil {
		fatal(err)
	}
	if err := os.WriteFile(filepath.Join(*repo, "manifest.json"), out, 0o644); err != nil {
		fatal(err)
	}
	unlabeled := writeLabels(*repo, files)
	fmt.Printf("声音包生成完成：%d 个文件，%.1f MB，version %d（清理 %d 个旧文件，%d 个待命名）\n→ %s\n",
		len(files), float64(total)/1024/1024, next, pruned, unlabeled, *repo)
}

// labelsFile is the hand-editable name mapping the app reads at runtime.
type labelsFile struct {
	Categories map[string]catalog.CategoryMeta `json:"categories"`
	Labels     map[string]string               `json:"labels"`
}

// writeLabels keeps <repo>/labels.json in step with the files: existing
// entries stay (they are hand-edited), missing labels are seeded from the
// app's shipped names, and entries for files that left are dropped. It
// returns how many labels fell back to a derived name, for the user to
// name properly.
func writeLabels(repo string, files []manifestFile) int {
	var lf labelsFile
	if data, err := os.ReadFile(filepath.Join(repo, "labels.json")); err == nil {
		json.Unmarshal(data, &lf)
	}
	if lf.Categories == nil {
		lf.Categories = map[string]catalog.CategoryMeta{}
	}
	if lf.Labels == nil {
		lf.Labels = map[string]string{}
	}
	for id, meta := range catalog.SeedCategories() {
		if _, ok := lf.Categories[id]; !ok {
			lf.Categories[id] = meta
		}
	}
	seed := catalog.SeedLabels()
	bases := map[string]bool{}
	unlabeled := 0
	for _, f := range files {
		base := f.ID[strings.LastIndex(f.ID, "/")+1:]
		cat := f.ID[:strings.LastIndex(f.ID, "/")]
		bases[base] = true
		if _, ok := lf.Labels[base]; !ok {
			if s, ok := seed[base]; ok {
				lf.Labels[base] = s
			} else {
				lf.Labels[base] = catalog.LabelOf(base)
				unlabeled++
			}
		}
		if cat == "" {
			continue
		}
		if _, ok := lf.Categories[cat]; !ok {
			lf.Categories[cat] = catalog.CategoryMeta{Title: cat, Emoji: ""}
		}
	}
	for base := range lf.Labels {
		if !bases[base] {
			delete(lf.Labels, base)
		}
	}
	out, err := json.MarshalIndent(lf, "", "  ")
	if err != nil {
		fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "labels.json"), out, 0o644); err != nil {
		fatal(err)
	}
	return unlabeled
}

// hashFile returns a file's SHA-256 hex digest and size.
func hashFile(path string) (string, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()
	h := sha256.New()
	size, err := io.Copy(h, f)
	if err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(h.Sum(nil)), size, nil
}

// copyIfChanged copies src to dst unless dst already holds the same bytes.
func copyIfChanged(src, dst, wantSum string) error {
	if sum, _, err := hashFile(dst); err == nil && sum == wantSum {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	tmp := dst + ".tmp"
	out, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(tmp)
		return err
	}
	if err := out.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, dst)
}

// pruneDir removes files under root whose repo-relative path is not in keep,
// and empty directories they leave. It returns how many files went.
func pruneDir(root string, keep map[string]bool) int {
	n := 0
	paths := []string{}
	filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			paths = append(paths, path)
		}
		return nil
	})
	sort.Sort(sort.Reverse(sort.StringSlice(paths))) // deepest first
	for _, p := range paths {
		rel, _ := filepath.Rel(root, p)
		if keep[filepath.ToSlash(rel)] {
			continue
		}
		if os.Remove(p) == nil {
			n++
		}
	}
	// Empty category directories.
	dirs := []string{}
	filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err == nil && d.IsDir() {
			dirs = append(dirs, path)
		}
		return nil
	})
	sort.Sort(sort.Reverse(sort.StringSlice(dirs)))
	for _, d := range dirs {
		os.Remove(d) // fails when not empty; that is the point
	}
	return n
}

// manifestChanged reports whether the pack repo's manifest lists exactly
// these files (by id, path, size and hash).
func manifestChanged(files []manifestFile, repo string) bool {
	data, err := os.ReadFile(filepath.Join(repo, "manifest.json"))
	if err != nil {
		return true
	}
	var m manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return true
	}
	if len(m.Files) != len(files) {
		return true
	}
	for i := range files {
		if m.Files[i] != files[i] {
			return true
		}
	}
	return false
}

// readManifestVersion reads just the version off an existing manifest.
func readManifestVersion(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	var m manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return 0, err
	}
	return m.Version, nil
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "packgen:", err)
	os.Exit(1)
}
