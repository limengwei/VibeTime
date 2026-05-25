<script setup>
import { ref } from 'vue'
import { getIconSvg } from '../audio/soundIcons.js'

const props = defineProps({
  sound: {
    type: Object,
    required: true
  }
})

const emit = defineEmits(['toggle', 'volume-change'])

const isHovered = ref(false)

function onToggle() {
  emit('toggle', props.sound.id)
}

function onVolumeChange(e) {
  const v = parseFloat(e.target.value)
  emit('volume-change', props.sound.id, v)
}
</script>

<template>
  <div
    class="sound-card"
    :class="{ active: sound.playing, hovered: isHovered }"
    @mouseenter="isHovered = true"
    @mouseleave="isHovered = false"
    @click="onToggle"
  >
    <div class="sound-icon" v-html="getIconSvg(sound.iconKey)"></div>
    <div class="sound-name">{{ sound.name }}</div>
    <div class="sound-desc">{{ sound.description }}</div>
    <div class="volume-control" @click.stop>
      <input
        type="range"
        min="0"
        max="1"
        step="0.01"
        :value="sound.volume"
        @input="onVolumeChange"
        class="volume-slider"
        :class="{ visible: sound.playing || isHovered }"
      />
    </div>
    <div class="volume-label" v-if="sound.playing">
      {{ Math.round(sound.volume * 100) }}%
    </div>
  </div>
</template>

<style scoped>
.sound-card {
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid rgba(255, 255, 255, 0.06);
  border-radius: 16px;
  padding: 20px 16px 16px;
  cursor: pointer;
  transition: all 0.3s ease;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  user-select: none;
  min-height: 160px;
}

.sound-card:hover {
  background: rgba(255, 255, 255, 0.08);
  border-color: rgba(255, 255, 255, 0.12);
  transform: translateY(-2px);
}

.sound-card.active {
  background: rgba(255, 255, 255, 0.1);
  border-color: rgba(255, 255, 255, 0.2);
  box-shadow: 0 4px 20px rgba(255, 255, 255, 0.05);
}

.sound-icon {
  width: 36px;
  height: 36px;
  color: rgba(255, 255, 255, 0.5);
  transition: color 0.3s ease;
}

.sound-card.active .sound-icon {
  color: rgba(255, 255, 255, 0.9);
}

.sound-card:hover .sound-icon {
  color: rgba(255, 255, 255, 0.7);
}

.sound-icon :deep(svg) {
  width: 100%;
  height: 100%;
}

.sound-name {
  font-size: 14px;
  font-weight: 600;
  color: rgba(255, 255, 255, 0.8);
  letter-spacing: 0.3px;
}

.sound-card.active .sound-name {
  color: rgba(255, 255, 255, 1);
}

.sound-desc {
  font-size: 11px;
  color: rgba(255, 255, 255, 0.35);
  line-height: 1.3;
}

.volume-control {
  width: 100%;
  margin-top: 4px;
}

.volume-slider {
  width: 100%;
  height: 4px;
  -webkit-appearance: none;
  appearance: none;
  background: rgba(255, 255, 255, 0.15);
  border-radius: 2px;
  outline: none;
  opacity: 0;
  transition: opacity 0.3s ease;
  cursor: pointer;
}

.volume-slider.visible {
  opacity: 1;
}

.volume-slider::-webkit-slider-thumb {
  -webkit-appearance: none;
  appearance: none;
  width: 14px;
  height: 14px;
  border-radius: 50%;
  background: white;
  cursor: pointer;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.3);
}

.volume-slider::-moz-range-thumb {
  width: 14px;
  height: 14px;
  border-radius: 50%;
  background: white;
  cursor: pointer;
  border: none;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.3);
}

.volume-label {
  font-size: 11px;
  color: rgba(255, 255, 255, 0.4);
  font-variant-numeric: tabular-nums;
}
</style>
