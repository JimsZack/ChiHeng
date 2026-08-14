import { Brain, DownloadSimple, Plus, UploadSimple } from "@phosphor-icons/react";
import { useEffect, useState } from "react";
import { MiniChart, PageHeader, PreviewNotice, StateSwitch } from "../components/ui";
import { exportHoldingsCSV, importHoldingsCSV } from "../services/api";
import { useAppStore, useHoldingsStore } from "../stores/app";

export const HoldingsPage = () => {
  const { holdings, state, load, add } = useHoldingsStore();
  const { wailsMode } = useAppStore();
  const [showForm, setShowForm] = useState(false);
  const [csvMessage, setCsvMessage] = useState<string | null>(null);
  const [form, setForm] = useState({
    fundCode: "",
    fundName: "",
    shares: "",
    costNav: "",
    openedOn: "",
  });

  useEffect(() => {
    void load();
  }, [load]);

  const onExportCSV = async () => {
    try {
      const result = await exportHoldingsCSV();
      setCsvMessage(`导出完成：${result.phase}`);
    } catch (err) {
      setCsvMessage(err instanceof Error ? err.message : "导出失败");
    }
  };

  const onImportCSV = async () => {
    try {
      const result = await importHoldingsCSV();
      setCsvMessage(`导入完成：${result.phase}`);
      await load();
    } catch (err) {
      setCsvMessage(err instanceof Error ? err.message : "导入失败");
    }
  };

  const submit = async () => {
    if (!form.fundCode || !form.shares || !form.costNav) {
      return;
    }
    await add({
      fundCode: form.fundCode,
      fundName: form.fundName || form.fundCode,
      shares: form.shares,
      costNav: form.costNav,
      openedOn: form.openedOn || new Date().toISOString().slice(0, 10),
    });
    setShowForm(false);
    setForm({ fundCode: "", fundName: "", shares: "", costNav: "", openedOn: "" });
  };

  const totalValue = holdings.reduce(
    (sum, h) => sum + (parseFloat(h.marketValue.replace(/,/g, "")) || 0),
    0,
  );
  const totalProfit = holdings.reduce(
    (sum, h) => sum + (parseFloat(h.profit.replace(/,/g, "")) || 0),
    0,
  );

  return (
    <div className="page">
      <PageHeader
        title="持仓管理"
        description="以实时估值审视仓位，买卖操作仅更新本地账本"
        actions={
          <div className="row">
            <button type="button" className="button" onClick={() => void onExportCSV()}>
              <DownloadSimple size={17} />
              导出 CSV
            </button>
            <button type="button" className="button" onClick={() => void onImportCSV()}>
              <UploadSimple size={17} />
              导入 CSV
            </button>
            <button type="button" className="button primary" onClick={() => setShowForm((v) => !v)}>
              <Plus size={17} />
              新增持仓
            </button>
          </div>
        }
      />
      {!wailsMode ? <PreviewNotice /> : null}
      {csvMessage ? (
        <div className="notice section" role="status">
          {csvMessage}
        </div>
      ) : null}
      {showForm ? (
        <section className="card section">
          <h2>新增持仓</h2>
          <div className="grid two section">
            <div className="field">
              <label htmlFor="new-code">基金代码</label>
              <input
                id="new-code"
                className="input"
                value={form.fundCode}
                onChange={(e) => setForm({ ...form, fundCode: e.target.value })}
              />
            </div>
            <div className="field">
              <label htmlFor="new-name">基金名称</label>
              <input
                id="new-name"
                className="input"
                value={form.fundName}
                onChange={(e) => setForm({ ...form, fundName: e.target.value })}
              />
            </div>
            <div className="field">
              <label htmlFor="new-shares">持有份额</label>
              <input
                id="new-shares"
                className="input"
                inputMode="decimal"
                value={form.shares}
                onChange={(e) => setForm({ ...form, shares: e.target.value })}
              />
            </div>
            <div className="field">
              <label htmlFor="new-cost">持仓成本</label>
              <input
                id="new-cost"
                className="input"
                inputMode="decimal"
                value={form.costNav}
                onChange={(e) => setForm({ ...form, costNav: e.target.value })}
              />
            </div>
          </div>
          <button type="button" className="button primary" onClick={() => void submit()}>
            保存持仓
          </button>
        </section>
      ) : null}
      <div className="grid metrics section">
        {[
          {
            l: "持仓市值",
            v: `¥ ${totalValue.toLocaleString("zh-CN", { minimumFractionDigits: 2 })}`,
          },
          { l: "当日预估", v: "待行情同步" },
          {
            l: "累计盈亏",
            v: `${totalProfit >= 0 ? "+" : ""}¥ ${totalProfit.toLocaleString("zh-CN", { minimumFractionDigits: 2 })}`,
          },
          { l: "持仓数量", v: `${holdings.length} 只` },
        ].map((x) => (
          <div className="card" key={x.l}>
            <div className="metric-label">{x.l}</div>
            <div className={`metric-value ${x.v.includes("+") ? "positive" : ""}`}>{x.v}</div>
          </div>
        ))}
      </div>
      <section className="card section table-wrap">
        <StateSwitch
          state={state}
          onRetry={() => void load()}
          emptyLabel="暂无持仓，点击右上角新增"
        >
          <table className="table">
            <thead>
              <tr>
                <th>基金名称</th>
                <th>持有份额</th>
                <th>持仓成本</th>
                <th>当前净值</th>
                <th>市值</th>
                <th>当日涨幅</th>
                <th>累计盈亏</th>
              </tr>
            </thead>
            <tbody>
              {holdings.map((h) => (
                <tr key={h.id}>
                  <td>
                    <strong>{h.name}</strong>
                    <div className="muted">{h.code}</div>
                  </td>
                  <td>{h.shares}</td>
                  <td>{h.costNav}</td>
                  <td>{h.currentNav}</td>
                  <td>{h.marketValue}</td>
                  <td className={h.dailyChange.startsWith("+") ? "positive" : "negative"}>
                    {h.dailyChange}
                  </td>
                  <td className={h.profit.startsWith("+") ? "positive" : "negative"}>{h.profit}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </StateSwitch>
      </section>
      <div className="row section" style={{ justifyContent: "space-between" }}>
        <h2>持仓当日走势</h2>
        <button type="button" className="button">
          <Brain size={17} />
          组合诊断
        </button>
      </div>
      <div className="grid three section">
        {holdings.map((h, index) => (
          <section className="card" key={h.id}>
            <strong>{h.name}</strong>
            <div className="muted">{h.code}</div>
            <MiniChart values={[42 + index * 5, 55, 48, 67, 61, 72, 64, 76, 70, 82]} />
          </section>
        ))}
      </div>
    </div>
  );
};
