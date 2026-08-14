import {
  ArrowClockwise,
  BookOpenText,
  CalendarCheck,
  ChartLineUp,
  GearSix,
  GlobeHemisphereEast,
  MagnifyingGlass,
  SquaresFour,
  Wallet,
} from "@phosphor-icons/react";
import type { ComponentType } from "react";
import { NavLink, Outlet, useNavigate } from "react-router-dom";

const items: readonly {
  readonly to: string;
  readonly label: string;
  readonly icon: ComponentType<{ size: number }>;
}[] = [
  { to: "/", label: "总览", icon: SquaresFour },
  { to: "/stocks", label: "股票行情", icon: ChartLineUp },
  { to: "/funds", label: "基金诊断", icon: MagnifyingGlass },
  { to: "/markets", label: "外汇商品", icon: GlobeHemisphereEast },
  { to: "/holdings", label: "持仓管理", icon: Wallet },
  { to: "/plans", label: "定投计划", icon: CalendarCheck },
  { to: "/knowledge", label: "投资知识", icon: BookOpenText },
];

export const AppShell = () => {
  const navigate = useNavigate();
  return (
    <div className="app">
      <header className="titlebar">
        <div className="brand">
          <svg className="brand-mark" viewBox="0 0 24 24" aria-hidden="true">
            <path
              d="M12 3v18M5 8h5v3H5zM14 13h5v3h-5z"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
            />
          </svg>
          <span className="brand-copy">
            持衡 <span className="muted">ChiHeng</span>
          </span>
        </div>
        <div className="title-actions">
          <span className="badge success">本地数据就绪</span>
          <button type="button" className="button icon-button" aria-label="刷新当前数据">
            <ArrowClockwise size={17} />
          </button>
          <button
            type="button"
            className="button icon-button"
            aria-label="打开设置"
            onClick={() => void navigate("/settings")}
          >
            <GearSix size={17} />
          </button>
        </div>
      </header>
      <aside className="sidebar">
        <nav className="nav" aria-label="主导航">
          {items.map(({ to, label, icon: Icon }) => (
            <NavLink key={to} to={to} end={to === "/"}>
              {({ isActive }) => (
                <>
                  <Icon size={19} />
                  <span>{label}</span>
                  <small className="route-mobile">{label.slice(0, 2)}</small>
                  {isActive ? <span className="sr-only" /> : null}
                </>
              )}
            </NavLink>
          ))}
        </nav>
        <div className="sidebar-foot">行情仅供参考，不构成投资建议</div>
      </aside>
      <main id="main">
        <Outlet />
      </main>
    </div>
  );
};
