// Package audio drives vibe-time's ambient mixing: any number of looping
// players with per-track volumes under one master volume, on nsgo's
// audio.Player.
//
// Quirk handled here: core's SetVolume treats 0 as 1 ("0 is 1"), so the
// effective volume is clamped to at least 0.01 and a zero master mutes via
// SetMuted instead.
package audio

import (
	"log"

	"github.com/nsgo-dev/nsgo/audio"
	"github.com/nsgo-dev/nsgo/core"
)

// track is one looping sound and its own volume (0.05–1).
type track struct {
	player *audio.Player
	volume float64
}

// Engine mixes concurrent looping sounds. All methods run on the engine's
// goroutine (from render, handlers and AfterFunc callbacks), like the
// players they drive.
type Engine struct {
	cx     *core.Scope
	master float64
	tracks map[string]*track
	// Count signals how many sounds play, for reactive UI.
	Count *core.Signal[int]

	source func(id string) ([]byte, error)
}

// NewEngine makes an engine at master volume 0.8, loading sound bytes
// through source.
func NewEngine(cx *core.Scope, source func(id string) ([]byte, error)) *Engine {
	return &Engine{
		cx:     cx,
		master: 0.8,
		tracks: map[string]*track{},
		Count:  core.NewSignal(cx, 0),
		source: source,
	}
}

// EffectiveVolume scales a track volume by the master volume, never reaching
// the 0 that core's SetVolume reads as 1.
func EffectiveVolume(trackVol, master float64) float64 {
	v := trackVol * master
	if v < 0.01 {
		v = 0.01
	}
	if v > 1 {
		v = 1
	}
	return v
}

// applyMaster sets a track's audible volume for the current master, muting
// rather than scaling to 0.
func (e *Engine) applyMaster(t *track) {
	if e.master <= 0.0001 {
		t.player.SetMuted(true)
		return
	}
	t.player.SetMuted(false)
	t.player.SetVolume(float32(EffectiveVolume(t.volume, e.master)))
}

// Play starts id looping at volume (clamped to 0.05–1); a playing sound just
// takes the new volume.
func (e *Engine) Play(id string, volume float64) {
	if volume < 0.05 {
		volume = 0.05
	}
	if volume > 1 {
		volume = 1
	}
	if t, ok := e.tracks[id]; ok {
		t.volume = volume
		e.applyMaster(t)
		return
	}
	data, err := e.source(id)
	if err != nil {
		log.Printf("[audio] load %s: %v", id, err)
		return
	}
	p := audio.NewPlayer(e.cx, audio.Source{Data: data}, audio.PlayerOptions{
		OnError: func(err error) { log.Printf("[audio] %s: %v", id, err) },
	})
	p.SetLoop(true)
	t := &track{player: p, volume: volume}
	e.applyMaster(t)
	p.Play(nil)
	e.tracks[id] = t
	e.Count.Set(len(e.tracks))
}

// Stop stops and releases one sound.
func (e *Engine) Stop(id string) {
	if t, ok := e.tracks[id]; ok {
		t.player.Close()
		delete(e.tracks, id)
		e.Count.Set(len(e.tracks))
	}
}

// StopAll stops every sound.
func (e *Engine) StopAll() {
	for id, t := range e.tracks {
		t.player.Close()
		delete(e.tracks, id)
	}
	e.Count.Set(0)
}

// SetVolume changes one playing sound's own volume.
func (e *Engine) SetVolume(id string, volume float64) {
	if t, ok := e.tracks[id]; ok {
		if volume < 0.05 {
			volume = 0.05
		}
		if volume > 1 {
			volume = 1
		}
		t.volume = volume
		e.applyMaster(t)
	}
}

// SetMaster scales every playing sound; 0 mutes them without stopping.
func (e *Engine) SetMaster(v float64) {
	if v < 0 {
		v = 0
	}
	if v > 1 {
		v = 1
	}
	e.master = v
	for _, t := range e.tracks {
		e.applyMaster(t)
	}
}

// Master is the current master volume.
func (e *Engine) Master() float64 { return e.master }

// Playing reports whether id is playing.
func (e *Engine) Playing(id string) bool {
	_, ok := e.tracks[id]
	return ok
}

// PlayingIDs lists the playing sound IDs (unordered).
func (e *Engine) PlayingIDs() []string {
	ids := make([]string, 0, len(e.tracks))
	for id := range e.tracks {
		ids = append(ids, id)
	}
	return ids
}
