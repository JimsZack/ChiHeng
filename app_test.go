package main

import (
	"context"
	"testing"

	"github.com/JimsZack/ChiHeng/internal/bindings"
	"github.com/JimsZack/ChiHeng/internal/schema"
	"github.com/JimsZack/ChiHeng/internal/service"
	"github.com/JimsZack/ChiHeng/internal/store"
)

// newTestApp 组装带临时数据库与密钥存储的 App，并完成启动/退出生命周期。
func newTestApp(t *testing.T) *App {
	t.Helper()
	db, err := store.Open(context.Background(), t.TempDir()+"/test.db")
	if err != nil {
		t.Fatalf("store.Open() error = %v", err)
	}
	if _, err := schema.EnsureSchema(context.Background(), db.DB()); err != nil {
		t.Fatalf("EnsureSchema() error = %v", err)
	}
	secrets, err := service.NewFileSecretStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileSecretStore() error = %v", err)
	}
	app := NewApp(db, secrets)
	app.onStartup(context.Background())
	t.Cleanup(func() { app.onShutdown(context.Background()) })
	return app
}

func TestListHoldings_whenEmpty(t *testing.T) {
	t.Parallel()
	app := newTestApp(t)

	resp := app.ListHoldings(bindings.RequestMeta{RequestID: "r1"})
	if resp.Error != nil {
		t.Fatalf("ListHoldings() error = %v", resp.Error)
	}
	if resp.Data == nil || len(resp.Data) != 0 {
		t.Fatalf("ListHoldings() data = %+v, want empty list", resp.Data)
	}
}

func TestCreateAndListHolding(t *testing.T) {
	t.Parallel()
	app := newTestApp(t)

	created := app.CreateHolding(bindings.RequestMeta{RequestID: "r1"}, bindings.HoldingRequest{
		FundCode: "000001", FundName: "测试基金",
		Shares: "100.00", CostNAV: "1.2345", OpenedOn: "2026-08-13",
	})
	if created.Error != nil {
		t.Fatalf("CreateHolding() error = %v", created.Error)
	}
	if created.Data == nil || created.Data.ID == "" {
		t.Fatalf("CreateHolding() data = %+v", created.Data)
	}

	listed := app.ListHoldings(bindings.RequestMeta{RequestID: "r2"})
	if listed.Error != nil || len(listed.Data) != 1 {
		t.Fatalf("ListHoldings() = %+v, err = %v; want 1 holding", listed.Data, listed.Error)
	}
	if listed.Data[0].FundCode != "000001" {
		t.Fatalf("ListHoldings()[0] = %+v", listed.Data[0])
	}
}

func TestCreateHolding_whenInvalidShares(t *testing.T) {
	t.Parallel()
	app := newTestApp(t)

	resp := app.CreateHolding(bindings.RequestMeta{RequestID: "r1"}, bindings.HoldingRequest{
		FundCode: "000001", Shares: "abc", CostNAV: "1.0",
	})
	if resp.Error == nil || resp.Error.Code != "VALIDATION" {
		t.Fatalf("CreateHolding() error = %+v, want VALIDATION", resp.Error)
	}
}

func TestBuySellHolding_whenRoundTrip(t *testing.T) {
	t.Parallel()
	app := newTestApp(t)

	created := app.CreateHolding(bindings.RequestMeta{}, bindings.HoldingRequest{
		FundCode: "000001", FundName: "测试基金", Shares: "100.00", CostNAV: "1.0000", OpenedOn: "2026-08-13",
	})
	version := created.Data.Version

	bought := app.BuyHolding(bindings.RequestMeta{}, bindings.TradeHoldingRequest{
		ID: created.Data.ID, Shares: "100.00", NAV: "2.0000", ExpectedVersion: version,
	})
	if bought.Error != nil {
		t.Fatalf("BuyHolding() error = %v", bought.Error)
	}
	// 加权平均成本 = (100*1 + 100*2) / 200 = 1.5
	if bought.Data.CostNAV != "1.5" {
		t.Fatalf("BuyHolding() costNav = %q, want 1.5", bought.Data.CostNAV)
	}

	sold := app.SellHolding(bindings.RequestMeta{}, bindings.TradeHoldingRequest{
		ID: created.Data.ID, Shares: "50.00", NAV: "2.5000", ExpectedVersion: bought.Data.Version,
	})
	if sold.Error != nil {
		t.Fatalf("SellHolding() error = %v", sold.Error)
	}
	if sold.Data.Shares != "150" {
		t.Fatalf("SellHolding() shares = %q, want 150", sold.Data.Shares)
	}
}

func TestPortfolioOverview_whenEmpty(t *testing.T) {
	t.Parallel()
	app := newTestApp(t)

	resp := app.PortfolioOverview(bindings.RequestMeta{})
	if resp.Error != nil {
		t.Fatalf("PortfolioOverview() error = %v", resp.Error)
	}
	if resp.Data == nil || len(resp.Data.Holdings) != 0 {
		t.Fatalf("PortfolioOverview() = %+v, want empty", resp.Data)
	}
}

func TestGetAppInfo(t *testing.T) {
	t.Parallel()
	app := newTestApp(t)

	info := app.GetAppInfo()
	if info["name"] == "" || info["version"] == "" {
		t.Fatalf("GetAppInfo() = %+v, want name and version", info)
	}
}

func TestSettingsBindings_whenRoundTrip(t *testing.T) {
	t.Parallel()
	app := newTestApp(t)

	saved := app.SavePreferences(bindings.RequestMeta{}, bindings.PreferencesRequest{
		Locale: "en-US", Theme: "dark", RefreshIntervalSeconds: 30,
		DisclaimerVersion: "2.0", LogLevel: "debug",
	})
	if saved.Error != nil {
		t.Fatalf("SavePreferences() error = %v", saved.Error)
	}
	got := app.GetPreferences(bindings.RequestMeta{})
	if got.Error != nil || got.Data == nil || got.Data.Theme != "dark" {
		t.Fatalf("GetPreferences() = %+v, err = %v", got.Data, got.Error)
	}
}

func TestPlanBindings_whenCreateAndPause(t *testing.T) {
	t.Parallel()
	app := newTestApp(t)

	created := app.CreatePlan(bindings.RequestMeta{}, bindings.PlanRequest{
		FundCode: "000001", FundName: "测试基金", Amount: "100.00",
		Frequency: "monthly", ExecutionDay: 15, StartDate: "2026-08-13",
	})
	if created.Error != nil {
		t.Fatalf("CreatePlan() error = %v", created.Error)
	}

	paused := app.PausePlan(bindings.RequestMeta{}, bindings.IDRequest{
		ID: created.Data.ID, ExpectedVersion: created.Data.Version,
	})
	if paused.Error != nil || paused.Data.Status != "paused" {
		t.Fatalf("PausePlan() = %+v, err = %v", paused.Data, paused.Error)
	}
}
