<script setup>
import { ref, computed, reactive, nextTick } from 'vue'
import { getIconSvg } from '../audio/soundIcons.js'
import { encodeRecipe, decodeRecipe } from '../utils/recipeCode.js'
import { generateRecipeCard } from '../utils/recipeCard.js'

const props = defineProps({
  categories: { type: Array, required: true }
})

const emit = defineEmits(['apply', 'play-sound', 'stop-sound', 'stop-all', 'update-volume'])

const builtinPresets = [
  {
    id: '__rainy_night',
    name: '雨夜',
    icon: '🌧',
    desc: '雨声 + 雷声 + 风',
    builtin: true,
    sounds: {
      'rain/heavy-rain': 0.7,
      'rain/thunder': 0.4,
      'nature/wind': 0.3
    }
  },
  {
    id: '__deep_focus',
    name: '深度专注',
    icon: '🧠',
    desc: '棕色噪音 + 咖啡厅 + 键盘',
    builtin: true,
    sounds: {
      'noise/brown-noise': 0.5,
      'places/cafe': 0.4,
      'things/keyboard': 0.2
    }
  },
  {
    id: '__forest_morning',
    name: '森林清晨',
    icon: '🌲',
    desc: '鸟鸣 + 溪流 + 林间风',
    builtin: true,
    sounds: {
      'animals/birds': 0.6,
      'nature/river': 0.4,
      'nature/wind-in-trees': 0.3
    }
  },
  {
    id: '__ocean_breeze',
    name: '海边微风',
    icon: '🌊',
    desc: '海浪 + 海鸥 + 风',
    builtin: true,
    sounds: {
      'nature/waves': 0.6,
      'animals/seagulls': 0.3,
      'nature/wind': 0.35
    }
  },
  {
    id: '__campfire',
    name: '篝火之夜',
    icon: '🔥',
    desc: '篝火 + 蟋蟀 + 猫头鹰',
    builtin: true,
    sounds: {
      'nature/campfire': 0.6,
      'animals/crickets': 0.4,
      'animals/owl': 0.25
    }
  },
  {
    id: '__city_night',
    name: '城市深夜',
    icon: '🌃',
    desc: '交通 + 远处人群 + 小雨',
    builtin: true,
    sounds: {
      'urban/traffic': 0.35,
      'urban/crowd': 0.2,
      'rain/light-rain': 0.5
    }
  },
  {
    id: '__meditation',
    name: '冥想',
    icon: '🧘',
    desc: '颂钵 + Alpha 波 + 风铃',
    builtin: true,
    sounds: {
      'things/singing-bowl': 0.5,
      'binaural/binaural-alpha': 0.4,
      'things/wind-chimes': 0.2
    }
  },
  {
    id: '__study_room',
    name: '自习室',
    icon: '📚',
    desc: '图书馆 + 时钟 + 翻书',
    builtin: true,
    sounds: {
      'places/library': 0.5,
      'things/clock': 0.2,
      'things/paper': 0.25
    }
  },
  {
    id: '__cozy_home',
    name: '温馨小屋',
    icon: '🏠',
    desc: '吊扇 + 猫咪 + 洗衣机',
    builtin: true,
    sounds: {
      'things/ceiling-fan': 0.4,
      'animals/cat-purring': 0.5,
      'things/washing-machine': 0.2
    }
  },
  {
    id: '__train_journey',
    name: '火车旅行',
    icon: '🚂',
    desc: '火车内 + 窗上雨声 + 键盘',
    builtin: true,
    sounds: {
      'transport/inside-a-train': 0.5,
      'rain/rain-on-window': 0.4,
      'things/keyboard': 0.15
    }
  },
  {
    id: '__snow_day',
    name: '下雪天',
    icon: '❄',
    desc: '踏雪 + 风声 + 篝火',
    builtin: true,
    sounds: {
      'nature/walk-in-snow': 0.4,
      'nature/wind': 0.35,
      'nature/campfire': 0.5
    }
  },
  {
    id: '__sleep',
    name: '助眠',
    icon: '🌙',
    desc: '粉色噪音 + 小雨 + 蛙鸣',
    builtin: true,
    sounds: {
      'noise/pink-noise': 0.5,
      'rain/light-rain': 0.45,
      'animals/frog': 0.2
    }
  }
]

const randomIcons = ['🎵', '🎶', '✨', '🌟', '💫', '🎪', '🎭', '🎨', '🎬', '🎮', '🎲', '🎯', '🎸', '🎹', '🎺', '🎻', '🥁', '🏕', '⛰', '🌅', '🌄', '🎆', '🎇', '🏙', '🏯', '⛵', '🚀', '🎡', '🎢', '🎠']

function getRandomIcon() {
  return randomIcons[Math.floor(Math.random() * randomIcons.length)]
}

function loadCustomPresets() {
  try {
    const raw = localStorage.getItem('vibe_custom_presets')
    if (!raw) return []
    return JSON.parse(raw)
  } catch {
    return []
  }
}

function loadBuiltinOverrides() {
  try {
    const raw = localStorage.getItem('vibe_builtin_overrides')
    if (!raw) return {}
    return JSON.parse(raw)
  } catch {
    return {}
  }
}

function saveCustomPresetsToStorage(presets) {
  localStorage.setItem('vibe_custom_presets', JSON.stringify(presets))
}

function saveBuiltinOverrides(overrides) {
  localStorage.setItem('vibe_builtin_overrides', JSON.stringify(overrides))
}

const builtinOverrides = reactive(loadBuiltinOverrides())

const customPresets = ref(loadCustomPresets())

const allPresets = computed(() => {
  return [...builtinPresets, ...customPresets.value]
})

const activePresetId = ref(null)
const activeVolumes = reactive({})

const activePreset = computed(() => {
  if (!activePresetId.value) return null
  return allPresets.value.find(p => p.id === activePresetId.value) || null
})

const activePresetSounds = computed(() => {
  if (!activePresetId.value) return []
  return Object.entries(activeVolumes).map(([id, vol]) => ({
    id,
    label: getSoundLabel(id),
    volume: vol
  }))
})

const showModal = ref(false)
const editingPreset = ref(null)
const formName = ref('')
const formIcon = ref('')
const formDesc = ref('')
const formSelectedSounds = reactive({})

const showImportModal = ref(false)
const importCode = ref('')
const toastVisible = ref(false)
const toastMessage = ref('')
let toastTimer = null

const showCardModal = ref(false)
const cardImageUrl = ref('')
const cardPresetName = ref('')
const shareRecipeCode = ref('')

const formSelectedCount = computed(() => Object.keys(formSelectedSounds).length)

function autoDesc() {
  const labels = Object.keys(formSelectedSounds).map(id => getSoundLabel(id))
  return labels.join(' + ')
}

function getSoundLabel(soundId) {
  for (const cat of props.categories) {
    for (const s of cat.sounds) {
      if (s.id === soundId) return s.label
    }
  }
  const parts = soundId.split('/')
  return parts[parts.length - 1]
}

function isSoundSelected(soundId) {
  return soundId in formSelectedSounds
}

function toggleFormSound(soundId) {
  if (soundId in formSelectedSounds) {
    delete formSelectedSounds[soundId]
    emit('stop-sound', soundId)
  } else {
    formSelectedSounds[soundId] = 0.5
    emit('play-sound', soundId, 0.5)
  }
  if (!formDesc.value || formDesc.value === autoDesc()) {
    formDesc.value = autoDesc()
  }
}

function setFormSoundVolume(soundId, vol) {
  if (soundId in formSelectedSounds) {
    formSelectedSounds[soundId] = vol
    emit('play-sound', soundId, vol)
  }
}

function openAddModal() {
  editingPreset.value = null
  formName.value = ''
  formIcon.value = getRandomIcon()
  formDesc.value = ''
  Object.keys(formSelectedSounds).forEach(k => delete formSelectedSounds[k])
  emit('stop-all')
  showModal.value = true
}

function openEditModal(preset) {
  editingPreset.value = preset
  formName.value = preset.name
  formIcon.value = preset.icon
  formDesc.value = preset.desc || ''
  Object.keys(formSelectedSounds).forEach(k => delete formSelectedSounds[k])
  emit('stop-all')
  if (preset.sounds) {
    for (const [id, vol] of Object.entries(preset.sounds)) {
      formSelectedSounds[id] = vol
      emit('play-sound', id, vol)
    }
  }
  showModal.value = true
}

function randomizeIcon() {
  formIcon.value = getRandomIcon()
}

function closeModal() {
  Object.keys(formSelectedSounds).forEach(id => {
    emit('stop-sound', id)
  })
  showModal.value = false
}

function savePreset() {
  const name = formName.value.trim()
  if (!name) return
  if (formSelectedCount.value === 0) return

  const sounds = {}
  for (const [id, vol] of Object.entries(formSelectedSounds)) {
    sounds[id] = vol
  }

  if (editingPreset.value && !editingPreset.value.builtin) {
    const idx = customPresets.value.findIndex(p => p.id === editingPreset.value.id)
    if (idx >= 0) {
      customPresets.value[idx] = {
        ...customPresets.value[idx],
        name,
        icon: formIcon.value,
        desc: formDesc.value || autoDesc(),
        sounds
      }
    }
  } else {
    customPresets.value.push({
      id: 'custom_' + Date.now(),
      name,
      icon: formIcon.value,
      desc: formDesc.value || autoDesc(),
      sounds,
      custom: true
    })
  }

  saveCustomPresetsToStorage(customPresets.value)
  const soundsCopy = { ...sounds }
  Object.keys(formSelectedSounds).forEach(id => {
    emit('stop-sound', id)
  })
  showModal.value = false
  emit('apply', soundsCopy)
}

function deleteCustom(preset) {
  customPresets.value = customPresets.value.filter(p => p.id !== preset.id)
  saveCustomPresetsToStorage(customPresets.value)
}

function applyPreset(preset) {
  if (activePresetId.value === preset.id) return
  activePresetId.value = preset.id
  Object.keys(activeVolumes).forEach(k => delete activeVolumes[k])
  const override = builtinOverrides[preset.id] || {}
  for (const [id, vol] of Object.entries(preset.sounds)) {
    activeVolumes[id] = override[id] !== undefined ? override[id] : vol
  }
  emit('apply', activeVolumes)
}

function clearSelection() {
  activePresetId.value = null
  Object.keys(activeVolumes).forEach(k => delete activeVolumes[k])
}

defineExpose({ clearSelection })

function onActiveVolumeChange(soundId, volume) {
  if (!(soundId in activeVolumes)) return
  activeVolumes[soundId] = volume
  const preset = activePreset.value
  if (!preset) return
  if (preset.builtin) {
    if (!builtinOverrides[preset.id]) builtinOverrides[preset.id] = {}
    builtinOverrides[preset.id][soundId] = volume
    saveBuiltinOverrides(builtinOverrides)
  } else {
    preset.sounds[soundId] = volume
    const idx = customPresets.value.findIndex(p => p.id === preset.id)
    if (idx >= 0) {
      customPresets.value[idx] = { ...customPresets.value[idx], sounds: { ...preset.sounds } }
      saveCustomPresetsToStorage(customPresets.value)
    }
  }
  emit('update-volume', soundId, volume)
}

function onOverlayClick(e) {
  if (e.target === e.currentTarget) closeModal()
}

function copyRecipeCode(preset) {
  let sounds = preset.sounds
  if (activePresetId.value === preset.id) {
    sounds = { ...activeVolumes }
  }
  const code = encodeRecipe({ name: preset.name, icon: preset.icon, desc: preset.desc, sounds })
  if (!code) {
    showToast('❌ 无法编码空配方')
    return
  }
  try {
    navigator.clipboard.writeText(code).then(() => {
      showToast(`✅ 已复制「${preset.name}」配方码`)
    }).catch(() => {
      showToast('📋 配方码已生成，请手动复制')
    })
  } catch {
    showToast('📋 配方码已生成')
  }
}

function openImportModal() {
  importCode.value = ''
  showImportModal.value = true
}

function closeImportModal() {
  showImportModal.value = false
  importCode.value = ''
}

function importPreset() {
  const code = importCode.value.trim()
  if (!code) return

  const decoded = decodeRecipe(code)
  if (!decoded) {
    showToast('❌ 配方码无效或已损坏')
    return
  }

  let name = decoded.name
  if (allPresets.value.some(p => p.name === name)) {
    name = decoded.name + ' 📥'
  }

  const newPreset = {
    id: 'custom_' + Date.now(),
    name,
    icon: decoded.icon || '🎵',
    desc: decoded.desc || '',
    sounds: decoded.sounds,
    custom: true
  }

  customPresets.value.push(newPreset)
  saveCustomPresetsToStorage(customPresets.value)
  closeImportModal()
  showToast(`✅ 已导入配方「${decoded.name}」`)

  const soundsCopy = { ...decoded.sounds }
  emit('apply', soundsCopy)
}

function showToast(msg) {
  toastMessage.value = msg
  toastVisible.value = true
  if (toastTimer) clearTimeout(toastTimer)
  toastTimer = setTimeout(() => {
    toastVisible.value = false
    toastTimer = null
  }, 2500)
}

async function openShareModal(preset) {
  let sounds = preset.sounds
  if (activePresetId.value === preset.id) {
    sounds = { ...activeVolumes }
  }
  const code = encodeRecipe({ name: preset.name, icon: preset.icon, desc: preset.desc, sounds })
  if (!code) {
    showToast('❌ 无法分享空配方')
    return
  }
  shareRecipeCode.value = code
  cardPresetName.value = preset.name
  try {
    const blob = await generateRecipeCard(
      { name: preset.name, icon: preset.icon, desc: preset.desc, sounds },
      code
    )
    cardImageUrl.value = URL.createObjectURL(blob)
  } catch {
    // 卡片生成失败不影响声音码分享
    cardImageUrl.value = ''
  }
  showCardModal.value = true
}

function closeCardModal() {
  showCardModal.value = false
  shareRecipeCode.value = ''
  if (cardImageUrl.value) {
    URL.revokeObjectURL(cardImageUrl.value)
    cardImageUrl.value = ''
  }
}

function downloadCard() {
  if (!cardImageUrl.value) return
  const a = document.createElement('a')
  a.href = cardImageUrl.value
  a.download = `VibeTime-${cardPresetName.value || 'recipe'}.png`
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
}

function copyShareCode() {
  const code = shareRecipeCode.value
  if (!code) return
  try {
    navigator.clipboard.writeText(code).then(() => {
      showToast(`✅ 已复制「${cardPresetName.value}」声音码`)
    }).catch(() => {
      showToast('📋 声音码已生成，请手动复制')
    })
  } catch {
    showToast('📋 声音码已生成')
  }
}
</script>

<template>
  <div class="scene-main">
    <div class="scene-grid">
      <div
        v-for="preset in allPresets"
        :key="preset.id"
        class="scene-card"
        :class="{ 'scene-card-active': activePresetId === preset.id }"
        @click="applyPreset(preset)"
      >
        <button
          v-if="preset.custom"
          class="card-action card-delete"
          @click.stop="deleteCustom(preset)"
          title="删除"
        >
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M18 6L6 18M6 6l12 12"/></svg>
        </button>
        <button
          v-if="preset.custom"
          class="card-action card-edit"
          @click.stop="openEditModal(preset)"
          title="编辑"
        >
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/></svg>
        </button>
        <button
          class="card-action card-share"
          @click.stop="openShareModal(preset)"
          title="分享"
        >
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="18" cy="5" r="3"/><circle cx="6" cy="12" r="3"/><circle cx="18" cy="19" r="3"/><line x1="8.59" y1="13.51" x2="15.42" y2="17.49"/><line x1="15.41" y1="6.51" x2="8.59" y2="10.49"/></svg>
        </button>
        <div class="card-icon">{{ preset.icon }}</div>
        <div class="card-name">{{ preset.name }}</div>
        <div class="card-desc">{{ preset.desc }}</div>
      </div>

      <div class="scene-card scene-card-add" @click="openAddModal">
        <div class="add-icon">
          <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round">
            <line x1="12" y1="5" x2="12" y2="19"/>
            <line x1="5" y1="12" x2="19" y2="12"/>
          </svg>
        </div>
        <div class="card-name">添加场景</div>
        <div class="card-desc">自定义声音组合</div>
      </div>

      <div class="scene-card scene-card-import" @click="openImportModal">
        <div class="add-icon">
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>
        </div>
        <div class="card-name">导入配方</div>
        <div class="card-desc">粘贴配方码</div>
      </div>
    </div>

    <div v-if="activePreset" class="active-sounds-panel">
      <div class="active-panel-header">
        <span class="active-panel-icon">{{ activePreset.icon }}</span>
        <span class="active-panel-name">{{ activePreset.name }}</span>
      </div>
      <div class="active-sound-grid">
        <div class="active-sound-card" v-for="s in activePresetSounds" :key="s.id">
          <div class="active-sound-icon" v-html="getIconSvg(s.id)"></div>
          <div class="active-sound-label">{{ s.label }}</div>
          <div class="active-sound-slider-wrap">
            <input
              type="range"
              class="active-sound-slider"
              min="0.05"
              max="1"
              step="0.05"
              :value="s.volume"
              @input="onActiveVolumeChange(s.id, parseFloat($event.target.value))"
            />
          </div>
          <span class="active-sound-vol">{{ Math.round(s.volume * 100) }}%</span>
        </div>
      </div>
    </div>

    <Teleport to="body">
      <div v-if="showModal" class="modal-overlay" @click="onOverlayClick">
        <div class="modal-container">
          <div class="modal-header">
            <h3>{{ editingPreset ? '编辑场景' : '添加场景' }}</h3>
            <button class="modal-close" @click="closeModal">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M18 6L6 18M6 6l12 12"/></svg>
            </button>
          </div>

          <div class="modal-body">
            <div class="form-row">
              <div class="form-icon-field">
                <label class="form-label">图标</label>
                <div class="icon-picker">
                  <span class="icon-preview">{{ formIcon }}</span>
                  <button class="icon-random-btn" @click="randomizeIcon" title="随机图标">🎲</button>
                </div>
              </div>
              <div class="form-name-field">
                <label class="form-label">名称</label>
                <input class="form-input" v-model="formName" placeholder="场景名称" maxlength="20" />
              </div>
            </div>

            <div class="form-row">
              <div class="form-full">
                <label class="form-label">描述</label>
                <input class="form-input" v-model="formDesc" placeholder="简短描述这个场景..." maxlength="40" />
              </div>
            </div>

            <div class="form-sounds-section">
              <div class="form-label">
                选择声音
                <span class="selected-count" v-if="formSelectedCount > 0">({{ formSelectedCount }})</span>
              </div>

              <div class="selected-sounds" v-if="formSelectedCount > 0">
                <div class="selected-tag" v-for="(vol, id) in formSelectedSounds" :key="id">
                  <span class="tag-name">{{ getSoundLabel(id) }}</span>
                  <input
                    type="range"
                    class="tag-volume"
                    min="0.05"
                    max="1"
                    step="0.05"
                    :value="vol"
                    @input="setFormSoundVolume(id, parseFloat($event.target.value))"
                  />
                  <button class="tag-remove" @click="toggleFormSound(id)">
                    <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round"><path d="M18 6L6 18M6 6l12 12"/></svg>
                  </button>
                </div>
              </div>

              <div class="sound-groups">
                <div class="sound-group" v-for="cat in categories" :key="cat.id">
                  <div class="group-title">{{ cat.title }}</div>
                  <div class="group-sounds">
                    <button
                      v-for="s in cat.sounds"
                      :key="s.id"
                      class="sound-chip"
                      :class="{ active: isSoundSelected(s.id) }"
                      @click="toggleFormSound(s.id)"
                    >
                      {{ s.label }}
                    </button>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <div class="modal-footer">
            <button class="btn-cancel" @click="closeModal">取消</button>
            <button class="btn-save" @click="savePreset" :disabled="!formName.trim() || formSelectedCount === 0">
              {{ editingPreset ? '保存修改' : '添加场景' }}
            </button>
          </div>
        </div>
      </div>
    </Teleport>

    <Teleport to="body">
      <div v-if="showImportModal" class="modal-overlay" @click.self="closeImportModal">
        <div class="modal-container import-modal">
          <div class="modal-header">
            <h3>导入配方码</h3>
            <button class="modal-close" @click="closeImportModal">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M18 6L6 18M6 6l12 12"/></svg>
            </button>
          </div>
          <div class="modal-body">
            <p class="import-hint">粘贴好友分享的 VibeTime 配方码</p>
            <textarea
              v-model="importCode"
              class="import-textarea"
              placeholder="VIBE:..."
              rows="2"
              @keydown.ctrl.enter="importPreset"
              @keydown.meta.enter="importPreset"
            ></textarea>
          </div>
          <div class="modal-footer">
            <button class="btn-cancel" @click="closeImportModal">取消</button>
            <button class="btn-save" @click="importPreset" :disabled="!importCode.trim()">导入</button>
          </div>
        </div>
      </div>
    </Teleport>

    <Teleport to="body">
      <Transition name="toast">
        <div v-if="toastVisible" class="toast">{{ toastMessage }}</div>
      </Transition>
    </Teleport>

    <Teleport to="body">
      <div v-if="showCardModal" class="modal-overlay" @click.self="closeCardModal">
        <div class="card-modal-container">
          <div class="card-modal-header">
            <h3>🎴 {{ cardPresetName }}</h3>
            <button class="modal-close" @click="closeCardModal">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M18 6L6 18M6 6l12 12"/></svg>
            </button>
          </div>
          <div class="card-modal-body">
            <img v-if="cardImageUrl" :src="cardImageUrl" class="card-preview" alt="声音卡片预览" />
          </div>
          <div class="card-modal-footer">
            <button class="btn-cancel" @click="closeCardModal">关闭</button>
            <button class="btn-share-code" @click="copyShareCode">📋 复制声音码</button>
            <button class="btn-save" @click="downloadCard" :disabled="!cardImageUrl">💾 保存图片</button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<style scoped>
.scene-main {
  flex: 1;
  overflow-y: auto;
  padding: 12px 32px 20px;
  align-content: start;
}

.scene-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
  gap: 12px;
}

.scene-card {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 20px 12px 14px;
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid rgba(255, 255, 255, 0.06);
  border-radius: 16px;
  cursor: pointer;
  transition: all 0.2s ease;
  text-align: center;
}

.scene-card:hover {
  background: rgba(255, 255, 255, 0.08);
  border-color: rgba(255, 255, 255, 0.12);
  transform: translateY(-2px);
}

.scene-card:active {
  transform: translateY(0);
}

.scene-card-active {
  background: rgba(139, 92, 246, 0.12);
  border-color: rgba(139, 92, 246, 0.4);
  box-shadow: 0 0 20px rgba(139, 92, 246, 0.1);
}

.scene-card-active:hover {
  background: rgba(139, 92, 246, 0.18);
  border-color: rgba(139, 92, 246, 0.5);
}

.card-action {
  position: absolute;
  top: 6px;
  background: none;
  border: none;
  color: rgba(255, 255, 255, 0.2);
  cursor: pointer;
  padding: 4px;
  border-radius: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.15s ease;
  opacity: 0;
}

.scene-card:hover .card-action {
  opacity: 1;
}

.card-edit {
  right: 24px;
}

.card-delete {
  right: 6px;
}

.card-delete:hover {
  color: rgba(255, 100, 100, 0.8);
  background: rgba(255, 80, 80, 0.15);
}

.card-edit:hover {
  color: rgba(255, 255, 255, 0.7);
  background: rgba(255, 255, 255, 0.1);
}

.card-icon {
  font-size: 36px;
  margin-bottom: 8px;
  line-height: 1;
}

.card-name {
  font-size: 13px;
  font-weight: 600;
  color: rgba(255, 255, 255, 0.85);
  margin-bottom: 4px;
}

.card-desc {
  font-size: 11px;
  color: rgba(255, 255, 255, 0.3);
  line-height: 1.4;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.scene-card-add {
  border-style: dashed;
  border-color: rgba(255, 255, 255, 0.1);
  background: transparent;
}

.scene-card-add:hover {
  border-color: rgba(139, 92, 246, 0.4);
  background: rgba(139, 92, 246, 0.06);
}

.add-icon {
  color: rgba(255, 255, 255, 0.25);
  margin-bottom: 8px;
  transition: color 0.2s ease;
}

.scene-card-add:hover .add-icon {
  color: rgba(167, 139, 250, 0.6);
}

.active-sounds-panel {
  margin-top: 16px;
  background: rgba(139, 92, 246, 0.06);
  border: 1px solid rgba(139, 92, 246, 0.15);
  border-radius: 16px;
  padding: 14px 18px;
}

.active-panel-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
}

.active-panel-icon {
  font-size: 18px;
}

.active-panel-name {
  font-size: 14px;
  font-weight: 600;
  color: rgba(255, 255, 255, 0.8);
}

.active-sound-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(120px, 1fr));
  gap: 10px;
}

.active-sound-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  padding: 12px 8px 10px;
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid rgba(255, 255, 255, 0.06);
  border-radius: 12px;
  transition: all 0.15s ease;
}

.active-sound-card:hover {
  background: rgba(255, 255, 255, 0.08);
  border-color: rgba(255, 255, 255, 0.12);
}

.active-sound-icon {
  width: 28px;
  height: 28px;
  color: rgba(167, 139, 250, 0.8);
}

.active-sound-icon :deep(svg) {
  width: 100%;
  height: 100%;
}

.active-sound-label {
  font-size: 12px;
  font-weight: 500;
  color: rgba(255, 255, 255, 0.7);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 100%;
  text-align: center;
}

.active-sound-slider-wrap {
  width: 100%;
}

.active-sound-slider {
  width: 100%;
  height: 3px;
  -webkit-appearance: none;
  appearance: none;
  background: rgba(255, 255, 255, 0.12);
  border-radius: 2px;
  outline: none;
  cursor: pointer;
}

.active-sound-slider::-webkit-slider-thumb {
  -webkit-appearance: none;
  appearance: none;
  width: 12px;
  height: 12px;
  border-radius: 50%;
  background: rgba(167, 139, 250, 0.9);
  cursor: pointer;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.3);
}

.active-sound-slider::-moz-range-thumb {
  width: 12px;
  height: 12px;
  border-radius: 50%;
  background: rgba(167, 139, 250, 0.9);
  cursor: pointer;
  border: none;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.3);
}

.active-sound-vol {
  font-size: 10px;
  color: rgba(255, 255, 255, 0.3);
  font-variant-numeric: tabular-nums;
}

.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background: rgba(0, 0, 0, 0.65);
  backdrop-filter: blur(6px);
  z-index: 200;
  display: flex;
  align-items: center;
  justify-content: center;
  animation: fadeIn 0.15s ease;
}

@keyframes fadeIn {
  from { opacity: 0; }
  to { opacity: 1; }
}

.modal-container {
  background: #141924;
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 20px;
  width: 520px;
  max-height: 80vh;
  display: flex;
  flex-direction: column;
  box-shadow: 0 24px 80px rgba(0, 0, 0, 0.6);
  animation: slideUp 0.2s ease;
}

@keyframes slideUp {
  from { opacity: 0; transform: translateY(20px); }
  to { opacity: 1; transform: translateY(0); }
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 20px 24px 0;
  flex-shrink: 0;
}

.modal-header h3 {
  font-size: 17px;
  font-weight: 600;
  color: rgba(255, 255, 255, 0.9);
  margin: 0;
}

.modal-close {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  border: none;
  background: rgba(255, 255, 255, 0.06);
  color: rgba(255, 255, 255, 0.4);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.15s ease;
}

.modal-close:hover {
  background: rgba(255, 80, 80, 0.15);
  color: rgba(255, 120, 120, 0.8);
}

.modal-body {
  flex: 1;
  overflow-y: auto;
  padding: 16px 24px;
}

.form-row {
  display: flex;
  gap: 12px;
  margin-bottom: 12px;
}

.form-icon-field {
  flex-shrink: 0;
  width: 90px;
}

.form-name-field {
  flex: 1;
}

.form-full {
  flex: 1;
}

.form-label {
  font-size: 11px;
  color: rgba(255, 255, 255, 0.4);
  margin-bottom: 6px;
  letter-spacing: 0.3px;
  text-transform: uppercase;
  font-weight: 500;
}

.selected-count {
  color: rgba(139, 92, 246, 0.8);
  font-weight: 600;
}

.form-input {
  width: 100%;
  padding: 9px 12px;
  border-radius: 10px;
  border: 1px solid rgba(255, 255, 255, 0.1);
  background: rgba(255, 255, 255, 0.05);
  color: rgba(255, 255, 255, 0.9);
  font-size: 13px;
  outline: none;
  transition: border-color 0.2s ease;
}

.form-input::placeholder {
  color: rgba(255, 255, 255, 0.25);
}

.form-input:focus {
  border-color: rgba(255, 255, 255, 0.25);
}

.icon-picker {
  display: flex;
  align-items: center;
  gap: 6px;
}

.icon-preview {
  width: 38px;
  height: 38px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 22px;
  background: rgba(255, 255, 255, 0.06);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 10px;
}

.icon-random-btn {
  width: 38px;
  height: 38px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(255, 255, 255, 0.06);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 10px;
  cursor: pointer;
  font-size: 16px;
  transition: all 0.15s ease;
}

.icon-random-btn:hover {
  background: rgba(255, 255, 255, 0.12);
  border-color: rgba(255, 255, 255, 0.2);
}

.form-sounds-section {
  margin-top: 12px;
}

.selected-sounds {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 12px;
  padding: 10px;
  background: rgba(139, 92, 246, 0.06);
  border: 1px solid rgba(139, 92, 246, 0.15);
  border-radius: 12px;
}

.selected-tag {
  display: flex;
  align-items: center;
  gap: 6px;
  background: rgba(255, 255, 255, 0.08);
  border-radius: 8px;
  padding: 4px 8px;
}

.tag-name {
  font-size: 11px;
  color: rgba(255, 255, 255, 0.7);
  white-space: nowrap;
}

.tag-volume {
  width: 60px;
  height: 3px;
  -webkit-appearance: none;
  appearance: none;
  background: rgba(255, 255, 255, 0.15);
  border-radius: 2px;
  outline: none;
  cursor: pointer;
}

.tag-volume::-webkit-slider-thumb {
  -webkit-appearance: none;
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: rgba(167, 139, 250, 0.9);
  cursor: pointer;
}

.tag-remove {
  background: none;
  border: none;
  color: rgba(255, 255, 255, 0.3);
  cursor: pointer;
  padding: 2px;
  display: flex;
  align-items: center;
  transition: color 0.15s;
}

.tag-remove:hover {
  color: rgba(255, 100, 100, 0.8);
}

.sound-groups {
  padding-right: 4px;
}

.sound-group {
  margin-bottom: 14px;
}

.group-title {
  font-size: 11px;
  font-weight: 600;
  color: rgba(255, 255, 255, 0.35);
  margin-bottom: 6px;
  letter-spacing: 0.3px;
}

.group-sounds {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.sound-chip {
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid rgba(255, 255, 255, 0.06);
  border-radius: 8px;
  color: rgba(255, 255, 255, 0.5);
  padding: 5px 12px;
  font-size: 12px;
  cursor: pointer;
  transition: all 0.15s ease;
}

.sound-chip:hover {
  background: rgba(255, 255, 255, 0.1);
  color: rgba(255, 255, 255, 0.8);
  border-color: rgba(255, 255, 255, 0.15);
}

.sound-chip.active {
  background: rgba(139, 92, 246, 0.2);
  border-color: rgba(139, 92, 246, 0.4);
  color: rgba(209, 191, 255, 0.95);
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  padding: 12px 24px 20px;
  flex-shrink: 0;
  border-top: 1px solid rgba(255, 255, 255, 0.06);
}

.btn-cancel,
.btn-save {
  padding: 9px 22px;
  border-radius: 10px;
  border: none;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
}

.btn-cancel {
  background: rgba(255, 255, 255, 0.06);
  color: rgba(255, 255, 255, 0.6);
}

.btn-cancel:hover {
  background: rgba(255, 255, 255, 0.12);
  color: rgba(255, 255, 255, 0.9);
}

.btn-save {
  background: rgba(139, 92, 246, 0.3);
  color: rgba(255, 255, 255, 0.95);
}

.btn-save:hover:not(:disabled) {
  background: rgba(139, 92, 246, 0.45);
}

.btn-save:disabled {
  opacity: 0.3;
  cursor: not-allowed;
}

.btn-share-code {
  padding: 9px 22px;
  border-radius: 10px;
  border: none;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
  background: rgba(139, 92, 246, 0.3);
  color: rgba(255, 255, 255, 0.95);
}

.btn-share-code:hover {
  background: rgba(139, 92, 246, 0.45);
}

.card-share {
  left: 6px;
}

.card-share:hover {
  color: rgba(139, 92, 246, 0.8);
  background: rgba(139, 92, 246, 0.15);
}

.scene-card-import {
  border-style: dashed;
  border-color: rgba(255, 255, 255, 0.1);
  background: transparent;
}

.scene-card-import:hover {
  border-color: rgba(52, 211, 153, 0.4);
  background: rgba(52, 211, 153, 0.06);
}

.scene-card-import:hover .add-icon {
  color: rgba(52, 211, 153, 0.6);
}

.import-modal {
  width: 420px;
}

.import-hint {
  font-size: 13px;
  color: rgba(255, 255, 255, 0.45);
  margin-bottom: 12px;
  line-height: 1.4;
}

.import-textarea {
  width: 100%;
  padding: 12px 14px;
  border-radius: 12px;
  border: 1px solid rgba(255, 255, 255, 0.1);
  background: rgba(255, 255, 255, 0.05);
  color: rgba(255, 255, 255, 0.9);
  font-size: 13px;
  font-family: 'SF Mono', 'Menlo', 'Consolas', monospace;
  outline: none;
  resize: vertical;
  transition: border-color 0.2s ease;
  line-height: 1.5;
}

.import-textarea:focus {
  border-color: rgba(139, 92, 246, 0.5);
}

.import-textarea::placeholder {
  color: rgba(255, 255, 255, 0.2);
}

.toast {
  position: fixed;
  bottom: 40px;
  left: 50%;
  transform: translateX(-50%);
  background: rgba(20, 25, 36, 0.95);
  border: 1px solid rgba(255, 255, 255, 0.12);
  border-radius: 12px;
  padding: 12px 24px;
  font-size: 14px;
  color: rgba(255, 255, 255, 0.9);
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.5);
  backdrop-filter: blur(12px);
  z-index: 300;
  white-space: nowrap;
  max-width: 90vw;
  overflow: hidden;
  text-overflow: ellipsis;
}

.toast-enter-active {
  transition: all 0.25s ease;
}

.toast-leave-active {
  transition: all 0.2s ease;
}

.toast-enter-from {
  opacity: 0;
  transform: translateX(-50%) translateY(12px);
}

.toast-leave-to {
  opacity: 0;
  transform: translateX(-50%) translateY(-4px);
}

.card-modal-container {
  background: #141924;
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 20px;
  width: 480px;
  max-height: 85vh;
  display: flex;
  flex-direction: column;
  box-shadow: 0 24px 80px rgba(0, 0, 0, 0.6);
  animation: slideUp 0.2s ease;
}

.card-modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 20px 24px 0;
  flex-shrink: 0;
}

.card-modal-header h3 {
  font-size: 17px;
  font-weight: 600;
  color: rgba(255, 255, 255, 0.9);
  margin: 0;
}

.card-modal-body {
  flex: 1;
  overflow-y: auto;
  padding: 16px 24px;
  display: flex;
  justify-content: center;
  align-items: center;
}

.card-preview {
  max-width: 100%;
  height: auto;
  display: block;
  border-radius: 12px;
  box-shadow: 0 4px 24px rgba(0, 0, 0, 0.4);
}

.card-modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  padding: 12px 24px 20px;
  flex-shrink: 0;
  border-top: 1px solid rgba(255, 255, 255, 0.06);
}
</style>
