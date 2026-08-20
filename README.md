# 我在地府打麻将 · 存档迁移

将安卓版《我在地府打麻将》的存档迁移到 Windows 平台的 Steam 版本。

![应用截图](images/app.png)

## 功能

- **自动检测 Steam 存档目录** — 通过注册表定位 Steam 安装路径，遍历 `userdata` 找到对应 appid `3444020` 的用户存档目录
- **三种安卓存档获取方式**
  - ADB 自动拉取（需 USB 调试）
  - 手动选择 `game.db` 文件
  - 同局域网 Wi-Fi 上传（手机扫码即可）
- **存档预览与选择** — 读取 SQLite 数据库，列出 slot 表中的所有存档，支持选择性迁移
- **安全迁移** — 迁移前自动备份原存档为 `.bak` 文件

## 工作流程

1. 选择 Steam 存档目录（自动检测 / 手动指定）
2. 获取安卓存档（ADB / 手动 / Wi-Fi）
3. 预览存档列表，勾选要迁移的存档
4. 点击「开始迁移」，完成

## 开发

### 环境要求

- [Go](https://go.dev/) 1.25+
- [Node.js](https://nodejs.org/) 18+
- [pnpm](https://pnpm.io/)
- [Wails](https://wails.io/) v2

### 安装 Wails

```bash
go install github.com/wailsapp/wails/v3/cmd/wails@latest
```

### 开发模式

```bash
wails dev
```

### 构建

```bash
wails build
```

## 项目结构

```
.
├── frontend/          # Vue 3 + TypeScript 前端
├── internal/
│   ├── android/       # 安卓存档获取（ADB / Wi-Fi / SQLite 解析）
│   ├── migrate/       # 迁移核心逻辑
│   └── steam/         # Steam 路径检测
├── build/             # 构建资源（图标、安装包配置）
├── app.go             # Wails 应用核心
├── main.go            # 入口
└── wails.json         # Wails 配置
```

## 许可证

[MIT](LICENSE)
