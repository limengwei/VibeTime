<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import FocusMode from './components/FocusMode.vue'
import ScenePresets from './components/ScenePresets.vue'
import { WindowMinimise } from '../wailsjs/runtime/runtime'
import { HideWindow, ListSounds } from '../wailsjs/go/main/App'
import {
  startSound,
  stopSound,
  setSoundVolume,
  setMasterVolume,
  stopAll,
  resumeAudioContext,
  registerSounds
} from './audio/audioEngine.js'

const categories = ref([])
const soundStates = reactive({})
const masterVolume = ref(0.8)
const focusMode = ref(false)
const scenePresetsRef = ref(null)

const activeCount = computed(() => {
  return Object.values(soundStates).filter(s => s.playing).length
})

function initSoundState(sound) {
  if (!soundStates[sound.id]) {
    soundStates[sound.id] = { playing: false, volume: 0.5 }
  }
}

function toggleFocusMode() {
  focusMode.value = !focusMode.value
}

function onExitFocusMode() {
  focusMode.value = false
  stopAllSounds()
}

function onMasterVolumeChange(e) {
  masterVolume.value = parseFloat(e.target.value)
  setMasterVolume(masterVolume.value)
}

function stopAllSounds(clearScene = true) {
  Object.keys(soundStates).forEach(id => {
    if (soundStates[id].playing) {
      soundStates[id].playing = false
      stopSound(id)
    }
  })
  if (clearScene) scenePresetsRef.value?.clearSelection()
}

async function applyPreset(soundsMap) {
  resumeAudioContext()
  stopAllSounds(false)
  for (const [id, vol] of Object.entries(soundsMap)) {
    if (!soundStates[id]) soundStates[id] = { playing: false, volume: 0.5 }
    soundStates[id].playing = true
    soundStates[id].volume = vol
    await startSound(id)
    setSoundVolume(id, vol)
  }
}

async function onPreviewPlay(id, vol) {
  resumeAudioContext()
  if (!soundStates[id]) soundStates[id] = { playing: false, volume: 0.5 }
  soundStates[id].playing = true
  soundStates[id].volume = vol
  await startSound(id, vol)
}

function onPreviewStop(id) {
  if (soundStates[id]) {
    soundStates[id].playing = false
    stopSound(id)
  }
}

function onUpdateVolume(id, vol) {
  if (soundStates[id]) {
    soundStates[id].volume = vol
  }
  setSoundVolume(id, vol)
}

async function loadSounds() {
  const cats = await ListSounds()
  categories.value = cats
  const allSnds = []
  cats.forEach(cat => {
    cat.sounds.forEach(s => {
      initSoundState(s)
      allSnds.push(s)
    })
  })
  registerSounds(allSnds)
}

onMounted(() => {
  loadSounds()
})
</script>

<template>
  <div class="app-container">
    <FocusMode v-if="focusMode" @exit="onExitFocusMode" />
    <div class="titlebar">
      <div class="titlebar-drag">
        <span class="titlebar-title">Vibe Time</span>
      </div>
      <div class="titlebar-controls">
        <button class="titlebar-btn" @click="WindowMinimise" title="最小化">
          <svg width="12" height="12" viewBox="0 0 12 12"><rect y="5" width="12" height="1.5" fill="currentColor"/></svg>
        </button>
        <button class="titlebar-btn titlebar-btn-close" @click="HideWindow" title="最小化到托盘">
          <svg width="12" height="12" viewBox="0 0 12 12"><path d="M1 1L11 11M11 1L1 11" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/></svg>
        </button>
      </div>
    </div>
    <header class="app-header">
      <div class="header-top">
        <h1 class="app-title">Vibe Time</h1>
        <p class="app-subtitle">轻松找到你的宁静</p>
      </div>
    </header>

    <ScenePresets
      ref="scenePresetsRef"
      :categories="categories"
      @apply="applyPreset"
      @play-sound="onPreviewPlay"
      @stop-sound="onPreviewStop"
      @stop-all="stopAllSounds"
      @update-volume="onUpdateVolume"
    />

    <footer class="app-footer">
      <div class="footer-left">
        <span class="active-count" v-if="activeCount > 0">
          {{ activeCount }} 个音效正在播放
        </span>
      </div>
      <div class="footer-center">
        <button class="stop-btn" @click="stopAllSounds" v-if="activeCount > 0">
          <svg viewBox="0 0 24 24" fill="currentColor" width="16" height="16">
            <rect x="6" y="6" width="12" height="12" rx="1"/>
          </svg>
          全部停止
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
        <button class="focus-btn" :class="{ 'focus-active': focusMode }" :disabled="activeCount === 0" @click="toggleFocusMode" :title="activeCount === 0 ? '请先播放音效' : focusMode ? '退出专注模式' : '专注模式'">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>
        </button>
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

.titlebar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 36px;
  flex-shrink: 0;
  user-select: none;
  --wails-draggable: drag;
}

.titlebar-drag {
  flex: 1;
  padding-left: 16px;
}

.titlebar-title {
  font-size: 12px;
  color: rgba(255, 255, 255, 0.35);
  letter-spacing: 0.5px;
}

.titlebar-controls {
  display: flex;
  --wails-draggable: no-drag;
}

.titlebar-btn {
  width: 46px;
  height: 36px;
  border: none;
  background: transparent;
  color: rgba(255, 255, 255, 0.5);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.15s ease;
}

.titlebar-btn:hover {
  background: rgba(255, 255, 255, 0.1);
  color: rgba(255, 255, 255, 0.8);
}

.titlebar-btn-close:hover {
  background: rgba(255, 60, 60, 0.8);
  color: white;
}

.app-header {
  padding: 24px 32px 0;
  flex-shrink: 0;
}

.header-top {
  text-align: center;
  margin-bottom: 16px;
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

.titlebar-btn.focus-active {
  color: rgba(139, 92, 246, 0.9);
  background: rgba(139, 92, 246, 0.15);
}

.titlebar-btn:disabled {
  opacity: 0.25;
  cursor: not-allowed;
}

.focus-btn {
  background: rgba(255, 255, 255, 0.08);
  border: none;
  border-radius: 8px;
  color: rgba(255, 255, 255, 0.7);
  cursor: pointer;
  padding: 4px 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s;
  margin-left: 8px;
}

.focus-btn:hover:not(:disabled) {
  background: rgba(255, 255, 255, 0.15);
  color: rgba(255, 255, 255, 0.9);
}

.focus-btn.focus-active {
  color: rgba(139, 92, 246, 0.9);
  background: rgba(139, 92, 246, 0.15);
}

.focus-btn:disabled {
  opacity: 0.25;
  cursor: not-allowed;
}
</style>
