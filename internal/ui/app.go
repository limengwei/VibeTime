// Package ui builds the vibe-time desktop interface with NSGo: a scene
// preset grid, an active-scene mixing panel, a footer with master volume,
// the preset editor dialog and the focus-mode overlay.
package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/ui"

	"vibe-time/internal/audio"
	"vibe-time/internal/catalog"
	"vibe-time/internal/pack"
)

// Deps carries what the UI needs from main.
type Deps struct {
	Catalog    *catalog.Catalog
	Store      *catalog.Store
	Engine     *audio.Engine
	Wallpapers []catalog.Wallpaper
	// WallpaperData reads one wallpaper file's bytes.
	WallpaperData func(file string) ([]byte, error)
	// Pack downloads the sounds that ship in the separate pack repository;
	// nil keeps the app to its embedded sounds.
	Pack *pack.Pack
}

// app is the main view's mutable state plus its handlers. It lives on the
// engine's goroutine like everything that touches the audio engine.
type app struct {
	deps    Deps
	refresh *core.Signal[int] // bumped when plain state changes
	// gridRev rebuilds the preset grid, only when the preset list itself
	// changes (a preset saved or deleted); volume tweaks restyle the cards
	// through refresh instead, so sliders never rebuild the grid.
	gridRev *core.Signal[int]

	focusOpen   *core.Signal[bool]
	editorOpen  *core.Signal[bool]
	editorID    *core.Signal[string] // "" = new preset, else the custom id
	master      *core.Signal[float64]
	welcomeOpen *core.Signal[bool] // first launch: the sound pack's download dialog
	packReady   *core.Signal[bool] // the manifest has arrived
	packFailed  *core.Signal[bool] // the manifest could not be fetched (offline)

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
		gridRev:       core.NewSignal(cx, 0),
		focusOpen:     core.NewSignal(cx, false),
		editorOpen:    core.NewSignal(cx, false),
		editorID:      core.NewSignal(cx, ""),
		master:        core.NewSignal(cx, 0.8),
		welcomeOpen:   core.NewSignal(cx, false),
		packReady:     core.NewSignal(cx, false),
		packFailed:    core.NewSignal(cx, false),
		activeVolumes: map[string]float64{},
	}

	// The sound pack runs in the background: its manifest fills the
	// (otherwise empty) catalog with every sound the repository lists —
	// labels and category names from its labels.json — then the cache
	// fills quietly. A cold start (nothing cached) holds the prefetch for
	// the welcome dialog instead, so the user consents to the download.
	coldStart := deps.Pack != nil && deps.Pack.CachedCount() == 0
	a.startPack(!coldStart)
	if coldStart {
		a.welcomeOpen.Set(true)
	}

	return core.View{
		Class: "w-full h-full flex-col overflow-hidden text-neutral-100",
		Style: appBackground,
		Children: []core.Node{
			header(),
			repoCorner(cx),
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
			core.Show{When: a.welcomeOpen.Get, Then: func(cx *core.Scope) core.Node {
				return welcomePanel(cx, welcomeState{
					open:   a.welcomeOpen,
					pack:   deps.Pack,
					ready:  a.packReady.Get,
					failed: a.packFailed.Get,
					onRetry: func() {
						a.packFailed.Set(false)
						a.startPack(false)
					},
					onReady: func() {
						a.welcomeOpen.Set(false)
						a.bumpRefresh()
					},
				})
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

// startPack fetches the manifest and merges it into the catalog, prefetching
// after (unless the welcome dialog owns the first fill). It is re-runnable:
// the welcome dialog's retry calls it again after a failure.
func (a *app) startPack(prefetch bool) {
	if a.deps.Pack == nil {
		return
	}
	a.deps.Pack.Start(func(m *pack.Manifest) {
		for _, f := range m.Files {
			if _, ok := a.deps.Catalog.Sound(f.ID); !ok {
				s := catalog.SoundFromID(f.ID)
				if _, base, _ := strings.Cut(f.ID, "/"); m.Labels[base] != "" {
					s.Label = m.Labels[base]
				}
				a.deps.Catalog.AddSound(s)
			}
		}
		for id, cm := range m.Categories {
			a.deps.Catalog.SetCategoryMeta(id, cm.Title, cm.Emoji)
		}
		a.packReady.Set(true)
		a.packFailed.Set(false)
		a.bumpRefresh()
	}, prefetch, func() {
		a.packFailed.Set(true)
		a.bumpRefresh()
	})
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

// repoURL is the project's GitHub repository, opened in the system browser.
const repoURL = "https://github.com/limengwei/VibeTime"

// repoCorner is the GitHub repository entry pinned at the window's
// top-right corner, out of the flow of the centered header.
func repoCorner(cx *core.Scope) core.Node {
	return ui.Press{
		OnPress: func() { cx.OpenURL(repoURL, nil) },
		Label:   "GitHub 仓库",
		Style: core.Style{
			Position: core.Fixed, Top: core.Pt(8), Right: core.Pt(8),
			Width:  core.Pt(30),
			Height: core.Pt(30), CornerRadius: 8,
			AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter,
		},
		StyleFunc: func() core.Style {
			return core.Style{Hover: &core.Style{Background: core.RGBA(0xffffff, 0.08)}}
		},
		Children: []core.Node{core.Icon{SVG: iconGitHub, Style: core.Style{Width: core.Pt(16), Height: core.Pt(16), Color: core.RGBA(0xffffff, 0.35)}}},
	}
}

// applyPreset stops everything and loops the preset's sounds at their
// (possibly overridden) volumes. A sound still downloading starts as
// soon as its download lands, if the preset is still applied.
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
		if a.soundReady(id) {
			a.deps.Engine.Play(id, v)
		} else if a.deps.Pack != nil {
			id, v := id, v
			a.deps.Pack.Ensure(id, func(ok bool) {
				if ok && a.activeID == p.ID {
					a.deps.Engine.Play(id, v)
					a.bumpRefresh()
				}
			})
		}
		order = append(order, id)
		vols[id] = v
	}
	a.activeID, a.activePreset, a.activeOrder, a.activeVolumes = p.ID, p, order, vols
	a.bumpRefresh()
}

// soundReady reports whether a sound's bytes are at hand: downloaded to
// the pack cache or shipped in the embedded library.
func (a *app) soundReady(id string) bool {
	if a.deps.Pack != nil && a.deps.Pack.Cached(id) {
		return true
	}
	s, ok := a.deps.Catalog.Sound(id)
	return ok && s.File != ""
}

// bumpRefresh re-renders what follows plain state. It reads untracked so a
// bump from inside an effect never makes that effect depend on refresh — a
// tracked read there would re-run the effect forever (the editor's open
// effect once did, saturating the render goroutine and lagging the dialog).
func (a *app) bumpRefresh() {
	a.refresh.Set(a.refresh.GetUntracked() + 1)
}

// stopAll stops every sound; clearScene also drops the applied preset.
func (a *app) stopAll(clearScene bool) {
	a.deps.Engine.StopAll()
	if clearScene {
		a.activeID = ""
		a.activeOrder = nil
		a.activeVolumes = map[string]float64{}
	}
	a.bumpRefresh()
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
	a.bumpRefresh()
}

// presetGrid renders the scene cards and the add card. A Dynamic, not a
// Show: the branch must rebuild whenever a preset is added, edited or
// deleted, which changes no boolean the grid could watch.
func presetGrid(a *app) core.Node {
	return core.Dynamic(func(cx *core.Scope) core.Node {
		a.gridRev.Get()
		cards := []core.Node{}
		for _, p := range a.presets() {
			cards = append(cards, presetCard(a, p))
		}
		cards = append(cards, addCard(a))
		return ui.Wrap{Children: cards}
	})
}

// presetCard is one scene card.
func presetCard(a *app, p catalog.Preset) core.Node {
	children := []core.Node{
		core.Text{Content: p.Icon, Style: core.Style{FontSize: 30}},
		core.Text{Content: p.Name, Style: core.Style{FontSize: 13, FontWeight: core.FontWeightMedium, Color: core.RGBA(0xffffff, 0.85), TextAlign: core.TextAlignCenter}},
		core.Text{Content: p.Desc, MaxLines: 1, Style: core.Style{FontSize: 11, Color: core.RGBA(0xffffff, 0.3), TextAlign: core.TextAlignCenter}},
	}
	cardChildren := []core.Node{core.View{
		Class: "flex-col items-center gap-1",
		// The full width bounds the description, so its MaxLines
		// ellipsizes instead of stretching the card with one long line.
		Style:    core.Style{Width: core.Pct(100)},
		Children: children,
	}}
	// Custom cards add edit and delete in the top-right corner, direct
	// children of the card: the card itself (Position: Relative below)
	// anchors the strip in its padding box at the rim. The two buttons sit
	// in one absolutely-placed row, so their gap is ordinary layout rather
	// than a Right inset (valued insets scale wrongly under display
	// scaling).
	if !p.Builtin {
		cardChildren = append(cardChildren, core.View{
			Style: core.Style{
				Position: core.Absolute, Top: core.Pt(2), Right: core.Pt(2),
				Direction: core.Row, Gap: 2,
			},
			Children: []core.Node{
				cardAction(iconX, "删除", func() {
					if i := a.deps.Store.FindCustom(p.ID); i >= 0 {
						a.deps.Store.Custom = append(a.deps.Store.Custom[:i], a.deps.Store.Custom[i+1:]...)
						_ = a.deps.Store.Save()
						if a.activeID == p.ID {
							a.stopAll(true)
						}
						a.gridRev.Set(a.gridRev.GetUntracked() + 1)
					}
				}),
				cardAction(iconPencil, "编辑", func() {
					a.editorID.Set(p.ID)
					a.editorOpen.Set(true)
				}),
			},
		})
	}
	return ui.Press{
		OnPress:  func() { a.applyPreset(p) },
		Label:    p.Name,
		Selected: func() bool { a.refresh.Get(); return a.activeID == p.ID },
		Style: core.Style{
			Width: core.Pt(150), Padding: core.Symmetric(20, 12), CornerRadius: 16,
			Direction: core.Row, AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter,
			Position: core.Relative,
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
		Children: cardChildren,
	}
}

// cardAction is a small corner button inside the card's action strip.
func cardAction(svg *core.SVG, label string, onPress func()) core.Node {
	return core.View{
		OnPress: func(core.PressEvent) { onPress() },
		NoFocus: true,
		Label:   label,
		Style: core.Style{
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

// mixerPanel is the applied preset's per-sound volume cards. The rows are
// an Each over the active order: switching presets swaps rows by sound ID,
// while a volume drag keeps each row (and its slider) in place.
func mixerPanel(a *app) core.Node {
	return core.Show{When: func() bool { a.refresh.Get(); return a.activeID != "" }, Then: func(cx *core.Scope) core.Node {
		return core.View{
			Class: "flex-col gap-3 p-4 rounded-2xl",
			Style: core.Style{
				Background:  core.RGBA(0x8b5cf6, 0.06),
				BorderColor: core.RGBA(0x8b5cf6, 0.15), BorderWidth: 1,
			},
			Children: []core.Node{
				core.View{Class: "flex-row items-center gap-2", Children: []core.Node{
					core.Text{ContentFunc: func() string { a.refresh.Get(); return a.activePreset.Icon }, Style: core.Style{FontSize: 18}},
					core.Text{ContentFunc: func() string { a.refresh.Get(); return a.activePreset.Name }, Style: core.Style{FontSize: 14, FontWeight: core.FontWeightMedium, Color: core.RGBA(0xffffff, 0.8)}},
				}},
				ui.Wrap{Children: []core.Node{core.Each(
					func() []string { a.refresh.Get(); return a.activeOrder },
					func(id string) string { return id },
					func(cx *core.Scope, item func() string, _ func() int) core.Node {
						return mixerCard(a, item())
					},
				)}},
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
				Min: 0.05, Max: 1,
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
					Min: 0, Max: 1,
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
