import { createBrowserRouter, RouterProvider } from "react-router-dom";
import { AppShell } from "./components/shell";
import { FundsPage } from "./pages/funds";
import { HoldingsPage } from "./pages/holdings";
import { KnowledgePage } from "./pages/knowledge";
import { MarketsPage } from "./pages/markets";
import { OverviewPage } from "./pages/overview";
import { PlansPage } from "./pages/plans";
import { SettingsPage } from "./pages/settings";
import { StocksPage } from "./pages/stocks";

const router = createBrowserRouter([
  {
    path: "/",
    element: <AppShell />,
    children: [
      { index: true, element: <OverviewPage /> },
      { path: "stocks", element: <StocksPage /> },
      { path: "funds", element: <FundsPage /> },
      { path: "markets", element: <MarketsPage /> },
      { path: "holdings", element: <HoldingsPage /> },
      { path: "plans", element: <PlansPage /> },
      { path: "knowledge", element: <KnowledgePage /> },
      { path: "settings", element: <SettingsPage /> },
    ],
  },
]);

export const App = () => <RouterProvider router={router} />;
