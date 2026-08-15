// 前后端契约类型：字段名与后端 bindings DTO（internal/bindings/*_dto.go）一一对应。
// 新增字段时以 Go 侧 json tag 为准，勿另起名。

export const routeIds = [
  "overview",
  "stocks",
  "funds",
  "markets",
  "holdings",
  "plans",
  "knowledge",
] as const;

export type RouteId = (typeof routeIds)[number];
export type Freshness = "live" | "stale" | "offline";
export type LoadState = "loading" | "ready" | "empty" | "error";

export type Metric = {
  readonly label: string;
  readonly value: string;
  readonly comparison: string;
  readonly trend: "positive" | "negative" | "neutral";
};

// 对应 bindings.HoldingData
export type Holding = {
  readonly id: string;
  readonly fundCode: string;
  readonly fundName: string;
  readonly shares: string;
  readonly costNav: string;
  readonly currentNav: string;
  readonly marketValue: string;
  readonly dailyChange: string;
  readonly dailyPnl: string;
  readonly cumulativePnl: string;
  readonly openedOn: string;
  readonly version: number;
};

// 对应 bindings.InstrumentData（搜索结果，不含报价）
export type Instrument = {
  readonly id: string;
  readonly code: string;
  readonly name: string;
  readonly assetType: string;
  readonly market: string;
  readonly providerId: string;
};

// 对应 bindings.QuoteData（报价：嵌套 instrument + 价格/涨跌）
export type Quote = {
  readonly instrument: Instrument;
  readonly price: string;
  readonly change: string;
  readonly open: string;
  readonly high: string;
  readonly low: string;
  readonly previousClose: string;
  readonly volume: string;
  readonly amount: string;
};

// 对应 bindings.PlanData
export type Plan = {
  readonly id: string;
  readonly fundCode: string;
  readonly fundName: string;
  readonly amount: string;
  readonly frequency: string;
  readonly executionDay: number;
  readonly startDate: string;
  readonly status: "active" | "paused";
  readonly version: number;
};

// 对应 bindings.NewsItem
export type NewsItem = {
  readonly id: string;
  readonly title: string;
  readonly publishedAt: string;
  readonly source: string;
  readonly url: string;
  readonly tag?: string;
};

export type PreviewData = {
  readonly metrics: readonly Metric[];
  readonly holdings: readonly Holding[];
  readonly instruments: readonly Instrument[];
  readonly quotes: readonly Quote[];
  readonly plans: readonly Plan[];
  readonly news: readonly NewsItem[];
};
