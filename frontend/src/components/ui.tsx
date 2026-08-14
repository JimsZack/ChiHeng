import { CircleNotch, Package, WarningCircle } from "@phosphor-icons/react";
import type { ReactNode } from "react";
import type { Metric } from "../shared/types";

export const PageHeader = ({
  title,
  description,
  actions,
}: {
  readonly title: string;
  readonly description: string;
  readonly actions?: ReactNode;
}) => (
  <header className="page-head">
    <div>
      <h1>{title}</h1>
      <div className="muted">{description}</div>
    </div>
    {actions}
  </header>
);

export const MetricCard = ({ metric }: { readonly metric: Metric }) => (
  <article className="card">
    <div className="metric-label">{metric.label}</div>
    <div className={`metric-value ${metric.trend}`}>{metric.value}</div>
    <div className={`muted ${metric.trend}`}>{metric.comparison}</div>
  </article>
);

export const PreviewNotice = () => (
  <div className="notice preview" role="status">
    <WarningCircle size={18} />
    <div>
      <strong>浏览器预览数据</strong>
      <div>当前未连接持衡桌面运行时，页面内容仅用于界面验收，不代表实时行情。</div>
    </div>
  </div>
);

// ---- 状态组件：Loading / Empty / Error ----

export const LoadingState = ({ label = "加载中" }: { readonly label?: string }) => (
  <div className="state" role="status">
    <CircleNotch size={22} className="spin" />
    <div>{label}</div>
  </div>
);

export const EmptyState = ({ label = "暂无数据" }: { readonly label?: string }) => (
  <div className="state" role="status">
    <Package size={22} />
    <div>{label}</div>
  </div>
);

export const ErrorState = ({
  message,
  onRetry,
}: {
  readonly message: string;
  readonly onRetry?: () => void;
}) => (
  <div className="state error" role="alert">
    <WarningCircle size={22} />
    <div>{message}</div>
    {onRetry ? (
      <button type="button" className="button" onClick={onRetry}>
        重试
      </button>
    ) : null}
  </div>
);

// 按 LoadState 渲染统一的状态容器。
export const StateSwitch = ({
  state,
  error,
  onRetry,
  children,
  emptyLabel,
  loadingLabel,
}: {
  readonly state: "loading" | "ready" | "empty" | "error";
  readonly error?: string | null;
  readonly onRetry?: () => void;
  readonly children: ReactNode;
  readonly emptyLabel?: string;
  readonly loadingLabel?: string;
}) => {
  switch (state) {
    case "loading":
      return <LoadingState {...(loadingLabel ? { label: loadingLabel } : {})} />;
    case "empty":
      return <EmptyState {...(emptyLabel ? { label: emptyLabel } : {})} />;
    case "error":
      return <ErrorState message={error ?? "加载失败"} {...(onRetry ? { onRetry } : {})} />;
    default:
      return <>{children}</>;
  }
};

export const MiniChart = ({
  values = [44, 52, 48, 63, 58, 72, 69, 81, 76, 88, 83, 92],
}: {
  readonly values?: readonly number[];
}) => (
  <div className="chart" role="img" aria-label="走势概览，整体呈上升趋势">
    {values.map((value, index) => (
      // biome-ignore lint/suspicious/noArrayIndexKey: 静态示意柱状图，值可能重复且无需跨渲染复用
      <div className="bar" key={index} style={{ height: `${value}%` }} />
    ))}
  </div>
);
