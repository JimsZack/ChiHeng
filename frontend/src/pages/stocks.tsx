import { MagnifyingGlass, PushPin } from "@phosphor-icons/react";
import { useState } from "react";
import { MiniChart, PageHeader, PreviewNotice, StateSwitch } from "../components/ui";
import { addUserIndex } from "../services/api";
import { useAppStore, useMarketStore } from "../stores/app";

export const StocksPage = () => {
  const { search } = useMarketStore();
  const { wailsMode } = useAppStore();
  const [keyword, setKeyword] = useState("");
  const [results, setResults] = useState<
    ReturnType<typeof search> extends Promise<infer T> ? T : never
  >([]);
  const [searching, setSearching] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [selected, setSelected] = useState<{
    name: string;
    code: string;
    price: string;
    change: string;
  } | null>(null);
  const [pinned, setPinned] = useState(false);

  const doSearch = async () => {
    if (!keyword.trim()) {
      return;
    }
    setSearching(true);
    setError(null);
    setPinned(false);
    try {
      const found = await search(keyword.trim(), "stock");
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
    } catch (err) {
      setError(err instanceof Error ? err.message : "搜索失败");
    } finally {
      setSearching(false);
    }
  };

  const onPin = async () => {
    if (!selected) {
      return;
    }
    try {
      await addUserIndex(selected.code);
      setPinned(true);
    } catch (err) {
      setError(err instanceof Error ? err.message : "添加自选失败");
    }
  };

  return (
    <div className="page">
      <PageHeader title="股票行情" description="搜索股票、指数并查看报价、走势与盘口" />
      {!wailsMode ? <PreviewNotice /> : null}
      <div className="search section">
        <input
          className="input"
          aria-label="股票代码或名称"
          placeholder="输入代码或名称，例如 600519"
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
          搜索
        </button>
      </div>
      {searching ? <div className="state">正在搜索...</div> : null}
      {error ? (
        <div className="state error" role="alert">
          {error}
        </div>
      ) : null}
      <div className="grid two section">
        <section>
          <div className="card">
            {selected ? (
              <>
                <div className="page-head">
                  <div>
                    <span className="badge">行情</span>
                    <h1 style={{ marginTop: 8 }}>
                      {selected.name} <span className="muted">{selected.code}</span>
                    </h1>
                    <div
                      className={`metric-value ${selected.change.startsWith("+") ? "positive" : "negative"}`}
                    >
                      {selected.price}
                    </div>
                    <div className={selected.change.startsWith("+") ? "positive" : "negative"}>
                      {selected.change}
                    </div>
                  </div>
                  <button
                    type="button"
                    className="button"
                    onClick={() => void onPin()}
                    disabled={pinned}
                  >
                    <PushPin size={17} />
                    {pinned ? "已加入自选" : "加入自选"}
                  </button>
                </div>
                <div className="grid metrics">
                  {["今开 --", "最高 --", "最低 --", "昨收 --", "成交量 --", "成交额 --"].map(
                    (value) => (
                      <div className="muted" key={value}>
                        {value}
                      </div>
                    ),
                  )}
                </div>
              </>
            ) : (
              <div className="state">搜索后展示报价、走势与盘口</div>
            )}
          </div>
          <div className="card section">
            <div className="tabs">
              <button type="button" className="tab active">
                分时
              </button>
              <button type="button" className="tab">
                日 K
              </button>
              <button type="button" className="tab">
                周 K
              </button>
              <button type="button" className="tab">
                月 K
              </button>
            </div>
            <MiniChart values={[48, 52, 49, 43, 55, 61, 58, 63, 54, 47, 51, 44, 39, 45, 42]} />
          </div>
        </section>
        <aside className="card">
          <h2>搜索结果</h2>
          <StateSwitch
            state={results.length > 0 ? "ready" : "empty"}
            emptyLabel="输入代码或名称开始搜索"
          >
            <div className="list">
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
        </aside>
      </div>
    </div>
  );
};
