// Package catalog scans vibe-time's embedded sound library and wallpapers and
// holds the app's fixed data model: categories, sounds and their Chinese
// labels, plus the twelve built-in scene presets.
package catalog

import (
	"io/fs"
	"path"
	"sort"
	"strings"
)

// Sound is one loopable ambient sound file.
type Sound struct {
	// ID is "category/file-base", the stable key presets reference.
	ID string
	// Label is the Chinese name shown in the UI.
	Label string
	// Category is the sound's directory (category ID).
	Category string
	// File is the path inside the audio FS ("category/name.ext").
	File string
	// Cloud marks an on-demand sound: listed by the pack repository's
	// cloud manifest, its bytes download only when picked.
	Cloud bool
}

// Category groups sounds the way the audio library's directories do.
type Category struct {
	ID     string
	Title  string
	Emoji  string
	Sounds []Sound
}

// Attribution records where a sound's bytes came from. The sound pack's
// labels.json carries it per base name — origin and license for a future
// credits view, and an optional direct URL the client races as one more
// download candidate. All fields may be empty.
type Attribution struct {
	Origin  string `json:"origin,omitempty"`
	License string `json:"license,omitempty"`
	URL     string `json:"url,omitempty"`
}

// Catalog is the loaded sound library.
type Catalog struct {
	Cats  []Category
	byID  map[string]Sound
	audio fs.FS
}

// categoryMeta pairs a category directory with its Chinese title and emoji.
type categoryMeta struct{ title, emoji string }

var categoryMetaByID = map[string]categoryMeta{
	"animals":   {"动物", "🐾"},
	"binaural":  {"双耳节拍", "🧠"},
	"nature":    {"自然", "🌿"},
	"noise":     {"噪音", "🌫️"},
	"places":    {"场所", "🏠"},
	"rain":      {"雨声", "🌧️"},
	"things":    {"物品", "🔔"},
	"transport": {"交通", "🚆"},
	"urban":     {"城市", "🏙️"},
}

// soundLabels maps a sound's file base name to its Chinese label.
var soundLabels = map[string]string{
	"beehive":           "蜂巢",
	"birds":             "鸟鸣",
	"cat-purring":       "猫咪呼噜",
	"chickens":          "鸡鸣",
	"cows":              "牛叫",
	"crickets":          "蟋蟀",
	"crows":             "乌鸦",
	"dog-barking":       "狗吠",
	"frog":              "蛙鸣",
	"horse-gallop":      "马蹄",
	"owl":               "猫头鹰",
	"seagulls":          "海鸥",
	"sheep":             "绵羊",
	"whale":             "鲸鱼",
	"wolf":              "狼嚎",
	"woodpecker":        "啄木鸟",
	"binaural-alpha":    "Alpha 波",
	"binaural-beta":     "Beta 波",
	"binaural-delta":    "Delta 波",
	"binaural-gamma":    "Gamma 波",
	"binaural-theta":    "Theta 波",
	"campfire":          "篝火",
	"droplets":          "水滴",
	"howling-wind":      "呼啸风",
	"jungle":            "丛林",
	"river":             "河流",
	"walk-in-snow":      "踏雪",
	"walk-on-gravel":    "踏碎石",
	"walk-on-leaves":    "踏落叶",
	"waterfall":         "瀑布",
	"waves":             "海浪",
	"wind-in-trees":     "林间风",
	"wind":              "风声",
	"brown-noise":       "棕色噪音",
	"pink-noise":        "粉色噪音",
	"white-noise":       "白噪音",
	"airport":           "机场",
	"cafe":              "咖啡厅",
	"carousel":          "旋转木马",
	"church":            "教堂",
	"construction-site": "工地",
	"crowded-bar":       "喧闹酒吧",
	"laboratory":        "实验室",
	"laundry-room":      "洗衣房",
	"library":           "图书馆",
	"night-village":     "夜村",
	"office":            "办公室",
	"restaurant":        "餐厅",
	"subway-station":    "地铁站",
	"supermarket":       "超市",
	"temple":            "寺庙",
	"underwater":        "水下",
	"heavy-rain":        "大雨",
	"light-rain":        "小雨",
	"rain-on-car-roof":  "车顶雨",
	"rain-on-leaves":    "叶上雨",
	"rain-on-tent":      "帐篷雨",
	"rain-on-umbrella":  "伞上雨",
	"rain-on-window":    "窗上雨",
	"thunder":           "雷声",
	"boiling-water":     "沸水",
	"bubbles":           "气泡",
	"ceiling-fan":       "吊扇",
	"clock":             "时钟",
	"dryer":             "烘干机",
	"keyboard":          "键盘",
	"morse-code":        "摩斯电码",
	"paper":             "纸张",
	"singing-bowl":      "颂钵",
	"slide-projector":   "幻灯机",
	"tuning-radio":      "收音机",
	"typewriter":        "打字机",
	"vinyl-effect":      "黑胶唱片",
	"washing-machine":   "洗衣机",
	"wind-chimes":       "风铃",
	"windshield-wipers": "雨刮器",
	"airplane":          "飞机",
	"inside-a-train":    "火车内",
	"rowing-boat":       "划船",
	"sailboat":          "帆船",
	"submarine":         "潜水艇",
	"train":             "火车",
	"ambulance-siren":   "救护车",
	"busy-street":       "繁忙街道",
	"crowd":             "人群",
	"fireworks":         "烟花",
	"highway":           "高速公路",
	"road":              "马路",
	"traffic":           "交通",
}

// LabelOf returns the Chinese label for a sound's base file name, falling
// back to a cleaned-up version of the name.
func LabelOf(base string) string {
	if l, ok := soundLabels[base]; ok {
		return l
	}
	repl := strings.NewReplacer("-", " ", "_", " ")
	return strings.TrimSpace(repl.Replace(base))
}

// CategoryMeta names a sound category, as the sound pack's labels.json
// carries it.
type CategoryMeta struct {
	Title string `json:"title"`
	Emoji string `json:"emoji"`
}

// SeedLabels returns the shipped Chinese sound labels, for seeding the
// sound pack repository's labels.json.
func SeedLabels() map[string]string {
	out := make(map[string]string, len(soundLabels))
	for k, v := range soundLabels {
		out[k] = v
	}
	return out
}

// SeedCategories returns the shipped category metadata, likewise.
func SeedCategories() map[string]CategoryMeta {
	out := make(map[string]CategoryMeta, len(categoryMetaByID))
	for k, v := range categoryMetaByID {
		out[k] = CategoryMeta{Title: v.title, Emoji: v.emoji}
	}
	return out
}

func categoryTitle(id string) string {
	if m, ok := categoryMetaByID[id]; ok {
		return m.title
	}
	return strings.Title(strings.ReplaceAll(id, "-", " "))
}

func categoryEmoji(id string) string {
	if m, ok := categoryMetaByID[id]; ok {
		return m.emoji
	}
	return "🎵"
}

// Load scans an audio FS of "category/file.ext" into a sorted catalog.
func Load(audio fs.FS) *Catalog {
	c := &Catalog{byID: map[string]Sound{}, audio: audio}
	entries, err := fs.ReadDir(audio, ".")
	if err != nil {
		return c
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		catID := e.Name()
		files, err := fs.ReadDir(audio, catID)
		if err != nil {
			continue
		}
		sounds := []Sound{}
		for _, f := range files {
			if f.IsDir() {
				continue
			}
			ext := strings.ToLower(path.Ext(f.Name()))
			if ext != ".mp3" && ext != ".wav" && ext != ".m4a" && ext != ".ogg" {
				continue
			}
			base := strings.TrimSuffix(f.Name(), path.Ext(f.Name()))
			s := Sound{
				ID:       catID + "/" + base,
				Label:    LabelOf(base),
				Category: catID,
				File:     catID + "/" + f.Name(),
			}
			sounds = append(sounds, s)
			c.byID[s.ID] = s
		}
		if len(sounds) == 0 {
			continue
		}
		sort.Slice(sounds, func(i, j int) bool { return sounds[i].Label < sounds[j].Label })
		c.Cats = append(c.Cats, Category{
			ID:     catID,
			Title:  categoryTitle(catID),
			Emoji:  categoryEmoji(catID),
			Sounds: sounds,
		})
	}
	sort.Slice(c.Cats, func(i, j int) bool { return c.Cats[i].Title < c.Cats[j].Title })
	return c
}

// Sound looks a sound up by ID.
func (c *Catalog) Sound(id string) (Sound, bool) {
	s, ok := c.byID[id]
	return s, ok
}

// SoundFromID derives a sound from a pack manifest ID ("category/base"),
// where its label and category metadata follow the naming conventions.
func SoundFromID(id string) Sound {
	cat, base, _ := strings.Cut(id, "/")
	return Sound{ID: id, Label: LabelOf(base), Category: cat}
}

// AddSound merges a runtime-discovered sound (the downloaded pack's
// manifest), adding its category when new and keeping both sorts.
func (c *Catalog) AddSound(s Sound) {
	if s.ID == "" {
		return
	}
	if _, ok := c.byID[s.ID]; ok {
		return
	}
	c.byID[s.ID] = s
	for i := range c.Cats {
		if c.Cats[i].ID == s.Category {
			c.Cats[i].Sounds = append(c.Cats[i].Sounds, s)
			sort.Slice(c.Cats[i].Sounds, func(a, b int) bool {
				return c.Cats[i].Sounds[a].Label < c.Cats[i].Sounds[b].Label
			})
			return
		}
	}
	c.Cats = append(c.Cats, Category{
		ID: s.Category, Title: categoryTitle(s.Category), Emoji: categoryEmoji(s.Category),
		Sounds: []Sound{s},
	})
	sort.Slice(c.Cats, func(a, b int) bool { return c.Cats[a].Title < c.Cats[b].Title })
}

// SyncAll replaces the catalog's sounds wholesale — the downloaded pack's
// manifest is the whole library, so sounds that left the repository drop
// out here and renamed labels land. It returns the IDs that were removed.
func (c *Catalog) SyncAll(sounds []Sound) (removed []string) {
	present := make(map[string]bool, len(sounds))
	for _, s := range sounds {
		present[s.ID] = true
	}
	for id := range c.byID {
		if !present[id] {
			removed = append(removed, id)
		}
	}
	c.byID = map[string]Sound{}
	c.Cats = nil
	for _, s := range sounds {
		c.AddSound(s)
	}
	return removed
}

// SetCategoryMeta renames a category (and its emoji), re-sorting the list;
// unknown IDs are ignored. The sound pack's labels.json drives it.
func (c *Catalog) SetCategoryMeta(id, title, emoji string) {
	if title == "" && emoji == "" {
		return
	}
	for i := range c.Cats {
		if c.Cats[i].ID != id {
			continue
		}
		if title != "" {
			c.Cats[i].Title = title
		}
		if emoji != "" {
			c.Cats[i].Emoji = emoji
		}
		sort.Slice(c.Cats, func(a, b int) bool { return c.Cats[a].Title < c.Cats[b].Title })
		return
	}
}

// Empty is a catalog with nothing in it: the app starts here, and the
// downloaded sound pack's manifest fills it in.
func Empty() *Catalog {
	return &Catalog{byID: map[string]Sound{}}
}

// Data reads a sound's bytes by ID for playback.
func (c *Catalog) Data(id string) ([]byte, error) {
	s, ok := c.byID[id]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return fs.ReadFile(c.audio, s.File)
}

// Total is the number of sounds in the catalog.
func (c *Catalog) Total() int { return len(c.byID) }

// Wallpaper is one focus-mode background image.
type Wallpaper struct {
	Name string // base name, for display
	File string // path inside the wallpaper FS
}

// Wallpapers lists the embedded wallpapers sorted by name.
func Wallpapers(walls fs.FS) []Wallpaper {
	out := []Wallpaper{}
	entries, err := fs.ReadDir(walls, ".")
	if err != nil {
		return out
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(path.Ext(e.Name()))
		if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" {
			continue
		}
		base := strings.TrimSuffix(e.Name(), path.Ext(e.Name()))
		out = append(out, Wallpaper{Name: base, File: e.Name()})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
