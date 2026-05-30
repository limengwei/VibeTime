package main

import (
	"context"
	_ "embed"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"fyne.io/systray"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed build/windows/icon.ico
var trayIcon []byte

func getAssetsDir() string {
	candidates := []string{}

	if exePath, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exePath), "frontend", "dist"))
	}

	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(wd, "frontend", "dist"))
	}

	for _, dir := range candidates {
		if stat, err := os.Stat(dir); err == nil && stat.IsDir() {
			return dir
		}
	}

	panic("frontend/dist not found in any candidate path")
}

func main() {
	app := NewApp()
	assetsDir := getAssetsDir()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	loop, _ := systray.RunWithExternalLoop(func() {
		systray.SetIcon(trayIcon)
		systray.SetTooltip("Vibe Time")
		systray.SetOnTapped(func() {
			app.ShowWindow()
		})

		mShow := systray.AddMenuItem("显示窗口", "")
		systray.AddSeparator()
		mQuit := systray.AddMenuItem("退出", "")

		go func() {
			for {
				select {
				case <-mShow.ClickedCh:
					app.ShowWindow()
				case <-mQuit.ClickedCh:
					systray.Quit()
				}
			}
		}()
	}, func() {
		os.Exit(0)
	})

	go loop()

	go func() {
		<-sigCh
		os.Exit(0)
	}()

	err := wails.Run(&options.App{
		Title:     "Vibe Time",
		Width:     900,
		Height:    680,
		MinWidth:  600,
		MinHeight: 500,
		Frameless: true,
		AssetServer: &assetserver.Options{
			Assets: os.DirFS(assetsDir),
		},
		BackgroundColour: &options.RGBA{R: 10, G: 14, B: 23, A: 255},
		OnStartup:        app.startup,
		OnBeforeClose: func(ctx context.Context) bool {
			runtime.WindowHide(ctx)
			return true
		},
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId: "vibe-time-a3f8c9d2-e7b1-4d6a-b5c3-9e8f2a1d0c7b",
			OnSecondInstanceLaunch: func(secondInstanceData options.SecondInstanceData) {
				app.ShowWindow()
			},
		},
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
