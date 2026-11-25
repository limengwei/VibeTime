package catalog

import "testing"

// referenceCode was checksummed independently (DJB2 over UTF-16 units,
// base36) to stay byte-compatible with the web app's recipeCode.js. Sounds
// encode in sorted ID order (Go maps have no insertion order); decoding
// accepts any order.
const referenceCode = "VIBE:雨夜:🌧:雨声_雷声_风:nature/wind|30+rain/heavy-rain|70+rain/thunder|40+H3"

func TestEncodeRecipeReferenceVector(t *testing.T) {
	p := Preset{
		Name: "雨夜",
		Icon: "🌧",
		Desc: "雨声+雷声+风", // separators are sanitized on encode
		Sounds: map[string]float64{
			"rain/heavy-rain": 0.7,
			"rain/thunder":    0.4,
			"nature/wind":     0.3,
		},
	}
	if got := EncodeRecipe(p); got != referenceCode {
		t.Fatalf("EncodeRecipe = %q, want %q", got, referenceCode)
	}
}

func TestRecipeRoundtrip(t *testing.T) {
	p := Preset{
		Name: "海边",
		Icon: "🌊",
		Desc: "海浪 + 海鸥",
		Sounds: map[string]float64{
			"nature/waves":            0.6,
			"animals/seagulls":        0.3,
			"nature/wind":             0.35,
			"binaural/binaural-alpha": 0.25,
		},
	}
	got, ok := DecodeRecipe(EncodeRecipe(p))
	if !ok {
		t.Fatal("decode failed")
	}
	if got.Name != "海边" {
		t.Fatalf("name = %q, want 海边", got.Name)
	}
	if got.Icon != "🌊" {
		t.Fatalf("icon = %q", got.Icon)
	}
	for id, v := range p.Sounds {
		if dv, ok := got.Sounds[id]; !ok || dv != v {
			t.Errorf("sound %q = %v, want %v", id, dv, v)
		}
	}
	if len(got.Sounds) != len(p.Sounds) {
		t.Fatalf("sound count = %d, want %d", len(got.Sounds), len(p.Sounds))
	}
}

func TestDecodeRecipeRejects(t *testing.T) {
	bad := []string{
		"",
		"nonsense",
		"VIBE:",
		referenceCode + "0",            // trailing junk breaks checksum
		"VIBE:名:🎵::rain/thunder|40+ZZ", // bad checksum
		"VIBE:名:🎵::bogus|nope+J0",      // volume not a number
	}
	for _, s := range bad {
		if _, ok := DecodeRecipe(s); ok {
			t.Errorf("DecodeRecipe(%q) accepted", s)
		}
	}
}
