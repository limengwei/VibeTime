// Package main embeds vibe-time's shipped assets: the ambient sound library
// and the focus-mode wallpapers, both served from their original
// frontend/public locations, plus the tray icon (PNG; nsgo takes PNG only).
package main

import (
	"embed"
	"io/fs"
)

//go:embed all:frontend/public/audio
var audioAssets embed.FS

//go:embed all:frontend/public/wallpapers
var wallpaperAssets embed.FS

// audioFS is the sound library rooted at the audio/ directory.
func audioFS() fs.FS {
	sub, err := fs.Sub(audioAssets, "frontend/public/audio")
	if err != nil {
		panic(err) // the embedded tree is fixed at build time
	}
	return sub
}

// wallpaperFS is the wallpaper set rooted at the wallpapers/ directory.
func wallpaperFS() fs.FS {
	sub, err := fs.Sub(wallpaperAssets, "frontend/public/wallpapers")
	if err != nil {
		panic(err) // the embedded tree is fixed at build time
	}
	return sub
}

//go:embed build/appicon.png
var trayIcon []byte
