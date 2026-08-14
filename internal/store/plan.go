package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/chiheng-app/chiheng/internal/domain"
)

var ErrNotFound = errors.New("not found")

// ---- 定投计划 ----

func (s *Store) ListPlans(ctx context.Context) ([]domain.Plan, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,fund_code,fund_name,amount_cents,frequency,execution_day,start_date,status,version,created_at,updated_at FROM investment_plans ORDER BY created_at`)
	if err != nil {
		return nil, fmt.Errorf("listing plans: %w", err)
	}
	defer rows.Close()
	plans := []domain.Plan{}
	for rows.Next() {
		var p domain.Plan
		if err := rows.Scan(&p.ID, &p.FundCode, &p.FundName, &p.Amount, &p.Frequency, &p.ExecutionDay, &p.StartDate, &p.Status, &p.Version, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning plan: %w", err)
		}
		plans = append(plans, p)
	}
	return plans, rows.Err()
}

func (s *Store) CreatePlan(ctx context.Context, plan domain.Plan) (domain.Plan, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.ExecContext(ctx, `INSERT INTO investment_plans(id,fund_code,fund_name,amount_cents,frequency,execution_day,start_date,status,version,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,1,?,?)`,
		plan.ID, plan.FundCode, plan.FundName, plan.Amount, plan.Frequency, plan.ExecutionDay, plan.StartDate, plan.Status, now, now)
	if err != nil {
		return domain.Plan{}, fmt.Errorf("creating plan: %w", err)
	}
	plan.Version, plan.CreatedAt, plan.UpdatedAt = 1, now, now
	return plan, nil
}

func (s *Store) FindPlan(ctx context.Context, id string) (domain.Plan, error) {
	var p domain.Plan
	err := s.db.QueryRowContext(ctx, `SELECT id,fund_code,fund_name,amount_cents,frequency,execution_day,start_date,status,version,created_at,updated_at FROM investment_plans WHERE id=?`, id).
		Scan(&p.ID, &p.FundCode, &p.FundName, &p.Amount, &p.Frequency, &p.ExecutionDay, &p.StartDate, &p.Status, &p.Version, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Plan{}, ErrNotFound
	}
	if err != nil {
		return domain.Plan{}, fmt.Errorf("finding plan: %w", err)
	}
	return p, nil
}

func (s *Store) UpdatePlanStatus(ctx context.Context, id, status string, expected int64) (domain.Plan, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := s.db.ExecContext(ctx, `UPDATE investment_plans SET status=?,version=version+1,updated_at=? WHERE id=? AND version=?`, status, now, id, expected)
	if err != nil {
		return domain.Plan{}, fmt.Errorf("updating plan status: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return domain.Plan{}, fmt.Errorf("checking plan update: %w", err)
	}
	if count != 1 {
		return domain.Plan{}, ErrConflict
	}
	return s.FindPlan(ctx, id)
}

func (s *Store) DeletePlan(ctx context.Context, id string, expected int64) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM investment_plans WHERE id=? AND version=?`, id, expected)
	if err != nil {
		return fmt.Errorf("deleting plan: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking plan delete: %w", err)
	}
	if count != 1 {
		return ErrConflict
	}
	return nil
}

// ---- 定投执行账本（幂等） ----

// RecordExecution 以 INSERT OR IGNORE 保证同日同计划只记录一次（幂等）。
// amount 为分（cents）、nav 为微元（micros）整数，直接写入 INTEGER 列避免类型漂移。
func (s *Store) RecordExecution(ctx context.Context, planID, scheduledDate, status string, amountCents, navMicros int64) (string, error) {
	id := newStoreID("exec")
	_, err := s.db.ExecContext(ctx, `INSERT OR IGNORE INTO plan_executions(id,plan_id,scheduled_date,status,amount_cents,nav_micros,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?)`,
		id, planID, scheduledDate, status, amountCents, navMicros, time.Now().UTC().Format(time.RFC3339), time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		return "", fmt.Errorf("recording execution: %w", err)
	}
	return id, nil
}

func (s *Store) UpdateExecutionResult(ctx context.Context, id, status string, navMicros int64, errorCode string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE plan_executions SET status=?,nav_micros=?,error_code=?,updated_at=? WHERE id=?`, status, navMicros, errorCode, time.Now().UTC().Format(time.RFC3339), id)
	if err != nil {
		return fmt.Errorf("updating execution: %w", err)
	}
	return nil
}

func (s *Store) ExecutionExists(ctx context.Context, planID, scheduledDate string) (bool, error) {
	var one int
	err := s.db.QueryRowContext(ctx, `SELECT 1 FROM plan_executions WHERE plan_id=? AND scheduled_date=? LIMIT 1`, planID, scheduledDate).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("checking execution: %w", err)
	}
	return true, nil
}

func (s *Store) ListExecutions(ctx context.Context, planID string, limit int) ([]domain.Execution, error) {
	query := `SELECT id,plan_id,scheduled_date,status,amount_cents,nav_micros,error_code,created_at FROM plan_executions`
	args := []any{}
	if planID != "" {
		query += ` WHERE plan_id=?`
		args = append(args, planID)
	}
	query += ` ORDER BY scheduled_date DESC`
	if limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing executions: %w", err)
	}
	defer rows.Close()
	executions := []domain.Execution{}
	for rows.Next() {
		var e domain.Execution
		var errorCode sql.NullString
		if err := rows.Scan(&e.ID, &e.PlanID, &e.ScheduledDate, &e.Status, &e.Amount, &e.NAV, &errorCode, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning execution: %w", err)
		}
		e.ErrorCode = errorCode.String
		executions = append(executions, e)
	}
	return executions, rows.Err()
}

// ---- 设置 ----

func (s *Store) GetSetting(ctx context.Context, key string) (string, error) {
	var value string
	err := s.db.QueryRowContext(ctx, `SELECT value FROM settings WHERE key=?`, key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("getting setting: %w", err)
	}
	return value, nil
}

func (s *Store) SetSetting(ctx context.Context, key, value string) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO settings(key,value,updated_at) VALUES(?,?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value,updated_at=excluded.updated_at`,
		key, value, time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("setting setting: %w", err)
	}
	return nil
}

func (s *Store) ListSettings(ctx context.Context) (map[string]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT key,value FROM settings`)
	if err != nil {
		return nil, fmt.Errorf("listing settings: %w", err)
	}
	defer rows.Close()
	result := map[string]string{}
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, fmt.Errorf("scanning setting: %w", err)
		}
		result[key] = value
	}
	return result, rows.Err()
}

// ---- 自选行情 ----

func (s *Store) ListUserIndices(ctx context.Context) ([]domain.Instrument, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT symbol,name,market FROM user_indices ORDER BY added_at`)
	if err != nil {
		return nil, fmt.Errorf("listing user indices: %w", err)
	}
	defer rows.Close()
	instruments := []domain.Instrument{}
	for rows.Next() {
		var in domain.Instrument
		if err := rows.Scan(&in.Code, &in.Name, &in.Market); err != nil {
			return nil, fmt.Errorf("scanning user index: %w", err)
		}
		in.AssetType = "index"
		instruments = append(instruments, in)
	}
	return instruments, rows.Err()
}

func (s *Store) AddUserIndex(ctx context.Context, symbol, name, market string) error {
	_, err := s.db.ExecContext(ctx, `INSERT OR IGNORE INTO user_indices(symbol,name,market,added_at) VALUES(?,?,?,?)`,
		symbol, name, market, time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("adding user index: %w", err)
	}
	return nil
}

func (s *Store) RemoveUserIndex(ctx context.Context, symbol string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM user_indices WHERE symbol=?`, symbol)
	if err != nil {
		return fmt.Errorf("removing user index: %w", err)
	}
	return nil
}

// ---- 搜索历史 ----

func (s *Store) ListSearchHistory(ctx context.Context, limit int) ([]domain.SearchRecord, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := s.db.QueryContext(ctx, `SELECT keyword,asset_type,searched_at FROM search_history ORDER BY searched_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("listing search history: %w", err)
	}
	defer rows.Close()
	records := []domain.SearchRecord{}
	for rows.Next() {
		var r domain.SearchRecord
		if err := rows.Scan(&r.Keyword, &r.AssetType, &r.SearchedAt); err != nil {
			return nil, fmt.Errorf("scanning search history: %w", err)
		}
		records = append(records, r)
	}
	return records, rows.Err()
}

func (s *Store) AddSearchHistory(ctx context.Context, keyword, assetType string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := s.db.ExecContext(ctx, `INSERT INTO search_history(keyword,asset_type,searched_at) VALUES(?,?,?) ON CONFLICT(keyword) DO UPDATE SET searched_at=excluded.searched_at`, keyword, assetType, now); err != nil {
		return fmt.Errorf("adding search history: %w", err)
	}
	// 仅保留最近 10 条
	if _, err := s.db.ExecContext(ctx, `DELETE FROM search_history WHERE keyword NOT IN (SELECT keyword FROM search_history ORDER BY searched_at DESC LIMIT 10)`); err != nil {
		return fmt.Errorf("trimming search history: %w", err)
	}
	return nil
}

// ---- 资产历史 ----

func (s *Store) UpsertAssetHistory(ctx context.Context, date string, marketValue, cost, dayProfit int64) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO asset_history(date,total_market_value_micros,total_cost_micros,day_profit_micros) VALUES(?,?,?,?) ON CONFLICT(date) DO UPDATE SET total_market_value_micros=excluded.total_market_value_micros,total_cost_micros=excluded.total_cost_micros,day_profit_micros=excluded.day_profit_micros`,
		date, marketValue, cost, dayProfit)
	if err != nil {
		return fmt.Errorf("upserting asset history: %w", err)
	}
	return nil
}

func (s *Store) ListAssetHistory(ctx context.Context) ([]domain.AssetPoint, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT date,total_market_value_micros,total_cost_micros,day_profit_micros FROM asset_history ORDER BY date`)
	if err != nil {
		return nil, fmt.Errorf("listing asset history: %w", err)
	}
	defer rows.Close()
	points := []domain.AssetPoint{}
	for rows.Next() {
		var p domain.AssetPoint
		if err := rows.Scan(&p.Date, &p.TotalMarketValue, &p.TotalCost, &p.DayProfit); err != nil {
			return nil, fmt.Errorf("scanning asset history: %w", err)
		}
		points = append(points, p)
	}
	return points, rows.Err()
}

// ---- 分时 ticks ----

func (s *Store) UpsertTick(ctx context.Context, code, recordTime string, pctBP, priceMicros int64) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO intraday_ticks(fund_code,record_time,pct_bp,price_micros) VALUES(?,?,?,?) ON CONFLICT(fund_code,record_time) DO UPDATE SET pct_bp=excluded.pct_bp,price_micros=excluded.price_micros`,
		code, recordTime, pctBP, priceMicros)
	if err != nil {
		return fmt.Errorf("upserting tick: %w", err)
	}
	return nil
}

func (s *Store) ListTicks(ctx context.Context, code string) ([]domain.SeriesPoint, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT record_time,pct_bp,price_micros FROM intraday_ticks WHERE fund_code=? ORDER BY record_time`, code)
	if err != nil {
		return nil, fmt.Errorf("listing ticks: %w", err)
	}
	defer rows.Close()
	points := []domain.SeriesPoint{}
	for rows.Next() {
		var p domain.SeriesPoint
		var timeStr string
		if err := rows.Scan(&timeStr, &p.Close, &p.High); err != nil {
			return nil, fmt.Errorf("scanning tick: %w", err)
		}
		p.Time = timeStr
		points = append(points, p)
	}
	return points, rows.Err()
}

// ---- 基金日绩效 ----

func (s *Store) UpsertFundPerformance(ctx context.Context, code, date string, nav, growthBP, confirmedNAV int64) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO fund_daily_performance(fund_code,date,nav_micros,daily_growth_bp,confirmed_nav_micros) VALUES(?,?,?,?,?) ON CONFLICT(fund_code,date) DO UPDATE SET nav_micros=excluded.nav_micros,daily_growth_bp=excluded.daily_growth_bp,confirmed_nav_micros=excluded.confirmed_nav_micros`,
		code, date, nav, growthBP, confirmedNAV)
	if err != nil {
		return fmt.Errorf("upserting fund performance: %w", err)
	}
	return nil
}

func (s *Store) ListFundPerformance(ctx context.Context, code string, limit int) ([]domain.SeriesPoint, error) {
	if limit <= 0 {
		limit = 365
	}
	rows, err := s.db.QueryContext(ctx, `SELECT date,nav_micros,daily_growth_bp FROM fund_daily_performance WHERE fund_code=? ORDER BY date DESC LIMIT ?`, code, limit)
	if err != nil {
		return nil, fmt.Errorf("listing fund performance: %w", err)
	}
	defer rows.Close()
	points := []domain.SeriesPoint{}
	for rows.Next() {
		var p domain.SeriesPoint
		if err := rows.Scan(&p.Time, &p.Close, &p.High); err != nil {
			return nil, fmt.Errorf("scanning fund performance: %w", err)
		}
		points = append(points, p)
	}
	// 升序返回
	for i, j := 0, len(points)-1; i < j; i, j = i+1, j-1 {
		points[i], points[j] = points[j], points[i]
	}
	return points, rows.Err()
}
