# 持衡 ChiHeng

本地优先的个人投资组合与行情分析桌面应用。看清持仓，理性权衡。

持衡基于 Wails v2 构建：Go 负责核心业务、数据存储与系统能力，前端使用 React 与 TypeScript 提供专业、克制的桌面界面。所有数据均保存在本地 SQLite 数据库中，不上传云端。

## 核心功能

- 持仓管理：持仓录入、加减仓、手动修正、删除，加权平均成本计算，支持 4 位小数成本精度，带乐观锁并发保护。
- 智能定投：按日/周/月创建定投计划，基于历史净值回测（中性/乐观/悲观三曲线），交易时点自动执行并写入执行账本，杜绝重复执行。
- 基金诊断：基金搜索、实时估值、本地量化评分（收益/回撤/夏普）、规则诊断报告，可选接入 DeepSeek 生成 AI 深度报告（需用户明确同意）。
- 行情分析：A 股/境外股票实时行情与五档盘口、分时与 K 线（日/周/月）、全球指数、外汇与贵金属/商品行情。
- 总览仪表盘：总资产、当日收益、累计盈亏、资产历史曲线、持仓分布、自选行情、财经快讯。
- 理财科普：投资知识库与实时财经快讯流。
- 数据安全：密钥存入操作系统钥匙串，不落明文；数据全程本地保存并自动备份。

## 技术架构

| 层 | 技术 |
|---|---|
| 桌面框架 | Wails v2（Go 1.25） |
| 后端 | Go：Service / Repository / Domain 分层，SQLite（modernc.org/sqlite，纯 Go 无 CGO） |
| 前端 | React 19 + TypeScript + Vite，ECharts 图表，Phosphor 图标 |
| 构建 | Wails CLI，三平台原生打包 |

架构细节见 [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)，功能映射见 [docs/FEATURE_MATRIX.md](docs/FEATURE_MATRIX.md)。

## 开发环境

- Go 1.25 及以上
- Node.js 22、pnpm 10
- Wails v2 CLI（`go install github.com/wailsapp/wails/v2/cmd/wails@latest`）
- 平台依赖：Windows 需 WebView2；macOS 需 Xcode Command Line Tools；Linux 需 `libgtk-3-dev` 与 `libwebkit2gtk-4.1-dev`

## 快速开始

```bash
# 安装依赖
go mod download
pnpm --dir frontend install --frozen-lockfile

# 开发模式（热重载）
wails dev

# 全量质量门禁（测试、lint、类型检查、构建）
./scripts/build-all.sh

# 打包当前平台
wails build -clean
```

产物输出至 `build/bin/`。

## 测试

```bash
go test -race ./internal/...
pnpm --dir frontend test
```

## 发布

三平台原生构建、签名与校验和规范见 [docs/BUILDING.md](docs/BUILDING.md)。发布产物必须通过 `./scripts/build-all.sh` 全部检查。

## 数据与隐私

- 数据库、日志、备份均位于系统用户配置目录下的 `chiheng` 文件夹。
- 持仓与配置绝不云端上传；AI 诊断仅在用户明确同意后才将脱敏指标发送至所选模型服务商。

## 免责声明

本软件提供的数据与分析结果仅供参考，不构成任何投资建议。


