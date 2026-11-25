package ui

import (
	"strings"

	"github.com/nsgo-dev/nsgo/base"
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/ui"

	"vibe-time/internal/catalog"
)

// openShare encodes a preset as a recipe code (the applied preset shares
// its current volumes) and shows the share dialog. The web app also draws a
// PNG recipe card here; the Go app shares the code itself.
func (a *app) openShare(p catalog.Preset) {
	sounds := p.Sounds
	if a.activeID == p.ID {
		sounds = a.activeVolumes
	}
	code := catalog.EncodeRecipe(catalog.Preset{Name: p.Name, Icon: p.Icon, Desc: p.Desc, Sounds: sounds})
	if code == "" {
		a.toaster.Show(ui.ToastContent{Type: ui.ToastError, Title: "无法分享空配方"})
		return
	}
	a.shareCode.Set(code)
	a.shareOpen.Set(true)
}

// copyCode puts the code on the clipboard and toasts about it.
func (a *app) copyCode(code, what string) {
	a.cx.SetClipboard(code)
	a.toaster.Show(ui.ToastContent{Type: ui.ToastSuccess, Title: "已复制", Description: what})
}

// shareDialog shows a preset's recipe code with a copy button.
func shareDialog(a *app) core.Node {
	return ui.Dialog{
		Open: a.shareOpen.Get,
		OnOpenChange: func(open bool, _ base.Reason) {
			if !open {
				a.shareOpen.Set(false)
			}
		},
		Title:       "分享配方",
		Description: "把这串配方码发给朋友，对方用「导入配方」即可还原你的声音组合。",
		Children: []core.Node{
			core.View{Style: core.Style{Width: core.Pct(100)}, Children: []core.Node{
				ui.Input{
					Value:    a.shareCode.Get,
					Readonly: true,
					Style:    core.Style{FontSize: 11, FontFamily: core.FontMonospace},
				},
				core.Text{Content: "提示：当前正在播放的预设分享的是调整后的音量。", Style: core.Style{FontSize: 10, Color: core.RGBA(0xffffff, 0.3), Margin: core.Edges{Top: 8}}},
			}},
		},
		Footer: []core.Node{
			ui.Button{Text: "关闭", OnPress: func() { a.shareOpen.Set(false) }},
			ui.Button{Text: "复制配方码", OnPress: func() {
				a.copyCode(a.shareCode.Get(), "配方码")
			}},
		},
	}
}

// importDialog pastes a friend's recipe code in and turns it into a custom
// preset, applying it right away.
func importDialog(a *app) core.Node {
	return ui.Dialog{
		Open: a.importOpen.Get,
		OnOpenChange: func(open bool, _ base.Reason) {
			if !open {
				a.importOpen.Set(false)
			}
		},
		Title:       "导入配方码",
		Description: "粘贴好友分享的 VibeTime 配方码",
		Children: []core.Node{
			core.View{Style: core.Style{Width: core.Pct(100)}, Children: []core.Node{
				ui.Input{
					Value:       a.importCode.Get,
					OnChange:    a.importCode.Set,
					OnSubmit:    func(string) { a.importRecipe() },
					Placeholder: "VIBE:…",
					AutoFocus:   true,
					Style:       core.Style{FontSize: 12, FontFamily: core.FontMonospace},
				},
			}},
		},
		Footer: []core.Node{
			ui.Button{Text: "取消", OnPress: func() { a.importOpen.Set(false) }},
			ui.Button{Text: "导入", Disabled: func() bool {
				return strings.TrimSpace(a.importCode.Get()) == ""
			}, OnPress: func() { a.importRecipe() }},
		},
	}
}

// importRecipe decodes the pasted code into a custom preset and applies it.
func (a *app) importRecipe() {
	p, ok := catalog.DecodeRecipe(a.importCode.Get())
	if !ok {
		a.toaster.Show(ui.ToastContent{Type: ui.ToastError, Title: "导入失败", Description: "配方码无效或已损坏"})
		return
	}
	// Filter to sounds this app actually ships, so a foreign code cannot
	// add silent tracks.
	for id := range p.Sounds {
		if _, exists := a.deps.Catalog.Sound(id); !exists {
			delete(p.Sounds, id)
		}
	}
	if len(p.Sounds) == 0 {
		a.toaster.Show(ui.ToastContent{Type: ui.ToastError, Title: "导入失败", Description: "配方中没有可用的音效"})
		return
	}
	name := p.Name
	for _, existing := range a.presets() {
		if existing.Name == name {
			name = p.Name + " 📥"
			break
		}
	}
	p.Name = name
	p.ID = a.deps.Store.NextCustomID()
	a.deps.Store.Custom = append(a.deps.Store.Custom, p)
	if err := a.deps.Store.Save(); err != nil {
		a.toaster.Show(ui.ToastContent{Type: ui.ToastError, Title: "保存失败", Description: err.Error()})
	}
	a.importOpen.Set(false)
	a.refresh.Set(a.refresh.Get() + 1)
	a.activeID = "" // apply fresh even if an equal-named preset was active
	a.applyPreset(p)
	a.toaster.Show(ui.ToastContent{Type: ui.ToastSuccess, Title: "已导入配方", Description: p.Name})
}
