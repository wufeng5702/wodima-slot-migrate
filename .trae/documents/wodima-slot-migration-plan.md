# 我滴妈存档迁移工具 - 实施计划

## 一、Summary（摘要）

构建一个 Windows 桌面应用，将游戏《我在地府打麻将》的安卓端 SQLite 存档迁移到 Steam 版（appid `3444020`）的 `Slot[012].json` 存档目录。后端使用 **Wails v2 (Go)** + 前端 **Vue 3 + TypeScript + Vite**，最终产物为单个 `.exe`（含内嵌 ADB），体积预期 25-40MB，满足 ≤100MB 限制。

### 已确认的关键决策
| 决策项 | 选定方案 |
|---|---|
| 后端框架 | Wails v2 (Go) |
| 前端框架 | Vue 3 + TS + Vite（Wails 默认模板） |
| SQLite 驱动 | `modernc.org/sqlite`（纯 Go，无 CGO） |
| 前端包管理 | pnpm |
| ADB 策略 | 内嵌 adb.exe + 2 个 DLL（运行时解压到 `%LOCALAPPDATA%`） |
| 槽位迁移策略 | UI 列出 DB 所有行，用户勾选要迁移的行 |
| 备份文件名格式 | `Slot{X}.json.{YYYYMMDD_HHMMSS}.bak`（注意：用户原文 `Slog` 视为 `Slot` 笔误） |
| 时间戳格式 | `YYYYMMDD_HHMMSS`（Windows 安全，无冒号） |

## 二、Current State Analysis（现状分析）

### 项目现状
- 空项目，仅有根目录的 `game.db`（3,530,752 字节，SQLite 3 格式），是用户手动从安卓拷贝的样本存档。
- 无 `.git`、无 `go.mod`、无 `package.json`，需要从零初始化。

### 环境探测结果
| 项 | 结果 |
|---|---|
| ADB | 未安装（PATH 与常见位置均无），需内嵌 |
| Steam 注册表 | `HKCU:\Software\Valve\Steam\SteamPath = e:/program files (x86)/steam`，`HKLM:\SOFTWARE\WOW6432Node\Valve\Steam\InstallPath` 一致 |
| game.db | 有效 SQLite 3，将在实施时验证表结构和数据 |

### 安卓存档访问路径分析
- 文件路径：`Android/data/com.itaotuo.wodima/files/game.db` → 即 `/sdcard/Android/data/com.itaotuo.wodima/files/game.db`
- adb 可访问性：Android 11+ 对普通 app 限制了 `Android/data/` 访问，但 `adb shell` 具有提升权限，**可以直接 `adb pull` 该路径**（无需 root、无需应用可调试）
- 前置条件：用户手机需开启 USB 调试；首次连接需授权 PC

## 三、Proposed Changes（具体修改）

### 3.1 项目目录结构

```
d:\__Playground\wodima-slot-migrate\
├── go.mod
├── go.sum
├── wails.json
├── main.go                              # Wails 应用入口
├── app.go                               # Wails App 结构体，绑定到前端的方法
├── internal/
│   ├── steam/
│   │   └── steam.go                     # Steam 安装目录 + steamID 探测
│   ├── android/
│   │   ├── adb.go                       # ADB 包装（设备探测、pull 文件）
│   │   ├── db.go                        # SQLite 读取
│   │   └── embedded/
│   │       ├── adb.exe                  # 内嵌的 ADB 主程序
│   │       ├── AdbWinApi.dll
│   │       └── AdbWinUsbApi.dll
│   └── migrate/
│       └── migrate.go                   # 备份 + 写入逻辑
├── frontend/
│   ├── package.json
│   ├── vite.config.ts
│   ├── tsconfig.json
│   ├── index.html
│   └── src/
│       ├── main.ts
│       ├── App.vue
│       ├── style.css
│       └── components/
│           ├── SteamPanel.vue           # Steam 路径/用户选择
│           ├── AndroidPanel.vue         # 安卓 game.db 获取（自动/手动）
│           ├── SlotTable.vue            # 槽位列表与勾选
│           └── MigratePanel.vue         # 迁移确认与结果日志
├── build/
│   └── windows/
│       ├── info.json                    # Wails NSIS 安装器配置
│       └── wails.exe.manifest           # UAC manifest (asInvoker)
└── .trae/
    └── documents/
        └── wodima-slot-migration-plan.md
```

### 3.2 后端实现细节

#### 3.2.1 `internal/steam/steam.go` — Steam 探测

**功能**：
1. `DetectSteamPath() (string, error)`：读注册表 `HKCU\Software\Valve\Steam` 的 `SteamPath`，失败回退到 `HKLM\SOFTWARE\WOW6432Node\Valve\Steam` 的 `InstallPath`。将正斜杠 `/` 转为 `\`，返回规范路径。
2. `DetectSteamUsers(steamPath string) ([]SteamUser, error)`：遍历 `{steamPath}\userdata\` 下的数字子目录，筛选存在 `3444020\remote\` 的子目录，返回 `[{SteamID, RemotePath}, ...]`。多个时由前端让用户选择。

**数据结构**：
```go
type SteamUser struct {
    SteamID    string `json:"steamId"`
    RemotePath string `json:"remotePath"` // e.g. E:\...\userdata\{steamId}\3444020\remote
}
```

**依赖**：`golang.org/x/sys/windows/registry`

#### 3.2.2 `internal/android/adb.go` — ADB 封装

**功能**：
1. `EnsureADB() (string, error)`：将 `//go:embed embedded/*` 的 adb.exe + 2 个 DLL 解压到 `%LOCALAPPDATA%\wodima-migrate\adb\<version>\`，并写入版本标记文件避免重复解压。返回解压后的 `adb.exe` 路径。
2. `ListDevices(adbPath string) ([]Device, error)`：执行 `adb devices`，解析输出。
3. `PullGameDB(adbPath, deviceSerial, dstPath string) error`：执行 `adb -s <serial> pull /sdcard/Android/data/com.itaotuo.wodima/files/game.db <dstPath>`。失败时返回结构化错误（设备未授权 / 路径不存在 / 应用未安装等）。

**内嵌 ADB 来源**：从 Google 官方 `platform-tools-latest-windows.zip` 下载，提取 3 个文件放入 `internal/android/embedded/`。版本约 5MB。

**错误分类**：
- `ErrNoDevice`：无设备连接
- `ErrUnauthorized`：手机未授权 PC
- `ErrPathNotFound`：路径不存在（应用未安装或存档未生成）
- `ErrUSbDebuggingOff`：USB 调试未开启（adb 启动 server 失败时推断）

#### 3.2.3 `internal/android/db.go` — SQLite 读取

**功能**：
1. `ReadSlots(dbPath string) ([]SlotRow, error)`：用 `modernc.org/sqlite` 以只读模式打开（`file:{path}?mode=ro` 或 `sql.Open` + `SetMaxOpenConns(1)`），执行 `SELECT id, slotIndex, userAccount, jsonString FROM slot ORDER BY slotIndex, userAccount`。

**数据结构**：
```go
type SlotRow struct {
    ID          int64  `json:"id"`
    SlotIndex   int    `json:"slotIndex"`
    UserAccount string `json:"userAccount"`
    JSONString  string `json:"jsonString"`
    JSONSize    int    `json:"jsonSize"`    // len(jsonString)，前端展示用
    JSONPreview string `json:"jsonPreview"` // 前 200 字符，前端展示用
}
```

**安全考虑**：使用 `?mode=ro` 只读打开，避免误写源 DB。

#### 3.2.4 `internal/migrate/migrate.go` — 迁移逻辑

**功能**：
1. `BackupAndWrite(remotePath string, slots []SlotSelection) ([]MigrateResult, error)`：对每条选中行：
   - 计算目标文件：`{remotePath}\Slot{slotIndex}.json`
   - 若目标存在：重命名为 `Slot{slotIndex}.json.{YYYYMMDD_HHMMSS}.bak`
   - 将 `jsonString` 写入目标文件（UTF-8，无 BOM）
2. 返回每条记录的结果（成功/失败 + 实际操作：已备份/直接写入）。

**数据结构**：
```go
type SlotSelection struct {
    ID         int64  `json:"id"`         // DB 行 ID，用于幂等
    SlotIndex  int    `json:"slotIndex"`
    JSONString string `json:"jsonString"`
}

type MigrateResult struct {
    ID         int64  `json:"id"`
    SlotIndex  int    `json:"slotIndex"`
    TargetFile string `json:"targetFile"`
    BackupFile string `json:"backupFile"` // 空字符串表示无备份
    Success    bool   `json:"success"`
    Error      string `json:"error"`
}
```

**原子性**：单文件迁移内若写入失败，已重命名的备份不回滚（避免数据丢失），由错误日志提示用户手动恢复。跨文件迁移不保证事务性（多文件部分成功是允许的）。

#### 3.2.5 `app.go` — Wails 绑定方法

暴露给前端的方法（通过 `context.WithCancel` 支持取消）：

```go
type App struct {
    ctx context.Context
}

// 返回 Steam 安装目录与可选用户列表
func (a *App) DetectSteam() (SteamInfo, error)

// 弹出文件夹选择对话框，返回用户选择的 remote 目录
func (a *App) PickRemoteManually() (string, error)

// 探测安卓设备；成功则自动 pull game.db 到临时目录，返回本地路径
func (a *App) AutoFetchAndroidDB() (string, error)

// 弹出文件选择对话框，让用户选 game.db
func (a *App) PickAndroidDBManually() (string, error)

// 读取指定 game.db 的所有 slot 行
func (a *App) ReadAndroidSlots(dbPath string) ([]SlotRow, error)

// 执行迁移
func (a *App) Migrate(remotePath string, selections []SlotSelection) ([]MigrateResult, error)
```

### 3.3 前端实现细节

#### 3.3.1 `App.vue` — 主布局
4 个面板，从上至下：
1. `<SteamPanel>` — 显示 Steam 路径、检测到的用户列表（radio 或 select）、手动选择按钮。
2. `<AndroidPanel>` — 两个按钮：「自动从手机获取」（调用 `AutoFetchAndroidDB`）、「手动选择 game.db」（调用 `PickAndroidDBManually`）。显示当前 game.db 路径与状态。
3. `<SlotTable>` — 仅当 game.db 路径有效时显示。表头：☑ | slotIndex | userAccount | JSON 大小 | JSON 预览。提供「全选/全不选」。
4. `<MigratePanel>` — 仅当 Steam 用户已选 且 至少一行 slot 已勾选 时启用「迁移」按钮。点击后显示进度与逐条结果日志。

#### 3.3.2 状态管理
- 不引入 Pinia，使用 Vue 3 Composition API + `provide/inject` 在 `App.vue` 中集中管理跨组件状态（Steam 信息、game.db 路径、选中的 slot 行）。

#### 3.3.3 UI 库选择
- 不引入完整 UI 框架（Element Plus 等），用原生 CSS + 少量自写组件，保持产物体积小、依赖少。
- 表格使用 `<table>` + 原生 checkbox，进度使用原生 `<progress>`。

#### 3.3.4 错误展示
- 所有后端调用错误以 toast / 行内提示方式展示。`AutoFetchAndroidDB` 失败时根据错误类型给出具体引导（开启 USB 调试 / 授权 PC / 应用未安装等）。

### 3.4 构建配置

#### 3.4.1 `wails.json`
```json
{
  "name": "wodima-slot-migrate",
  "outputfilename": "WodimaSlotMigrate",
  "frontend:install": "pnpm install",
  "frontend:build": "pnpm build",
  "frontend:dev:watcher": "pnpm dev",
  "frontend:dev:serverUrl": "auto",
  "author": "itaotuo"
}
```

#### 3.4.2 内嵌 ADB 的获取（实施前需完成）
1. 下载 `https://dl.google.com/android/repository/platform-tools-latest-windows.zip`
2. 解压后取 `platform-tools\adb.exe`、`AdbWinApi.dll`、`AdbWinUsbApi.dll`
3. 放入 `internal/android/embedded/` 目录
4. 在 `internal/android/adb.go` 顶部添加：
   ```go
   //go:embed embedded/adb.exe embedded/AdbWinApi.dll embedded/AdbWinUsbApi.dll
   var adbFS embed.FS
   ```

#### 3.4.3 `go.mod` 依赖
```
github.com/wailsapp/wails/v2 v2.x.x
modernc.org/sqlite v1.x.x
golang.org/x/sys
```

### 3.5 实施步骤（按顺序）

1. **初始化项目骨架**：`wails init -n wodima-slot-migrate -t vue-ts`，配置 `wails.json` 用 pnpm。
2. **下载并放入 ADB 二进制**：执行 3.4.2 步骤。
3. **实现 `internal/steam/steam.go`**：注册表读取 + userdata 遍历。
4. **实现 `internal/android/db.go`**：SQLite 只读查询。本地用 `game.db` 验证读取正确性。
5. **实现 `internal/android/adb.go`**：解压、设备枚举、pull。
6. **实现 `internal/migrate/migrate.go`**：备份与写入。
7. **实现 `app.go` 绑定方法**：组合上述模块。
8. **实现前端组件**：从 `SteamPanel` 到 `MigratePanel` 依次完成。
9. **本地联调**：用项目根 `game.db` + 一个测试 Steam userdata 目录验证完整流程。
10. **构建**：`wails build` 生成 `.exe`，检查产物体积。

## 四、Assumptions & Decisions（假设与决策）

### 关键假设
1. **`Slog` 是 `Slot` 的笔误**：用户原文 `Slog[012].json.[时间戳].bak`，按照上下文（PC 端文件名为 `Slot[012].json`）判断为笔误，统一使用 `Slot`。若理解有误请在评审时指出。
2. **安卓 `adb pull` 可访问 `Android/data/{pkg}/files/`**：基于 ADB 在 Android 11+ 仍有提升权限的常规认知。若实测发现部分 ROM 仍限制，需文档化「手动拷贝」退路。
3. **`jsonString` 直接写入 `Slot{X}.json`**：用户明确说明「`jsonString` 字段的内容对应着 Windows 下的 `Slot[012].json`」，不需要任何格式转换。
4. **同一 slotIndex 多行 userAccount**：UI 列出全部，用户勾选；若勾选同一 slotIndex 多行，后者覆盖前者（前端默认禁用同 slotIndex 多选，提示用户）。
5. **Steam userdata 数字子目录即 steamID**：符合 Steam 实际行为。
6. **不需要 UAC 提权**：Steam userdata 目录通常用户可写；如遇权限问题，文档化让用户检查 Steam 是否运行（占用文件）。

### 决策依据
- **纯 Go SQLite**：避免 CGO 编译复杂度，Wails 默认构建链不需要 mingw。
- **内嵌 ADB**：用户已选；体积增量约 5MB 可接受。
- **不引入 UI 框架**：界面简单，原生 CSS 即可，减少依赖体积与构建时间。
- **备份格式 `Slot{X}.json.{YYYYMMDD_HHMMSS}.bak`**：Windows 文件名不能含冒号，使用下划线分隔。

## 五、Verification（验证步骤）

### 5.1 单元/集成验证
1. **Steam 探测**：在用户机器运行 `DetectSteam()`，应返回 `E:\Program Files (x86)\Steam`。
2. **Steam 用户遍历**：调用 `DetectSteamUsers`，确认能找到包含 `3444020` 的 steamID。
3. **SQLite 读取**：对项目根 `game.db` 调用 `ReadSlots`，确认能解析出至少一行，字段值合理（jsonString 非空、可解析为 JSON）。
4. **ADB 解压**：首次启动应用后检查 `%LOCALAPPDATA%\wodima-migrate\adb\` 下 3 个文件存在，版本文件正确。
5. **迁移逻辑**：构造测试 remote 目录（含 `Slot0.json`），传入 mock 选中行，验证：
   - 原 `Slot0.json` 被重命名为 `Slot0.json.{timestamp}.bak`，文件名格式正确
   - 新 `Slot0.json` 内容等于 `jsonString`
   - 同一 remote 目录无 `Slot1.json` 时，直接创建新文件，无 `.bak` 产生

### 5.2 端到端验证
1. 启动应用（`wails dev`）。
2. Steam 面板自动显示路径与用户。
3. 手动选择项目根 `game.db` → 槽位表显示行。
4. 勾选一行 → 迁移按钮启用。
5. 点迁移 → 检查 Steam remote 目录：原文件已备份，新文件内容正确。
6. 打开游戏确认存档可加载（用户手动）。

### 5.3 产物体积验证
- `wails build` 后检查 `build\bin\` 下 `.exe` 体积应 ≤ 50MB（保守，含 ADB）。

## 六、风险与回退

| 风险 | 概率 | 回退方案 |
|---|---|---|
| 部分 ROM 限制 adb 访问 `Android/data/` | 低 | 文档化引导用户手动拷贝 |
| Steam 注册表被迁移工具改写（如 Portable Steam） | 低 | 同时检查 `HKLM\SOFTWARE\Valve\Steam`（非 WOW6432Node） |
| `modernc.org/sqlite` 性能不足 | 极低（3.5MB DB） | 切换 `mattn/go-sqlite3`（CGO） |
| Wails v2 在某些 Windows 10 LTSC WebView2 缺失 | 中 | 文档化安装 WebView2 Runtime（Wails 官方已知问题） |
