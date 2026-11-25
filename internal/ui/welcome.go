package ui

import (
	"context"
	"fmt"

	"github.com/nsgo-dev/nsgo/base"
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/ui"

	"vibe-time/internal/pack"
)

// welcomeState feeds the first-launch download dialog: the sound pack is
// not shipped with the app, so a cold start walks the user through
// fetching it before anything can play.
type welcomeState struct {
	open    *core.Signal[bool]
	pack    *pack.Pack
	ready   func() bool // the manifest has arrived
	failed  func() bool // the manifest could not be fetched
	onRetry func()      // try the manifest again
	onReady func()      // download complete: close and enter the app
}

// dlProgress is the download's latest report: a phase ("下载" the archive
// in bytes, "入库"/"补齐" files) with its done/total pair.
type dlProgress struct {
	phase     string
	done, tot int64
}

// welcomePanel is the modal first-launch dialog: an introduction, a
// download button with live progress, and retry on failure. It cannot be
// dismissed — downloading is the only way on.
func welcomePanel(cx *core.Scope, s welcomeState) core.Node {
	stateSig := core.NewSignal(cx, "idle") // idle | running | error
	progSig := core.NewSignal(cx, dlProgress{})

	start := func() {
		if stateSig.Get() == "running" {
			return
		}
		stateSig.Set("running")
		go func() {
			err := s.pack.Download(context.Background(), func(phase string, done, total int64) {
				p := dlProgress{phase: phase, done: done, tot: total}
				cx.Post(func() { progSig.Set(p) })
			})
			if err != nil {
				cx.Post(func() {
					stateSig.Set("error")
					progSig.Set(dlProgress{phase: err.Error()})
				})
				return
			}
			cx.Post(s.onReady)
		}()
	}

	return ui.Dialog{
		Open:                   s.open.Get,
		OnOpenChange:           func(open bool, _ base.Reason) {},
		NoCloseOnEscape:        true,
		NoCloseOnBackdropPress: true,
		Title:                  "欢迎来到 Vibe Time",

		Children: []core.Node{
			core.View{Class: "gap-1.5 flex-col items-center", Children: []core.Node{
				core.Text{Content: "89 个环境音效、9 大分类、12 个内置场景，自由组合属于你的宁静空间。", Style: core.Style{FontSize: 12, Color: core.RGBA(0xffffff, 0.55), TextAlign: core.TextAlignCenter}},
				core.Text{Content: "声音库不随安装包分发，首次使用需下载（约 48.5 MB，经镜像加速的 GitHub 仓库），仅此一次，之后离线可用。", Style: core.Style{FontSize: 12, Color: core.RGBA(0xffffff, 0.4), TextAlign: core.TextAlignCenter}},
			}},

			// download button / progress / error. TextFunc keeps the label
			// live: the branch builds before the manifest arrives, and the
			// button must switch from fetching to ready without a rebuild.
			core.Show{When: func() bool { return stateSig.Get() != "running" }, Then: func(cx *core.Scope) core.Node {
				return core.View{Class: "gap-2 flex-col items-center", Children: []core.Node{
					ui.Button{
						TextFunc: func() string {
							switch {
							case s.failed():
								return "网络连接失败，点击重试"
							case !s.ready():
								return "正在获取声音列表…"
							case stateSig.Get() == "error":
								return "重试下载"
							}
							return "下载声音库（约 48.5 MB）"
						},
						Disabled: func() bool { return !s.ready() && !s.failed() },
						OnPress: func() {
							if s.failed() && !s.ready() {
								s.onRetry()
								return
							}
							start()
						},
					},
					core.Show{When: func() bool { return stateSig.Get() == "error" }, Then: func(cx *core.Scope) core.Node {
						return core.Text{ContentFunc: func() string { return progSig.Get().phase },
							Style: core.Style{FontSize: 11, Color: core.RGB(0xf87171), TextAlign: core.TextAlignCenter}}
					}},
				}}
			}},
			core.Show{When: func() bool { return stateSig.Get() == "running" }, Then: func(cx *core.Scope) core.Node {
				return core.View{Class: "w-[420px] max-w-full gap-2 flex-col items-center", Children: []core.Node{
					core.Text{ContentFunc: func() string {
						p := progSig.Get()
						switch p.phase {
						case "下载": // the archive, in bytes
							if p.tot > 0 {
								return fmt.Sprintf("下载声音包 %s / %s", humanBytes(p.done), humanBytes(p.tot))
							}
							if p.done > 0 {
								return fmt.Sprintf("下载声音包 %s", humanBytes(p.done))
							}
							return "下载声音包…"
						case "入库": // files verified and stored out of the archive
							return fmt.Sprintf("校验入库 %d / %d 个文件", p.done, p.tot)
						case "补齐": // files fetched one by one
							return fmt.Sprintf("补齐声音文件 %d / %d", p.done, p.tot)
						}
						return "准备下载…"
					}, Style: core.Style{FontSize: 12, Color: core.RGBA(0xffffff, 0.55)}},
					ui.Progress{
						Value: func() float32 {
							p := progSig.Get()
							if p.tot <= 0 {
								return 0
							}
							return min(float32(p.done)/float32(p.tot), 1) * 100
						},
						Indeterminate: func() bool { return progSig.Get().tot <= 0 },
						Style:         core.Style{Width: core.Pct(100)},
					},
				}}
			}},

			core.Text{ContentFunc: func() string {
				return fmt.Sprintf("缓存位置：%s", s.pack.Dir())
			}, Style: core.Style{FontSize: 10, Color: core.RGBA(0xffffff, 0.3)}},
		},
	}
}

// humanBytes formats a byte count for display.
func humanBytes(n int64) string {
	switch {
	case n >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(n)/(1<<30))
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.0f KB", float64(n)/(1<<10))
	default:
		return fmt.Sprintf("%d B", n)
	}
}
