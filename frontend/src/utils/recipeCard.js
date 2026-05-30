/**
 * VibeTime 声音卡片生成器
 *
 * 在 canvas 上绘制一张含 QR 码 + 配方信息的分享图片
 */

import QRCode from 'qrcode'

const CARD_W = 600
const CARD_H = 780
const PAD = 32
const QR_SZ = 110

/**
 * 生成声音卡片（PNG Blob）
 * @param {Object} preset - { name, icon, desc, sounds }
 * @param {string} recipeCode - 已编码的配方码字符串
 * @returns {Promise<Blob>} PNG blob
 */
export async function generateRecipeCard(preset, recipeCode) {
  const canvas = document.createElement('canvas')
  canvas.width = CARD_W
  canvas.height = CARD_H
  const ctx = canvas.getContext('2d')

  // ── 背景渐变 ──
  const grad = ctx.createLinearGradient(0, 0, 0, CARD_H)
  grad.addColorStop(0, '#0e121f')
  grad.addColorStop(0.45, '#141a2a')
  grad.addColorStop(1, '#080c14')
  fillRect(ctx, 0, 0, CARD_W, CARD_H, grad)

  let y = PAD

  // ── 标题 ──
  setFont(ctx, '600 22px')
  ctx.fillStyle = 'rgba(255,255,255,0.4)'
  ctx.fillText('🎵  Vibe Time', PAD, y + 22)
  y += 44

  // ── 分割线 ──
  line(ctx, PAD, y, CARD_W - PAD, y, 'rgba(255,255,255,0.06)', 1)
  y += 28

  // ── 配方图标 ──
  ctx.textAlign = 'center'
  setFont(ctx, '56px sans-serif')
  ctx.fillText(preset.icon || '🎵', CARD_W / 2, y + 56)
  y += 72

  // ── 配方名称 ──
  setFont(ctx, '700 26px')
  ctx.fillStyle = 'rgba(255,255,255,0.92)'
  ctx.fillText(preset.name || '未命名', CARD_W / 2, y + 26)
  y += 38

  // ── 描述 ──
  if (preset.desc) {
    setFont(ctx, '400 15px')
    ctx.fillStyle = 'rgba(255,255,255,0.45)'
    ctx.fillText(preset.desc, CARD_W / 2, y + 15)
    y += 30
  }

  // ── 分隔标题 ──
  setFont(ctx, '500 11px')
  ctx.fillStyle = 'rgba(255,255,255,0.25)'
  ctx.fillText('— 声音组合 —', CARD_W / 2, y)
  y += 28
  ctx.textAlign = 'left'

  // ── 声音条目 ──
  const sounds = Object.entries(preset.sounds)
  const barX = PAD + 90
  const barW = CARD_W - barX - QR_SZ - 52

  for (const [id, vol] of sounds) {
    const label = getLabel(id)
    const pct = Math.round(vol * 100)
    const rowH = 20
    const barH = 14
    const barY = y + (rowH - barH) / 2

    // 标签
    setFont(ctx, '400 14px')
    ctx.fillStyle = 'rgba(255,255,255,0.8)'
    ctx.fillText(label, PAD, y + 14)

    // 音量条背景
    roundRect(ctx, barX, barY, barW, barH, 7, 'rgba(255,255,255,0.08)')

    // 音量条填充
    if (pct > 0) {
      const fillW = Math.max(barW * vol, 8)
      const barGrad = ctx.createLinearGradient(barX, 0, barX + barW, 0)
      barGrad.addColorStop(0, '#7c3aed')
      barGrad.addColorStop(1, '#a78bfa')
      roundRect(ctx, barX, barY, fillW, barH, 7, barGrad)

      // 发光
      ctx.globalAlpha = 0.15
      roundRect(ctx, barX, barY, fillW, barH, 7, '#a78bfa')
      ctx.globalAlpha = 1
    }

    // 百分比
    ctx.textAlign = 'right'
    setFont(ctx, '500 12px')
    ctx.fillStyle = 'rgba(255,255,255,0.5)'
    ctx.fillText(`${pct}%`, CARD_W - PAD - QR_SZ - 24, y + 14)
    ctx.textAlign = 'left'

    y += rowH + 10
  }

  // ── 底部间隙 ──
  const bottomAreaY = CARD_H - PAD - QR_SZ - 48
  if (y < bottomAreaY) y = bottomAreaY

  // ── QR 码 ──
  const qrX = CARD_W - PAD - QR_SZ
  const qrY = y

  // QR 白色底
  roundRect(ctx, qrX - 10, qrY - 10, QR_SZ + 20, QR_SZ + 20, 12, '#ffffff')

  // 生成 QR
  const qrCanvas = document.createElement('canvas')
  await QRCode.toCanvas(qrCanvas, recipeCode, {
    width: QR_SZ,
    margin: 1,
    color: { dark: '#000000', light: '#ffffff' }
  })
  ctx.drawImage(qrCanvas, qrX, qrY, QR_SZ, QR_SZ)

  // "扫码获取配方" 文字
  ctx.textAlign = 'center'
  setFont(ctx, '400 12px')
  ctx.fillStyle = 'rgba(255,255,255,0.3)'
  ctx.fillText('扫码获取配方', CARD_W - PAD - QR_SZ / 2, qrY + QR_SZ + 28)
  ctx.textAlign = 'left'

  // ── 底部配方码 ──
  const bottomY = CARD_H - PAD - 12
  setFont(ctx, '400 10px "SF Mono", "Menlo", "Consolas", monospace')
  ctx.fillStyle = 'rgba(255,255,255,0.15)'
  ctx.textAlign = 'center'
  const truncated = recipeCode.length > 58 ? recipeCode.slice(0, 58) + '…' : recipeCode
  ctx.fillText(truncated, CARD_W / 2, bottomY)

  return new Promise((resolve, reject) => {
    canvas.toBlob(blob => {
      if (blob) resolve(blob)
      else reject(new Error('生成图片失败'))
    }, 'image/png')
  })
}

// ── Canvas 工具函数 ──

function fillRect(ctx, x, y, w, h, fill) {
  ctx.fillStyle = fill
  ctx.fillRect(x, y, w, h)
}

function setFont(ctx, str) {
  ctx.font = str + ' -apple-system, "Segoe UI", "Noto Color Emoji", sans-serif'
}

function line(ctx, x1, y1, x2, y2, stroke, width) {
  ctx.strokeStyle = stroke
  ctx.lineWidth = width
  ctx.beginPath()
  ctx.moveTo(x1, y1)
  ctx.lineTo(x2, y2)
  ctx.stroke()
}

function roundRect(ctx, x, y, w, h, r, fill) {
  ctx.beginPath()
  ctx.moveTo(x + r, y)
  ctx.lineTo(x + w - r, y)
  ctx.arcTo(x + w, y, x + w, y + r, r)
  ctx.lineTo(x + w, y + h - r)
  ctx.arcTo(x + w, y + h, x + w - r, y + h, r)
  ctx.lineTo(x + r, y + h)
  ctx.arcTo(x, y + h, x, y + h - r, r)
  ctx.lineTo(x, y + r)
  ctx.arcTo(x, y, x + r, y, r)
  ctx.closePath()
  ctx.fillStyle = fill
  ctx.fill()
}

// ── 声音 ID 转短标签 ──

const LABEL_MAP = {
  'heavy-rain': '大雨',
  'light-rain': '小雨',
  'thunder': '雷声',
  'rain-on-window': '窗上雨',
  'rain-on-tent': '帐篷雨',
  'rain-on-leaves': '叶上雨',
  'rain-on-umbrella': '伞上雨',
  'rain-on-car-roof': '车顶雨',
  'birds': '鸟鸣',
  'cat-purring': '猫咪呼噜',
  'chickens': '鸡鸣',
  'cows': '牛叫',
  'crickets': '蟋蟀',
  'crows': '乌鸦',
  'dog-barking': '狗吠',
  'frog': '蛙鸣',
  'horse-gallop': '马蹄',
  'owl': '猫头鹰',
  'seagulls': '海鸥',
  'sheep': '绵羊',
  'whale': '鲸鱼',
  'wolf': '狼嚎',
  'woodpecker': '啄木鸟',
  'beehive': '蜂巢',
  'campfire': '篝火',
  'droplets': '水滴',
  'howling-wind': '呼啸风',
  'jungle': '丛林',
  'river': '河流',
  'walk-in-snow': '踏雪',
  'walk-on-gravel': '踏碎石',
  'walk-on-leaves': '踏落叶',
  'waterfall': '瀑布',
  'waves': '海浪',
  'wind-in-trees': '林间风',
  'wind': '风声',
  'brown-noise': '棕色噪音',
  'pink-noise': '粉色噪音',
  'white-noise': '白噪音',
  'airport': '机场',
  'cafe': '咖啡厅',
  'carousel': '旋转木马',
  'church': '教堂',
  'construction-site': '工地',
  'crowded-bar': '喧闹酒吧',
  'laboratory': '实验室',
  'laundry-room': '洗衣房',
  'library': '图书馆',
  'night-village': '夜村',
  'office': '办公室',
  'restaurant': '餐厅',
  'subway-station': '地铁站',
  'supermarket': '超市',
  'temple': '寺庙',
  'underwater': '水下',
  'boiling-water': '沸水',
  'bubbles': '气泡',
  'ceiling-fan': '吊扇',
  'clock': '时钟',
  'dryer': '烘干机',
  'keyboard': '键盘',
  'morse-code': '摩斯电码',
  'paper': '纸张',
  'singing-bowl': '颂钵',
  'slide-projector': '幻灯机',
  'tuning-radio': '收音机',
  'typewriter': '打字机',
  'vinyl-effect': '黑胶唱片',
  'washing-machine': '洗衣机',
  'wind-chimes': '风铃',
  'windshield-wipers': '雨刮器',
  'airplane': '飞机',
  'inside-a-train': '火车内',
  'rowing-boat': '划船',
  'sailboat': '帆船',
  'submarine': '潜水艇',
  'train': '火车',
  'ambulance-siren': '救护车',
  'busy-street': '繁忙街道',
  'crowd': '人群',
  'fireworks': '烟花',
  'highway': '高速公路',
  'road': '马路',
  'traffic': '交通',
  'binaural-alpha': 'Alpha 波',
  'binaural-beta': 'Beta 波',
  'binaural-delta': 'Delta 波',
  'binaural-gamma': 'Gamma 波',
  'binaural-theta': 'Theta 波',
}

function getLabel(id) {
  const name = id.split('/').pop()
  return LABEL_MAP[name] || name.replace(/-/g, ' ')
}
