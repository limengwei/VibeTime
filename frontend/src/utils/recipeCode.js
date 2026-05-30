/**
 * VibeTime 配方码编解码器
 *
 * 格式: VIBE:NAME:ICON:DESC:soundID|vol+soundID|vol+CHK
 *
 * 示例: VIBE:雨夜:🌧:雨声+雷声+风:rain/heavy-rain|70+rain/thunder|40+nature/wind|30+A7
 *
 * 校验: DJB2 hash → 2字符 Base36 大写
 */

const PREFIX = 'VIBE:'

/**
 * 将预设编码为可分享的配方码
 * @param {Object} preset - { name, icon, desc, sounds: { id: volume(0-1) } }
 * @returns {string} 配方码，若无效返回空字符串
 */
export function encodeRecipe(preset) {
  if (!preset || !preset.sounds || Object.keys(preset.sounds).length === 0) {
    return ''
  }

  const name = (preset.name || '未命名')
    .replace(/[:+|]/g, '_')
    .slice(0, 30)

  const icon = preset.icon || '🎵'
  const desc = (preset.desc || '')
    .replace(/[:+|]/g, '_')
    .slice(0, 50)

  const soundParts = []
  for (const [id, vol] of Object.entries(preset.sounds)) {
    const volInt = Math.round(Math.max(0, Math.min(1, vol)) * 100)
    soundParts.push(`${id}|${volInt}`)
  }

  if (soundParts.length === 0) return ''

  const payload = `${name}:${icon}:${desc}:${soundParts.join('+')}`
  const checksum = computeChecksum(payload)

  return `${PREFIX}${payload}+${checksum}`
}

/**
 * 将配方码解码为预设对象
 * @param {string} code
 * @returns {Object|null} { name, icon, desc, sounds } 或 null
 */
export function decodeRecipe(code) {
  if (!code || typeof code !== 'string') return null

  const trimmed = code.trim()
  if (!trimmed.startsWith(PREFIX)) return null

  const inner = trimmed.slice(PREFIX.length)

  // 定位校验码 (最后一个 + 之后)
  const lastPlus = inner.lastIndexOf('+')
  if (lastPlus < 0) return null

  const payload = inner.slice(0, lastPlus)
  const checksum = inner.slice(lastPlus + 1)

  // 校验
  if (computeChecksum(payload) !== checksum) return null

  // 解析 payload: name:icon:desc:sounds
  const firstColon = payload.indexOf(':')
  if (firstColon < 0) return null
  const secondColon = payload.indexOf(':', firstColon + 1)
  if (secondColon < 0) return null
  const thirdColon = payload.indexOf(':', secondColon + 1)
  if (thirdColon < 0) return null

  const name = payload.slice(0, firstColon) || '未命名'
  const icon = payload.slice(firstColon + 1, secondColon) || '🎵'
  const desc = payload.slice(secondColon + 1, thirdColon) || ''
  const soundStr = payload.slice(thirdColon + 1)

  // 解析声音条目
  const sounds = {}
  const entries = soundStr.split('+')
  for (const entry of entries) {
    const barIdx = entry.lastIndexOf('|')
    if (barIdx < 0) continue
    const id = entry.slice(0, barIdx)
    const volInt = parseInt(entry.slice(barIdx + 1), 10)
    if (isNaN(volInt) || volInt < 0 || volInt > 100) continue
    sounds[id] = volInt / 100
  }

  if (Object.keys(sounds).length === 0) return null

  return { name, icon, desc: desc || '', sounds }
}

/**
 * DJB2 哈希 → 2字符 Base36 校验码
 * @param {string} str
 * @returns {string} 2字符大写字母数字
 */
function computeChecksum(str) {
  let hash = 5381
  for (let i = 0; i < str.length; i++) {
    hash = ((hash << 5) + hash) + str.charCodeAt(i)
    hash = hash & 0x7fffffff
  }
  return Math.abs(hash).toString(36).slice(0, 2).toUpperCase()
}
