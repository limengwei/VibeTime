package main

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"fyne.io/systray"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx context.Context
}

type SoundInfo struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Category string `json:"category"`
	Src      string `json:"src"`
}

type CategoryInfo struct {
	ID     string      `json:"id"`
	Title  string      `json:"title"`
	Sounds []SoundInfo `json:"sounds"`
}

type WallpaperInfo struct {
	Name string `json:"name"`
	File string `json:"file"`
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) ShowWindow() {
	if a.ctx != nil {
		runtime.WindowSetAlwaysOnTop(a.ctx, true)
		runtime.WindowShow(a.ctx)
		runtime.WindowSetAlwaysOnTop(a.ctx, false)
	}
}

func (a *App) HideWindow() {
	if a.ctx != nil {
		runtime.WindowHide(a.ctx)
	}
}

func (a *App) QuitApp() {
	systray.Quit()
	os.Exit(0)
}

func (a *App) getFS() fs.FS {
	return os.DirFS(getAssetsDir())
}

func (a *App) ListSounds() []CategoryInfo {
	categories := []CategoryInfo{}

	fsys := a.getFS()

	audioDir := "audio"
	entries, err := fs.ReadDir(fsys, audioDir)
	if err != nil {
		return categories
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		catID := entry.Name()
		catTitle := categoryTitle(catID)

		soundFiles, err := fs.ReadDir(fsys, audioDir+"/"+catID)
		if err != nil {
			continue
		}

		sounds := []SoundInfo{}
		for _, sf := range soundFiles {
			if sf.IsDir() {
				continue
			}
			ext := strings.ToLower(filepath.Ext(sf.Name()))
			if ext != ".mp3" && ext != ".wav" && ext != ".m4a" && ext != ".ogg" {
				continue
			}
			baseName := strings.TrimSuffix(sf.Name(), filepath.Ext(sf.Name()))
			soundID := catID + "/" + baseName
			label := soundLabel(baseName)
			sounds = append(sounds, SoundInfo{
				ID:       soundID,
				Label:    label,
				Category: catID,
				Src:      "/audio/" + catID + "/" + sf.Name(),
			})
		}

		if len(sounds) > 0 {
			sort.Slice(sounds, func(i, j int) bool {
				return sounds[i].Label < sounds[j].Label
			})
			categories = append(categories, CategoryInfo{
				ID:     catID,
				Title:  catTitle,
				Sounds: sounds,
			})
		}
	}

	sort.Slice(categories, func(i, j int) bool {
		return categories[i].Title < categories[j].Title
	})

	return categories
}

func (a *App) ListWallpapers() []WallpaperInfo {
	wallpapers := []WallpaperInfo{}

	fsys := a.getFS()

	entries, err := fs.ReadDir(fsys, "wallpapers")
	if err != nil {
		return wallpapers
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" {
			continue
		}
		baseName := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		wallpapers = append(wallpapers, WallpaperInfo{
			Name: baseName,
			File: entry.Name(),
		})
	}

	sort.Slice(wallpapers, func(i, j int) bool {
		return wallpapers[i].Name < wallpapers[j].Name
	})

	return wallpapers
}

func categoryTitle(id string) string {
	titles := map[string]string{
		"animals":   "动物",
		"binaural":  "双耳节拍",
		"nature":    "自然",
		"noise":     "噪音",
		"places":    "场所",
		"rain":      "雨声",
		"things":    "物品",
		"transport": "交通",
		"urban":     "城市",
	}
	if t, ok := titles[id]; ok {
		return t
	}
	return strings.Title(id)
}

func soundLabel(filename string) string {
	labels := map[string]string{
		"beehive":           "蜂巢",
		"birds":             "鸟鸣",
		"cat-purring":       "猫咪呼噜",
		"chickens":          "鸡鸣",
		"cows":              "牛叫",
		"crickets":          "蟋蟀",
		"crows":             "乌鸦",
		"dog-barking":       "狗吠",
		"frog":              "青蛙",
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
		"construction-site": "建筑工地",
		"crowded-bar":       "喧闹酒吧",
		"laboratory":        "实验室",
		"laundry-room":      "洗衣房",
		"library":           "图书馆",
		"night-village":     "夜晚村庄",
		"office":            "办公室",
		"restaurant":        "餐厅",
		"subway-station":    "地铁站",
		"supermarket":       "超市",
		"temple":            "寺庙",
		"underwater":        "水下",
		"heavy-rain":        "大雨",
		"light-rain":        "小雨",
		"rain-on-car-roof":  "车顶雨声",
		"rain-on-leaves":    "叶上雨声",
		"rain-on-tent":      "帐篷雨声",
		"rain-on-umbrella":  "伞上雨声",
		"rain-on-window":    "窗上雨声",
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
		"tuning-radio":      "调频收音机",
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
	if l, ok := labels[filename]; ok {
		return l
	}
	return strings.ReplaceAll(strings.Title(strings.ReplaceAll(filename, "-", " ")), "-", " ")
}
