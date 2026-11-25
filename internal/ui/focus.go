package ui

import (
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/ui"

	"vibe-time/internal/catalog"
)

// durationOptions are the focus timer's choices; 0 means unlimited.
var durationOptions = []struct {
	Label string
	Min   int
}{
	{"5 分钟", 5}, {"10 分钟", 10}, {"20 分钟", 20},
	{"30 分钟", 30}, {"60 分钟", 60}, {"不限制", 0},
}

// focusView renders the focus-mode overlay: a wallpaper, a big timer with a
// progress ring, running controls and a settings panel. Its state is created
// per entry (Show rebuilds Then), and it enters running an unlimited count-up
// with the panel hidden, as the web app did.
func focusView(cx *core.Scope, a *app) core.Node {
	walls := a.deps.Wallpapers

	duration := core.NewSignal(cx, 0)  // chosen minutes, 0 = unlimited
	remaining := core.NewSignal(cx, 0) // seconds left (or elapsed, unlimited)
	running := core.NewSignal(cx, true)
	paused := core.NewSignal(cx, false)
	panelOpen := core.NewSignal(cx, false)
	wpIndex := core.NewSignal(cx, rand.IntN(max(len(walls), 1)))

	// wpCache memoizes wallpaper bytes across re-renders.
	wpCache := map[string][]byte{}
	wpData := func(i int) []byte {
		if i < 0 || i >= len(walls) {
			return nil
		}
		if d, ok := wpCache[walls[i].File]; ok {
			return d
		}
		d, err := a.deps.WallpaperData(walls[i].File)
		if err != nil {
			return nil
		}
		wpCache[walls[i].File] = d
		return d
	}

	// The timer ticks with AfterFunc re-arming, which runs on the engine's
	// goroutine — where the signals and the audio engine belong.
	var stopTick func() bool
	halt := func() {
		if stopTick != nil {
			stopTick()
			stopTick = nil
		}
	}
	stopTimer := func() {
		halt()
		running.Set(false)
		paused.Set(false)
		if duration.Get() == 0 {
			remaining.Set(0)
		} else {
			remaining.Set(duration.Get() * 60)
		}
	}
	var tick func()
	tick = func() {
		if !running.Get() || paused.Get() {
			return
		}
		if duration.Get() == 0 {
			remaining.Set(remaining.Get() + 1)
		} else if r := remaining.Get() - 1; r > 0 {
			remaining.Set(r)
		} else {
			remaining.Set(0)
			stopTimer()
			panelOpen.Set(true)
			return
		}
		stopTick = cx.AfterFunc(time.Second, tick)
	}
	selectDuration := func(min int) {
		halt()
		duration.Set(min)
		remaining.Set(min * 60)
		running.Set(true)
		paused.Set(false)
		panelOpen.Set(false)
		stopTick = cx.AfterFunc(time.Second, tick)
	}
	togglePause := func() {
		if !running.Get() {
			return
		}
		if paused.Get() {
			paused.Set(false)
			halt()
			stopTick = cx.AfterFunc(time.Second, tick)
		} else {
			halt()
			paused.Set(true)
		}
	}

	exit := func() {
		halt()
		a.stopAll(true)
		a.focusOpen.Set(false)
	}

	// Keys, bound in this view's scope so they rest once it leaves.
	cx.OnKey("escape", func() {
		if panelOpen.Get() {
			panelOpen.Set(false)
		} else {
			exit()
		}
	})
	cx.OnKey("space", togglePause)

	// Enter running the unlimited count-up (the web app's onMounted
	// selectDuration(0)); scheduling during build commits no state.
	stopTick = cx.AfterFunc(time.Second, tick)

	// Stretch an absolute view over its parent (Length zero needs Pt(0)).
	fill := core.Style{Position: core.Absolute,
		Top: core.Pt(0), Right: core.Pt(0), Bottom: core.Pt(0), Left: core.Pt(0)}

	return core.View{
		Style: core.Style{
			Position: core.Absolute,
			Top:      core.Pt(0), Right: core.Pt(0), Bottom: core.Pt(0), Left: core.Pt(0),
			ZIndex: 100, Overflow: core.Hidden,
		},
		Children: []core.Node{
			// Wallpaper.
			core.Image{
				SourceFunc: func() core.ImageSource {
					return core.ImageSource{Data: wpData(wpIndex.Get())}
				},
				ResizeMode: core.Cover,
				Style:      fill,
			},
			// Vignette over the wallpaper.
			core.View{Style: overlay(fill, core.RadialGradient(
				core.RGBA(0x000000, 0.2), core.RGBA(0x000000, 0.6)))},

			// Timer and controls.
			core.View{Class: "w-full h-full flex-col items-center justify-center", Children: []core.Node{
				timerDisplay(duration, remaining),
				core.Show{When: func() bool { return running.Get() && !panelOpen.Get() }, Then: func(cx *core.Scope) core.Node {
					return core.View{Class: "flex-row gap-4", Children: []core.Node{
						roundButton(func() *core.SVG {
							if paused.Get() {
								return iconPlay
							}
							return iconPause
						}, "暂停/继续", togglePause),
						roundButton(func() *core.SVG { return iconStop }, "停止专注", exit, danger),
						roundButton(func() *core.SVG { return iconGear }, "专注设置", func() { panelOpen.Set(true) }),
					}}
				}},
			}},

			// Panel backdrop closes the panel on a press.
			core.Show{When: panelOpen.Get, Then: func(cx *core.Scope) core.Node {
				return core.View{
					OnPress: func(core.PressEvent) { panelOpen.Set(false) },
					NoFocus: true,
					Style:   fill,
				}
			}},

			// Settings panel: durations and wallpapers.
			core.Show{When: panelOpen.Get, Then: func(cx *core.Scope) core.Node {
				return settingsPanel(a, walls, wpIndex, wpData, duration, selectDuration)
			}},
		},
	}
}

// overlay puts a scrim over the given fill style.
func overlay(fill core.Style, grad core.Gradient) core.Style {
	fill.Gradients = []core.Gradient{grad}
	return fill
}

// timerDisplay is the ring (countdown only) with the big time text.
func timerDisplay(duration *core.Signal[int], remaining *core.Signal[int]) core.Node {
	timeText := core.Text{ContentFunc: func() string {
		r := remaining.Get()
		return fmt.Sprintf("%02d:%02d", r/60%100, r%60)
	}, Style: core.Style{FontSize: 72, FontWeight: core.FontWeightLight, LetterSpacing: 4, Color: core.RGBA(0xffffff, 0.95),
		TextShadowColor: core.RGBA(0x000000, 0.3), TextShadowRadius: 20}}
	return core.View{
		Style: core.Style{Width: core.Pt(280), Height: core.Pt(280),
			AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter,
			Margin: core.Edges{Bottom: 32}},
		Children: []core.Node{
			core.Show{When: func() bool { return duration.Get() != 0 }, Then: func(cx *core.Scope) core.Node {
				return ui.ProgressRing{
					Value: func() float32 {
						total := duration.Get() * 60
						if total <= 0 {
							return 0
						}
						return float32(total-remaining.Get()) / float32(total) * 100
					},
					Size:      270,
					LineWidth: 3,
					Tint:      func() core.Color { return core.RGBA(0xffffff, 0.7) },
					Style:     core.Style{Position: core.Absolute, Top: core.Pt(0), Left: core.Pt(0)},
				}
			}},
			timeText,
		},
	}
}

// roundButton is a glassy circular control; NoFocus keeps Space free for the
// pause shortcut (focused buttons would take it).
func roundButton(icon func() *core.SVG, label string, onPress func(), mods ...func(*core.Style)) core.Node {
	style := core.Style{
		Width:        core.Pt(48),
		Height:       core.Pt(48),
		CornerRadius: 24,
		Background:   core.RGBA(0xffffff, 0.1),
		BorderColor:  core.RGBA(0xffffff, 0.2), BorderWidth: 1,
		AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter,
	}
	for _, m := range mods {
		m(&style)
	}
	return core.View{
		OnPress: func(core.PressEvent) { onPress() },
		NoFocus: true,
		Label:   label,
		Style:   style,
		StyleFunc: func() core.Style {
			return core.Style{Hover: &core.Style{Background: core.RGBA(0xffffff, 0.2), BorderColor: core.RGBA(0xffffff, 0.3)}}
		},
		Children: []core.Node{core.Icon{SVGFunc: icon, Style: core.Style{Width: core.Pt(20), Height: core.Pt(20), Color: core.RGBA(0xffffff, 0.8)}}},
	}
}

// danger reddens a round button's hover.
func danger(s *core.Style) {
	s.BorderColor = core.RGBA(0xff5050, 0.4)
}

// settingsPanel is the floating card with the duration choices and wallpaper
// thumbnails.
func settingsPanel(a *app, walls []catalog.Wallpaper, wpIndex *core.Signal[int],
	wpData func(int) []byte, duration *core.Signal[int], selectDuration func(int)) core.Node {
	durations := make([]core.Node, 0, len(durationOptions))
	for _, opt := range durationOptions {
		opt := opt
		active := false
		if duration.Get() == opt.Min && opt.Min != 0 {
			active = true
		}
		base := core.Style{
			Padding: core.Symmetric(8, 16), CornerRadius: 12,
			Background:  core.RGBA(0xffffff, 0.06),
			BorderColor: core.RGBA(0xffffff, 0.08), BorderWidth: 1,
			Color: core.RGBA(0xffffff, 0.6),
		}
		if active {
			base.Background = core.RGBA(0xffffff, 0.15)
			base.BorderColor = core.RGBA(0xffffff, 0.25)
			base.Color = core.RGB(0xffffff)
		}
		durations = append(durations, ui.Press{
			OnPress: func() { selectDuration(opt.Min) },
			Label:   opt.Label,
			Style:   base,
			StyleFunc: func() core.Style {
				return core.Style{Hover: &core.Style{Background: core.RGBA(0xffffff, 0.12), Color: core.RGBA(0xffffff, 0.9)}}
			},
			Children: []core.Node{core.Text{Content: opt.Label, Style: core.Style{FontSize: 13}}},
		})
	}

	thumbs := make([]core.Node, 0, len(walls))
	for i := range walls {
		i := i
		thumbStyle := core.Style{
			Width: core.Pt(80), Height: core.Pt(50), CornerRadius: 8,
			BorderColor: core.RGBA(0xffffff, 0.3), BorderWidth: 2,
		}
		if wpIndex.Get() == i {
			thumbStyle.BorderColor = core.RGBA(0xffffff, 0.6)
		}
		thumbs = append(thumbs, core.View{
			OnPress: func(core.PressEvent) { wpIndex.Set(i) },
			NoFocus: true,
			Style:   thumbStyle,
			StyleFunc: func() core.Style {
				return core.Style{Hover: &core.Style{BorderColor: core.RGBA(0xffffff, 0.5)}}
			},
			Children: []core.Node{
				core.Image{
					SourceFunc: func() core.ImageSource {
						return core.ImageSource{Data: wpData(i)}
					},
					ResizeMode: core.Cover,
					Style:      core.Style{Width: core.Pct(100), Height: core.Pct(100), CornerRadius: 6},
				},
			},
		})
	}

	return core.View{
		Style: core.Style{
			Position: core.Absolute, Bottom: core.Pt(60),
			Width: core.Pt(460), MaxHeight: core.Pt(430),
			CornerRadius: 20, Padding: core.Symmetric(24, 28),
			Background:  core.RGBA(0x0f141e, 0.85),
			BorderColor: core.RGBA(0xffffff, 0.08), BorderWidth: 1,
		},
		Children: []core.Node{
			core.View{Class: "flex-col gap-2.5", Children: []core.Node{
				core.Text{Content: "倒计时", Style: labelStyle},
				ui.Wrap{Children: durations},
			}},
			core.View{Class: "flex-col gap-2.5 pt-4", Children: []core.Node{
				core.Text{Content: "背景", Style: labelStyle},
				ui.Wrap{Children: thumbs},
			}},
		},
	}
}
