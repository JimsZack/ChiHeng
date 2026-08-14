import { Brain, MagnifyingGlass, Plus } from "@phosphor-icons/react";
import { useEffect, useState } from "react";
import { MiniChart, PageHeader, PreviewNotice, StateSwitch } from "../components/ui";
import { getSearchHistory } from "../services/api";
import { useAppStore, useHoldingsStore, useMarketStore } from "../stores/app";

export const FundsPage = () => {
  const { search } = useMarketStore();
  const { add } = useHoldingsStore();
  const { wailsMode } = useAppStore();
  const [keyword, setKeyword] = useState("");
  const [results, setResults] = useState<
    { id: string; name: string; code: string; price: string; change: string }[]
  >([]);
  const [searching, setSearching] = useState(false);
  const [history, setHistory] = useState<string[]>([]);
  const [selected, setSelected] = useState<{
    name: string;
    code: string;
    price: string;
    change: string;
  } | null>(null);
  const [amount, setAmount] = useState("1000.00");

  useEffect(() => {
    void (async () => {
      const records = await getSearchHistory();
      setHistory(records.map((r) => r.name || r.code));
    })();
  }, []);

  const doSearch = async (term = keyword) => {
    const query = term.trim();
    if (!query) {
      return;
    }
    setSearching(true);
    try {
      const found = await search(query, "fund");
      setResults(found);
      const first = found[0];
      if (first) {
        setSelected({
          name: first.name,
          code: first.code,
          price: first.price,
          change: first.change,
        });
      }
      // 更新搜索历史（去重置顶）
      setHistory((prev) => [query, ...prev.filter((h) => h !== query)].slice(0, 10));
    } finally {
      setSearching(false);
    }
  };

  const recordHolding = async () => {
    if (!selected) {
      return;
    }
    // 按金额买入：份额 = 金额 / 净值（此处用当前价近似，正式实现由后端按 15:00 规则计算）
    const nav = parseFloat(selected.price.replace(/,/g, "")) || 1;
    const shares = (parseFloat(amount) / nav).toFixed(2);
    await add({
      fundCode: selected.code,
      fundName: selected.name,
      shares,
      costNav: nav.toFixed(4),
      openedOn: new Date().toISOString().slice(0, 10),
    });
  };

  return (
    <div className="page">
      <PageHeader title="基金查询与诊断" description="查询净值、量化表现并安全地记录持仓" />
      {!wailsMode ? <PreviewNotice /> : null}
      <div className="search section">
        <input
          className="input"
          aria-label="基金代码或名称"
          placeholder="输入基金代码或名称"
          value={keyword}
          onChange={(e) => setKeyword(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter") {
              void doSearch();
            }
          }}
        />
        <button type="button" className="button primary" onClick={() => void doSearch()}>
          <MagnifyingGlass size={17} />
          查询
        </button>
      </div>
      <div className="row section">
        <span className="muted">最近搜索</span>
        {history.length === 0 ? (
          <span className="muted">暂无记录</span>
        ) : (
          history.map((x) => (
            <button
              type="button"
              className="badge"
              key={x}
              onClick={() => {
                setKeyword(x);
                void doSearch(x);
              }}
            >
              {x}
            </button>
          ))
        )}
      </div>
      {searching ? <div className="state">正在查询...</div> : null}
      <div className="grid two section">
        <section>
          <div className="card">
            {selected ? (
              <>
                <span className="badge success">已查询</span>
                <h1 style={{ marginTop: 8 }}>
                  {selected.name} <span className="muted">{selected.code}</span>
                </h1>
                <div className="grid metrics section">
                  <div>
                    <div className="muted">参考价格</div>
                    <div
                      className={`metric-value ${selected.change.startsWith("+") ? "positive" : "negative"}`}
                    >
                      {selected.price}
                    </div>
                  </div>
                  <div>
                    <div className="muted">涨跌幅</div>
                    <div
                      className={`metric-value ${selected.change.startsWith("+") ? "positive" : "negative"}`}
                    >
                      {selected.change}
                    </div>
                  </div>
                  <div>
                    <div className="muted">基金类型</div>
                    <strong>待详情接口</strong>
                  </div>
                  <div>
                    <div className="muted">数据来源</div>
                    <strong>天天基金</strong>
                  </div>
                </div>
              </>
            ) : (
              <div className="state">查询后展示净值、估值与诊断</div>
            )}
            <MiniChart />
          </div>
          <div className="card section">
            <h2>智能诊断</h2>
            <div className="metric-value">
              -- <span className="muted">/ 100 · 待数据</span>
            </div>
            <p className="muted">
              查询基金后，将展示本地量化评分与规则诊断；AI 深度分析需在设置中配置密钥并明确同意。
            </p>
            <div className="row">
              <button type="button" className="button">
                <Brain size={17} />
                本地量化诊断
              </button>
              <button type="button" className="button">
                <Brain size={17} />
                AI 深度分析
              </button>
            </div>
          </div>
        </section>
        <aside className="card">
          <h2>记录持仓</h2>
          <div className="tabs section">
            <button type="button" className="tab active">
              按金额
            </button>
            <button type="button" className="tab">
              按份额
            </button>
          </div>
          <div className="field section">
            <label htmlFor="amount">投入金额</label>
            <input
              id="amount"
              className="input"
              inputMode="decimal"
              value={amount}
              onChange={(e) => setAmount(e.target.value)}
            />
          </div>
          <button
            type="button"
            className="button primary section"
            onClick={() => void recordHolding()}
          >
            <Plus size={17} />
            复核并添加
          </button>
          <div className="notice section">金额与份额仅记录在本机，不会发起交易。</div>
        </aside>
      </div>
      <StateSwitch
        state={results.length > 0 ? "ready" : "empty"}
        emptyLabel="查询基金后在此展示结果"
      >
        <div className="list section">
          {results.map((item) => (
            <button
              type="button"
              className="list-item"
              key={item.id}
              style={{
                width: "100%",
                textAlign: "left",
                background: "none",
                border: "none",
                cursor: "pointer",
              }}
              onClick={() =>
                setSelected({
                  name: item.name,
                  code: item.code,
                  price: item.price,
                  change: item.change,
                })
              }
            >
              <span>
                <strong>{item.name}</strong>
                <div className="muted">{item.code}</div>
              </span>
              <span>
                <div>{item.price}</div>
                <div className={item.change.startsWith("+") ? "positive" : "negative"}>
                  {item.change}
                </div>
              </span>
            </button>
          ))}
        </div>
      </StateSwitch>
    </div>
  );
};
