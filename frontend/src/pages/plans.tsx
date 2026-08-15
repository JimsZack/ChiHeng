import { Calculator, Pause, Play, Plus, Trash } from "@phosphor-icons/react";
import { useEffect, useState } from "react";
import { MiniChart, PageHeader, PreviewNotice, StateSwitch } from "../components/ui";
import { type BacktestResult, backtestPlan } from "../services/api";
import { useAppStore, usePlansStore } from "../stores/app";

const frequencyLabels: Record<string, string> = {
  monthly: "每月",
  weekly: "每周",
  daily: "每日",
};

const cadenceLabel = (frequency: string, executionDay: number): string => {
  switch (frequency) {
    case "daily":
      return "每日";
    case "weekly":
      return `每周${["一", "二", "三", "四", "五", "六", "日"][executionDay - 1] ?? ""}`;
    case "monthly":
      return `每月 ${executionDay} 日`;
    default:
      return "每月";
  }
};

export const PlansPage = () => {
  const { plans, state, load, create, pause, resume, remove } = usePlansStore();
  const { wailsMode } = useAppStore();
  const [form, setForm] = useState({
    fundCode: "",
    fundName: "",
    amount: "1000.00",
    frequency: "monthly",
    executionDay: "15",
    startDate: new Date().toISOString().slice(0, 10),
  });
  const [years, setYears] = useState("3");
  const [backtest, setBacktest] = useState<BacktestResult | null>(null);
  const [backtestLoading, setBacktestLoading] = useState(false);
  const [backtestError, setBacktestError] = useState<string | null>(null);

  useEffect(() => {
    void load();
  }, [load]);

  const savePlan = async () => {
    if (!form.fundCode || !form.amount) {
      return;
    }
    await create({
      fundCode: form.fundCode,
      fundName: form.fundName || form.fundCode,
      amount: form.amount,
      frequency: form.frequency,
      executionDay: Number(form.executionDay),
      startDate: form.startDate,
    });
  };

  const runBacktest = async () => {
    if (!form.fundCode || !form.amount) {
      return;
    }
    setBacktestLoading(true);
    setBacktestError(null);
    try {
      const result = await backtestPlan({
        fundCode: form.fundCode,
        amount: form.amount,
        frequency: form.frequency,
        executionDay: Number(form.executionDay),
        years: Number(years),
      });
      setBacktest(result);
    } catch (err) {
      setBacktestError(err instanceof Error ? err.message : "回测失败");
    } finally {
      setBacktestLoading(false);
    }
  };

  const chartValues = backtest
    ? backtest.neutral.map((p) => {
        const max = Math.max(
          ...backtest.neutral.map((n) => Number(n.value)),
          ...backtest.optimistic.map((n) => Number(n.value)),
          ...backtest.pessimistic.map((n) => Number(n.value)),
        );
        return max > 0 ? Math.max(8, Math.round((Number(p.value) / max) * 100)) : 8;
      })
    : [22, 26, 31, 35, 41, 48, 52, 61, 68, 75, 83, 91];

  return (
    <div className="page">
      <PageHeader
        title="定投计划"
        description="基于历史净值测算，并在持衡运行期间自动记录到本地账本"
      />
      {!wailsMode ? <PreviewNotice /> : null}
      <div className="grid two section">
        <section className="card">
          <h2>定投测算器</h2>
          <div className="grid two section">
            <div className="field">
              <label htmlFor="fund-code">基金代码</label>
              <input
                id="fund-code"
                className="input"
                value={form.fundCode}
                onChange={(e) => setForm({ ...form, fundCode: e.target.value })}
              />
            </div>
            <div className="field">
              <label htmlFor="fund-name">基金名称</label>
              <input
                id="fund-name"
                className="input"
                value={form.fundName}
                onChange={(e) => setForm({ ...form, fundName: e.target.value })}
              />
            </div>
            <div className="field">
              <label htmlFor="plan-amount">每期金额</label>
              <input
                id="plan-amount"
                className="input"
                defaultValue="1000.00"
                inputMode="decimal"
                value={form.amount}
                onChange={(e) => setForm({ ...form, amount: e.target.value })}
              />
            </div>
            <div className="field">
              <label htmlFor="cadence">定投周期</label>
              <select
                id="cadence"
                className="input"
                value={form.frequency}
                onChange={(e) => setForm({ ...form, frequency: e.target.value })}
              >
                <option value="monthly">每月</option>
                <option value="weekly">每周</option>
                <option value="daily">每日</option>
              </select>
            </div>
            <div className="field">
              <label htmlFor="exec-day">执行日</label>
              <input
                id="exec-day"
                className="input"
                inputMode="numeric"
                value={form.executionDay}
                onChange={(e) => setForm({ ...form, executionDay: e.target.value })}
              />
            </div>
            <div className="field">
              <label htmlFor="start-date">开始日期</label>
              <input
                id="start-date"
                className="input"
                type="date"
                value={form.startDate}
                onChange={(e) => setForm({ ...form, startDate: e.target.value })}
              />
            </div>
            <div className="field">
              <label htmlFor="years">回测年限</label>
              <select
                id="years"
                className="input"
                value={years}
                onChange={(e) => setYears(e.target.value)}
              >
                <option value="1">1 年</option>
                <option value="3">3 年</option>
                <option value="5">5 年</option>
                <option value="10">10 年</option>
              </select>
            </div>
          </div>
          <button
            type="button"
            className="button primary section"
            onClick={() => void runBacktest()}
          >
            <Calculator size={17} />
            {backtestLoading ? "测算中..." : "开始历史测算"}
          </button>
          {backtestError ? (
            <div className="state error" role="alert">
              {backtestError}
            </div>
          ) : null}
          {backtest ? (
            <>
              <MiniChart values={chartValues} />
              <div className="grid three">
                <div>
                  <span className="muted">累计投入</span>
                  <strong> ¥ {backtest.invested}</strong>
                </div>
                <div>
                  <span className="muted">期末市值</span>
                  <strong> ¥ {backtest.currentValue}</strong>
                </div>
                <div>
                  <span className="muted">历史收益</span>
                  <strong className={backtest.yieldRate.startsWith("-") ? "negative" : "positive"}>
                    {" "}
                    {backtest.yieldRate}
                  </strong>
                </div>
              </div>
            </>
          ) : null}
          <button
            type="button"
            className="button section"
            onClick={() => void savePlan()}
            disabled={!backtest}
          >
            <Plus size={17} />
            保存为计划
          </button>
        </section>
        <aside>
          <div className="notice">
            <strong>计划执行边界</strong>
            <span>持衡退出后不会后台执行；计划仅模拟本地持仓记账，不连接券商。</span>
          </div>
          <div className="notice">
            <strong>回测说明</strong>
            <span>回测基于历史净值区间，过去表现不代表未来收益；乐观/悲观为波动率偏移估算。</span>
          </div>
        </aside>
      </div>
      <h2 className="section">已保存计划</h2>
      <StateSwitch
        state={state}
        onRetry={() => void load()}
        emptyLabel="暂无定投计划，使用左侧表单创建"
      >
        <div className="grid two section">
          {plans.map((p) => (
            <article className="card" key={p.id}>
              <div className="row" style={{ justifyContent: "space-between" }}>
                <span className={`badge ${p.status === "active" ? "success" : "warning"}`}>
                  {p.status === "active" ? "运行中" : "已暂停"}
                </span>
                <div className="row">
                  <button
                    type="button"
                    className="button icon-button"
                    aria-label={p.status === "active" ? "暂停计划" : "恢复计划"}
                    onClick={() => void (p.status === "active" ? pause(p.id, p.version) : resume(p.id, p.version))}
                  >
                    {p.status === "active" ? <Pause size={16} /> : <Play size={16} />}
                  </button>
                  <button
                    type="button"
                    className="button danger icon-button"
                    aria-label="删除计划"
                    onClick={() => void remove(p.id, p.version)}
                  >
                    <Trash size={16} />
                  </button>
                </div>
              </div>
              <h2>{p.fundName}</h2>
              <div className="list">
                <div className="list-item">
                  <span>频率</span>
                  <strong>{frequencyLabels[p.frequency]}</strong>
                </div>
                <div className="list-item">
                  <span>每期金额</span>
                  <strong>¥ {p.amount}</strong>
                </div>
                <div className="list-item">
                  <span>执行</span>
                  <strong>{cadenceLabel(p.frequency, p.executionDay)}</strong>
                </div>
              </div>
            </article>
          ))}
        </div>
      </StateSwitch>
    </div>
  );
};
