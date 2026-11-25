package catalog

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Preset is a scene: a named set of sounds with per-sound volumes.
type Preset struct {
	// ID identifies the preset; builtins use "__name", customs "custom_<n>".
	ID string `json:"id"`
	// Name is the display name.
	Name string `json:"name"`
	// Icon is an emoji.
	Icon string `json:"icon"`
	// Desc is the short description (defaults to the joined sound labels).
	Desc string `json:"desc"`
	// Builtin marks the twelve shipped presets; only customs are editable
	// or deletable.
	Builtin bool `json:"builtin,omitempty"`
	// Sounds maps sound IDs to volumes in [0.05, 1].
	Sounds map[string]float64 `json:"sounds"`
}

// Clone returns a deep copy of the preset.
func (p Preset) Clone() Preset {
	sounds := make(map[string]float64, len(p.Sounds))
	for k, v := range p.Sounds {
		sounds[k] = v
	}
	p.Sounds = sounds
	return p
}

// builtinPresets are the twelve shipped scenes (ported from the Vue app).
var builtinPresets = []Preset{
	{ID: "__rainy_night", Name: "雨夜", Icon: "🌧", Desc: "雨声 + 雷声 + 风", Builtin: true, Sounds: map[string]float64{
		"rain/heavy-rain": 0.7, "rain/thunder": 0.4, "nature/wind": 0.3}},
	{ID: "__deep_focus", Name: "深度专注", Icon: "🧠", Desc: "棕色噪音 + 咖啡厅 + 键盘", Builtin: true, Sounds: map[string]float64{
		"noise/brown-noise": 0.5, "places/cafe": 0.4, "things/keyboard": 0.2}},
	{ID: "__forest_morning", Name: "森林清晨", Icon: "🌲", Desc: "鸟鸣 + 溪流 + 林间风", Builtin: true, Sounds: map[string]float64{
		"animals/birds": 0.6, "nature/river": 0.4, "nature/wind-in-trees": 0.3}},
	{ID: "__ocean_breeze", Name: "海边微风", Icon: "🌊", Desc: "海浪 + 海鸥 + 风", Builtin: true, Sounds: map[string]float64{
		"nature/waves": 0.6, "animals/seagulls": 0.3, "nature/wind": 0.35}},
	{ID: "__campfire", Name: "篝火之夜", Icon: "🔥", Desc: "篝火 + 蟋蟀 + 猫头鹰", Builtin: true, Sounds: map[string]float64{
		"nature/campfire": 0.6, "animals/crickets": 0.4, "animals/owl": 0.25}},
	{ID: "__city_night", Name: "城市深夜", Icon: "🌃", Desc: "交通 + 远处人群 + 小雨", Builtin: true, Sounds: map[string]float64{
		"urban/traffic": 0.35, "urban/crowd": 0.2, "rain/light-rain": 0.5}},
	{ID: "__meditation", Name: "冥想", Icon: "🧘", Desc: "颂钵 + Alpha 波 + 风铃", Builtin: true, Sounds: map[string]float64{
		"things/singing-bowl": 0.5, "binaural/binaural-alpha": 0.4, "things/wind-chimes": 0.2}},
	{ID: "__study_room", Name: "自习室", Icon: "📚", Desc: "图书馆 + 时钟 + 翻书", Builtin: true, Sounds: map[string]float64{
		"places/library": 0.5, "things/clock": 0.2, "things/paper": 0.25}},
	{ID: "__cozy_home", Name: "温馨小屋", Icon: "🏠", Desc: "吊扇 + 猫咪 + 洗衣机", Builtin: true, Sounds: map[string]float64{
		"things/ceiling-fan": 0.4, "animals/cat-purring": 0.5, "things/washing-machine": 0.2}},
	{ID: "__train_journey", Name: "火车旅行", Icon: "🚂", Desc: "火车内 + 窗上雨声 + 键盘", Builtin: true, Sounds: map[string]float64{
		"transport/inside-a-train": 0.5, "rain/rain-on-window": 0.4, "things/keyboard": 0.15}},
	{ID: "__snow_day", Name: "下雪天", Icon: "❄", Desc: "踏雪 + 风声 + 篝火", Builtin: true, Sounds: map[string]float64{
		"nature/walk-in-snow": 0.4, "nature/wind": 0.35, "nature/campfire": 0.5}},
	{ID: "__sleep", Name: "助眠", Icon: "🌙", Desc: "粉色噪音 + 小雨 + 蛙鸣", Builtin: true, Sounds: map[string]float64{
		"noise/pink-noise": 0.5, "rain/light-rain": 0.45, "animals/frog": 0.2}},
}

// BuiltinPresets returns copies of the twelve shipped scenes.
func BuiltinPresets() []Preset {
	out := make([]Preset, len(builtinPresets))
	for i, p := range builtinPresets {
		out[i] = p.Clone()
	}
	return out
}

// presetFile is the JSON persisted under the user's config directory.
type presetFile struct {
	Custom    []Preset                      `json:"custom"`
	Overrides map[string]map[string]float64 `json:"builtin_overrides"`
}

// Store persists custom presets and built-in volume overrides as JSON in the
// user's config directory (replacing the web app's localStorage).
type Store struct {
	path      string
	Custom    []Preset
	Overrides map[string]map[string]float64
}

// OpenStore loads (or creates) the preset store at
// <user config dir>/vibe-time/presets.json.
func OpenStore() (*Store, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	return openStore(filepath.Join(dir, "vibe-time", "presets.json"))
}

// openStore loads (or creates) the preset store at path.
func openStore(path string) (*Store, error) {
	s := &Store{
		path:      path,
		Overrides: map[string]map[string]float64{},
	}
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	var f presetFile
	if err := json.Unmarshal(data, &f); err != nil {
		// A corrupt file should not take the app down; start fresh.
		return s, nil
	}
	if f.Custom != nil {
		s.Custom = f.Custom
	}
	if f.Overrides != nil {
		s.Overrides = f.Overrides
	}
	return s, nil
}

// Save writes the store atomically (temp file + rename).
func (s *Store) Save() error {
	f := presetFile{Custom: s.Custom, Overrides: s.Overrides}
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return err
	}
	return nil
}

// Override returns the saved volume override for a sound in a preset.
func (s *Store) Override(presetID, soundID string) (float64, bool) {
	m, ok := s.Overrides[presetID]
	if !ok {
		return 0, false
	}
	v, ok := m[soundID]
	return v, ok
}

// SetOverride records a volume override for a sound in a built-in preset.
func (s *Store) SetOverride(presetID, soundID string, v float64) {
	m, ok := s.Overrides[presetID]
	if !ok {
		m = map[string]float64{}
		s.Overrides[presetID] = m
	}
	m[soundID] = v
}

// FindCustom returns the index of a custom preset by ID, or -1.
func (s *Store) FindCustom(id string) int {
	for i, p := range s.Custom {
		if p.ID == id {
			return i
		}
	}
	return -1
}

// NextCustomID returns a fresh custom preset ID.
func (s *Store) NextCustomID() string {
	return fmt.Sprintf("custom_%d", time.Now().UnixMilli())
}
