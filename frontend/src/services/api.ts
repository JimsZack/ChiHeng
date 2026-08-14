// API 层：统一访问 Go 后端 Bindings，非 Wails 环境回退到预览数据。
// Wails v2 生成的绑定形如 window.go.main.App.<Method>(args...)，返回 Promise。

import type { Holding, Instrument, LoadState, Metric, NewsItem, Plan } from "../shared/types";
import { previewData } from "./preview-data";

type WailsApp = {
  [method: string]: (...args: unknown[]) => Promise<unknown>;
};

type ResponseEnvelope<T> = {
  meta?: { requestId?: string; servedAt?: string; stale?: boolean };
  data: T | null;
  error?: { code?: string; message?: string } | null;
};

const meta = (): { requestId: string; locale: string } => ({
  requestId: crypto.randomUUID(),
  locale: "zh-CN",
});

const getApp = (): WailsApp | null => {
  const app = (globalThis as { go?: { main?: { App?: WailsApp } } }).go?.main?.App;
  return app && typeof app === "object" ? app : null;
};

export const isWailsMode = (): boolean => getApp() !== null;

// 调用绑定：成功返回 data，绑定不可用或信封带 error 时抛出。
const call = async <T>(method: string, ...args: unknown[]): Promise<T> => {
  const app = getApp();
  if (!app || typeof app[method] !== "function") {
    throw new Error(`wails-binding-unavailable:${method}`);
  }
  const envelope = (await app[method](meta(), ...args)) as ResponseEnvelope<T>;
  if (envelope?.error) {
    throw new Error(`${envelope.error.message || envelope.error.code || "unknown-error"}`);
  }
  return envelope?.data as T;
};

const withFallback = async <T>(method: string, fallback: T, args: unknown[] = []): Promise<T> => {
  try {
    return await call<T>(method, ...args);
  } catch {
    return fallback;
  }
};

// 预览数据映射
const metricOf = (m: Metric): Metric => m;

// ---- 持仓 ----

export const listHoldings = (): Promise<Holding[]> =>
  withFallback("ListHoldings", previewData.holdings as unknown as Holding[]);

export const createHolding = (input: {
  fundCode: string;
  fundName: string;
  shares: string;
  costNav: string;
  openedOn: string;
}): Promise<Holding> =>
  call<Holding>("CreateHolding", input).catch(() => {
    throw new Error("创建持仓失败");
  });

export const buyHolding = (input: {
  id: string;
  shares: string;
  nav: string;
  expectedVersion: number;
}): Promise<Holding> =>
  call<Holding>("BuyHolding", input).catch(() => {
    throw new Error("加仓失败");
  });

export const sellHolding = (input: {
  id: string;
  shares: string;
  nav: string;
  expectedVersion: number;
}): Promise<Holding> =>
  call<Holding>("SellHolding", input).catch(() => {
    throw new Error("减仓失败");
  });

export const deleteHolding = (id: string, expectedVersion: number): Promise<void> =>
  call<void>("DeleteHolding", { id, expectedVersion }).catch(() => {
    throw new Error("删除失败");
  });

export const exportHoldingsCSV = (): Promise<{ status: string; phase: string }> =>
  call<{ status: string; phase: string }>("ExportHoldingsCSV").catch(() => {
    throw new Error("导出失败，请检查保存路径");
  });

export const importHoldingsCSV = (): Promise<{ status: string; phase: string }> =>
  call<{ status: string; phase: string }>("ImportHoldingsCSV").catch(() => {
    throw new Error("导入失败，请检查 CSV 文件格式");
  });

export const assetHistory = (): Promise<{ date: string; value: string }[]> =>
  withFallback("AssetHistory", [
    { date: "2025-09", value: "241201.58" },
    { date: "2025-12", value: "253812.90" },
    { date: "2026-03", value: "261004.37" },
    { date: "2026-06", value: "273129.65" },
    { date: "2026-08", value: "286420.36" },
  ]);

// ---- 定投 ----

export const listPlans = (): Promise<Plan[]> =>
  withFallback("ListPlans", previewData.plans as unknown as Plan[]);

export const createPlan = (input: {
  fundCode: string;
  fundName: string;
  amount: string;
  frequency: string;
  executionDay: number;
  startDate: string;
}): Promise<Plan> =>
  call<Plan>("CreatePlan", input).catch(() => {
    throw new Error("创建定投计划失败");
  });

export const pausePlan = (id: string, expectedVersion: number): Promise<Plan> =>
  call<Plan>("PausePlan", { id, expectedVersion }).catch(() => {
    throw new Error("暂停计划失败");
  });

export const resumePlan = (id: string, expectedVersion: number): Promise<Plan> =>
  call<Plan>("ResumePlan", { id, expectedVersion }).catch(() => {
    throw new Error("恢复计划失败");
  });

export const deletePlan = (id: string, expectedVersion: number): Promise<void> =>
  call<void>("DeletePlan", { id, expectedVersion }).catch(() => {
    throw new Error("删除计划失败");
  });

export type BacktestResult = {
  invested: string;
  currentValue: string;
  yieldRate: string;
  neutral: { date: string; value: string }[];
  optimistic: { date: string; value: string }[];
  pessimistic: { date: string; value: string }[];
};

export const backtestPlan = (input: {
  fundCode: string;
  amount: string;
  frequency: string;
  executionDay: number;
  years: number;
}): Promise<BacktestResult> =>
  call<BacktestResult>("Backtest", input).catch(() => {
    throw new Error("回测失败，请检查基金代码或稍后重试");
  });

// ---- 行情 ----

export const searchInstruments = (keyword: string, assetType = ""): Promise<Instrument[]> =>
  withFallback("SearchInstruments", previewData.instruments as unknown as Instrument[], [
    { keyword, assetType, limit: 20 },
  ]);

export const getGlobalIndices = (): Promise<Instrument[]> =>
  withFallback("GetGlobalIndices", previewData.instruments.slice(0, 3) as unknown as Instrument[]);

export const getNews = (): Promise<NewsItem[]> =>
  withFallback("GetNews", previewData.news as unknown as NewsItem[]);

export const getQuote = (instrumentId: string): Promise<Instrument> =>
  withFallback("GetQuote", previewData.instruments[0] as unknown as Instrument, [{ instrumentId }]);

export const getUserIndices = (): Promise<Instrument[]> =>
  withFallback("GetUserIndices", previewData.instruments.slice(0, 2) as unknown as Instrument[]);

export const addUserIndex = (instrumentId: string): Promise<void> =>
  call<void>("AddUserIndex", { instrumentId }).catch(() => {
    throw new Error("添加自选失败");
  });

export const removeUserIndex = (instrumentId: string): Promise<void> =>
  call<void>("RemoveUserIndex", { instrumentId }).catch(() => {
    throw new Error("移除自选失败");
  });

export const getSearchHistory = (): Promise<Instrument[]> =>
  withFallback("GetSearchHistory", [
    { id: "h1", name: "161725", code: "161725", kind: "fund", price: "", change: "" },
    { id: "h2", name: "600519", code: "600519", kind: "stock", price: "", change: "" },
  ]);

// ---- 诊断 ----

export const diagnoseFund = (instrumentId: string): Promise<unknown> =>
  withFallback("DiagnoseFund", { score: 4, conclusion: "预览诊断结果" }, [{ instrumentId }]);

// ---- 设置 ----

export const getAppInfo = (): Promise<{ name: string; version: string; tagline: string }> =>
  withFallback("GetAppInfo", {
    name: "持衡 ChiHeng",
    version: "0.1.0",
    tagline: "看清持仓，理性权衡。",
  });

export type Preferences = {
  locale: string;
  theme: string;
  refreshIntervalSeconds: number;
  disclaimerVersion: string;
  logLevel: string;
};

export const getPreferences = (): Promise<Preferences> =>
  withFallback("GetPreferences", {
    locale: "zh-CN",
    theme: "system",
    refreshIntervalSeconds: 60,
    disclaimerVersion: "1.0",
    logLevel: "info",
  });

export const savePreferences = (input: Preferences): Promise<Preferences> =>
  call<Preferences>("SavePreferences", input).catch(() => {
    throw new Error("保存偏好失败");
  });

export type AIProfile = {
  provider: string;
  model: string;
  hasSecret: boolean;
  available: boolean;
  unavailableReason: string;
};

export const getAIProfile = (): Promise<AIProfile> =>
  withFallback("GetAIProfile", {
    provider: "deepseek",
    model: "deepseek-chat",
    hasSecret: false,
    available: false,
    unavailableReason: "预览模式",
  });

export const saveAIProfile = (input: {
  provider: string;
  model: string;
  apiKey: string;
}): Promise<AIProfile> =>
  call<AIProfile>("SaveAIProfile", input).catch(() => {
    throw new Error("保存 AI 配置失败");
  });

export type Load<T> = { state: LoadState; data: T };

export { metricOf };
