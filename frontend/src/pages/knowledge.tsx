import { ArrowSquareOut, BookOpenText, ShieldCheck } from "@phosphor-icons/react";
import { useEffect } from "react";
import { PageHeader, PreviewNotice, StateSwitch } from "../components/ui";
import { useAppStore, useMarketStore } from "../stores/app";

export const KnowledgePage = () => {
  const { news, state, load } = useMarketStore();
  const { wailsMode } = useAppStore();

  useEffect(() => {
    void load();
  }, [load]);

  return (
    <div className="page">
      <PageHeader title="投资知识" description="理解估值、风险与长期投资中的关键概念" />
      {!wailsMode ? <PreviewNotice /> : null}
      <div className="grid two section">
        <article className="card">
          <BookOpenText size={24} />
          <h2>定投的核心不是预测</h2>
          <p>
            定期投入可以分散择时压力，但无法消除市场风险。回测展示过去区间的结果，不代表未来收益。
          </p>
          <h2 className="section">选择基金时关注什么</h2>
          <p>
            同时查看跟踪标的、费用、回撤、长期业绩稳定性与自己的持有期限。短期排名不应成为唯一依据。
          </p>
          <h2 className="section">读懂净值与估值</h2>
          <p>
            盘中估值来自公开行情推算，可能与最终公布净值存在差异。持衡会保留来源与更新时间，并在过期时继续展示最后成功数据。
          </p>
        </article>
        <aside>
          <div className="card">
            <ShieldCheck size={24} />
            <h2>安全边界</h2>
            <div className="list">
              <div className="list-item">本地持仓账本，不连接券商</div>
              <div className="list-item">AI 组合分析前逐次确认发送范围</div>
              <div className="list-item">分析仅供参考，不构成投资建议</div>
            </div>
          </div>
          <div className="card section">
            <h2>市场快讯</h2>
            <StateSwitch state={state} onRetry={() => void load()} emptyLabel="暂无快讯">
              <div className="list">
                {news.map((n) => (
                  <a
                    className="list-item"
                    href={n.url}
                    target="_blank"
                    rel="noreferrer"
                    key={n.id}
                    style={{ color: "inherit", textDecoration: "none" }}
                  >
                    <span>
                      {n.title}
                      <span className="muted"> · {n.time}</span>
                    </span>
                    <ArrowSquareOut size={16} />
                  </a>
                ))}
              </div>
            </StateSwitch>
          </div>
        </aside>
      </div>
    </div>
  );
};
