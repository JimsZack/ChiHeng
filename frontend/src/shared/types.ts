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

export type Holding = {
  readonly id: string;
  readonly code: string;
  readonly name: string;
  readonly shares: string;
  readonly costNav: string;
  readonly currentNav: string;
  readonly marketValue: string;
  readonly dailyChange: string;
  readonly profit: string;
};

export type Instrument = {
  readonly id: string;
  readonly name: string;
  readonly code: string;
  readonly kind: "stock" | "index" | "fund" | "fx" | "metal" | "commodity";
  readonly price: string;
  readonly change: string;
};

export type Plan = {
  readonly id: string;
  readonly fund: string;
  readonly cadence: string;
  readonly amount: string;
  readonly nextDate: string;
  readonly status: "active" | "paused";
  readonly frequency?: string;
  readonly executionDay?: number;
};

export type NewsItem = {
  readonly id: string;
  readonly time: string;
  readonly tag: string;
  readonly title: string;
  readonly url: string;
};

export type PreviewData = {
  readonly metrics: readonly Metric[];
  readonly holdings: readonly Holding[];
  readonly instruments: readonly Instrument[];
  readonly plans: readonly Plan[];
  readonly news: readonly NewsItem[];
};
