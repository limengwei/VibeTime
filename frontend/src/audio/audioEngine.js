let audioCtx = null
let masterGain = null

const soundFiles = {
  rain: '/audio/rain.m4a',
  thunder: '/audio/thunder.m4a',
  wind: '/audio/wind.m4a',
  fire: '/audio/fire.m4a',
  ocean: '/audio/waves.m4a',
  birds: '/audio/birds.m4a',
  whiteNoise: '/audio/whitenoise.m4a',
  cafe: '/audio/cafe.m4a',
  forest: '/audio/forest.m4a',
  singingBowl: '/audio/singing-bowl.m4a',
  fan: '/audio/fan.m4a',
  piano: '/audio/piano.m4a'
}

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

async function loadAudioBuffer(type) {
  if (audioBufferCache[type]) {
    return audioBufferCache[type]
  }
  const url = soundFiles[type]
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
    console.warn(`Failed to load audio file for ${type}, falling back to synthesis:`, e)
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

class SynthGenerator {
  constructor(type) {
    this.type = type
    this.ctx = getAudioContext()
    this.output = this.ctx.createGain()
    this.output.gain.value = 0
    this.output.connect(getMasterGain())
    this.sources = []
    this.playing = false
    this.volume = 0.5
  }

  start() {
    if (this.playing) return
    this.playing = true
    this._cleanupSources()
    this._createSources()
    this._fadeIn()
  }

  stop() {
    this.playing = false
    this._fadeOut(() => {
      this._cleanupSources()
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

  _cleanupSources() {
    this.sources.forEach(s => {
      try { s.stop() } catch (e) { /* already stopped */ }
    })
    this.sources = []
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

  _createNoiseBuffer(duration = 2) {
    const sampleRate = this.ctx.sampleRate
    const length = sampleRate * duration
    const buffer = this.ctx.createBuffer(2, length, sampleRate)
    for (let ch = 0; ch < 2; ch++) {
      const data = buffer.getChannelData(ch)
      for (let i = 0; i < length; i++) {
        data[i] = Math.random() * 2 - 1
      }
    }
    return buffer
  }

  _createFilteredNoise(filterType, frequency, Q = 1, duration = 2) {
    const buffer = this._createNoiseBuffer(duration)
    const source = this.ctx.createBufferSource()
    source.buffer = buffer
    source.loop = true
    const filter = this.ctx.createBiquadFilter()
    filter.type = filterType
    filter.frequency.value = frequency
    filter.Q.value = Q
    source.connect(filter)
    filter.connect(this.output)
    source.start()
    this.sources.push(source)
    return source
  }

  _createSources() {
    switch (this.type) {
      case 'rain': this._createRain(); break
      case 'thunder': this._createThunder(); break
      case 'wind': this._createWind(); break
      case 'fire': this._createFire(); break
      case 'ocean': this._createOcean(); break
      case 'birds': this._createBirds(); break
      case 'whiteNoise': this._createWhiteNoise(); break
      case 'cafe': this._createCafe(); break
      case 'forest': this._createForest(); break
      case 'singingBowl': this._createSingingBowl(); break
      case 'fan': this._createFan(); break
      case 'piano': this._createPiano(); break
      default: this._createFilteredNoise('lowpass', 1000)
    }
  }

  _createRain() {
    this._createFilteredNoise('bandpass', 8000, 0.5)
    const bufLen = this.ctx.sampleRate * 0.02
    const buf = this.ctx.createBuffer(1, bufLen, this.ctx.sampleRate)
    const d = buf.getChannelData(0)
    for (let i = 0; i < bufLen; i++) d[i] = Math.random() * 2 - 1
    const tick = this.ctx.createBufferSource()
    tick.buffer = buf
    tick.loop = true
    const tickGain = this.ctx.createGain()
    tickGain.gain.value = 0.15
    const hp = this.ctx.createBiquadFilter()
    hp.type = 'highpass'
    hp.frequency.value = 4000
    tick.connect(hp)
    hp.connect(tickGain)
    tickGain.connect(this.output)
    tick.start()
    this.sources.push(tick)
  }

  _createThunder() {
    this._createFilteredNoise('lowpass', 100, 0.7)
    const lfo = this.ctx.createOscillator()
    lfo.type = 'sine'
    lfo.frequency.value = 0.05
    const lfoGain = this.ctx.createGain()
    lfoGain.gain.value = 0.3
    lfo.connect(lfoGain)
    lfoGain.connect(this.output.gain)
    lfo.start()
    this.sources.push(lfo)
  }

  _createWind() {
    const source = this._createNoiseBuffer(4)
    const s = this.ctx.createBufferSource()
    s.buffer = source
    s.loop = true
    const filter = this.ctx.createBiquadFilter()
    filter.type = 'bandpass'
    filter.frequency.value = 500
    filter.Q.value = 0.3
    const lfo = this.ctx.createOscillator()
    lfo.type = 'sine'
    lfo.frequency.value = 0.15
    const lfoGain = this.ctx.createGain()
    lfoGain.gain.value = 400
    lfo.connect(lfoGain)
    lfoGain.connect(filter.frequency)
    lfo.start()
    s.connect(filter)
    filter.connect(this.output)
    s.start()
    this.sources.push(s, lfo)
  }

  _createFire() {
    this._createFilteredNoise('bandpass', 1000, 0.8)
    const lfo = this.ctx.createOscillator()
    lfo.type = 'sawtooth'
    lfo.frequency.value = 3
    const lfoGain = this.ctx.createGain()
    lfoGain.gain.value = 0.15
    lfo.connect(lfoGain)
    lfoGain.connect(this.output.gain)
    lfo.start()
    this.sources.push(lfo)
  }

  _createOcean() {
    const source = this._createNoiseBuffer(6)
    const s = this.ctx.createBufferSource()
    s.buffer = source
    s.loop = true
    const filter = this.ctx.createBiquadFilter()
    filter.type = 'lowpass'
    filter.frequency.value = 600
    filter.Q.value = 0.5
    const lfo = this.ctx.createOscillator()
    lfo.type = 'sine'
    lfo.frequency.value = 0.08
    const lfoGain = this.ctx.createGain()
    lfoGain.gain.value = 500
    lfo.connect(lfoGain)
    lfoGain.connect(filter.frequency)
    lfo.start()
    const waveLfo = this.ctx.createOscillator()
    waveLfo.type = 'sine'
    waveLfo.frequency.value = 0.08
    const waveGain = this.ctx.createGain()
    waveGain.gain.value = 0.25
    waveLfo.connect(waveGain)
    waveGain.connect(this.output.gain)
    waveLfo.start()
    s.connect(filter)
    filter.connect(this.output)
    s.start()
    this.sources.push(s, lfo, waveLfo)
  }

  _createBirds() {
    const chirp = () => {
      if (!this.playing) return
      const now = this.ctx.currentTime
      const osc = this.ctx.createOscillator()
      const oscGain = this.ctx.createGain()
      const freq = 2000 + Math.random() * 3000
      osc.type = 'sine'
      osc.frequency.setValueAtTime(freq, now)
      osc.frequency.linearRampToValueAtTime(freq * (0.8 + Math.random() * 0.4), now + 0.05)
      osc.frequency.linearRampToValueAtTime(freq * 0.9, now + 0.1)
      oscGain.gain.setValueAtTime(0, now)
      oscGain.gain.linearRampToValueAtTime(0.15, now + 0.02)
      oscGain.gain.linearRampToValueAtTime(0, now + 0.12)
      osc.connect(oscGain)
      oscGain.connect(this.output)
      osc.start(now)
      osc.stop(now + 0.15)
      setTimeout(chirp, 300 + Math.random() * 2000)
    }
    chirp()
    const chirp2 = () => {
      if (!this.playing) return
      const now = this.ctx.currentTime
      const osc = this.ctx.createOscillator()
      const oscGain = this.ctx.createGain()
      osc.type = 'sine'
      const baseFreq = 1500 + Math.random() * 2000
      osc.frequency.setValueAtTime(baseFreq, now)
      osc.frequency.linearRampToValueAtTime(baseFreq * 1.2, now + 0.08)
      osc.frequency.linearRampToValueAtTime(baseFreq * 0.8, now + 0.16)
      oscGain.gain.setValueAtTime(0, now)
      oscGain.gain.linearRampToValueAtTime(0.1, now + 0.02)
      oscGain.gain.linearRampToValueAtTime(0, now + 0.2)
      osc.connect(oscGain)
      oscGain.connect(this.output)
      osc.start(now)
      osc.stop(now + 0.22)
      setTimeout(chirp2, 500 + Math.random() * 3000)
    }
    setTimeout(chirp2, 500)
  }

  _createWhiteNoise() {
    const buf = this._createNoiseBuffer(2)
    const s = this.ctx.createBufferSource()
    s.buffer = buf
    s.loop = true
    s.connect(this.output)
    s.start()
    this.sources.push(s)
  }

  _createCafe() {
    this._createFilteredNoise('bandpass', 2000, 0.3)
    const murmur = this._createNoiseBuffer(3)
    const ms = this.ctx.createBufferSource()
    ms.buffer = murmur
    ms.loop = true
    const mFilter = this.ctx.createBiquadFilter()
    mFilter.type = 'bandpass'
    mFilter.frequency.value = 500
    mFilter.Q.value = 2
    const mGain = this.ctx.createGain()
    mGain.gain.value = 0.08
    const lfo = this.ctx.createOscillator()
    lfo.type = 'sine'
    lfo.frequency.value = 0.2
    const lfoGain = this.ctx.createGain()
    lfoGain.gain.value = 0.04
    lfo.connect(lfoGain)
    lfoGain.connect(mGain.gain)
    lfo.start()
    ms.connect(mFilter)
    mFilter.connect(mGain)
    mGain.connect(this.output)
    ms.start()
    this.sources.push(ms, lfo)
  }

  _createForest() {
    const buf = this._createNoiseBuffer(4)
    const s = this.ctx.createBufferSource()
    s.buffer = buf
    s.loop = true
    const filter = this.ctx.createBiquadFilter()
    filter.type = 'bandpass'
    filter.frequency.value = 3000
    filter.Q.value = 0.3
    const gain = this.ctx.createGain()
    gain.gain.value = 0.06
    s.connect(filter)
    filter.connect(gain)
    gain.connect(this.output)
    s.start()
    this.sources.push(s)
    const chirp = () => {
      if (!this.playing) return
      const now = this.ctx.currentTime
      const osc = this.ctx.createOscillator()
      const oscGain = this.ctx.createGain()
      osc.type = 'sine'
      const freq = 3000 + Math.random() * 2000
      osc.frequency.setValueAtTime(freq, now)
      osc.frequency.linearRampToValueAtTime(freq * 1.1, now + 0.03)
      osc.frequency.linearRampToValueAtTime(freq * 0.95, now + 0.08)
      oscGain.gain.setValueAtTime(0, now)
      oscGain.gain.linearRampToValueAtTime(0.08, now + 0.01)
      oscGain.gain.linearRampToValueAtTime(0, now + 0.1)
      osc.connect(oscGain)
      oscGain.connect(this.output)
      osc.start(now)
      osc.stop(now + 0.12)
      setTimeout(chirp, 800 + Math.random() * 4000)
    }
    chirp()
  }

  _createSingingBowl() {
    const freqs = [261.6, 392, 523.3, 659.3]
    const playBowl = () => {
      if (!this.playing) return
      const now = this.ctx.currentTime
      const freq = freqs[Math.floor(Math.random() * freqs.length)]
      const osc = this.ctx.createOscillator()
      osc.type = 'sine'
      osc.frequency.value = freq
      const oscGain = this.ctx.createGain()
      oscGain.gain.setValueAtTime(0, now)
      oscGain.gain.linearRampToValueAtTime(0.2, now + 0.5)
      oscGain.gain.exponentialRampToValueAtTime(0.001, now + 4)
      osc.connect(oscGain)
      oscGain.connect(this.output)
      osc.start(now)
      osc.stop(now + 4.1)
      const osc2 = this.ctx.createOscillator()
      osc2.type = 'sine'
      osc2.frequency.value = freq * 2
      const oscGain2 = this.ctx.createGain()
      oscGain2.gain.setValueAtTime(0, now)
      oscGain2.gain.linearRampToValueAtTime(0.05, now + 0.3)
      oscGain2.gain.exponentialRampToValueAtTime(0.001, now + 2)
      osc2.connect(oscGain2)
      oscGain2.connect(this.output)
      osc2.start(now)
      osc2.stop(now + 2.1)
      setTimeout(playBowl, 3000 + Math.random() * 5000)
    }
    playBowl()
  }

  _createFan() {
    const buf = this._createNoiseBuffer(2)
    const s = this.ctx.createBufferSource()
    s.buffer = buf
    s.loop = true
    const filter = this.ctx.createBiquadFilter()
    filter.type = 'bandpass'
    filter.frequency.value = 300
    filter.Q.value = 1
    const filter2 = this.ctx.createBiquadFilter()
    filter2.type = 'lowpass'
    filter2.frequency.value = 800
    s.connect(filter)
    filter.connect(filter2)
    filter2.connect(this.output)
    s.start()
    this.sources.push(s)
    const hum = this.ctx.createOscillator()
    hum.type = 'sine'
    hum.frequency.value = 120
    const humGain = this.ctx.createGain()
    humGain.gain.value = 0.06
    hum.connect(humGain)
    humGain.connect(this.output)
    hum.start()
    this.sources.push(hum)
  }

  _createPiano() {
    const pentatonic = [261.6, 293.7, 329.6, 392, 440, 523.3, 587.3, 659.3]
    const playNote = () => {
      if (!this.playing) return
      const now = this.ctx.currentTime
      const freq = pentatonic[Math.floor(Math.random() * pentatonic.length)]
      const osc = this.ctx.createOscillator()
      osc.type = 'triangle'
      osc.frequency.value = freq
      const oscGain = this.ctx.createGain()
      oscGain.gain.setValueAtTime(0, now)
      oscGain.gain.linearRampToValueAtTime(0.12, now + 0.05)
      oscGain.gain.exponentialRampToValueAtTime(0.001, now + 3)
      osc.connect(oscGain)
      oscGain.connect(this.output)
      osc.start(now)
      osc.stop(now + 3.1)
      const osc2 = this.ctx.createOscillator()
      osc2.type = 'sine'
      osc2.frequency.value = freq * 2
      const oscGain2 = this.ctx.createGain()
      oscGain2.gain.setValueAtTime(0, now)
      oscGain2.gain.linearRampToValueAtTime(0.03, now + 0.03)
      oscGain2.gain.exponentialRampToValueAtTime(0.001, now + 1.5)
      osc2.connect(oscGain2)
      oscGain2.connect(this.output)
      osc2.start(now)
      osc2.stop(now + 1.6)
      setTimeout(playNote, 1500 + Math.random() * 3000)
    }
    playNote()
  }
}

const activePlayers = {}

export async function startSound(type) {
  getAudioContext()
  if (activePlayers[type]) {
    activePlayers[type].stop()
    delete activePlayers[type]
  }
  const filePlayer = new FileAudioPlayer(type)
  filePlayer.volume = 0.5
  const loaded = await filePlayer.start()
  if (loaded) {
    activePlayers[type] = filePlayer
    return
  }
  const synth = new SynthGenerator(type)
  synth.start()
  activePlayers[type] = synth
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
