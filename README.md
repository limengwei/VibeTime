# Vibe Time

轻松找到你的宁静 — 一款桌面白噪音/环境音应用，帮助你专注、放松和入眠。

## 功能特性

- **100+ 高品质环境音** — 覆盖 9 大分类：动物、自然、雨声、噪音、场所、物品、交通、城市、双耳节拍
- **场景预设** — 内置「雨夜」「深度专注」「森林清晨」等多个一键场景，也支持自定义预设
- **专注模式** — 全屏沉浸体验，支持倒计时（5/10/20/30/60 分钟）和精美壁纸背景
- **独立音量控制** — 每个音效可单独调节音量，全局主音量控制
- **多音混合播放** — 同时播放多个音效，自由叠加创造属于你的声景
- **系统托盘** — 关闭窗口后最小化到系统托盘，后台继续播放

## 技术栈

| 层 | 技术 |
|---|---|
| 后端 | Go 1.23 + [Wails v2](https://wails.io/) |
| 前端 | Vue 3 + Vite |
| 系统托盘 | [fyne.io/systray](https://github.com/fyne-io/systray) |
| 音频引擎 | Web Audio API（浏览器原生） |

## 项目结构

```
vibe-time/
├── app.go                          # Go 后端：音效/壁纸列表、窗口控制
├── main.go                         # 应用入口：Wails 配置、系统托盘
├── wails.json                      # Wails 项目配置
├── frontend/
│   ├── src/
│   │   ├── App.vue                 # 主界面：音效面板、全局控制
│   │   ├── components/
│   │   │   ├── FocusMode.vue       # 专注模式（全屏+计时+壁纸）
│   │   │   ├── ScenePresets.vue    # 场景预设选择器
│   │   │   └── SoundCard.vue       # 单个音效卡片
│   │   └── audio/
│   │       ├── audioEngine.js      # Web Audio 音频引擎
│   │       └── soundIcons.js       # 音效分类图标
│   └── public/
│       ├── audio/                  # 音效资源（MP3/WAV）
│       └── wallpapers/             # 专注模式壁纸
└── build/                          # 构建资源（图标、清单等）
```

## 开发

### 前置要求

- [Go](https://go.dev/dl/) 1.23+
- [Node.js](https://nodejs.org/) 16+
- [Wails CLI](https://wails.io/docs/gettingstarted/installation) v2

### 安装 Wails CLI

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

### 开发模式

```bash
wails dev
```

启动后自带 Vite 热更新，前端修改即时生效。同时会在 `http://localhost:34115` 启动开发服务器，可在浏览器中调试 Go 方法调用。

### 构建

```bash
wails build
```

生成的可执行文件位于 `build/bin/` 目录下。

## 音效分类

| 分类 | 示例 |
|---|---|
| 🐾 动物 | 鸟鸣、猫呼噜、蟋蟀、鲸鱼、狼嚎… |
| 🌿 自然 | 篝火、河流、瀑布、海浪、丛林… |
| 🌧 雨声 | 大雨、小雨、帐篷雨声、雷声… |
| 🔊 噪音 | 白噪音、棕色噪音、粉色噪音 |
| 🏢 场所 | 咖啡厅、图书馆、机场、地铁… |
| 🔔 物品 | 时钟、打字机、颂钵、风铃… |
| 🚂 交通 | 飞机、火车、帆船、潜水艇… |
| 🏙 城市 | 繁忙街道、烟花、高速公路… |
| 🧠 双耳节拍 | Alpha、Beta、Delta、Theta、Gamma |

## 声明

- 声音文件来源：[Moodist](https://github.com/remvze/moodist)
- 背景图片来源：网络

## License

MIT
