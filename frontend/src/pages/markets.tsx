import { PushPin } from "@phosphor-icons/react";
import { useCallback, useEffect, useState } from "react";
import { MiniChart, PageHeader, PreviewNotice, StateSwitch } from "../components/ui";
import { getQuote } from "../services/api";
import type { Instrument } from "../shared/types";
import { useAppStore, useMarketStore } from "../stores/app";

export const MarketsPage = () => {
  const { search } = useMarketStore();
  const { wailsMode } = useAppStore();
  const [results, setResults] = useState<Instrument[]>([]);
  const [state, setState] = useState<"loading" | "ready" | "empty" | "error">("loading");
  const [selected, setSelected] = useState<{
    name: string;
    code: string;
    price: string;
    change: string;
  } | null>(null);

  const selectInstrument = async (instrument: Instrument) => {
    setSelected({ name: instrument.name, code: instrument.code, price: "--", change: "--" });
    try {
      const quote = await getQuote(instrument.id);
      setSelected({
        name: quote.instrument.name,
        code: quote.instrument.code,
        price: quote.price,
        change: quote.change,
      });
    } catch {
      // 报价获取失败时保留占位符
    }
  };

  const load = useCallback(async () => {
    setState("loading");
    try {
      const found = await search("美元", "forex");
      setResults(found);
      setState(found.length > 0 ? "ready" : "empty");
      if (found[0]) {
        await selectInstrument(found[0]);
      }
    } catch {
      setState("error");
    }
  }, [search]);

  useEffect(() => {
    void load();
  }, [load]);

  return (
    <div className="page">
      <PageHeader
        title="外汇与商品"
        description="查看受支持数据源提供的外汇、贵金属与大宗商品行情"
      />
      {!wailsMode ? <PreviewNotice /> : null}
      <div className="notice section">
        <strong>数据源说明</strong>
        <span>
          外汇与商品行情来自公开行情接口，更新频率受数据源限制；盘口与分时以实际返回为准。
        </span>
      </div>
      <StateSwitch state={state} onRetry={() => void load()} emptyLabel="暂无外汇商品行情">
        <div className="grid three section">
          {results.map((i) => (
            <button
              type="button"
              className="card list-item"
              key={i.id}
              style={{ color: "inherit", textAlign: "left", cursor: "pointer" }}
              onClick={() => void selectInstrument(i)}
            >
              <div>
                <span className="badge">forex</span>
                <h2>{i.name}</h2>
                <div className="muted">{i.code}</div>
              </div>
              <div className="quote">
                <strong>{selected && selected.code === i.code ? selected.price : "--"}</strong>
                <div
                  className={
                    selected && selected.code === i.code && selected.change.startsWith("+")
                      ? "positive"
                      : "negative"
                  }
                >
                  {selected && selected.code === i.code ? selected.change : "--"}
                </div>
              </div>
            </button>
          ))}
        </div>
      </StateSwitch>
      <div className="grid two section">
        <section className="card">
          <div className="page-head">
            <div>
              <span className="badge">外汇</span>
              <h1>
                {selected?.name ?? "美元 / 人民币"}{" "}
                <span className="muted">{selected?.code ?? "USD/CNY"}</span>
              </h1>
              <div
                className={`metric-value ${selected?.change.startsWith("+") ? "positive" : "negative"}`}
              >
                {selected?.price ?? "--"}
              </div>
              <div className={selected?.change.startsWith("+") ? "positive" : "negative"}>
                {selected?.change ?? "--"}
              </div>
            </div>
            <button type="button" className="button">
              <PushPin size={17} />
              加入自选
            </button>
          </div>
          <MiniChart />
        </section>
        <aside className="card">
          <h2>行情摘要</h2>
          <div className="list">
            {["今开 --", "最高 --", "最低 --", "昨收 --", "更新时间 --"].map((x) => (
              <div className="list-item" key={x}>
                {x}
              </div>
            ))}
          </div>
          <div className="tabs section">
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
        </aside>
      </div>
    </div>
  );
};
