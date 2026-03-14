# OSSAM - Open Source Software Application Market

[![Go](https://img.shields.io/badge/Go-1.23-00ADD8?style=flat&logo=go)](https://go.dev/)
[![Wails](https://img.shields.io/badge/Wails-v2.11.0-00C7B7?style=flat)](https://wails.io/)
[![Vue](https://img.shields.io/badge/Vue-3.5-4FC08D?style=flat&logo=vue.js)](https://vuejs.org/)
[![License](https://img.shields.io/badge/License-MIT-blue?style=flat)](LICENSE)

**OSSAM** 是一款面向 GitHub 开源软件的桌面应用市场，提供一键下载、集中浏览、多线程加速等能力，让工程师快速装机、高效投产。

---

## 📖 目录

- [项目概述](#-项目概述)
- [解决痛点](#-解决痛点)
- [核心功能](#-核心功能)
- [技术栈](#-技术栈)
- [快速开始](#-快速开始)
- [项目结构](#-项目结构)
- [配置说明](#-配置说明)
- [开发指南](#-开发指南)
- [构建与发布](#-构建与发布)
- [贡献指南](#-贡献指南)
- [License](#-license)

---

## 📋 项目概述

OSSAM (Open Source Software Application Market) 是一个基于 **Go + Wails + Vue 3** 构建的跨平台桌面应用，旨在为开发者提供统一的 GitHub 开源软件管理体验。

### 设计理念

- **一键下载**：自动识别 GitHub Release，智能匹配平台架构
- **集中管理**：分类浏览、模糊搜索、批量下载
- **加速体验**：内置 CDN 加速，多线程并发下载
- **开源可扩展**：软件源配置开源，社区共同维护

---

## 🎯 解决痛点

| 场景 | 传统方式 | OSSAM 方案 |
|------|----------|------------|
| 新电脑装机 | 手动搜索、逐个下载 | 分类浏览、一键批量下载 |
| GitHub 访问 | 需要 VPN、下载慢 | 内置 CDN 加速、稳定下载 |
| 版本管理 | 手动检查更新 | 自动检测、一键升级 |
| 软件发现 | 信息分散、难以筛选 | 分类管理、快速搜索 |

---

## ✨ 核心功能

### 1. 多线程高速下载
- 基于 Go `net/http` 实现并发下载
- 支持断点续传（预留扩展点）
- 下载进度实时显示

### 2. 智能平台匹配
- 自动识别 Windows / macOS / Linux
- 识别 AMD64 / ARM64 / 386 架构
- 智能匹配 Release 资产

### 3. CDN 加速
- 内置 `ghproxy.net` / `ghfast.top` 加速源
- 支持自定义 CDN 源
- 可配置开关与优先级

### 4. 应用分类管理
```
🛡️ 安全工具 (Security)
🛠️ 开发利器 (DevTools)
💻 系统增强 (System)
🌐 网络插件 (Network)
📊 数据管理 (Database)
📟 终端艺术 (Terminal)
🎨 效率办公 (Utility)
```

### 5. 自动更新
- 检测 GitHub Release 版本
- 对比本地版本号
- 提示并执行更新

### 6. 安全校验
- 下载文件 Hash 校验（预留）
- 完整性验证
- 异常处理与重试

---

## 🛠️ 技术栈

### 后端
| 技术 | 版本 | 说明 |
|------|------|------|
| Go | 1.23 | 核心后端逻辑 |
| Wails | v2.11.0 | 桌面应用框架 |

### 前端
| 技术 | 版本 | 说明 |
|------|------|------|
| Vue | 3.5.30 | 响应式框架 |
| Vite | 8.0.0 | 构建工具 |
| Element Plus | 2.13.5 | UI 组件库 |
| Vue Router | 5.0.3 | 路由管理 |
| Marked | 17.0.4 | Markdown 渲染 |
| DOMPurify | 3.3.3 | XSS 防护 |

### 开发环境要求
```bash
Node.js: 24.14.x (锁定范围：>=24.14.0 <24.15.0)
npm:     11.x    (锁定范围：>=11.0.0 <12.0.0)
Registry: https://registry.npmmirror.com
```

---

## 🚀 快速开始

### 环境准备

```bash
# 安装 Go 1.23+
go version

# 安装 Node.js 24.14.x (推荐使用 nvm)
nvm install 24.14
nvm use 24.14

# 安装 Wails CLI
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

### 克隆项目

```bash
git clone https://github.com/HEUdbh/ossam.git
cd ossam
```

### 安装依赖

```bash
# 前端依赖
cd frontend
npm ci
npm run build
cd ..
```

### 开发模式

```bash
# 启动开发服务器（热重载）
wails dev
```

### 生产构建

```bash
# 构建桌面应用
wails build
```

构建产物位于 `build/bin/` 目录。

---

## 📁 项目结构

```
ossam/
├── main.go                 # Wails 应用入口
├── app.go                  # 核心业务逻辑
├── go.mod                  # Go 模块定义
├── go.sum                  # Go 依赖锁定
├── wails.json              # Wails 配置
│
├── config/                 # 配置文件目录
│   ├── rules.json          # 全局规则与分类定义
│   └── categories/         # 分类应用配置
│       ├── security.json
│       ├── devtools.json
│       ├── system.json
│       ├── network.json
│       ├── database.json
│       ├── terminal.json
│       └── utility.json
│
├── frontend/               # 前端源码
│   ├── src/
│   │   ├── App.vue         # 根组件
│   │   ├── main.js         # 入口文件
│   │   ├── router/         # 路由配置
│   │   ├── stores/         # 状态管理
│   │   ├── views/          # 页面组件
│   │   │   ├── MarketView.vue
│   │   │   ├── AppDetailView.vue
│   │   │   ├── MineView.vue
│   │   │   └── Settings*.vue
│   │   ├── components/     # 通用组件
│   │   ├── utils/          # 工具函数
│   │   └── assets/         # 静态资源
│   ├── index.html
│   ├── package.json
│   ├── vite.config.js
│   └── scripts/
│       └── check-node-version.cjs
│
├── build/                  # 构建输出
│   ├── appicon.png         # 应用图标
│   └── README.md
│
└── .github/
    └── workflows/
        └── build.yml       # CI/CD 配置
```

---

## ⚙️ 配置说明

### rules.json - 全局配置

```json
{
  "market_name": "ossam",
  "last_updated": "20260311",
  "default_match": ".*\\.(zip|tar\\.gz|tgz|7z|dmg|pkg|msi|exe|AppImage|deb|rpm)$",
  "platform_keywords": {
    "windows": ["windows", "win", "win32", "win64"],
    "linux": ["linux", "gnu", "musl"],
    "macos": ["mac", "macos", "darwin", "osx"]
  },
  "platform_match": {
    "windows": "(windows|win)",
    "linux": "linux",
    "macos": "(mac|macos)"
  },
  "categories": [
    { "name": "Security", "file": "config/categories/security.json" },
    { "name": "DevTools", "file": "config/categories/devtools.json" },
    ...
  ]
}
```

### 分类配置示例 (security.json)

```json
{
  "apps": [
    {
      "name": "OneForAll",
      "repo": "shmilylty/OneForAll",
      "photo": "",
      "summary": "OneForAll：子域名发现工具。"
    },
    {
      "name": "dirsearch",
      "repo": "maurosoria/dirsearch",
      "photo": "",
      "summary": "dirsearch：安全测试与资产发现工具。",
      "match": "dirsearch-.*\\.tar.gz"
    }
  ]
}
```

### 配置字段说明

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `name` | string | ✅ | 应用展示名称 |
| `repo` | string | ✅ | GitHub 仓库标识 (`owner/repo`) |
| `photo` | string | ❌ | 应用图标 URL（空则使用默认头像） |
| `summary` | string | ✅ | 应用简介 |
| `match` | string | ❌ | 文件名匹配正则（空则使用 `default_match`） |

---

## 📖 开发指南

### 前端开发

```bash
cd frontend

# 开发模式（热重载）
npm run dev

# 生产构建
npm run build

# 预览构建产物
npm run preview
```

### 后端开发

```bash
# 运行测试
go test ./...

# 格式化代码
go fmt ./...

# 代码检查
go vet ./...
```

### 推荐 IDE 配置

- **VS Code** + [Volar](https://marketplace.visualstudio.com/items?itemName=Vue.volar) (Vue 3 支持)
- **Go** + [Go 插件](https://marketplace.visualstudio.com/items?itemName=golang.Go)

---

## 📦 构建与发布

### 跨平台构建

```bash
# Windows
wails build -platform windows/amd64

# macOS
wails build -platform darwin/universal

# Linux
wails build -platform linux/amd64
```

### CI/CD

项目使用 GitHub Actions 自动化构建，配置位于 `.github/workflows/build.yml`。

---

## 🤝 贡献指南

### 添加新软件

1. 在 `config/categories/` 对应分类文件中添加应用配置
2. 确保 `repo` 字段格式正确 (`owner/repo`)
3. 提交 PR 并说明软件用途

### 报告问题

- 使用 GitHub Issues 报告 Bug
- 提供复现步骤、环境信息、日志截图

### 功能建议

- 在 Issues 中描述使用场景
- 说明期望的行为与价值

---

## 📄 License

MIT License - 详见 [LICENSE](LICENSE) 文件

---

## 👤 作者

- **HEUdbh**
- Email: 15399211657@163.com

---

## 🙏 致谢

感谢以下开源项目：

- [Wails](https://wails.io/) - Go 桌面应用框架
- [Vue.js](https://vuejs.org/) - 渐进式前端框架
- [Element Plus](https://element-plus.org/) - Vue 3 UI 组件库
- [ghproxy](https://ghproxy.net/) - GitHub 加速服务

---

<div align="center">

**如果这个项目对你有帮助，请给一个 ⭐️ Star！**

</div>
