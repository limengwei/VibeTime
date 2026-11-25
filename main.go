// Command vibe-time is an ambient sound mixer: any number of looping sounds
// under scene presets, with a focus timer over random wallpapers. It stays in
// the tray when its window closes.
package main

import (
	"fmt"
	"io/fs"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/nsgo-dev/nsgo"
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/desktop"

	"vibe-time/internal/audio"
	"vibe-time/internal/catalog"
	"vibe-time/internal/pack"
	"vibe-time/internal/ui"
)

// singleInstanceAddr is the local lock that keeps one GUI instance: the
// running process listens on it.
const singleInstanceAddr = "127.0.0.1:9395"

// singletonLock holds the single-instance listener for the process's
// lifetime: unreferenced, its os.File finalizer would close it and free the
// port while the app keeps running.
var singletonLock net.Listener

// acquireSingleInstance takes the single-instance lock. A previous instance
// still holding it (hiding in the tray) is killed, so the newest launch
// always wins and shows its own window.
func acquireSingleInstance() {
	ln, err := net.Listen("tcp", singleInstanceAddr)
	if err != nil {
		killPreviousInstance()
		for i := 0; i < 30 && err != nil; i++ {
			time.Sleep(100 * time.Millisecond)
			ln, err = net.Listen("tcp", singleInstanceAddr)
		}
		if err != nil {
			// A foreign program squats the port; run without the lock.
			fmt.Fprintf(os.Stderr, "[单实例] %s 被其他程序占用，跳过单实例保护\n", singleInstanceAddr)
			return
		}
	}
	singletonLock = ln
}

// killPreviousInstance force-kills the process listening on the singleton
// port — the previous GUI instance.
func killPreviousInstance() {
	out, err := exec.Command("netstat", "-ano", "-p", "tcp").Output()
	if err != nil {
		return
	}
	self := os.Getpid()
	for _, line := range strings.Split(string(out), "\n") {
		if !strings.Contains(line, "LISTENING") || !strings.Contains(line, singleInstanceAddr) {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		pid := fields[len(fields)-1]
		if pid == fmt.Sprint(self) {
			continue
		}
		_ = exec.Command("taskkill", "/F", "/PID", pid).Run()
	}
}

func main() {
	// The lock lives for the process's lifetime; there is no in-app restart.
	acquireSingleInstance()
	nsgo.Run(func(app *core.Scope) {
		win := desktop.OpenWindow(app, desktop.WindowOptions{
			Title: "Vibe Time", Width: 1014, Height: 755, Key: "main",
		}, func(cx *core.Scope) core.Node {
			// The catalog starts empty: every sound lives in the separate
			// vibe-time-sound repository and arrives through the pack.
			cats := catalog.Empty()
			store, err := catalog.OpenStore()
			if err != nil {
				fmt.Fprintf(os.Stderr, "[预设] 读取失败，使用空白预设：%v\n", err)
				store = &catalog.Store{}
			}
			pk, err := pack.Open(cx)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[声音包] 缓存目录不可用，无法下载声音：%v\n", err)
				pk = nil
			}
			var engine *audio.Engine
			if pk != nil {
				engine = audio.NewEngine(cx, pk.Read)
			} else {
				engine = audio.NewEngine(cx, func(string) ([]byte, error) {
					return nil, fs.ErrNotExist
				})
			}
			walls := catalog.Wallpapers(wallpaperFS())
			wallFS := wallpaperFS()
			return ui.Main(cx, ui.Deps{
				Catalog: cats, Store: store, Engine: engine, Pack: pk,
				Wallpapers: walls,
				WallpaperData: func(file string) ([]byte, error) {
					return fs.ReadFile(wallFS, file)
				},
			})
		})

		// Closing the window hides it to the tray instead of quitting, so
		// the sounds keep playing; the tray menu shows the window or quits.
		win.OnClosing(app, func(c *core.WindowClosing) {
			c.Cancel()
			win.Hide()
		})
		desktop.NewTray(app, desktop.TrayOptions{
			Icon:    trayIcon, // PNG; nsgo takes PNG only
			Tooltip: "Vibe Time · 轻松找到你的宁静",
			OnClick: win.Show,
			Menu: func() []core.MenuItem {
				return []core.MenuItem{
					{Label: "显示窗口", OnSelect: win.Show},
					{Separator: true},
					{Label: "退出", OnSelect: app.Quit},
				}
			},
		})
	}, nsgo.Appearance(core.AppearanceDark))
}
