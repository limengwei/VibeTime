// Package main embeds vibe-time's shipped assets: the focus-mode
// wallpapers and the tray icon (PNG; nsgo takes PNG only). The sound
// library is not shipped — it downloads from the vibe-time-sound
// repository at runtime.
package main

import (
	"embed"
	"io/fs"
)

//go:embed all:assets/wallpapers
var wallpaperAssets embed.FS

// wallpaperFS is the wallpaper set rooted at the wallpapers/ directory.
func wallpaperFS() fs.FS {
	sub, err := fs.Sub(wallpaperAssets, "assets/wallpapers")
	if err != nil {
		panic(err) // the embedded tree is fixed at build time
	}
	return sub
}

//go:embed assets/appicon.png
var trayIcon []byte
