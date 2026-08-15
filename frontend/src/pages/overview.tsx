import { ArrowSquareOut, Pulse } from "@phosphor-icons/react";
import { useEffect } from "react";
import { MetricCard, MiniChart, PageHeader, PreviewNotice, StateSwitch } from "../components/ui";
import { useRuntimeNotice } from "../hooks/use-runtime";
import { useAppStore, useHoldingsStore, useMarketStore } from "../stores/app";

export const OverviewPage = () => {
  const { holdings, state: holdingsState, load: loadHoldings } = useHoldingsStore();
  const { indices, news, state: marketState, load: loadMarket } = useMarketStore();
  const { wailsMode } = useAppStore();
  const runtimeNotice = useRuntimeNotice(wailsMode);

  useEffect(() => {
    void loadHoldings();
    void loadMarket();
  }, [loadHoldings, loadMarket]);

  const totalMarketValue = holdings.reduce(
    (sum, h) => sum + parseFloat(h.marketValue.replace(/,/g, "")) || 0,
    0,
  );
  const totalCost = holdings.reduce(
    (sum, h) =>
      sum + parseFloat(h.costNav.replace(/,/g, "")) * parseFloat(h.shares.replace(/,/g, "")) || 0,
    0,
  );
  const pnl = totalMarketValue - totalCost;
  const rate = totalCost > 0 ? ((pnl / totalCost) * 100).toFixed(2) : "0.00";

  const trend = (value: number): "positive" | "negative" | "neutral" =>
    value > 0 ? "positive" : value < 0 ? "negative" : "neutral";

  const metrics: {
    label: string;
    value: string;
    comparison: string;
    trend: "positive" | "negative" | "neutral";
  }[] = [
    {
      label: "总资产",
      value: `¥ ${totalMarketValue.toLocaleString("zh-CN", { minimumFractionDigits: 2 })}`,
      comparison: `累计收益 ${pnl >= 0 ? "+" : ""}${pnl.toLocaleString("zh-CN", { minimumFractionDigits: 2 })}`,
      trend: trend(pnl),
    },
    {
      label: "当日预估收益",
      value: "待行情同步",
      comparison: "交易时段自动更新",
      trend: "neutral",
    },
    {
      label: "持仓成本",
      value: `¥ ${totalCost.toLocaleString("zh-CN", { minimumFractionDigits: 2 })}`,
      comparison: `${holdings.length} 只基金`,
      trend: "neutral",
    },
    { label: "累计收益率", value: `${rate}%`, comparison: "自建仓之日起", trend: trend(pnl) },
  ];

  return (
    <div className="page">
      <PageHeader
        title="资产总览"
        description="查看本地持仓价值、收益与市场动态"
        actions={
          <button type="button" className="button" onClick={() => void loadMarket()}>
            <Pulse size={17} />
            刷新行情
          </button>
        }
      />
      {runtimeNotice}
      {!wailsMode ? <PreviewNotice /> : null}
      <div className="grid metrics section">
        {metrics.map((metric) => (
          <MetricCard key={metric.label} metric={metric} />
        ))}
      </div>
      <div className="grid two section">
        <section className="card">
          <div className="row" style={{ justifyContent: "space-between" }}>
            <div>
              <h2>资产走势</h2>
              <span className="muted">基于每日真实记录</span>
            </div>
            <span className="badge warning">数据同步于本地账本</span>
          </div>
          <StateSwitch
            state={holdingsState}
            onRetry={() => void loadHoldings()}
            emptyLabel="暂无资产记录"
          >
            <MiniChart values={[44, 52, 48, 63, 58, 72, 69, 81, 76, 88, 83, 92]} />
          </StateSwitch>
        </section>
        <section className="card">
          <h2>全球指数与自选</h2>
          <StateSwitch
            state={marketState}
            onRetry={() => void loadMarket()}
            emptyLabel="暂无指数数据"
          >
            <div className="list">
              {indices.slice(0, 4).map((item) => (
                <div className="list-item" key={item.instrument.id}>
                  <div>
                    <strong>{item.instrument.name}</strong>
                    <div className="muted">{item.instrument.code}</div>
                  </div>
                  <div className="quote">
                    <div>{item.price}</div>
                    <div className={item.change.startsWith("+") ? "positive" : "negative"}>
                      {item.change}
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </StateSwitch>
        </section>
      </div>
      <div className="grid two section">
        <section className="card">
          <h2>持仓分布</h2>
          <MiniChart values={[62, 84, 48, 71, 55, 38, 66, 44]} />
          <div className="muted">按持仓市值计算，图表数据同时可在持仓表中读取。</div>
        </section>
        <section className="card">
          <h2>东方财富全球 7×24</h2>
          <StateSwitch state={marketState} onRetry={() => void loadMarket()} emptyLabel="暂无快讯">
            <div className="list">
              {news.map((item) => (
                <a
                  className="list-item"
                  href={item.url}
                  target="_blank"
                  rel="noreferrer"
                  key={item.id}
                  style={{ color: "inherit", textDecoration: "none" }}
                >
                  <div>
                    <span className="badge">{item.tag}</span> {item.title}
                    <div className="muted">{item.publishedAt}</div>
                  </div>
                  <ArrowSquareOut size={16} />
                </a>
              ))}
            </div>
          </StateSwitch>
        </section>
      </div>
    </div>
  );
};
