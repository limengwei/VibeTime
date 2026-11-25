// Package main embeds vibe-time's shipped assets: the ambient sound library,
// the focus-mode wallpapers and the tray icon (PNG; nsgo takes PNG only).
package main

import (
	"embed"
	"io/fs"
)

//go:embed all:assets/audio
var audioAssets embed.FS

//go:embed all:assets/wallpapers
var wallpaperAssets embed.FS

// audioFS is the sound library rooted at the audio/ directory.
func audioFS() fs.FS {
	sub, err := fs.Sub(audioAssets, "assets/audio")
	if err != nil {
		panic(err) // the embedded tree is fixed at build time
	}
	return sub
}

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
