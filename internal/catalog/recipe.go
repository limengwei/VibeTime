package catalog

import (
	"math"
	"strconv"
	"strings"
	"unicode/utf16"
)

// recipePrefix starts every shareable recipe code.
const recipePrefix = "VIBE:"

// EncodeRecipe renders a preset as a shareable recipe code:
//
//	VIBE:NAME:ICON:DESC:soundID|vol+soundID|vol+CHK
//
// The checksum is DJB2 over the payload's UTF-16 code units (as the web
// app's charCodeAt loop does) rendered as up to two base-36 characters, so
// codes interoperate with the Vue app's recipeCode.js.
func EncodeRecipe(p Preset) string {
	if len(p.Sounds) == 0 {
		return ""
	}
	name := sanitizeRecipeField(p.Name, 30)
	if name == "" {
		name = "未命名"
	}
	icon := p.Icon
	if icon == "" {
		icon = "🎵"
	}
	desc := sanitizeRecipeField(p.Desc, 50)
	parts := make([]string, 0, len(p.Sounds))
	for _, id := range sortedSoundIDs(p.Sounds) {
		v := p.Sounds[id]
		vol := int(math.Round(math.Min(math.Max(v, 0), 1) * 100))
		parts = append(parts, id+"|"+strconv.Itoa(vol))
	}
	payload := name + ":" + icon + ":" + desc + ":" + strings.Join(parts, "+")
	return recipePrefix + payload + "+" + recipeChecksum(payload)
}

// DecodeRecipe parses a recipe code back into a preset (without an ID).
// It returns false when the code is malformed or its checksum fails.
func DecodeRecipe(code string) (Preset, bool) {
	trimmed := strings.TrimSpace(code)
	inner, ok := strings.CutPrefix(trimmed, recipePrefix)
	if !ok {
		return Preset{}, false
	}
	lastPlus := strings.LastIndex(inner, "+")
	if lastPlus < 0 {
		return Preset{}, false
	}
	payload, checksum := inner[:lastPlus], inner[lastPlus+1:]
	if recipeChecksum(payload) != checksum {
		return Preset{}, false
	}

	first := strings.IndexByte(payload, ':')
	if first < 0 {
		return Preset{}, false
	}
	second := strings.IndexByte(payload[first+1:], ':')
	if second < 0 {
		return Preset{}, false
	}
	second += first + 1
	third := strings.IndexByte(payload[second+1:], ':')
	if third < 0 {
		return Preset{}, false
	}
	third += second + 1

	name := payload[:first]
	if name == "" {
		name = "未命名"
	}
	icon := payload[first+1 : second]
	if icon == "" {
		icon = "🎵"
	}
	desc := payload[second+1 : third]

	sounds := map[string]float64{}
	for _, entry := range strings.Split(payload[third+1:], "+") {
		bar := strings.LastIndexByte(entry, '|')
		if bar < 0 {
			continue
		}
		id := entry[:bar]
		vol, err := strconv.Atoi(entry[bar+1:])
		if err != nil || vol < 0 || vol > 100 || id == "" {
			continue
		}
		sounds[id] = float64(vol) / 100
	}
	if len(sounds) == 0 {
		return Preset{}, false
	}
	return Preset{Name: name, Icon: icon, Desc: desc, Sounds: sounds}, true
}

// sanitizeRecipeField strips the code's separators and truncates by runes.
func sanitizeRecipeField(s string, max int) string {
	repl := strings.NewReplacer(":", "_", "+", "_", "|", "_")
	s = repl.Replace(s)
	runes := []rune(s)
	if len(runes) > max {
		runes = runes[:max]
	}
	return string(runes)
}

// sortedSoundIDs gives sounds a stable code order.
func sortedSoundIDs(sounds map[string]float64) []string {
	ids := make([]string, 0, len(sounds))
	for id := range sounds {
		ids = append(ids, id)
	}
	// small n: insertion sort keeps it dependency-free
	for i := 1; i < len(ids); i++ {
		for j := i; j > 0 && ids[j] < ids[j-1]; j-- {
			ids[j], ids[j-1] = ids[j-1], ids[j]
		}
	}
	return ids
}

// recipeChecksum is DJB2 over UTF-16 units masked to 31 bits, as the web
// app computes it, formatted as (up to) two uppercase base-36 characters.
func recipeChecksum(s string) string {
	hash := uint32(5381)
	for _, u := range utf16.Encode([]rune(s)) {
		hash = (hash<<5 + hash) + uint32(u)
		hash &= 0x7fffffff
	}
	b36 := strconv.FormatUint(uint64(hash), 36)
	return strings.ToUpper(b36[:min(2, len(b36))])
}
