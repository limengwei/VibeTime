<script setup>
import { ref, reactive, computed } from 'vue'
import SoundCard from './components/SoundCard.vue'
import {
  startSound,
  stopSound,
  setSoundVolume,
  setMasterVolume,
  stopAll,
  resumeAudioContext
} from './audio/audioEngine.js'

const sounds = reactive([
  { id: 'rain', name: 'Rain', description: 'Heavy rainfall', iconKey: 'rain', playing: false, volume: 0.5 },
  { id: 'thunder', name: 'Thunder', description: 'Distant storm', iconKey: 'thunder', playing: false, volume: 0.5 },
  { id: 'wind', name: 'Wind', description: 'Howling wind', iconKey: 'wind', playing: false, volume: 0.5 },
  { id: 'fire', name: 'Fireplace', description: 'Crackling fire', iconKey: 'fire', playing: false, volume: 0.5 },
  { id: 'ocean', name: 'Ocean Waves', description: 'Calming sea waves', iconKey: 'ocean', playing: false, volume: 0.5 },
  { id: 'birds', name: 'Birds', description: 'Morning chirping', iconKey: 'birds', playing: false, volume: 0.5 },
  { id: 'whiteNoise', name: 'White Noise', description: 'Consistent static', iconKey: 'whiteNoise', playing: false, volume: 0.5 },
  { id: 'cafe', name: 'Cafe', description: 'Coffee shop ambience', iconKey: 'cafe', playing: false, volume: 0.5 },
  { id: 'forest', name: 'Forest', description: 'Birds and leaves', iconKey: 'forest', playing: false, volume: 0.5 },
  { id: 'singingBowl', name: 'Singing Bowl', description: 'Tibetan meditation', iconKey: 'singingBowl', playing: false, volume: 0.5 },
  { id: 'fan', name: 'Fan', description: 'Room fan noise', iconKey: 'fan', playing: false, volume: 0.5 },
  { id: 'piano', name: 'Piano', description: 'Soft melodies', iconKey: 'piano', playing: false, volume: 0.5 }
])

const masterVolume = ref(0.8)
const activeCount = computed(() => sounds.filter(s => s.playing).length)

const presets = [
  { name: 'Relax', sounds: { rain: 0.6, thunder: 0.2, fire: 0.4 } },
  { name: 'Meditation', sounds: { singingBowl: 0.5, rain: 0.2 } },
  { name: 'Sleep', sounds: { rain: 0.5, thunder: 0.3, whiteNoise: 0.3 } },
  { name: 'Focus', sounds: { whiteNoise: 0.4, cafe: 0.3, fan: 0.3 } },
  { name: 'Study', sounds: { cafe: 0.5, rain: 0.3 } },
  { name: 'Ocean', sounds: { ocean: 0.6, wind: 0.3, birds: 0.2 } },
  { name: 'Cafe', sounds: { cafe: 0.6, rain: 0.2 } },
  { name: 'Work', sounds: { fan: 0.4, whiteNoise: 0.3 } }
]

const activePreset = ref(null)

async function toggleSound(id) {
  resumeAudioContext()
  const sound = sounds.find(s => s.id === id)
  if (!sound) return
  if (sound.playing) {
    sound.playing = false
    stopSound(id)
  } else {
    sound.playing = true
    await startSound(id)
    setSoundVolume(id, sound.volume)
  }
  activePreset.value = null
}

function changeVolume(id, volume) {
  const sound = sounds.find(s => s.id === id)
  if (!sound) return
  sound.volume = volume
  if (sound.playing) {
    setSoundVolume(id, volume)
  }
}

function onMasterVolumeChange(e) {
  masterVolume.value = parseFloat(e.target.value)
  setMasterVolume(masterVolume.value)
}

async function applyPreset(preset) {
  resumeAudioContext()
  sounds.forEach(s => {
    s.playing = false
    stopSound(s.id)
  })
  for (const [id, vol] of Object.entries(preset.sounds)) {
    const sound = sounds.find(s => s.id === id)
    if (sound) {
      sound.volume = vol
      sound.playing = true
      await startSound(id)
      setSoundVolume(id, vol)
    }
  }
  activePreset.value = preset.name
}

function stopAllSounds() {
  sounds.forEach(s => {
    s.playing = false
    stopSound(s.id)
  })
  activePreset.value = null
}
</script>

<template>
  <div class="app-container">
    <header class="app-header">
      <div class="header-top">
        <h1 class="app-title">Vibe Time</h1>
        <p class="app-subtitle">Find Your Peace with Ease</p>
      </div>
      <div class="preset-bar">
        <button
          v-for="preset in presets"
          :key="preset.name"
          class="preset-btn"
          :class="{ active: activePreset === preset.name }"
          @click="applyPreset(preset)"
        >
          {{ preset.name }}
        </button>
      </div>
    </header>

    <main class="sound-grid">
      <SoundCard
        v-for="sound in sounds"
        :key="sound.id"
        :sound="sound"
        @toggle="toggleSound"
        @volume-change="changeVolume"
      />
    </main>

    <footer class="app-footer">
      <div class="footer-left">
        <span class="active-count" v-if="activeCount > 0">
          {{ activeCount }} sound{{ activeCount > 1 ? 's' : '' }} playing
        </span>
      </div>
      <div class="footer-center">
        <button class="stop-btn" @click="stopAllSounds" v-if="activeCount > 0">
          <svg viewBox="0 0 24 24" fill="currentColor" width="16" height="16">
            <rect x="6" y="6" width="12" height="12" rx="1"/>
          </svg>
          Stop All
        </button>
      </div>
      <div class="footer-right">
        <svg class="volume-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" width="18" height="18">
          <polygon points="11 5 6 9 2 9 2 15 6 15 11 19 11 5"/>
          <path d="M19.07 4.93a10 10 0 0 1 0 14.14"/>
          <path d="M15.54 8.46a5 5 0 0 1 0 7.07"/>
        </svg>
        <input
          type="range"
          min="0"
          max="1"
          step="0.01"
          :value="masterVolume"
          @input="onMasterVolumeChange"
          class="master-slider"
        />
      </div>
    </footer>
  </div>
</template>

<style scoped>
.app-container {
  display: flex;
  flex-direction: column;
  height: 100vh;
  overflow: hidden;
  background: linear-gradient(145deg, #0a0e17 0%, #111827 50%, #0f1520 100%);
}

.app-header {
  padding: 24px 32px 0;
  flex-shrink: 0;
}

.header-top {
  text-align: center;
  margin-bottom: 20px;
}

.app-title {
  font-size: 28px;
  font-weight: 700;
  color: rgba(255, 255, 255, 0.95);
  margin: 0;
  letter-spacing: 1px;
}

.app-subtitle {
  font-size: 13px;
  color: rgba(255, 255, 255, 0.35);
  margin: 4px 0 0;
  letter-spacing: 0.5px;
}

.preset-bar {
  display: flex;
  gap: 8px;
  justify-content: center;
  flex-wrap: wrap;
  padding-bottom: 16px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.06);
}

.preset-btn {
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 20px;
  color: rgba(255, 255, 255, 0.55);
  padding: 6px 16px;
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
  letter-spacing: 0.3px;
}

.preset-btn:hover {
  background: rgba(255, 255, 255, 0.1);
  color: rgba(255, 255, 255, 0.8);
  border-color: rgba(255, 255, 255, 0.15);
}

.preset-btn.active {
  background: rgba(255, 255, 255, 0.15);
  color: white;
  border-color: rgba(255, 255, 255, 0.25);
}

.sound-grid {
  flex: 1;
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
  gap: 12px;
  padding: 20px 32px;
  overflow-y: auto;
  align-content: start;
}

.app-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 32px;
  border-top: 1px solid rgba(255, 255, 255, 0.06);
  flex-shrink: 0;
  background: rgba(0, 0, 0, 0.2);
}

.footer-left, .footer-right {
  flex: 1;
}

.footer-right {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
}

.footer-center {
  display: flex;
  justify-content: center;
}

.active-count {
  font-size: 12px;
  color: rgba(255, 255, 255, 0.4);
}

.stop-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  background: rgba(255, 80, 80, 0.15);
  border: 1px solid rgba(255, 80, 80, 0.2);
  border-radius: 8px;
  color: rgba(255, 120, 120, 0.9);
  padding: 6px 14px;
  font-size: 12px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.stop-btn:hover {
  background: rgba(255, 80, 80, 0.25);
  border-color: rgba(255, 80, 80, 0.3);
}

.stop-btn svg {
  width: 12px;
  height: 12px;
}

.volume-icon {
  color: rgba(255, 255, 255, 0.4);
  flex-shrink: 0;
}

.master-slider {
  width: 100px;
  height: 4px;
  -webkit-appearance: none;
  appearance: none;
  background: rgba(255, 255, 255, 0.15);
  border-radius: 2px;
  outline: none;
  cursor: pointer;
}

.master-slider::-webkit-slider-thumb {
  -webkit-appearance: none;
  appearance: none;
  width: 14px;
  height: 14px;
  border-radius: 50%;
  background: white;
  cursor: pointer;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.3);
}

.master-slider::-moz-range-thumb {
  width: 14px;
  height: 14px;
  border-radius: 50%;
  background: white;
  cursor: pointer;
  border: none;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.3);
}
</style>
