import { create } from "zustand";
import * as api from "../services/api";
import type { Holding, Instrument, LoadState, NewsItem, Plan, Quote } from "../shared/types";

// ---- 持仓 store ----

type HoldingsState = {
  state: LoadState;
  holdings: Holding[];
  error: string | null;
  load: () => Promise<void>;
  add: (input: {
    fundCode: string;
    fundName: string;
    shares: string;
    costNav: string;
    openedOn: string;
  }) => Promise<void>;
  buy: (id: string, shares: string, nav: string, version: number) => Promise<void>;
  sell: (id: string, shares: string, nav: string, version: number) => Promise<void>;
  remove: (id: string, version: number) => Promise<void>;
};

export const useHoldingsStore = create<HoldingsState>((set) => ({
  state: "loading",
  holdings: [],
  error: null,
  load: async () => {
    set({ state: "loading", error: null });
    try {
      const holdings = await api.listHoldings();
      set({ state: holdings.length === 0 ? "empty" : "ready", holdings });
    } catch (err) {
      set({ state: "error", error: err instanceof Error ? err.message : "加载失败" });
    }
  },
  add: async (input) => {
    await api.createHolding(input);
    await useHoldingsStore.getState().load();
  },
  buy: async (id, shares, nav, version) => {
    await api.buyHolding({ id, shares, nav, expectedVersion: version });
    await useHoldingsStore.getState().load();
  },
  sell: async (id, shares, nav, version) => {
    await api.sellHolding({ id, shares, nav, expectedVersion: version });
    await useHoldingsStore.getState().load();
  },
  remove: async (id, version) => {
    await api.deleteHolding(id, version);
    await useHoldingsStore.getState().load();
  },
}));

// ---- 定投 store ----

type PlansState = {
  state: LoadState;
  plans: Plan[];
  error: string | null;
  load: () => Promise<void>;
  create: (input: {
    fundCode: string;
    fundName: string;
    amount: string;
    frequency: string;
    executionDay: number;
    startDate: string;
  }) => Promise<void>;
  pause: (id: string, version: number) => Promise<void>;
  resume: (id: string, version: number) => Promise<void>;
  remove: (id: string, version: number) => Promise<void>;
};

export const usePlansStore = create<PlansState>((set) => ({
  state: "loading",
  plans: [],
  error: null,
  load: async () => {
    set({ state: "loading", error: null });
    try {
      const plans = await api.listPlans();
      set({ state: plans.length === 0 ? "empty" : "ready", plans });
    } catch (err) {
      set({ state: "error", error: err instanceof Error ? err.message : "加载失败" });
    }
  },
  create: async (input) => {
    await api.createPlan(input);
    await usePlansStore.getState().load();
  },
  pause: async (id, version) => {
    await api.pausePlan(id, version);
    await usePlansStore.getState().load();
  },
  resume: async (id, version) => {
    await api.resumePlan(id, version);
    await usePlansStore.getState().load();
  },
  remove: async (id, version) => {
    await api.deletePlan(id, version);
    await usePlansStore.getState().load();
  },
}));

// ---- 行情 store ----

type MarketState = {
  state: LoadState;
  indices: Quote[];
  news: NewsItem[];
  error: string | null;
  load: () => Promise<void>;
  search: (keyword: string, assetType?: string) => Promise<Instrument[]>;
};

export const useMarketStore = create<MarketState>((set) => ({
  state: "loading",
  indices: [],
  news: [],
  error: null,
  load: async () => {
    set({ state: "loading", error: null });
    try {
      const [indices, news] = await Promise.all([api.getGlobalIndices(), api.getNews()]);
      set({ state: "ready", indices, news });
    } catch (err) {
      set({ state: "error", error: err instanceof Error ? err.message : "加载失败" });
    }
  },
  search: async (keyword, assetType = "") => api.searchInstruments(keyword, assetType),
}));

// ---- 应用信息 ----

type AppState = {
  info: { name: string; version: string; tagline: string };
  wailsMode: boolean;
  loadInfo: () => Promise<void>;
};

export const useAppStore = create<AppState>((set) => ({
  info: { name: "持衡 ChiHeng", version: "0.1.0", tagline: "看清持仓，理性权衡。" },
  wailsMode: api.isWailsMode(),
  loadInfo: async () => {
    const info = await api.getAppInfo();
    set({ info, wailsMode: api.isWailsMode() });
  },
}));
