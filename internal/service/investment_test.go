package service

import (
	"context"
	"errors"
	"testing"

	"github.com/chiheng-app/chiheng/internal/domain"
	"github.com/chiheng-app/chiheng/internal/schema"
	"github.com/chiheng-app/chiheng/internal/store"
)

func openStore(t *testing.T) *store.Store {
	t.Helper()
	db, err := store.Open(context.Background(), t.TempDir()+"/test.db")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := schema.EnsureSchema(context.Background(), db.DB()); err != nil {
		t.Fatalf("ensure schema: %v", err)
	}
	return db
}

// fakeNAV 提供可编程的基金净值。
type fakeNAV struct {
	value int64
	err   error
}

func (f *fakeNAV) FundNAV(_ context.Context, _ string) (int64, error) {
	return f.value, f.err
}

func TestCreatePlan_whenValidInput(t *testing.T) {
	t.Parallel()
	s := NewInvestment(openStore(t), &fakeNAV{value: domain.Scale}, NewPortfolio(openStore(t)))

	plan, err := s.CreatePlan(context.Background(), domain.Plan{
		FundCode:     "000001",
		FundName:     "测试基金",
		Amount:       100_00, // 100 元 = 10000 分
		Frequency:    "monthly",
		ExecutionDay: 15,
	})

	if err != nil {
		t.Fatalf("CreatePlan() error = %v", err)
	}
	if plan.ID == "" || plan.Status != "active" || plan.StartDate == "" {
		t.Fatalf("CreatePlan() plan = %+v", plan)
	}
}

func TestCreatePlan_whenInvalidFrequency(t *testing.T) {
	t.Parallel()
	s := NewInvestment(openStore(t), &fakeNAV{}, NewPortfolio(openStore(t)))

	_, err := s.CreatePlan(context.Background(), domain.Plan{
		FundCode: "000001", Amount: 100_00, Frequency: "yearly",
	})
	if !errors.Is(err, domain.ErrInvalidDecimal) {
		t.Fatalf("CreatePlan() error = %v, want ErrInvalidDecimal", err)
	}
}

func TestExecuteDue_whenPlanDue(t *testing.T) {
	t.Parallel()
	db := openStore(t)
	portfolio := NewPortfolio(db)
	s := NewInvestment(db, &fakeNAV{value: 1_000_000}, portfolio)

	// 每日执行计划，从今天开始，必然到期。
	plan, err := s.CreatePlan(context.Background(), domain.Plan{
		FundCode: "000001", FundName: "测试基金",
		Amount: 100_00, Frequency: "daily",
	})
	if err != nil {
		t.Fatalf("CreatePlan() error = %v", err)
	}

	count, err := s.ExecuteDue(context.Background())
	if err != nil {
		t.Fatalf("ExecuteDue() error = %v", err)
	}
	if count != 1 {
		t.Fatalf("ExecuteDue() count = %d, want 1", count)
	}

	// 同日重复执行应幂等（账本唯一约束），不新增执行。
	count, err = s.ExecuteDue(context.Background())
	if err != nil {
		t.Fatalf("ExecuteDue() second call error = %v", err)
	}
	if count != 0 {
		t.Fatalf("ExecuteDue() second call count = %d, want 0 (idempotent)", count)
	}

	// 计划执行应写入持仓。
	holdings, err := portfolio.List(context.Background())
	if err != nil || len(holdings) != 1 {
		t.Fatalf("portfolio holdings = %d, err = %v; want 1", len(holdings), err)
	}
	if holdings[0].FundCode != plan.FundCode {
		t.Fatalf("holding fund = %q, want %q", holdings[0].FundCode, plan.FundCode)
	}
}

func TestExecuteDue_whenPausedPlan(t *testing.T) {
	t.Parallel()
	db := openStore(t)
	s := NewInvestment(db, &fakeNAV{value: 1_000_000}, NewPortfolio(db))

	plan, err := s.CreatePlan(context.Background(), domain.Plan{
		FundCode: "000001", FundName: "测试基金",
		Amount: 100_00, Frequency: "daily",
	})
	if err != nil {
		t.Fatalf("CreatePlan() error = %v", err)
	}
	if _, err := s.PausePlan(context.Background(), plan.ID, plan.Version); err != nil {
		t.Fatalf("PausePlan() error = %v", err)
	}

	count, err := s.ExecuteDue(context.Background())
	if err != nil {
		t.Fatalf("ExecuteDue() error = %v", err)
	}
	if count != 0 {
		t.Fatalf("ExecuteDue() count = %d, want 0 for paused plan", count)
	}
}

func TestBacktest_whenEmptySeries(t *testing.T) {
	t.Parallel()
	s := NewInvestment(openStore(t), &fakeNAV{}, NewPortfolio(openStore(t)))

	result := s.Backtest(context.Background(), nil, 100_00, 3)
	if result.Invested != 0 {
		t.Fatalf("Backtest() invested = %d, want 0", result.Invested)
	}
}

func TestBacktest_whenValidSeries(t *testing.T) {
	t.Parallel()
	s := NewInvestment(openStore(t), &fakeNAV{}, NewPortfolio(openStore(t)))

	series := make([]domain.SeriesPoint, 40)
	for i := range series {
		series[i] = domain.SeriesPoint{Time: "2026-01-01", Close: 1_000_000 + int64(i)*10_000}
	}
	result := s.Backtest(context.Background(), series, 100_00, 3)
	if result.Invested == 0 || result.CurrentValue == 0 {
		t.Fatalf("Backtest() = %+v, want non-zero", result)
	}
	if len(result.Neutral) == 0 || len(result.Optimistic) == 0 || len(result.Pessimistic) == 0 {
		t.Fatalf("Backtest() series empty: neutral=%d optimistic=%d pessimistic=%d",
			len(result.Neutral), len(result.Optimistic), len(result.Pessimistic))
	}
}
