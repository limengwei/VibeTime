package audio

import "testing"

func TestEffectiveVolume(t *testing.T) {
	cases := []struct {
		track, master, want float64
	}{
		{0.05, 1, 0.05},    // minimum slider value at full master
		{1, 1, 1},          // full everything
		{0.5, 0.8, 0.4},    // ordinary scaling
		{0.5, 0, 0.01},     // zero master never reaches SetVolume's 0-is-1
		{0.05, 0.01, 0.01}, // tiny products clamp up
		{2, 1, 1},          // out-of-range clamps down
	}
	for _, c := range cases {
		if got := EffectiveVolume(c.track, c.master); got != c.want {
			t.Errorf("EffectiveVolume(%v, %v) = %v, want %v", c.track, c.master, got, c.want)
		}
	}
}
