package ui

import (
	"math/rand/v2"
	"strings"

	"github.com/nsgo-dev/nsgo/base"
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/ui"

	"vibe-time/internal/catalog"
)

// randomIcons feeds the editor's random icon button.
var randomIcons = []string{
	"🎵", "🎶", "✨", "🌟", "💫", "🎪", "🎭", "🎨", "🎬", "🎮",
	"🎲", "🎯", "🎸", "🎹", "🎺", "🎻", "🥁", "🏕", "⛰", "🌅",
	"🌄", "🎆", "🎇", "🏙", "🏯", "⛵", "🚀", "🎡", "🎢", "🎠",
}

// editor is the preset dialog's state, rebuilt on every open.
type editor struct {
	a *app

	name *core.Signal[string]
	icon *core.Signal[string]
	desc *core.Signal[string]

	// selected keeps the chosen sounds in pick order; selRefresh bumps on
	// every change (a map alone cannot re-render).
	selected   map[string]float64
	order      []string
	selRefresh *core.Signal[int]

	// editingID is the custom preset being edited, "" for a new one.
	editingID string
}

// presetEditor renders the add/edit scene dialog. Its state is created per
// open (Show rebuilds Then), so toggling sounds previews immediately.
func presetEditor(cx *core.Scope, a *app) core.Node {
	e := &editor{
		a:          a,
		name:       core.NewSignal(cx, ""),
		icon:       core.NewSignal(cx, randomIcons[0]),
		desc:       core.NewSignal(cx, ""),
		selected:   map[string]float64{},
		selRefresh: core.NewSignal(cx, 0),
		editingID:  a.editorID.Get(),
	}

	// Opening the editor stops the current mix, as the web app did; editing
	// preloads the preset and previews it. The work runs as an effect, after
	// this build commits, so no signal is written while it renders.
	cx.Effect(func() {
		a.stopAll(false)
		if e.editingID != "" {
			if i := a.deps.Store.FindCustom(e.editingID); i >= 0 {
				p := a.deps.Store.Custom[i]
				e.name.Set(p.Name)
				e.icon.Set(p.Icon)
				e.desc.Set(p.Desc)
				for id, v := range p.Sounds {
					e.pick(id, v)
				}
			}
		} else {
			e.icon.Set(randomIcon())
		}
	})

	editing := e.editingID != ""
	title := "添加场景"
	if editing {
		title = "编辑场景"
	}

	return ui.Dialog{
		Open: a.editorOpen.Get,
		OnOpenChange: func(open bool, _ base.Reason) {
			if !open {
				e.close()
			}
		},
		Title: title,
		Children: []core.Node{
			core.Scroll{
				HideIndicator: true,
				Style:         core.Style{Width: core.Pct(100), MinHeight: core.Pt(0), MaxHeight: core.Pt(430), Gap: 14},
				Children: []core.Node{
					// Name and icon.
					core.View{Class: "flex-row gap-3", Children: []core.Node{
						core.View{Class: "flex-col gap-1", Style: core.Style{Width: core.Pt(96)}, Children: []core.Node{
							core.Text{Content: "图标", Style: labelStyle},
							core.View{Class: "flex-row items-center gap-1.5", Children: []core.Node{
								core.View{Style: core.Style{
									Width: core.Pt(38), Height: core.Pt(38), CornerRadius: 10,
									Background:  core.RGBA(0xffffff, 0.06),
									BorderColor: core.RGBA(0xffffff, 0.1), BorderWidth: 1,
									AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter,
								}, Children: []core.Node{
									core.Text{ContentFunc: e.icon.Get, Style: core.Style{FontSize: 20}},
								}},
								ui.Button{Icon: func() *core.SVG { return iconDice }, Label: "随机图标", OnPress: func() {
									e.icon.Set(randomIcon())
								}, Style: core.Style{Width: core.Pt(38), Height: core.Pt(38)}, IconSize: 16},
							}},
						}},
						core.View{Class: "flex-col gap-1 flex-1 min-w-0", Children: []core.Node{
							core.Text{Content: "名称", Style: labelStyle},
							ui.Input{Value: e.name.Get, OnChange: e.name.Set, Placeholder: "场景名称", Style: core.Style{FontSize: 13}},
						}},
					}},

					// Description.
					core.View{Class: "flex-col gap-1", Children: []core.Node{
						core.Text{Content: "描述", Style: labelStyle},
						ui.Input{Value: e.desc.Get, OnChange: e.desc.Set, Placeholder: "简短描述这个场景…（留空自动生成）", Style: core.Style{FontSize: 13}},
					}},

					// Chosen sounds with their volumes.
					core.View{Class: "flex-col gap-2", Children: []core.Node{
						core.Text{ContentFunc: func() string {
							e.selRefresh.Get()
							if n := len(e.selected); n > 0 {
								return "已选声音（" + itoaOf(n) + "）"
							}
							return "已选声音"
						}, Style: labelStyle},
						core.Show{When: func() bool { e.selRefresh.Get(); return len(e.selected) > 0 }, Then: func(cx *core.Scope) core.Node {
							return core.View{
								Class: "flex-col gap-2 p-2.5 rounded-xl",
								Style: core.Style{
									Background:  core.RGBA(0x8b5cf6, 0.06),
									BorderColor: core.RGBA(0x8b5cf6, 0.15), BorderWidth: 1,
								},
								Children: []core.Node{e.selectedTags()},
							}
						}},
					}},

					// Every sound as chips, grouped by category.
					core.View{Class: "flex-col gap-3", Children: e.categoryGroups()},
				},
			},
		},
		Footer: []core.Node{
			ui.Button{Text: "取消", OnPress: e.close},
			ui.Button{Text: "保存", Disabled: func() bool { return !e.valid() }, OnPress: e.save},
		},
	}
}

// labelStyle is the small uppercase-ish section label.
var labelStyle = core.Style{FontSize: 11, Color: core.RGBA(0xffffff, 0.4), FontWeight: core.FontWeightMedium}

// itoaOf formats a small count.
func itoaOf(n int) string {
	if n == 0 {
		return "0"
	}
	digits := []byte{}
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

// randomIcon picks a random emoji.
func randomIcon() string {
	return randomIcons[rand.IntN(len(randomIcons))]
}

// pick adds a sound (previewing it) at volume v.
func (e *editor) pick(id string, v float64) {
	if _, ok := e.selected[id]; ok {
		return
	}
	if v < 0.05 {
		v = 0.05
	}
	e.selected[id] = v
	e.order = append(e.order, id)
	e.a.deps.Engine.Play(id, v)
	e.selRefresh.Set(e.selRefresh.Get() + 1)
}

// unpick removes a sound and stops its preview.
func (e *editor) unpick(id string) {
	if _, ok := e.selected[id]; ok {
		delete(e.selected, id)
		for i, x := range e.order {
			if x == id {
				e.order = append(e.order[:i], e.order[i+1:]...)
				break
			}
		}
		e.a.deps.Engine.Stop(id)
		e.selRefresh.Set(e.selRefresh.Get() + 1)
	}
}

// toggle flips a sound's membership.
func (e *editor) toggle(id string) {
	if _, ok := e.selected[id]; ok {
		e.unpick(id)
	} else {
		e.pick(id, 0.5)
	}
}

// close stops the previews and dismisses the dialog.
func (e *editor) close() {
	e.a.stopAll(false)
	e.a.editorOpen.Set(false)
}

// valid reports whether the form can be saved.
func (e *editor) valid() bool {
	return strings.TrimSpace(e.name.Get()) != "" && len(e.selected) > 0
}

// autoDesc joins the chosen sounds' labels.
func (e *editor) autoDesc() string {
	out := ""
	for i, id := range e.order {
		if i > 0 {
			out += " + "
		}
		if s, ok := e.a.deps.Catalog.Sound(id); ok {
			out += s.Label
		} else {
			out += id
		}
	}
	return out
}

// save persists the preset and applies it.
func (e *editor) save() {
	if !e.valid() {
		return
	}
	name := strings.TrimSpace(e.name.Get())
	desc := e.desc.Get()
	if desc == "" {
		desc = e.autoDesc()
	}
	sounds := make(map[string]float64, len(e.selected))
	for id, v := range e.selected {
		sounds[id] = v
	}
	st := e.a.deps.Store
	p := catalog.Preset{Name: name, Icon: e.icon.Get(), Desc: desc, Sounds: sounds}
	if i := st.FindCustom(e.editingID); e.editingID != "" && i >= 0 {
		p.ID = e.editingID
		st.Custom[i] = p
	} else {
		p.ID = st.NextCustomID()
		st.Custom = append(st.Custom, p)
	}
	if err := st.Save(); err != nil {
		// Saved presets fail only on a broken config dir; keep the session
		// running either way.
		_ = err
	}
	e.a.editorOpen.Set(false)
	// Apply fresh: applyPreset skips a repeated ID, so clear it first.
	e.a.activeID = ""
	e.a.applyPreset(p)
}

// selectedTags renders the picked sounds with volume sliders and removes.
func (e *editor) selectedTags() core.Node {
	tags := make([]core.Node, 0, len(e.order))
	for _, id := range e.order {
		id := id
		sound, _ := e.a.deps.Catalog.Sound(id)
		tags = append(tags, core.View{
			Class: "flex-row items-center gap-1.5 px-2 py-1 rounded-lg",
			Style: core.Style{Background: core.RGBA(0xffffff, 0.08)},
			Children: []core.Node{
				core.Text{Content: sound.Label, Style: core.Style{FontSize: 11, Color: core.RGBA(0xffffff, 0.7)}},
				ui.Slider{
					Min: 0.05, Max: 1, Step: 0.05,
					Value: func() float64 { e.selRefresh.Get(); return e.selected[id] },
					OnChange: func(v float64) {
						e.selected[id] = v
						e.a.deps.Engine.SetVolume(id, v)
						e.selRefresh.Set(e.selRefresh.Get() + 1)
					},
					Style: core.Style{Width: core.Pt(64)},
					Label: sound.Label,
				},
				core.View{
					OnPress: func(core.PressEvent) { e.unpick(id) },
					NoFocus: true,
					Label:   "移除 " + sound.Label,
					Style:   core.Style{Padding: core.All(2)},
					StyleFunc: func() core.Style {
						return core.Style{Hover: &core.Style{Background: core.RGBA(0xffffff, 0.1)}}
					},
					Children: []core.Node{core.Icon{SVG: iconX, Style: core.Style{Width: core.Pt(10), Height: core.Pt(10), Color: core.RGBA(0xffffff, 0.3)}}},
				},
			},
		})
	}
	return ui.Wrap{Children: tags}
}

// categoryGroups renders every category's sound chips.
func (e *editor) categoryGroups() []core.Node {
	groups := make([]core.Node, 0, len(e.a.deps.Catalog.Cats))
	for _, cat := range e.a.deps.Catalog.Cats {
		chips := make([]core.Node, 0, len(cat.Sounds))
		for _, s := range cat.Sounds {
			s := s
			chips = append(chips, ui.Press{
				OnPress: func() { e.toggle(s.ID) },
				Label:   s.Label,
				Selected: func() bool {
					e.selRefresh.Get()
					_, ok := e.selected[s.ID]
					return ok
				},
				Style: core.Style{
					Padding: core.Symmetric(5, 12), CornerRadius: 8,
					Background:  core.RGBA(0xffffff, 0.04),
					BorderColor: core.RGBA(0xffffff, 0.06), BorderWidth: 1,
					Color: core.RGBA(0xffffff, 0.5),
				},
				StyleFunc: func() core.Style {
					e.selRefresh.Get()
					if _, ok := e.selected[s.ID]; ok {
						return core.Style{
							Background:  core.RGBA(0x8b5cf6, 0.2),
							BorderColor: core.RGBA(0x8b5cf6, 0.4),
							Color:       core.RGBA(0xd1bfff, 0.95),
						}
					}
					return core.Style{Hover: &core.Style{Background: core.RGBA(0xffffff, 0.1), Color: core.RGBA(0xffffff, 0.8)}}
				},
				Children: []core.Node{core.Text{Content: s.Label, Style: core.Style{FontSize: 12}}},
			})
		}
		groups = append(groups, core.View{Class: "flex-col gap-1.5", Children: []core.Node{
			core.Text{Content: cat.Title, Style: core.Style{FontSize: 11, FontWeight: core.FontWeightMedium, Color: core.RGBA(0xffffff, 0.35)}},
			ui.Wrap{Children: chips},
		}})
	}
	return groups
}
