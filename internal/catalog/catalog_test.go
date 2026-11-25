package catalog

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

// stubFS maps a sound library of IDs onto stub files; Load only reads the
// directory, so the contents never matter.
func stubFS(ids ...string) fstest.MapFS {
	fs := fstest.MapFS{}
	for _, id := range ids {
		fs[id+".m4a"] = &fstest.MapFile{}
	}
	return fs
}

// builtinFS stubs exactly the built-in scenes' sounds, as the sound pack
// will list them.
func builtinFS() fstest.MapFS {
	var ids []string
	for _, p := range builtinPresets {
		for id := range p.Sounds {
			ids = append(ids, id)
		}
	}
	return stubFS(ids...)
}

func TestCatalogLoadAndBuiltinPresets(t *testing.T) {
	c := Load(builtinFS())
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
			s, ok := c.Sound(id)
			if !ok {
				t.Errorf("preset %q references unknown sound %q", p.ID, id)
				continue
			}
			if s.Label == "" {
				t.Errorf("%s has an empty label", id)
			}
			if v < 0.05 || v > 1 {
				t.Errorf("preset %q sound %q volume %v out of range", p.ID, id, v)
			}
		}
	}
	for _, cat := range c.Cats {
		if cat.Title == "" || cat.Emoji == "" {
			t.Errorf("category %q missing title or emoji", cat.ID)
		}
		for i := 1; i < len(cat.Sounds); i++ {
			if cat.Sounds[i-1].Label > cat.Sounds[i].Label {
				t.Errorf("category %q not sorted by label", cat.ID)
			}
		}
	}
	for i := 1; i < len(c.Cats); i++ {
		if c.Cats[i-1].Title > c.Cats[i].Title {
			t.Fatal("categories not sorted by title")
		}
	}
}

func TestAddSoundMergesAndSorts(t *testing.T) {
	c := Empty()
	c.AddSound(SoundFromID("nature/river"))
	c.AddSound(SoundFromID("nature/campfire"))
	c.AddSound(SoundFromID("urban/traffic"))
	if got := c.Total(); got != 3 {
		t.Fatalf("sounds = %d, want 3", got)
	}
	var nature *Category
	for i := range c.Cats {
		if c.Cats[i].ID == "nature" {
			nature = &c.Cats[i]
		}
	}
	if nature == nil || len(nature.Sounds) != 2 {
		t.Fatalf("nature category missing: %+v", c.Cats)
	}
	// 河流 (U+6CB3) sorts before 篝火 (U+7BDD) by label.
	if nature.Sounds[0].ID != "nature/river" {
		t.Errorf("nature not sorted by label: %+v", nature.Sounds)
	}
	// A repeated add is a no-op.
	c.AddSound(SoundFromID("nature/river"))
	if c.Total() != 3 {
		t.Errorf("repeated add duplicated: %d", c.Total())
	}
	// Metadata renames a category.
	c.SetCategoryMeta("urban", "城市街道", "🛣")
	for _, cat := range c.Cats {
		if cat.ID == "urban" && (cat.Title != "城市街道" || cat.Emoji != "🛣") {
			t.Errorf("SetCategoryMeta did not apply: %+v", cat)
		}
	}
	c.SetCategoryMeta("missing", "无", "")
	if c.Total() != 3 {
		t.Error("SetCategoryMeta must not create sounds")
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
	dir := filepath.Join("..", "..", "assets", "wallpapers")
	if _, err := os.Stat(dir); err != nil {
		t.Skipf("wallpapers not present: %v", err)
	}
	ws := Wallpapers(os.DirFS(dir))
	if len(ws) != 8 {
		t.Fatalf("wallpapers = %d, want 8", len(ws))
	}
}
