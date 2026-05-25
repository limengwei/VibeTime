let audioCtx = null
let masterGain = null

const soundFileMap = {}
const audioBufferCache = {}

function getAudioContext() {
  if (!audioCtx) {
    audioCtx = new (window.AudioContext || window.webkitAudioContext)()
    masterGain = audioCtx.createGain()
    masterGain.gain.value = 0.8
    masterGain.connect(audioCtx.destination)
  }
  if (audioCtx.state === 'suspended') {
    audioCtx.resume()
  }
  return audioCtx
}

function getMasterGain() {
  getAudioContext()
  return masterGain
}

export function registerSound(id, src) {
  soundFileMap[id] = src
}

export function registerSounds(sounds) {
  sounds.forEach(s => {
    soundFileMap[s.id] = s.src
  })
}

async function loadAudioBuffer(type) {
  if (audioBufferCache[type]) {
    return audioBufferCache[type]
  }
  const url = soundFileMap[type]
  if (!url) return null
  try {
    const response = await fetch(url)
    if (!response.ok) return null
    const arrayBuffer = await response.arrayBuffer()
    const ctx = getAudioContext()
    const audioBuffer = await ctx.decodeAudioData(arrayBuffer)
    audioBufferCache[type] = audioBuffer
    return audioBuffer
  } catch (e) {
    console.warn(`Failed to load audio file for ${type}:`, e)
    return null
  }
}

class FileAudioPlayer {
  constructor(type) {
    this.type = type
    this.ctx = getAudioContext()
    this.output = this.ctx.createGain()
    this.output.gain.value = 0
    this.output.connect(getMasterGain())
    this.source = null
    this.playing = false
    this.volume = 0.5
    this.buffer = null
  }

  async start() {
    if (this.playing) return
    this.playing = true
    if (!this.buffer) {
      this.buffer = await loadAudioBuffer(this.type)
    }
    if (!this.buffer || !this.playing) {
      return false
    }
    this._playBuffer()
    return true
  }

  _playBuffer() {
    this.source = this.ctx.createBufferSource()
    this.source.buffer = this.buffer
    this.source.loop = true
    this.source.connect(this.output)
    this._fadeIn()
    this.source.start()
  }

  stop() {
    this.playing = false
    this._fadeOut(() => {
      if (this.source) {
        try { this.source.stop() } catch (e) { /* already stopped */ }
        this.source = null
      }
    })
  }

  setVolume(v) {
    this.volume = v
    if (this.playing) {
      const now = this.ctx.currentTime
      this.output.gain.cancelScheduledValues(now)
      this.output.gain.setValueAtTime(this.output.gain.value, now)
      this.output.gain.linearRampToValueAtTime(v, now + 0.05)
    }
  }

  _fadeIn() {
    const now = this.ctx.currentTime
    this.output.gain.cancelScheduledValues(now)
    this.output.gain.setValueAtTime(0, now)
    this.output.gain.linearRampToValueAtTime(this.volume, now + 0.5)
  }

  _fadeOut(callback) {
    const now = this.ctx.currentTime
    this.output.gain.cancelScheduledValues(now)
    this.output.gain.setValueAtTime(this.output.gain.value, now)
    this.output.gain.linearRampToValueAtTime(0, now + 0.3)
    if (callback) {
      setTimeout(callback, 350)
    }
  }
}

const activePlayers = {}

export async function startSound(type) {
  getAudioContext()
  if (activePlayers[type]) {
    activePlayers[type].stop()
    delete activePlayers[type]
  }
  const player = new FileAudioPlayer(type)
  player.volume = 0.5
  const loaded = await player.start()
  if (loaded) {
    activePlayers[type] = player
    return
  }
  console.warn(`No audio file found for ${type}`)
}

export function stopSound(type) {
  if (activePlayers[type]) {
    activePlayers[type].stop()
    delete activePlayers[type]
  }
}

export function setSoundVolume(type, volume) {
  if (activePlayers[type]) {
    activePlayers[type].setVolume(volume)
  }
}

export function setMasterVolume(volume) {
  const mg = getMasterGain()
  const now = getAudioContext().currentTime
  mg.gain.cancelScheduledValues(now)
  mg.gain.setValueAtTime(mg.gain.value, now)
  mg.gain.linearRampToValueAtTime(volume, now + 0.05)
}

export function stopAll() {
  Object.keys(activePlayers).forEach(type => {
    activePlayers[type].stop()
    delete activePlayers[type]
  })
}

export function resumeAudioContext() {
  getAudioContext()
}
