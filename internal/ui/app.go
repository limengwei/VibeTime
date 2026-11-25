// Package ui builds the vibe-time desktop interface with NSGo: a scene
// preset grid, an active-scene mixing panel, a footer with master volume,
// the preset editor dialog and the focus-mode overlay.
package ui

import (
	"fmt"
	"sort"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/ui"

	"vibe-time/internal/audio"
	"vibe-time/internal/catalog"
)

// Deps carries what the UI needs from main.
type Deps struct {
	Catalog    *catalog.Catalog
	Store      *catalog.Store
	Engine     *audio.Engine
	Wallpapers []catalog.Wallpaper
	// WallpaperData reads one wallpaper file's bytes.
	WallpaperData func(file string) ([]byte, error)
}

// app is the main view's mutable state plus its handlers. It lives on the
// engine's goroutine like everything that touches the audio engine.
type app struct {
	deps    Deps
	refresh *core.Signal[int] // bumped when plain state changes

	focusOpen  *core.Signal[bool]
	editorOpen *core.Signal[bool]
	editorID   *core.Signal[string] // "" = new preset, else the custom id
	master     *core.Signal[float64]

	activeID      string // applied preset ID, "" when none
	activePreset  catalog.Preset
	activeOrder   []string
	activeVolumes map[string]float64
}

// Main renders the whole app window content.
func Main(cx *core.Scope, deps Deps) core.Node {
	a := &app{
		deps:          deps,
		refresh:       core.NewSignal(cx, 0),
		focusOpen:     core.NewSignal(cx, false),
		editorOpen:    core.NewSignal(cx, false),
		editorID:      core.NewSignal(cx, ""),
		master:        core.NewSignal(cx, 0.8),
		activeVolumes: map[string]float64{},
	}

	return core.View{
		Class: "w-full h-full flex-col overflow-hidden text-neutral-100",
		Style: appBackground,
		Children: []core.Node{
			header(),
			core.Scroll{
				Class: "flex-1 min-h-0",
				Style: core.Style{Padding: core.Symmetric(4, 32), Gap: 16},
				Children: []core.Node{
					presetGrid(a),
					mixerPanel(a),
				},
			},
			footer(cx, a),
			core.Show{When: a.editorOpen.Get, Then: func(cx *core.Scope) core.Node {
				return presetEditor(cx, a)
			}},
			core.Show{When: a.focusOpen.Get, Then: func(cx *core.Scope) core.Node {
				return focusView(cx, a)
			}},
		},
	}
}

// appBackground is the window's deep gradient, as the web app's.
var appBackground = core.Style{
	Background: core.RGB(0x0a0e17),
	Gradients: []core.Gradient{
		core.LinearGradient(145, core.RGB(0x0a0e17), core.RGB(0x111827), core.RGB(0x0f1520)),
	},
}

// header is the title block.
func header() core.Node {
	return core.View{Class: "flex-col items-center gap-1 pt-6 pb-2", Children: []core.Node{
		core.Text{Content: "Vibe Time", Style: core.Style{FontSize: 26, FontWeight: core.FontWeightBold, LetterSpacing: 1}},
		core.Text{Content: "轻松找到你的宁静", Style: core.Style{FontSize: 13, Color: core.RGBA(0xffffff, 0.35), LetterSpacing: 0.5}},
	}}
}

// presets lists the built-ins followed by the customs.
func (a *app) presets() []catalog.Preset {
	all := append(catalog.BuiltinPresets(), a.deps.Store.Custom...)
	return all
}

// applyPreset stops everything and loops the preset's sounds at their
// (possibly overridden) volumes.
func (a *app) applyPreset(p catalog.Preset) {
	if a.activeID == p.ID {
		return
	}
	a.deps.Engine.StopAll()
	ids := make([]string, 0, len(p.Sounds))
	for id := range p.Sounds {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	order, vols := []string{}, map[string]float64{}
	for _, id := range ids {
		v := p.Sounds[id]
		if ov, ok := a.deps.Store.Override(p.ID, id); ok {
			v = ov
		}
		a.deps.Engine.Play(id, v)
		order = append(order, id)
		vols[id] = v
	}
	a.activeID, a.activePreset, a.activeOrder, a.activeVolumes = p.ID, p, order, vols
	a.refresh.Set(a.refresh.Get() + 1)
}

// stopAll stops every sound; clearScene also drops the applied preset.
func (a *app) stopAll(clearScene bool) {
	a.deps.Engine.StopAll()
	if clearScene {
		a.activeID = ""
		a.activeOrder = nil
		a.activeVolumes = map[string]float64{}
	}
	a.refresh.Set(a.refresh.Get() + 1)
}

// setSoundVolume changes one playing sound's volume and persists it as the
// applied preset's value (a built-in's as an override).
func (a *app) setSoundVolume(id string, v float64) {
	a.activeVolumes[id] = v
	a.deps.Engine.SetVolume(id, v)
	st := a.deps.Store
	if a.activePreset.Builtin {
		st.SetOverride(a.activeID, id, v)
		_ = st.Save()
	} else if i := st.FindCustom(a.activeID); i >= 0 {
		cp := st.Custom[i]
		if cp.Sounds == nil {
			cp.Sounds = map[string]float64{}
		}
		cp.Sounds[id] = v
		st.Custom[i] = cp
		_ = st.Save()
	}
	a.refresh.Set(a.refresh.Get() + 1)
}

// presetGrid renders the scene cards and the add card.
func presetGrid(a *app) core.Node {
	cards := []core.Node{}
	for _, p := range a.presets() {
		cards = append(cards, presetCard(a, p))
	}
	cards = append(cards, addCard(a))
	return ui.Wrap{Children: cards}
}

// presetCard is one scene card.
func presetCard(a *app, p catalog.Preset) core.Node {
	children := []core.Node{
		core.Text{Content: p.Icon, Style: core.Style{FontSize: 30}},
		core.Text{Content: p.Name, Style: core.Style{FontSize: 13, FontWeight: core.FontWeightMedium, Color: core.RGBA(0xffffff, 0.85), TextAlign: core.TextAlignCenter}},
		core.Text{Content: p.Desc, MaxLines: 2, Style: core.Style{FontSize: 11, Color: core.RGBA(0xffffff, 0.3), TextAlign: core.TextAlignCenter}},
	}
	// Custom cards add edit and delete in the corners.
	if !p.Builtin {
		children = append(children,
			cardAction(iconPencil, "编辑", core.Pt(6), func() {
				a.editorID.Set(p.ID)
				a.editorOpen.Set(true)
			}),
			cardAction(iconX, "删除", core.Pt(28), func() {
				if i := a.deps.Store.FindCustom(p.ID); i >= 0 {
					a.deps.Store.Custom = append(a.deps.Store.Custom[:i], a.deps.Store.Custom[i+1:]...)
					_ = a.deps.Store.Save()
					if a.activeID == p.ID {
						a.stopAll(true)
					}
					a.refresh.Set(a.refresh.Get() + 1)
				}
			}),
		)
	}
	return ui.Press{
		OnPress:  func() { a.applyPreset(p) },
		Label:    p.Name,
		Selected: func() bool { a.refresh.Get(); return a.activeID == p.ID },
		Style: core.Style{
			Width: core.Pt(150), Padding: core.Symmetric(20, 12), CornerRadius: 16,
			Direction: core.Row, AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter,
		},
		StyleFunc: func() core.Style {
			a.refresh.Get()
			base := core.Style{
				Direction:    core.Row,
				AlignItems:   core.AlignCenter,
				CornerRadius: 16,
				Background:   core.RGBA(0xffffff, 0.04),
				BorderColor:  core.RGBA(0xffffff, 0.06), BorderWidth: 1,
				Hover: &core.Style{Background: core.RGBA(0xffffff, 0.08), BorderColor: core.RGBA(0xffffff, 0.12)},
			}
			if a.activeID == p.ID {
				base.Background = core.RGBA(0x8b5cf6, 0.12)
				base.BorderColor = core.RGBA(0x8b5cf6, 0.4)
				base.Hover = &core.Style{Background: core.RGBA(0x8b5cf6, 0.18), BorderColor: core.RGBA(0x8b5cf6, 0.5)}
			}
			return base
		},
		Children: []core.Node{core.View{Class: "flex-col items-center gap-1", Children: children}},
	}
}

// cardAction is a small absolute corner button on custom cards.
func cardAction(svg *core.SVG, label string, right core.Length, onPress func()) core.Node {
	return core.View{
		OnPress: func(core.PressEvent) { onPress() },
		NoFocus: true,
		Label:   label,
		Style: core.Style{
			Position: core.Absolute, Top: core.Pt(6), Right: right,
			Width: core.Pt(20), Height: core.Pt(20), CornerRadius: 6,
			AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter,
		},
		StyleFunc: func() core.Style {
			return core.Style{Hover: &core.Style{Background: core.RGBA(0xffffff, 0.1)}}
		},
		Children: []core.Node{core.Icon{SVG: svg, Style: core.Style{Width: core.Pt(12), Height: core.Pt(12), Color: core.RGBA(0xffffff, 0.35)}}},
	}
}

// addCard is the dashed "add scene" card.
func addCard(a *app) core.Node {
	return ui.Press{
		OnPress: func() {
			a.editorID.Set("")
			a.editorOpen.Set(true)
		},
		Label: "添加场景",
		Style: core.Style{
			Width: core.Pt(150), Padding: core.Symmetric(20, 12), CornerRadius: 16,
			Direction: core.Row, AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter,
			BorderStyle: core.BorderDashed, BorderColor: core.RGBA(0xffffff, 0.1), BorderWidth: 1,
		},
		StyleFunc: func() core.Style {
			return core.Style{Hover: &core.Style{Background: core.RGBA(0x8b5cf6, 0.06), BorderColor: core.RGBA(0x8b5cf6, 0.4)}}
		},
		Children: []core.Node{core.View{Class: "flex-col items-center gap-1", Children: []core.Node{
			core.Icon{SVG: iconPlus, Style: core.Style{Width: core.Pt(26), Height: core.Pt(26), Color: core.RGBA(0xffffff, 0.25)}},
			core.Text{Content: "添加场景", Style: core.Style{FontSize: 13, FontWeight: core.FontWeightMedium, Color: core.RGBA(0xffffff, 0.85)}},
			core.Text{Content: "自定义声音组合", Style: core.Style{FontSize: 11, Color: core.RGBA(0xffffff, 0.3)}},
		}}},
	}
}

// mixerPanel is the applied preset's per-sound volume cards.
func mixerPanel(a *app) core.Node {
	return core.Show{When: func() bool { a.refresh.Get(); return a.activeID != "" }, Then: func(cx *core.Scope) core.Node {
		p := a.activePreset
		cards := make([]core.Node, 0, len(a.activeOrder))
		for _, id := range a.activeOrder {
			cards = append(cards, mixerCard(a, id))
		}
		return core.View{
			Class: "flex-col gap-3 p-4 rounded-2xl",
			Style: core.Style{
				Background:  core.RGBA(0x8b5cf6, 0.06),
				BorderColor: core.RGBA(0x8b5cf6, 0.15), BorderWidth: 1,
			},
			Children: []core.Node{
				core.View{Class: "flex-row items-center gap-2", Children: []core.Node{
					core.Text{Content: p.Icon, Style: core.Style{FontSize: 18}},
					core.Text{Content: p.Name, Style: core.Style{FontSize: 14, FontWeight: core.FontWeightMedium, Color: core.RGBA(0xffffff, 0.8)}},
				}},
				ui.Wrap{Children: cards},
			},
		}
	}}
}

// mixerCard is one sound's card in the mixing panel.
func mixerCard(a *app, id string) core.Node {
	sound, _ := a.deps.Catalog.Sound(id)
	emoji := "🎵"
	if cat, ok := categoryOf(a, sound.Category); ok {
		emoji = cat.Emoji
	}
	return core.View{
		Class: "flex-col items-center gap-1.5 px-2 py-3 rounded-xl",
		Style: core.Style{
			Width:       core.Pt(120),
			Background:  core.RGBA(0xffffff, 0.05),
			BorderColor: core.RGBA(0xffffff, 0.06), BorderWidth: 1,
		},
		Children: []core.Node{
			core.Text{Content: emoji, Style: core.Style{FontSize: 22}},
			core.Text{Content: sound.Label, MaxLines: 1, Style: core.Style{FontSize: 12, Color: core.RGBA(0xffffff, 0.7), TextAlign: core.TextAlignCenter}},
			ui.Slider{
				Min: 0.05, Max: 1, Step: 0.05,
				Value:    func() float64 { a.refresh.Get(); return a.activeVolumes[id] },
				OnChange: func(v float64) { a.setSoundVolume(id, v) },
				Style:    core.Style{Width: core.Pct(100)},
				Label:    sound.Label,
			},
			core.Text{ContentFunc: func() string {
				a.refresh.Get()
				return percentLabel(a.activeVolumes[id])
			}, Style: core.Style{FontSize: 10, Color: core.RGBA(0xffffff, 0.3)}},
		},
	}
}

// categoryOf finds a category by ID.
func categoryOf(a *app, id string) (catalog.Category, bool) {
	for _, c := range a.deps.Catalog.Cats {
		if c.ID == id {
			return c, true
		}
	}
	return catalog.Category{}, false
}

// footer is the bottom bar: playing count, stop-all, master volume, focus.
func footer(cx *core.Scope, a *app) core.Node {
	return core.View{
		Class: "flex-row items-center justify-between px-8 py-3",
		Style: core.Style{
			Borders: core.Edges{Top: 1}, BorderColor: core.RGBA(0xffffff, 0.06),
			Background: core.RGBA(0x000000, 0.2),
		},
		Children: []core.Node{
			// Left: playing count.
			core.View{Style: core.Style{FlexGrow: 1}, Children: []core.Node{
				core.Text{ContentFunc: func() string {
					n := a.deps.Engine.Count.Get()
					if n == 0 {
						return ""
					}
					return pluralSounds(n)
				}, Style: core.Style{FontSize: 12, Color: core.RGBA(0xffffff, 0.4)}},
			}},
			// Center: stop all.
			core.Show{When: func() bool { return a.deps.Engine.Count.Get() > 0 }, Then: func(cx *core.Scope) core.Node {
				return ui.Press{
					OnPress: func() { a.stopAll(true) },
					Label:   "全部停止",
					Style: core.Style{
						Padding: core.Symmetric(6, 14), CornerRadius: 8, Direction: core.Row, Gap: 6,
						AlignItems:  core.AlignCenter,
						Background:  core.RGBA(0xff5050, 0.15),
						BorderColor: core.RGBA(0xff5050, 0.2), BorderWidth: 1,
					},
					StyleFunc: func() core.Style {
						return core.Style{Hover: &core.Style{Background: core.RGBA(0xff5050, 0.25), BorderColor: core.RGBA(0xff5050, 0.3)}}
					},
					Children: []core.Node{
						core.Icon{SVG: iconStop, Style: core.Style{Width: core.Pt(12), Height: core.Pt(12), Color: core.RGBA(0xff7878, 0.9)}},
						core.Text{Content: "全部停止", Style: core.Style{FontSize: 12, Color: core.RGBA(0xff7878, 0.9)}},
					},
				}
			}},
			// Right: master volume + focus.
			core.View{Class: "flex-row items-center justify-end gap-2", Style: core.Style{FlexGrow: 1}, Children: []core.Node{
				core.Icon{SVG: iconVolume, Style: core.Style{Width: core.Pt(16), Height: core.Pt(16), Color: core.RGBA(0xffffff, 0.4)}},
				ui.Slider{
					Min: 0, Max: 1, Step: 0.01,
					Value:    a.master.Get,
					OnChange: func(v float64) { a.master.Set(v); a.deps.Engine.SetMaster(v) },
					Style:    core.Style{Width: core.Pt(110)},
					Label:    "总音量",
				},
				focusButton(a),
			}},
		},
	}
}

// focusButton enters focus mode; disabled with nothing playing.
func focusButton(a *app) core.Node {
	return ui.Press{
		OnPress:  func() { a.focusOpen.Set(true) },
		Disabled: func() bool { return a.deps.Engine.Count.Get() == 0 },
		Label:    "专注模式",
		Style: core.Style{
			Width: core.Pt(34), Height: core.Pt(34), CornerRadius: 17,
			AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter,
			Background:  core.RGBA(0xffffff, 0.05),
			BorderColor: core.RGBA(0xffffff, 0.08), BorderWidth: 1,
		},
		StyleFunc: func() core.Style {
			if a.deps.Engine.Count.Get() == 0 {
				return core.Style{Opacity: core.Alpha(0.3)}
			}
			return core.Style{Hover: &core.Style{Background: core.RGBA(0x8b5cf6, 0.2), BorderColor: core.RGBA(0x8b5cf6, 0.4)}}
		},
		Children: []core.Node{core.Icon{SVG: iconClock, Style: core.Style{Width: core.Pt(16), Height: core.Pt(16), Color: core.RGBA(0xffffff, 0.7)}}},
	}
}

// percentLabel formats a volume as a percentage.
func percentLabel(v float64) string {
	return fmt.Sprintf("%d%%", int(v*100+0.5))
}

// pluralSounds renders the playing count line.
func pluralSounds(n int) string {
	return fmt.Sprintf("%d 个音效正在播放", n)
}
