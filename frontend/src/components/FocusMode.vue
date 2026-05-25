<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { ListWallpapers } from '../../wailsjs/go/main/App'

const emit = defineEmits(['exit'])

const wallpapers = ref([])

const durationOptions = [
  { label: '5 分钟', value: 5 },
  { label: '10 分钟', value: 10 },
  { label: '20 分钟', value: 20 },
  { label: '30 分钟', value: 30 },
  { label: '60 分钟', value: 60 },
  { label: '不限制', value: 0 }
]

const selectedDuration = ref(0)
const currentWallpaper = ref(0)
const showPanel = ref(true)
const timerRunning = ref(false)
const timerPaused = ref(false)
const remainingSeconds = ref(0)
let intervalId = null

const wallpaperUrl = computed(() => {
  if (wallpapers.value.length === 0) return ''
  return `/wallpapers/${wallpapers.value[currentWallpaper.value].file}`
})

const formattedTime = computed(() => {
  if (selectedDuration.value === 0) {
    const elapsed = remainingSeconds.value
    const m = Math.floor(elapsed / 60)
    const s = elapsed % 60
    return `${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`
  }
  const m = Math.floor(remainingSeconds.value / 60)
  const s = remainingSeconds.value % 60
  return `${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`
})

const progressPercent = computed(() => {
  if (selectedDuration.value === 0) return 0
  const total = selectedDuration.value * 60
  return ((total - remainingSeconds.value) / total) * 100
})

function pickRandomWallpaper() {
  if (wallpapers.value.length === 0) return
  let idx
  do {
    idx = Math.floor(Math.random() * wallpapers.value.length)
  } while (idx === currentWallpaper.value && wallpapers.value.length > 1)
  currentWallpaper.value = idx
}

function selectWallpaper(index) {
  currentWallpaper.value = index
}

function clearTimer() {
  if (intervalId) {
    clearInterval(intervalId)
    intervalId = null
  }
}

function selectDuration(value) {
  clearTimer()
  selectedDuration.value = value
  remainingSeconds.value = value === 0 ? 0 : value * 60
  timerRunning.value = true
  timerPaused.value = false
  showPanel.value = false
  if (value === 0) {
    intervalId = setInterval(() => {
      remainingSeconds.value++
    }, 1000)
  } else {
    intervalId = setInterval(() => {
      remainingSeconds.value--
      if (remainingSeconds.value <= 0) {
        remainingSeconds.value = 0
        stopTimer()
        showPanel.value = true
      }
    }, 1000)
  }
}

function pauseTimer() {
  clearTimer()
  timerPaused.value = true
}

function resumeTimer() {
  clearTimer()
  timerPaused.value = false
  if (selectedDuration.value === 0) {
    intervalId = setInterval(() => {
      remainingSeconds.value++
    }, 1000)
  } else {
    intervalId = setInterval(() => {
      remainingSeconds.value--
      if (remainingSeconds.value <= 0) {
        remainingSeconds.value = 0
        stopTimer()
        showPanel.value = true
      }
    }, 1000)
  }
}

function stopTimer() {
  clearTimer()
  timerRunning.value = false
  timerPaused.value = false
  remainingSeconds.value = selectedDuration.value === 0 ? 0 : selectedDuration.value * 60
}

function handleKeydown(e) {
  if (e.key === 'Escape') {
    if (showPanel.value) {
      emit('exit')
    } else {
      showPanel.value = true
    }
  }
  if (e.key === ' ' && timerRunning.value) {
    e.preventDefault()
    if (timerPaused.value) {
      resumeTimer()
    } else {
      pauseTimer()
    }
  }
}

onMounted(async () => {
  const wps = await ListWallpapers()
  wallpapers.value = wps
  pickRandomWallpaper()
  window.addEventListener('keydown', handleKeydown)
  selectDuration(0)
})

onUnmounted(() => {
  clearTimer()
  window.removeEventListener('keydown', handleKeydown)
})
</script>

<template>
  <div class="focus-overlay">
    <div class="wallpaper-layer" :style="{ backgroundImage: `url(${wallpaperUrl})` }"></div>
    <div class="wallpaper-overlay"></div>

    <div class="focus-content">
      <div class="timer-display" :class="{ 'timer-done': remainingSeconds === 0 && timerRunning && selectedDuration !== 0 }">
        <div class="timer-ring" v-if="selectedDuration !== 0">
          <svg viewBox="0 0 200 200">
            <circle class="ring-bg" cx="100" cy="100" r="90" />
            <circle class="ring-progress" cx="100" cy="100" r="90"
              :style="{ strokeDashoffset: 565.48 - (565.48 * progressPercent / 100) }" />
          </svg>
        </div>
        <div class="timer-text">{{ formattedTime }}</div>
      </div>

      <div class="running-controls" v-if="timerRunning && !showPanel">
        <button class="ctrl-btn" @click="timerPaused ? resumeTimer() : pauseTimer()">
          <svg v-if="!timerPaused" width="24" height="24" viewBox="0 0 24 24" fill="currentColor">
            <rect x="6" y="4" width="4" height="16" rx="1" />
            <rect x="14" y="4" width="4" height="16" rx="1" />
          </svg>
          <svg v-else width="24" height="24" viewBox="0 0 24 24" fill="currentColor">
            <polygon points="5,3 19,12 5,21" />
          </svg>
        </button>
        <button class="ctrl-btn ctrl-btn-stop" @click="emit('exit')">
          <svg width="24" height="24" viewBox="0 0 24 24" fill="currentColor">
            <rect x="6" y="6" width="12" height="12" rx="1" />
          </svg>
        </button>
        <button class="ctrl-btn" @click="showPanel = true">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"
            stroke-linecap="round">
            <circle cx="12" cy="12" r="3" />
            <path
              d="M12 1v2m0 18v2m-9-11h2m18 0h2m-3.3-7.7-1.4 1.4M6.7 17.3l-1.4 1.4m0-13.4 1.4 1.4m10.6 10.6 1.4 1.4" />
          </svg>
        </button>
      </div>

      <div class="panel-backdrop" v-if="showPanel" @click="showPanel = false"></div>

      <Transition name="panel">
        <div class="settings-panel" v-if="showPanel" @click.stop>
          <div class="panel-header">
            <h2>专注模式</h2>
            <div class="panel-header-actions">
              <button class="close-btn" @click="showPanel = false">
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"
                  stroke-linecap="round">
                  <path d="M18 6L6 18M6 6l12 12" />
                </svg>
              </button>
            </div>
          </div>

          <div class="panel-section">
            <div class="section-label">倒计时</div>
            <div class="duration-options">
              <button v-for="opt in durationOptions" :key="opt.value" class="duration-btn"
                :class="{ active: selectedDuration === opt.value }" @click="selectDuration(opt.value)">
                {{ opt.label }}
              </button>
            </div>
          </div>

          <div class="panel-section">
            <div class="section-label">背景</div>
            <div class="wallpaper-grid">
              <div v-for="(wp, index) in wallpapers" :key="wp.file" class="wallpaper-thumb"
                :class="{ active: currentWallpaper === index }"
                :style="{ backgroundImage: `url(/wallpapers/${wp.file})` }" @click="selectWallpaper(index)">
              </div>
            </div>
          </div>
        </div>
      </Transition>
    </div>
  </div>
</template>

<style scoped>
.focus-overlay {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  z-index: 100;
  animation: fadeIn 0.5s ease;
}

@keyframes fadeIn {
  from {
    opacity: 0;
  }

  to {
    opacity: 1;
  }
}

.wallpaper-layer {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background-size: cover;
  background-position: center;
  transition: background-image 0.8s ease;
}

.wallpaper-overlay {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background: radial-gradient(ellipse at center, rgba(0, 0, 0, 0.2) 0%, rgba(0, 0, 0, 0.6) 100%);
}

.focus-content {
  position: relative;
  z-index: 1;
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
}

.timer-display {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 32px;
}

.timer-ring {
  position: absolute;
  width: 260px;
  height: 260px;
}

.timer-ring svg {
  width: 100%;
  height: 100%;
  transform: rotate(-90deg);
}

.ring-bg {
  fill: none;
  stroke: rgba(255, 255, 255, 0.1);
  stroke-width: 3;
}

.ring-progress {
  fill: none;
  stroke: rgba(255, 255, 255, 0.7);
  stroke-width: 3;
  stroke-linecap: round;
  stroke-dasharray: 565.48;
  stroke-dashoffset: 565.48;
  transition: stroke-dashoffset 1s linear;
}

.timer-text {
  font-size: 72px;
  font-weight: 200;
  color: rgba(255, 255, 255, 0.95);
  letter-spacing: 4px;
  font-variant-numeric: tabular-nums;
  text-shadow: 0 2px 20px rgba(0, 0, 0, 0.3);
}

.timer-done .timer-text {
  animation: pulse 2s ease-in-out infinite;
}

@keyframes pulse {

  0%,
  100% {
    opacity: 1;
  }

  50% {
    opacity: 0.4;
  }
}

.running-controls {
  display: flex;
  gap: 16px;
}

.ctrl-btn {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  border: 1px solid rgba(255, 255, 255, 0.2);
  background: rgba(255, 255, 255, 0.1);
  backdrop-filter: blur(10px);
  color: rgba(255, 255, 255, 0.8);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s ease;
}

.ctrl-btn:hover {
  background: rgba(255, 255, 255, 0.2);
  border-color: rgba(255, 255, 255, 0.3);
}

.ctrl-btn-stop:hover {
  background: rgba(255, 80, 80, 0.3);
  border-color: rgba(255, 80, 80, 0.4);
}

.panel-backdrop {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  z-index: 0;
}

.settings-panel {
  position: absolute;
  bottom: 60px;
  background: rgba(15, 20, 30, 0.85);
  backdrop-filter: blur(30px);
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 20px;
  padding: 28px 32px;
  width: 460px;
  max-height: 70vh;
  overflow-y: auto;
}

.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 24px;
}

.panel-header h2 {
  font-size: 18px;
  font-weight: 600;
  color: rgba(255, 255, 255, 0.9);
  margin: 0;
}

.close-btn {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  border: none;
  background: rgba(255, 255, 255, 0.06);
  color: rgba(255, 255, 255, 0.5);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s ease;
}

.close-btn:hover {
  background: rgba(255, 255, 255, 0.15);
  color: rgba(255, 255, 255, 0.9);
}

.panel-header-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.panel-section {
  margin-bottom: 20px;
}

.section-label {
  font-size: 12px;
  color: rgba(255, 255, 255, 0.4);
  margin-bottom: 10px;
  letter-spacing: 0.5px;
  text-transform: uppercase;
}

.duration-options {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.duration-btn {
  background: rgba(255, 255, 255, 0.06);
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 12px;
  color: rgba(255, 255, 255, 0.6);
  padding: 8px 16px;
  font-size: 13px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.duration-btn:hover {
  background: rgba(255, 255, 255, 0.12);
  color: rgba(255, 255, 255, 0.9);
}

.duration-btn.active {
  background: rgba(255, 255, 255, 0.15);
  border-color: rgba(255, 255, 255, 0.25);
  color: white;
}

.wallpaper-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(80px, 1fr));
  gap: 8px;
}

.wallpaper-thumb {
  aspect-ratio: 16 / 10;
  border-radius: 8px;
  background-size: cover;
  background-position: center;
  cursor: pointer;
  border: 2px solid transparent;
  transition: all 0.2s ease;
  display: flex;
  align-items: flex-end;
  justify-content: center;
  overflow: hidden;
}

.wallpaper-thumb:hover {
  border-color: rgba(255, 255, 255, 0.3);
  transform: scale(1.05);
}

.wallpaper-thumb.active {
  border-color: rgba(255, 255, 255, 0.6);
  box-shadow: 0 0 12px rgba(255, 255, 255, 0.15);
}

.wp-name {
  font-size: 10px;
  color: white;
  background: rgba(0, 0, 0, 0.5);
  padding: 2px 6px;
  border-radius: 4px;
  margin-bottom: 4px;
}

.panel-enter-active,
.panel-leave-active {
  transition: all 0.3s ease;
}

.panel-enter-from,
.panel-leave-to {
  opacity: 0;
  transform: translateY(20px);
}
</style>
