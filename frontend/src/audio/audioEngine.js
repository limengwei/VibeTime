let audioCtx = null
let masterGain = null

const soundFileMap = {}
const audioBufferCache = {}
const MAX_CACHE_SIZE = 8

const cacheOrder = []

function getAudioContext() {
  if (!audioCtx) {
    audioCtx = new (window.AudioContext || window.webkitAudioContext)({
      latencyHint: 'playback'
    })
    masterGain = audioCtx.createGain()
    masterGain.gain.value = 0.8
    masterGain.connect(audioCtx.destination)
  }
  if (audioCtx.state === 'suspended') {
    audioCtx.resume()
  }
  return audioCtx
}

function getMasterGainNode() {
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

function evictCache() {
  while (cacheOrder.length > MAX_CACHE_SIZE) {
    const oldId = cacheOrder.shift()
    if (oldId && audioBufferCache[oldId] && !activePlayers[oldId]) {
      delete audioBufferCache[oldId]
    }
  }
}

function touchCache(id) {
  const idx = cacheOrder.indexOf(id)
  if (idx !== -1) {
    cacheOrder.splice(idx, 1)
  }
  cacheOrder.push(id)
  evictCache()
}

async function loadAudioBuffer(id) {
  if (audioBufferCache[id]) {
    touchCache(id)
    return audioBufferCache[id]
  }
  const url = soundFileMap[id]
  if (!url) return null
  try {
    const response = await fetch(url)
    if (!response.ok) return null
    const arrayBuffer = await response.arrayBuffer()
    const ctx = getAudioContext()
    const audioBuffer = await ctx.decodeAudioData(arrayBuffer)
    audioBufferCache[id] = audioBuffer
    touchCache(id)
    return audioBuffer
  } catch (e) {
    console.warn('Failed to load audio:', id, e)
    return null
  }
}

export function releaseBuffer(id) {
  if (audioBufferCache[id] && !activePlayers[id]) {
    delete audioBufferCache[id]
    const idx = cacheOrder.indexOf(id)
    if (idx !== -1) cacheOrder.splice(idx, 1)
  }
}

class FileAudioPlayer {
  constructor(id) {
    this.id = id
    this.source = null
    this.gainNode = null
    this.playing = false
    this.volume = 0.5
    this._stopping = false
  }

  start(buffer) {
    if (this.playing) return
    this.playing = true
    const ctx = getAudioContext()

    this.gainNode = ctx.createGain()
    this.gainNode.gain.value = 0
    this.gainNode.connect(masterGain)

    this.source = ctx.createBufferSource()
    this.source.buffer = buffer
    this.source.loop = true
    this.source.connect(this.gainNode)

    const now = ctx.currentTime
    this.gainNode.gain.setValueAtTime(0, now)
    this.gainNode.gain.linearRampToValueAtTime(this.volume, now + 0.3)

    this.source.start()
  }

  stop() {
    if (!this.playing || this._stopping) return
    this._stopping = true
    this.playing = false

    const ctx = getAudioContext()
    const gain = this.gainNode
    const src = this.source

    if (gain) {
      const now = ctx.currentTime
      gain.gain.cancelScheduledValues(now)
      gain.gain.setValueAtTime(gain.gain.value, now)
      gain.gain.linearRampToValueAtTime(0, now + 0.15)
    }

    setTimeout(() => {
      try { src && src.stop() } catch (e) {}
      try { gain && gain.disconnect() } catch (e) {}
      this._stopping = false
    }, 200)
  }

  setVolume(v) {
    this.volume = v
    if (this.gainNode && this.playing) {
      const ctx = getAudioContext()
      this.gainNode.gain.cancelScheduledValues(ctx.currentTime)
      this.gainNode.gain.value = v
    }
  }
}

const activePlayers = {}

export async function startSound(id, startVolume = 0.5) {
  getAudioContext()

  if (activePlayers[id]) {
    activePlayers[id].stop()
    delete activePlayers[id]
  }

  const buffer = await loadAudioBuffer(id)
  if (!buffer) {
    console.warn('No audio buffer for:', id)
    return
  }

  const player = new FileAudioPlayer(id)
  player.volume = startVolume
  player.start(buffer)
  activePlayers[id] = player
}

export function stopSound(id) {
  if (activePlayers[id]) {
    activePlayers[id].stop()
    delete activePlayers[id]
    releaseBuffer(id)
  }
}

export function setSoundVolume(id, volume) {
  if (activePlayers[id]) {
    activePlayers[id].setVolume(volume)
  }
}

export function setMasterVolume(volume) {
  const mg = getMasterGainNode()
  mg.gain.value = volume
}

export function stopAll() {
  Object.keys(activePlayers).forEach(id => {
    activePlayers[id].stop()
    delete activePlayers[id]
    releaseBuffer(id)
  })
}

export function resumeAudioContext() {
  getAudioContext()
}
