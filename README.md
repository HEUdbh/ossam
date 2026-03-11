# 一、项目概述

一款针对 GitHub 的开源软件应用市场（Open source software application market），对开源软件进行综合管理。支持一键下载最新版本，集中浏览，方便快捷。多线程下载，避免长久等待。

# 二、解决痛点

传统工程师换新电脑，或者去到新公司，需要快速装机，投入生产。去 GitHub 手动下载存在需要一个一个搜索并下载，程序繁琐，容易疏漏等问题；使用应用市场统一管理，方便高效，同时可以避免疏漏的情况。
面对新电脑，通常还存在需要 VPN 才能稳定访问 GitHub 和开源 VPN 在 GitHub 上下载的尴尬问题，软件提供统一下载体验，并且新人友好。

# 三、功能分析

## 1.多线程下载

使用golang实现多线程同时下载

## 2.稳定下载能力

提供 GitHub 资源的统一下载流程

## 3.可以修改文件下载位置

允许用户修改默认存储位置并保存

## 4.自动更新

检测 GitHub 仓库 release 版本实现远程自动更新

## 5.读取管理GitHub相关开源项目

通过配置文件读取目标仓库地址，自动拼接release版本，正则匹配相关文件名称进行版本管理和下载。
（下载中断处理？）

## 6.创建软件分类

对软件进行分类处理，方便用户寻找；加入搜索框允许进行模糊搜索
🛡️ 安全工具 (Security),扫描器 (Scanner)、爆破 (Brute Force)、内网渗透、取证
🛠️ 开发利器 (DevTools),IDE、编译器、代码格式化、Git 增强工具
💻 系统增强 (System),包管理器、文件搜索、快速启动、虚拟机、容器
🌐 网络插件 (Network),代理/VPN、抓包工具、API 调试 (Postman 类)、下载增强
📊 数据管理 (Database),SQL 客户端、Redis 管理器、数据可视化、日志分析
终端艺术 (Terminal),Shell、终端模拟器、常用 CLI 工具、Zsh 插件
🎨 效率办公 (Utility),截图工具、录屏、Markdown 编辑器、笔记软件

## 7.创建用户管理（待议）

允许登录（非强制），登录可以保存自己的个人软件配置包实现个人偏好一键下载，无需再去寻找等；相关用户信息通过 Cloudflare 保存；

## 8.安全校验

对下载文件的 Hash 进行校验保证下载的完整性；

## 9.跨平台架构识别

识别本地计算机架构，实现精准匹配（Windows、macOS）

## 10.开源的软件源清单

对软件源配置文件实行开源，欢迎补充丰富

## 11.依赖环境检测

检测系统环境，检测必要依赖是否安装等

# 四、前端设计

1. 默认主界面即为应用市场，一级侧边栏：“应用市场”、“我的”，一级侧边栏底部设置窗口
2. 应用市场：二级侧边栏为软件分类；主界面为具体的软件展示界面，具体的软件内容读取软件配置文件；展示方式采用图标+名称的方式展示，图标来自于远程连接或者读取 GitHub 的作者头像；点击任意软件图标，跳转新页面显示软件详细：名称、介绍、版本、开发者、原仓库连接，以及不同平台的不同下载按钮；
3. 我的：需要登录才可以使用，用于管理个人偏好配置（待议）
4. 登录界面
5. 设置界面：两个子页面，采用弹出新窗口的模式显示，分为“下载地址设置”和“关于作者”两个界面；

# 五、存储数据设计

## 1.用户信息存储

使用 Cloudflare 的 KV 数据库存储即可

## 2.用户偏好存储

使用 Cloudflare 的 KV 数据库存储

## 3.软件配置表

采用json格式存储

```
{
  "market_name": "ossam",
  "last_updated": "20260309",
  "apps": {
    "Security": [
      {
        "name": "oneforall",
        "repo": "shmilylty/OneForAll",
        "photo": "https://github.com/<用户名>.png", 
        "match": ".*\\.zip"
      },
      {
        "name": "dirsearch",
        "repo": "maurosoria/dirsearch",
        "photo": "https://cas.hrbeu.edu.cn/favicon.ico",
        "match": "dirsearch-.*\\.tar.gz"
      }
    ],
    "DevTools": [
      {
        "name": "fzf",
        "repo": "junegunn/fzf",
        "photo": "",
        "match": "fzf-.*-windows_amd64.zip"
      }
    ],
    "System": []
  }
}
```

# 六、开发阶段设计

## 阶段 1：项目骨架与配置读取

- 阶段目标：搭建可运行的桌面应用最小闭环，完成配置驱动能力。
- 功能清单：初始化并稳定现有 Go + Wails + Vue 工程；读取本地 `appsconfig.json`；建立配置解析与异常提示（文件缺失、JSON 格式错误）。
- 阶段产出：应用可启动；可从本地配置加载并展示基础应用数据。

## 阶段 2：GitHub Release 拉取与下载核心

- 阶段目标：打通“仓库版本发现 -> 资源匹配 -> 下载执行”的核心链路。
- 功能清单：接入 GitHub Release 信息拉取；按 `match` 正则匹配目标资源；实现并发下载；预留断点续传扩展点（任务元数据结构与接口）。
- 阶段产出：可稳定下载目标版本文件；可追踪下载任务状态（开始、进行中、失败、完成）。

## 阶段 3：应用市场 UI

- 阶段目标：完成面向用户的应用浏览与操作入口。
- 功能清单：实现分类导航与应用列表；支持关键字模糊搜索；实现应用详情页（名称、介绍、版本、开发者、仓库链接）；按平台显示下载按钮。
- 阶段产出：形成可用的应用市场主流程（浏览、检索、查看、下载）。

## 阶段 4：可用性增强

- 阶段目标：提升日常使用效率与维护便捷性。
- 功能清单：支持下载目录设置并持久化；实现应用自更新检查；增加依赖环境检测（必要命令/运行环境检查）。
- 阶段产出：用户可配置下载行为；系统可在启动或设置页触发环境自检与更新提示。

## 阶段 5：安全与发布

- 阶段目标：提升交付质量与安全可靠性，支持正式分发。
- 功能清单：增加下载文件 Hash 校验；完善下载异常与网络异常处理；完成 Windows/macOS 打包流程与发布产物整理。
- 阶段产出：具备可发布桌面安装包；关键下载链路具备完整性校验与错误反馈。

## 阶段 6：增强能力（后置）

- 阶段目标：补充账号体系与跨设备偏好同步能力。
- 功能清单：实现可选登录；支持个人偏好配置保存与读取；接入 Cloudflare KV 存储用户信息与偏好数据。
- 阶段产出：用户可在多设备同步个人偏好配置；核心下载能力仍可在未登录状态独立使用。

# 七、相关技术栈与选型说明

## 1. 当前技术栈（已落地）

- 后端：Go 1.23（见 `go.mod`）
- 桌面框架：Wails v2
- 前端：Vue 3 + Vite 3（见 `frontend/package.json`）
- 数据驱动：本地 JSON 配置文件（`appsconfig.json`）

## 2. 可选升级方向

- 前端构建链与状态管理增强：Vite 升级、引入 TypeScript 与状态管理（如 Pinia）提升可维护性。
- 下载任务持久化：将下载任务状态持久化到本地存储，支持重启恢复与断点续传落地。
- 日志与监控补充：增加结构化日志、关键链路埋点与错误聚合，提升问题定位效率。

## 3. appsconfig 配置契约说明（文档约定）

本小节用于明确当前配置字段语义，不修改 `appsconfig.json` 既有结构。

- `name`：应用名称（展示名），建议在同一分类内保持唯一。
- `repo`：GitHub 仓库标识，格式为 `owner/repo`。
- `photo`：应用图标地址，可为空；为空时可回退到默认图标或仓库作者头像策略。
- `match`：用于匹配 Release 资产文件名的正则表达式，需与目标平台产物命名规则一致。

## GitHub Request Mapping

| Request Purpose | Trigger Location | URL Template | Proxy Prefix Applied | Notes |
| --- | --- | --- | --- | --- |
| Release list query | `fetchReleases -> newGitHubRequest` | `https://api.github.com/repos/{owner}/{repo}/releases?per_page=30` | No | Sent directly by `newGitHubRequest`. |
| Repository stars query | `fetchRepoStars -> newGitHubRequest` | `https://api.github.com/repos/{owner}/{repo}` | No | Sent directly by `newGitHubRequest`. |
| README fetch (ghproxy + raw) | `fetchReadme -> buildReadmeRawCandidates` | `https://ghproxy.net/https://raw.githubusercontent.com/{owner}/{repo}/main/readme.md`<br>`https://ghproxy.net/https://raw.githubusercontent.com/{owner}/{repo}/main/README.md`<br>`https://ghproxy.net/https://raw.githubusercontent.com/{owner}/{repo}/master/readme.md`<br>`https://ghproxy.net/https://raw.githubusercontent.com/{owner}/{repo}/master/README.md` | Yes | Candidates are requested in order until first success. |
| Release asset download URL (`browser_download_url`) | `selectAssetForPlatform` | `https://github.com/{owner}/{repo}/releases/download/{tag}/{asset}` (typical shape) | No | `downloads[*].download_url` keeps original GitHub URL. |
| `StartDownload` input URL | `StartDownload` | Frontend-provided `download_url` | No | Uses original incoming URL after validation. |
| Default app avatar | `resolveAppPhoto -> buildGitHubAvatarURL` | `https://avatars.githubusercontent.com/{owner}` | No | Default avatar uses direct URL. |
| Default placeholder icon | `resolveAppPhoto` | `https://github.githubassets.com/favicons/favicon.png` | No | Placeholder icon uses direct URL. |
| Custom `photo` field | `resolveAppPhoto` | Original value from `appsconfig.json` | No | Returned as-is, no URL rewrite. |

- Runtime GitHub requests use direct URLs except README fetch, which prepends `https://ghproxy.net/`.
- Non-GitHub URLs are passed through as-is.

