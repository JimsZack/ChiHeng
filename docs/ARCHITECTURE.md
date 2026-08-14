# Wails 目标架构设计：持衡 ChiHeng Desktop

> 版本：v1.0
> 依据：docs/FEATURE_MATRIX.md（56 个 FEAT）、内部代码分析报告、Wails v2 官方 API。
> 技术栈：Go 1.25（Wails v2）+ React 19 + TypeScript + Vite（前端保持现状，避免无谓重写）+ modernc.org/sqlite。

## 1. 总体架构

```
┌─────────────────────────────────────────────────────┐
│                    Wails Desktop                    │
│  ┌───────────────────────────────────────────────┐  │
│  │  frontend/  React 19 + TS + Vite (webview)    │  │
│  │  pages · components · stores · services       │  │
│  │  wailsjs/runtime (Events) · wailsjs/bindings  │  │
│  └───────────────┬───────────────────────────────┘  │
│                  │ Bindings (JSON) + Events          │
│  ┌───────────────▼───────────────────────────────┐  │
│  │  app.go  App (Wails 绑定对象)                  │  │
│  │  onStartup / onShutdown / 方法=Service 调用    │  │
│  ├───────────────────────────────────────────────┤  │
│  │  internal/service  业务层（六域）               │  │
│  │  portfolio · investment · market ·            │  │
│  │  diagnosis · settings · task                  │  │
│  ├───────────────────────────────────────────────┤  │
│  │  internal/adapter  外部数据适配器               │  │
│  │  sina · eastmoney · fundgz · xueqiu · deepseek│  │
│  ├───────────────────────────────────────────────┤  │
│  │  internal/store  SQLite 数据层（WAL/单连接）    │  │
│  │  internal/domain 领域模型 + 定点货币           │  │
│  │  internal/schema 数据库结构与版本管理         │  │
│  └───────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────┘
```

## 2. 目录结构（实际方案，基于现有代码演化）

```
chiheng/
├── main.go                  # Wails 入口：打开 DB → 初始化结构 → 装配 → wails.Run
├── app.go                   # App 结构体：生命周期 + 六个域的方法绑定
├── wails.json               # Wails 项目配置（app name=ChiHeng）
├── Taskfile.yml             # wails 任务（可选）
├── internal/
│   ├── app/                 # 装配与依赖注入（config/db/services 组装）
│   ├── bindings/            # DTO 契约（现有，保留并补全）
│   ├── domain/              # 领域模型 + money（现有，保留）
│   ├── store/               # SQLite 数据层（扩展 8 张表方法）
│   ├── schema/              # 数据库结构与版本管理
│   ├── service/             # 业务层（现有 portfolio + 新增五域）
│   ├── adapter/             # 外部数据适配器（新增）
│   │   ├── sina/            # 新浪行情（A股五档/指数/外汇/商品）
│   │   ├── eastmoney/       # 东方财富（suggest/报价/K线/快讯）
│   │   ├── fundgz/          # 天天基金实时估值
│   │   ├── xueqiu/          # 雪球基金详情（可选）
│   │   └── deepseek/        # DeepSeek AI 诊断（OpenAI 兼容）
│   ├── scheduler/           # 定投调度器（新增，plan_executions 账本消费方）
│   └── config/              # 配置管理（settings 表 + SecretStore keyring）
├── frontend/                # React 19 + TS + Vite（现有，接入 bindings）
│   └── src/
│       ├── wailsjs/         # wails generate 产物（bindings + runtime）
│       ├── pages/ components/ stores/ services/ shared/ assets/ styles/
├── build/                   # Wails 平台资源（现有 + 补齐 Info.plist/info.json）
├── scripts/                 # 构建脚本（现有 build-all.sh / quality-gate.sh）
└── docs/                    # 文档（FEATURE_MATRIX / ARCHITECTURE / BUILDING）
```

## 3. Wails 生命周期

| 阶段 | 实现 |
|---|---|
| 程序启动前 | `main()`：解析配置 → `store.Open`（用户数据目录）→ `schema.EnsureSchema` → 装配 services/scheduler |
| `onStartup(ctx)` | 保存 ctx；加载偏好；启动定投调度器；恢复托盘 |
| 运行期 | 前端经 Bindings 调 App 方法；长任务进度经 `runtime.EventsEmit` 推送 |
| `onShutdown` | 停止调度器；关闭 DB；刷新托盘状态 |

```go
// main.go
func main() {
    cfg := config.Load()                     // 用户数据目录: os.UserConfigDir()/chiheng
    db, _ := store.Open(cfg.DBPath)
    schema.EnsureSchema(context.Background(), db.DB())
    app := app.NewApp(cfg, db, services...)  // 或直接 NewApp(cfg, db)
    wails.Run(&options.App{
        Title: "持衡 ChiHeng",
        Width: 1280, Height: 800, MinWidth: 1024, MinHeight: 700,
        AssetServer: &assetserver.Options{Assets: assets},
        Bind: []interface{}{app},
        OnStartup: app.onStartup, OnShutdown: app.onShutdown,
        Systray: 托盘菜单,
        Windows: &windows.Options{...},
        Mac: &mac.Options{...},
    })
}
```

## 4. 前后端通信方式

- **Bindings（主通道）**：`App` 结构体方法按六域分组（`Portfolio`、`Investment`、`Market`、`Diagnosis`、`Settings`、`System`），前端 `window.go.main.App.Xxx()`（wailsjs 生成）。请求/响应统一走现有 `bindings` 信封（`ResponseMeta`/`APIError`），金额字段为字符串（micros/cents 精确传递）。
- **Events（异步进度）**：`runtime.EventsEmit(ctx, "task:progress", TaskData)` → 前端 `Events.On("task:progress")`。用于回测、诊断、批量导入、快讯推送。
- **对话框/系统能力**：`runtime.OpenFileDialog/SaveFileDialog`（CSV 导入导出）、`runtime.WindowSetTitle` 等。
- **本地行情轮询**：前端定时调 `Market.GetQuote`（交易时段 1s），由 Go 侧适配器做短缓存（避免高频打外部 API）。

## 5. Service 层设计（六域）

| Service | 职责 | 关键方法（均为 App 绑定入口） |
|---|---|---|
| `Portfolio`（现有） | 持仓 CRUD/加减仓/修正 | List/Create/Buy/Sell/Correct/Delete/Overview/History |
| `Investment`（新增） | 定投计划/回测/执行账本 | CreatePlan/ListPlans/DeletePlan/Backtest/ListExecutions/ExecuteDue |
| `Market`（新增） | 行情聚合与搜索 | Search/SearchStocks/SearchFunds/Quote/Series/OrderBook/News/Rates/Metals/Commodities |
| `Diagnosis`（新增） | 本地评分 + 规则诊断 + AI 诊断 | DiagnoseFund/DiagnosePortfolio（校验 consent）/AIProfile 联动 |
| `Settings`（新增） | 偏好 + AI 配置 + 隐私 | GetPreferences/SavePreferences/GetAIProfile/SaveAIProfile/GetPrivacy |
| `Task`（新增） | 后台任务管理 | Submit/Status/Cancel（配合 TaskData 事件） |

设计约束：Service 只依赖 store/domain/adapter，不依赖 Wails runtime；App 层做 DTO 转换（domain → bindings）与错误映射（domain.Err → APIError）。

## 6. 数据层设计（store）

- 单连接 + WAL + busy_timeout（现有 `store.Open` 已具备）。
- 表：现有 6 张（holdings/investment_plans/plan_executions/settings/schema_versions）+ 新增 `user_indices`、`search_history`、`asset_history`、`intraday_ticks`、`fund_daily_performance`（金额列改定点整数）。
- 金额规范：份额/净值 micros（scale=1e6）、定投金额 cents（scale=100），一律 int64，DTO 层字符串化。
- 事务：`plan_executions` 唯一约束 `(plan_id, scheduled_date)` 保证执行幂等；写操作带乐观锁版本号。

## 7. 配置管理

- `settings` 表存偏好（自动刷新、语言、主题）。
- 密钥（DeepSeek API Key）走 `SecretStore` 接口：默认 OS keyring（`github.com/zalando/go-keyring` 或 `keyring`），桌面环境不可用时回退加密文件（0600）。DTO 只回传 `HasSecret`。
- `config` 包：用户数据目录、DB 路径、日志路径、适配器超时/缓存时长集中定义。

## 8. 日志系统

- Go 侧：`log/slog`（Go 1.21+），TextHandler 输出到用户数据目录 `logs/chiheng.log` + 控制台，分级 DEBUG/INFO/WARN/ERROR。
- 禁止在日志中输出 API Key、持仓金额明细等敏感数据。
- 前端：`console.*`，错误经 bindings 信封返回，不落 emoji、不带明文密钥。

## 9. 错误处理机制

- 分层：adapter（网络/解析错误，标记 retryable）→ service（领域错误，`domain.ErrInvalidDecimal`/`store.ErrConflict` 等）→ App 层映射为 `bindings.APIError{Code, Message, Field, Retryable}`。
- 错误码规范：`VALIDATION` / `NOT_FOUND` / `CONFLICT` / `RATE_LIMITED` / `UPSTREAM` / `UNAVAILABLE` / `AUTH_REQUIRED` / `INTERNAL`。
- 前端统一错误处理：`useQuery` 错误状态 + 信封 error 渲染，Loading/Empty/Error/Success 四态组件。

## 10. 异步任务机制（Task）

- `TaskManager`：submit → goroutine 执行 → `EventsEmit("task:progress", TaskData{TaskID, Kind, Status, Progress, Phase})` → 完成事件 `task:done`。
- 场景：定投回测、AI 诊断、批量导入。
- 执行带 ctx 取消，支持 Cancel；状态存内存 map + 可选持久化。

## 11. 系统托盘与窗口管理

- 托盘：`options.App.Systray` + 菜单（显示主窗口 / 隐藏 / 退出）。图标用 `build/tray/chiheng-tray-*.png`（黑白两套），macOS 用 template 图标。
- 窗口：标题「持衡 ChiHeng」；关闭行为：macOS 常规，Windows/Linux 关闭时最小化到托盘（可配置）。
- 窗口状态（尺寸/位置）持久化到 settings。

## 12. 文件操作与系统 API

- 数据文件：`os.UserConfigDir()/chiheng/chiheng.db`（Linux: ~/.config/chiheng，macOS: ~/Library/Application Support/chiheng，Windows: %AppData%\chiheng）。
- 对话框：`runtime.OpenFileDialog`（导入 CSV）、`runtime.SaveFileDialog`（导出 CSV）。
- 打开外链：`runtime.BrowserOpenURL`（快讯原文、帮助文档）。

## 13. 构建与发布机制

- `wails.json`：`{"name":"chiheng","outputfilename":"ChiHeng","frontend:install":"pnpm install","frontend:build":"pnpm run build","frontend:dir":"frontend"}`。
- `go:embed all:frontend/dist` 内嵌前端产物（main.go 现有注释已预写）。
- `build/` 补齐：`darwin/Info.plist` + `Info.dev.plist`（CFBundleName=ChiHeng、版本）、`windows/info.json`（名字 ChiHeng、图标）。
- `scripts/build-all.sh`（现有）：quality-gate → go test → vet → 前端 check → `wails build -clean`。
- 三平台产物：`build/bin/ChiHeng.exe` / `ChiHeng.app` / `chiheng`；文档按 `docs/BUILDING.md` 规范（签名/公证/SHA-256）。
- 命名统一：模块 `github.com/chiheng-app/chiheng`、包名 `chiheng`、二进制 `ChiHeng`、应用名「持衡 ChiHeng」，清除 `com.followplane` 残留。

## 14. 关键决策记录

| 决策 | 选择 | 理由 |
|---|---|---|
| 前端框架 | 保持 React 19 + TS + Vite | 已有 7 页完整 UI 与设计系统，重写 Vue 无收益且违反"复用成熟逻辑" |
| 数据库 | modernc.org/sqlite（现有） | 纯 Go、无 CGO，跨平台打包简单 |
| UI 组件 | 手写 CSS + Phosphor 图标（现状） | DESIGN.md 已定义完整 Design System，组件化封装即可 |
| 图表 | ECharts（已装未用） | 替代手写 MiniChart，满足 K 线/分时/面积图专业需求 |
| 密钥存储 | OS keyring + 加密文件回退 | 修复明文存储缺陷 |
| 外部行情 | Go 原生 HTTP 适配器（sina/eastmoney/fundgz） | 原生实现，避免进程内嵌外部运行时 |
| 定投执行 | Go 调度器 + plan_executions 账本 | 修复重复执行致命缺陷 |
| AI 诊断 | DeepSeek（OpenAI 兼容协议），强制 consent | 修复无同意外发持仓的隐私缺陷 |
