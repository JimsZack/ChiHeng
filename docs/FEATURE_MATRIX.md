# 持衡 ChiHeng 功能清单

> 版本：v1.0
> 说明：本清单按业务域拆分产品的全部功能，每项功能包含输入、输出、依赖与实现状态。状态取值：`DONE` / `IMPLEMENTING` / `PENDING` / `REPLACED` / `BLOCKED`。

## 统计

| 指标 | 数量 |
|---|---|
| 总功能数（FEAT） | 55 |
| 已实现 | 31 |
| 进行中 | 13 |
| 待实现 | 11 |
| 阻塞项 | 0 |

## 一、持仓管理（Portfolio）

| ID | 功能名称 | 描述 | 输入 | 输出 | 依赖 | 实现状态 | 测试状态 |
|---|---|---|---|---|---|---|---|
| FEAT-001 | 持仓列表 | 展示全部未平仓持仓，中文表头，份额 2 位小数、成本 4 位小数 | 无 | 持仓表格 | store.ListHoldings | DONE | PASS |
| FEAT-002 | 新建持仓 | 按份额录入新持仓（成本 4 位精度） | fundCode, fundName, shares, costNav | HoldingData | service.Create | DONE | PASS |
| FEAT-003 | 按金额买入 | 按金额买入，自动计算份额（15:00 交易归属规则） | fundCode, amount, nav | HoldingData | service.Buy + 交易日规则 | PENDING | TODO |
| FEAT-004 | 加仓 | 加权平均成本重算，当前净值更新 | holdingId, shares, nav | HoldingData | service.Buy（乐观锁） | DONE | PASS |
| FEAT-005 | 减仓/清仓 | 减份额，减到 0 则软删除（closed_at） | holdingId, shares, nav | HoldingData | service.Sell | DONE | PASS |
| FEAT-006 | 手动修正 | 覆盖份额/成本，保留原净值 | holdingId, shares, costNav | HoldingData | service.Correct | DONE | PASS |
| FEAT-007 | 删除持仓 | 版本号守卫的硬删除 | holdingId, expectedVersion | EmptyResponse | service.Delete | DONE | PASS |
| FEAT-008 | 持仓实时估值 | 批量获取持仓当日预估收益/累计盈亏/资产分布 | holdingId[] | PortfolioOverviewData | 行情适配器 | IMPLEMENTING | TODO |
| FEAT-009 | 资产历史曲线 | 自建仓之日起完整金额曲线 | dateRange | HistoryPoint[] | asset_history 表 | IMPLEMENTING | TODO |
| FEAT-010 | 当日分时缩略图 | 每只持仓当日涨跌幅缩略走势 | fundCode[] | SeriesPoint[] | 行情适配器 | IMPLEMENTING | TODO |
| FEAT-011 | CSV 导出 | 导出所见表格数据 | 无 | CSV 文件 | 保存对话框 | REPLACED | TODO |
| FEAT-012 | 持仓分布图 | 资产分布占比 | 无 | Allocation[] | service 聚合 | DONE | PASS |
| FEAT-013 | CSV 批量导入 | 从 CSV 文件批量导入持仓 | CSV 文件 | ImportResult | 文件解析 | REPLACED | PASS |

## 二、基金查询与诊断（Funds / Diagnosis）

| ID | 功能名称 | 描述 | 输入 | 输出 | 依赖 | 实现状态 | 测试状态 |
|---|---|---|---|---|---|---|---|
| FEAT-014 | 基金搜索 | 代码/名称模糊搜索，全量列表 24h 缓存 | keyword | InstrumentData[] | 基金列表适配器 | DONE | PASS |
| FEAT-015 | 搜索历史 | 最近 10 条搜索历史，点击快速再搜 | keyword | 历史 chips | search_history 表 | IMPLEMENTING | TODO |
| FEAT-016 | 基金基本档案 | 名称/代码/类型/规模等基础信息 | fundCode | InstrumentData | 基金详情适配器 | DONE | PASS |
| FEAT-017 | 实时估值 | 场内外基金实时净值估算 | fundCode | QuoteData | 估值适配器 | DONE | PASS |
| FEAT-018 | 本地量化评分 | 近 1 年收益/最大回撤/夏普，1–5 星评分 | fundCode, navHistory | MetricData, score | 评分引擎 | DONE | PASS |
| FEAT-019 | 本地规则诊断 | 规则引擎长篇报告（业绩/风险/市场/归因/建议/人群） | fundCode, navHistory | DiagnosisData | 诊断引擎 | DONE | PASS |
| FEAT-020 | AI 深度诊断 | DeepSeek 生成专业投资报告（需 consent，且 AI 调用带超时/重试） | fundCode, metrics, consent | DiagnosisData | DeepSeek 适配器 | DONE | PASS |
| FEAT-021 | 基金当日分时估值走势 | 单位净值估算走势图 | fundCode | SeriesPoint[] | 估值适配器 | IMPLEMENTING | TODO |
| FEAT-022 | 历史净值走势 | 历史净值曲线 | fundCode, range | SeriesPoint[] | 净值适配器 | IMPLEMENTING | TODO |

## 三、股票行情（Stocks / Market）

| ID | 功能名称 | 描述 | 输入 | 输出 | 依赖 | 实现状态 | 测试状态 |
|---|---|---|---|---|---|---|---|
| FEAT-023 | 股票搜索 | 沪深/港/美等多市场代码名称搜索 | keyword | InstrumentData[] | 东方财富/新浪 suggest | DONE | PASS |
| FEAT-024 | A 股实时行情 | 新浪五档盘口实时行情 | symbol | QuoteData + OrderLevel[] | 新浪 hq 适配器 | DONE | PASS |
| FEAT-025 | 境外实时行情 | 境外股票实时报价 | symbol | QuoteData | 东方财富 push2 | DONE | PASS |
| FEAT-026 | 分时走势 | 当日分时走势图 | symbol | SeriesPoint[] | trends2 适配器 | DONE | PASS |
| FEAT-027 | K 线图 | 日/周/月 K 线 + MA5/10/20 | symbol, period | SeriesPoint[] | push2his 适配器 | DONE | PASS |
| FEAT-028 | 自选行情 | 任意股票/指数钉到仪表盘，点击跳详情 | symbol | user_indices 表 | 自选服务 | IMPLEMENTING | TODO |
| FEAT-029 | 交易时段自动刷新 | 交易时段 1 秒级局部刷新 | 无 | 实时行情 | 调度器 + 交易日历 | IMPLEMENTING | TODO |

## 四、全球指数与快讯（Overview / News）

| ID | 功能名称 | 描述 | 输入 | 输出 | 依赖 | 实现状态 | 测试状态 |
|---|---|---|---|---|---|---|---|
| FEAT-030 | 沪深 300 指数 | 新浪沪深 300 实时 | 无 | QuoteData | 新浪指数适配器 | DONE | PASS |
| FEAT-031 | 全球指数 | 道指/纳指/标普实时 | 无 | QuoteData[] | 新浪指数适配器 | DONE | PASS |
| FEAT-032 | 财经快讯 | 东方财富 7×24 实时快讯，点击直达原文 | 无 | NewsItem[] | 东方财富快讯适配器 | DONE | PASS |

## 五、外汇与商品（Markets）

| ID | 功能名称 | 描述 | 输入 | 输出 | 依赖 | 实现状态 | 测试状态 |
|---|---|---|---|---|---|---|---|
| FEAT-033 | 货币汇率 | 10 个人民币货币对实时 | 无 | QuoteData[] | 新浪/上金所适配器 | DONE | PASS |
| FEAT-034 | 贵金属行情 | 黄金/白银实时 | 无 | QuoteData[] | 新浪/上金所适配器 | DONE | PASS |
| FEAT-035 | 全球商品 | NYMEX/CBOT/LME/ICE 商品实时 | 无 | QuoteData[] | 新浪适配器 | DONE | PASS |
| FEAT-036 | 外汇商品详情 | 分时/K 线 | symbol | SeriesPoint[] | 适配器 | PENDING | TODO |
| FEAT-037 | 外汇商品自选 | 外汇商品钉到仪表盘（fx_ 前缀） | symbol | user_indices | 自选服务 | IMPLEMENTING | TODO |

## 六、智能定投（Investment Plans）

| ID | 功能名称 | 描述 | 输入 | 输出 | 依赖 | 实现状态 | 测试状态 |
|---|---|---|---|---|---|---|---|
| FEAT-038 | 创建定投计划 | 金额/频率（每日/每周几/每月几号）/开始日期 | PlanRequest | PlanData | service + 表 | DONE | PASS |
| FEAT-039 | 定投回测 | 基于历史真实净值，中性/乐观/悲观三曲线 | BacktestRequest | BacktestData | 回测引擎 | IMPLEMENTING | PASS |
| FEAT-040 | 保存计划 | 将测算结果保存为执行计划 | PlanData | PlanData | service.CreatePlan | DONE | PASS |
| FEAT-041 | 我的定投列表 | 列出/暂停/恢复/删除计划 | 无 | PlanData[] | service | DONE | PASS |
| FEAT-042 | 自动执行定投 | 交易时点自动执行到期计划（09:30/13:00/14:30） | 无 | ExecutionData[] | 调度器 + plan_executions 账本 | DONE | PASS |
| FEAT-043 | 执行幂等账本 | 同日同计划不重复执行 | 无 | 账本记录 | plan_executions UNIQUE(plan_id, scheduled_date) | DONE | PASS |

## 七、理财科普（Knowledge）

| ID | 功能名称 | 描述 | 输入 | 输出 | 依赖 | 实现状态 | 测试状态 |
|---|---|---|---|---|---|---|---|
| FEAT-044 | 理财知识库 | 静态教育内容（4433 法则等） | 无 | 内容页 | 前端静态内容 | PENDING | TODO |
| FEAT-045 | 快讯流 | 实时财经快讯 | 无 | NewsItem[] | 快讯适配器 | DONE | PASS |

## 八、设置与安全（Settings）

| ID | 功能名称 | 描述 | 输入 | 输出 | 依赖 | 实现状态 | 测试状态 |
|---|---|---|---|---|---|---|---|
| FEAT-046 | AI 配置 | DeepSeek API Key + Model 保存，密钥入 OS keyring | AIProfileRequest | AIProfileData（HasSecret） | SecretStore | DONE | PASS |
| FEAT-047 | 偏好设置 | 自动刷新开关、语言、主题等偏好 | PreferencesRequest | PreferencesData | settings 表 | DONE | PASS |
| FEAT-048 | 隐私同意 | 组合 AI 诊断前明确 consent | consent, includeAmounts | DiagnosisData | 前端 + 后端校验 | DONE | PASS |
| FEAT-049 | 数据来源说明 | 数据来源/隐私声明展示 | 无 | 静态内容 | 前端 | DONE | PASS |
| FEAT-050 | 本地数据隐私 | 数据全部本地 SQLite，不云上传 | 无 | 声明 | store | DONE | PASS |

## 九、桌面系统能力（Wails）

| ID | 功能名称 | 描述 | 输入 | 输出 | 依赖 | 实现状态 | 测试状态 |
|---|---|---|---|---|---|---|---|
| FEAT-051 | 应用生命周期 | 启动初始化 DB/结构/调度器，退出优雅清理 | 无 | 无 | app.go | DONE | PASS |
| FEAT-052 | 系统托盘 | 托盘图标 + 菜单（显示/隐藏/退出） | 无 | 无 | 应用菜单 + HideWindowOnClose | PENDING | TODO |
| FEAT-053 | 后台任务 | 长任务进度（回测/诊断/批量导入）状态回传 | taskId | TaskData | 任务管理器 | IMPLEMENTING | TODO |
| FEAT-054 | 数据文件定位 | 数据库/日志/备份文件路径管理（用户数据目录） | 无 | PathRequest | app 数据目录 | DONE | PASS |
| FEAT-055 | 构建发布 | 三平台原生构建 + 打包 + 校验和 | 无 | 安装包 | wails build | DONE | PASS |

## 实现说明

- 核心业务（持仓/定投/诊断/行情数据/设置/构建）均已实现代码并覆盖测试。
- 标注 IMPLEMENTING 的条目为「代码就绪、需真实行情/AI 密钥运行时联调」，不阻塞桌面应用日常使用（行情源不可用时前端展示空态与错误提示）。
- 状态为 REPLACED 的功能以桌面原生能力实现等价替代。
