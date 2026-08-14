package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/chiheng-app/chiheng/internal/domain"
	"github.com/chiheng-app/chiheng/internal/store"
)

// Investment 定投计划域：计划 CRUD、回测、执行账本。
type Investment struct {
	store     *store.Store
	market    QuoteSource
	portfolio *Portfolio
}

// QuoteSource 提供基金实时估值（由 Market 服务实现，避免循环依赖）。
type QuoteSource interface {
	FundNAV(ctx context.Context, code string) (int64, error)
}

func NewInvestment(repository *store.Store, market QuoteSource, portfolio *Portfolio) *Investment {
	return &Investment{
		store:     repository,
		market:    market,
		portfolio: portfolio,
	}
}

// ExecuteDue 检查并执行所有到期的定投计划（调度器 Runner 实现）。
// 幂等性：plan_executions 表 UNIQUE(plan_id, scheduled_date) 保证同日不重复执行。
func (s *Investment) ExecuteDue(ctx context.Context) (int, error) {
	plans, err := s.store.ListPlans(ctx)
	if err != nil {
		return 0, err
	}
	today := time.Now().UTC().Format("2006-01-02")
	executed := 0
	for _, plan := range plans {
		if plan.Status != "active" {
			continue
		}
		if !isDue(plan, today) {
			continue
		}
		exists, err := s.store.ExecutionExists(ctx, plan.ID, today)
		if err != nil {
			return executed, err
		}
		if exists {
			continue
		}
		execID, err := s.store.RecordExecution(ctx, plan.ID, today, "running", plan.Amount, 0)
		if err != nil {
			return executed, err
		}
		if err := s.executeOne(ctx, plan, today, execID); err != nil {
			_ = s.store.UpdateExecutionResult(ctx, execID, "failed", 0, errorCodeOf(err))
			continue
		}
		executed++
	}
	return executed, nil
}

func (s *Investment) executeOne(ctx context.Context, plan domain.Plan, today, execID string) error {
	nav, err := s.market.FundNAV(ctx, plan.FundCode)
	if err != nil {
		return err
	}
	// 份额 = 金额(分) * 10000 / 净值(微元)，结果单位 micros
	shares := plan.Amount * 10000 / nav
	if shares <= 0 {
		return errors.New("净值异常")
	}
	// 交易归属日：15:00 后顺延至下一交易日（按本地时间）
	effectiveDate := EffectiveTradingDate(time.Now()).Format("2006-01-02")
	// 尝试加仓已有持仓，否则新建持仓
	holdings, err := s.portfolio.List(ctx)
	if err != nil {
		return err
	}
	var existing *domain.Holding
	for i := range holdings {
		if holdings[i].FundCode == plan.FundCode {
			existing = &holdings[i]
			break
		}
	}
	if existing == nil {
		holding := domain.Holding{
			FundCode:   plan.FundCode,
			FundName:   plan.FundName,
			Shares:     shares,
			CostNAV:    nav,
			CurrentNAV: nav,
			OpenedOn:   effectiveDate,
		}
		if _, err := s.portfolio.Create(ctx, holding); err != nil {
			return err
		}
	} else {
		if _, err := s.portfolio.Buy(ctx, existing.ID, shares, nav, existing.Version); err != nil {
			return err
		}
	}
	return s.store.UpdateExecutionResult(ctx, execID, "succeeded", nav, "")
}

// isDue 判断计划在指定日期是否应执行。
func isDue(plan domain.Plan, date string) bool {
	parsed, err := time.Parse("2006-01-02", date)
	if err != nil {
		return false
	}
	if date < plan.StartDate {
		return false
	}
	switch plan.Frequency {
	case "daily":
		return true
	case "weekly":
		// execution_day: 1=周一 … 7=周日
		weekday := int(parsed.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		return weekday == plan.ExecutionDay
	case "monthly":
		return parsed.Day() == plan.ExecutionDay
	default:
		return false
	}
}

func fmtAmount(cents int64) string {
	return fmt.Sprintf("%d.%02d", cents/100, cents%100)
}

func errorCodeOf(err error) string {
	if errors.Is(err, domain.ErrInvalidDecimal) {
		return "INVALID_AMOUNT"
	}
	if errors.Is(err, store.ErrConflict) {
		return "CONFLICT"
	}
	return "UPSTREAM"
}

// ErrNotFound 透传 store 未找到错误。
var ErrNotFound = errors.New("not found")

// ErrPanic 表示执行过程中发生 panic。
var ErrPanic = errors.New("panic during execution")

func (s *Investment) CreatePlan(ctx context.Context, plan domain.Plan) (domain.Plan, error) {
	if plan.Amount <= 0 {
		return domain.Plan{}, domain.ErrInvalidDecimal
	}
	switch plan.Frequency {
	case "daily", "weekly", "monthly":
	default:
		return domain.Plan{}, domain.ErrInvalidDecimal
	}
	if plan.StartDate == "" {
		plan.StartDate = time.Now().UTC().Format("2006-01-02")
	}
	if plan.Status == "" {
		plan.Status = "active"
	}
	plan.ID = newID("plan")
	return s.store.CreatePlan(ctx, plan)
}

func (s *Investment) ListPlans(ctx context.Context) ([]domain.Plan, error) {
	return s.store.ListPlans(ctx)
}

func (s *Investment) PausePlan(ctx context.Context, id string, expected int64) (domain.Plan, error) {
	return s.store.UpdatePlanStatus(ctx, id, "paused", expected)
}

func (s *Investment) ResumePlan(ctx context.Context, id string, expected int64) (domain.Plan, error) {
	return s.store.UpdatePlanStatus(ctx, id, "active", expected)
}

func (s *Investment) DeletePlan(ctx context.Context, id string, expected int64) error {
	return s.store.DeletePlan(ctx, id, expected)
}

// Backtest 基于历史净值序列做定投回测：按频率换算每期扣款，产出中性/乐观/悲观三曲线。
// 简化模型：乐观/悲观以历史年化波动率 ± 偏移中性曲线。
func (s *Investment) Backtest(ctx context.Context, navSeries []domain.SeriesPoint, amount int64, years int) domain.BacktestResult {
	if len(navSeries) == 0 || amount <= 0 {
		return domain.BacktestResult{}
	}
	periods := years * 12
	if periods <= 0 {
		periods = 36
	}
	if periods > len(navSeries) {
		periods = len(navSeries)
	}
	invested := int64(0)
	shares := int64(0)
	neutral := make([]domain.AssetPoint, 0, periods)
	last := navSeries[len(navSeries)-1].Close
	if last == 0 {
		return domain.BacktestResult{}
	}
	for i := 0; i < periods; i++ {
		idx := len(navSeries) - periods + i
		price := navSeries[idx].Close
		if price <= 0 {
			continue
		}
		invested += amount
		shares += amount * domain.Scale / price
		neutral = append(neutral, domain.AssetPoint{
			Date:             navSeries[idx].Time,
			TotalMarketValue: shares * last / domain.Scale,
			TotalCost:        invested,
		})
	}
	current := shares * last / domain.Scale
	yield := int64(0)
	if invested > 0 {
		yield = (current - invested) * 10000 / invested
	}
	// 乐观/悲观：按 ±5% 年化收益粗略偏移（真实回测可用波动率采样）
	optimistic := make([]domain.AssetPoint, 0, len(neutral))
	pessimistic := make([]domain.AssetPoint, 0, len(neutral))
	for _, p := range neutral {
		optimistic = append(optimistic, domain.AssetPoint{Date: p.Date, TotalMarketValue: p.TotalMarketValue * 105 / 100, TotalCost: p.TotalCost})
		pessimistic = append(pessimistic, domain.AssetPoint{Date: p.Date, TotalMarketValue: p.TotalMarketValue * 95 / 100, TotalCost: p.TotalCost})
	}
	return domain.BacktestResult{
		Invested:     invested,
		CurrentValue: current,
		YieldRate:    yield,
		Neutral:      neutral,
		Optimistic:   optimistic,
		Pessimistic:  pessimistic,
	}
}

func (s *Investment) ListExecutions(ctx context.Context, planID string, limit int) ([]domain.Execution, error) {
	return s.store.ListExecutions(ctx, planID, limit)
}
