package catalog

import (
	"os"
	"path/filepath"
	"testing"
)

// realAudio points the tests at the shipped sound library.
func realAudio(t *testing.T) *Catalog {
	t.Helper()
	dir := filepath.Join("..", "..", "frontend", "public", "audio")
	if _, err := os.Stat(dir); err != nil {
		t.Skipf("sound library not present: %v", err)
	}
	return Load(os.DirFS(dir))
}

func TestCatalogCounts(t *testing.T) {
	c := realAudio(t)
	if got := len(c.Cats); got != 9 {
		t.Fatalf("categories = %d, want 9", got)
	}
	if got := c.Total(); got != 89 {
		t.Fatalf("sounds = %d, want 89", got)
	}
}

func TestCatalogLabelsAndSorting(t *testing.T) {
	c := realAudio(t)
	for _, cat := range c.Cats {
		if cat.Title == "" || cat.Emoji == "" {
			t.Errorf("category %q missing title or emoji", cat.ID)
		}
		for i, s := range cat.Sounds {
			if s.Label == "" {
				t.Errorf("%s has an empty label", s.ID)
			}
			if i > 0 && cat.Sounds[i-1].Label > s.Label {
				t.Errorf("category %q not sorted by label", cat.ID)
			}
		}
	}
	for i, cat := range c.Cats {
		if i > 0 && c.Cats[i-1].Title > cat.Title {
			t.Fatal("categories not sorted by title")
		}
	}
}

func TestBuiltinPresetsReferenceExistingSounds(t *testing.T) {
	c := realAudio(t)
	presets := BuiltinPresets()
	if len(presets) != 12 {
		t.Fatalf("built-in presets = %d, want 12", len(presets))
	}
	for _, p := range presets {
		if !p.Builtin {
			t.Errorf("preset %q not marked builtin", p.ID)
		}
		if p.Name == "" || p.Icon == "" {
			t.Errorf("preset %q missing name or icon", p.ID)
		}
		for id, v := range p.Sounds {
			if _, ok := c.Sound(id); !ok {
				t.Errorf("preset %q references unknown sound %q", p.Name, id)
			}
			if v < 0.05 || v > 1 {
				t.Errorf("preset %q sound %q volume %v out of range", p.Name, id, v)
			}
		}
	}
}

func TestStoreRoundtrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "presets.json")
	s, err := openStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Custom) != 0 || len(s.Overrides) != 0 {
		t.Fatal("a fresh store should be empty")
	}
	s.Custom = []Preset{{ID: "custom_1", Name: "测试", Icon: "🎵", Sounds: map[string]float64{"rain/thunder": 0.4}}}
	s.SetOverride("__rainy_night", "rain/heavy-rain", 0.9)
	if err := s.Save(); err != nil {
		t.Fatal(err)
	}
	got, err := openStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Custom) != 1 || got.Custom[0].Name != "测试" || got.Custom[0].Sounds["rain/thunder"] != 0.4 {
		t.Fatalf("custom presets lost: %+v", got.Custom)
	}
	if v, ok := got.Override("__rainy_night", "rain/heavy-rain"); !ok || v != 0.9 {
		t.Fatalf("override lost: %v %v", v, ok)
	}
	if _, ok := got.Override("__rainy_night", "rain/thunder"); ok {
		t.Fatal("unexpected override present")
	}
}

func TestStoreCorruptFileStartsFresh(t *testing.T) {
	path := filepath.Join(t.TempDir(), "presets.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := openStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Custom) != 0 {
		t.Fatal("a corrupt file should start fresh")
	}
}

func TestWallpapers(t *testing.T) {
	dir := filepath.Join("..", "..", "frontend", "public", "wallpapers")
	if _, err := os.Stat(dir); err != nil {
		t.Skipf("wallpapers not present: %v", err)
	}
	ws := Wallpapers(os.DirFS(dir))
	if len(ws) != 8 {
		t.Fatalf("wallpapers = %d, want 8", len(ws))
	}
}
