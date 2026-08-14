package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	_ "modernc.org/sqlite"

	"github.com/chiheng-app/chiheng/internal/domain"
)

var ErrConflict = errors.New("version conflict")

func newStoreID(prefix string) string {
	value := make([]byte, 12)
	if _, err := rand.Read(value); err != nil {
		return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
	}
	return prefix + "-" + hex.EncodeToString(value)
}

type Store struct {
	db *sql.DB
}

func Open(ctx context.Context, path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}
	db.SetMaxOpenConns(1)
	for _, pragma := range []string{"PRAGMA foreign_keys=ON", "PRAGMA journal_mode=WAL", "PRAGMA busy_timeout=5000"} {
		if _, err := db.ExecContext(ctx, pragma); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("configuring database: %w", err)
		}
	}
	return &Store{db: db}, nil
}

func (s *Store) DB() *sql.DB { return s.db }

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) ListHoldings(ctx context.Context) ([]domain.Holding, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,fund_code,fund_name,shares_micros,cost_nav_micros,current_nav_micros,daily_change_bp,opened_on,version,created_at,updated_at FROM holdings WHERE closed_at IS NULL ORDER BY created_at`)
	if err != nil {
		return nil, fmt.Errorf("listing holdings: %w", err)
	}
	defer rows.Close()
	holdings := []domain.Holding{}
	for rows.Next() {
		var holding domain.Holding
		if err := rows.Scan(&holding.ID, &holding.FundCode, &holding.FundName, &holding.Shares, &holding.CostNAV, &holding.CurrentNAV, &holding.DailyChangeBP, &holding.OpenedOn, &holding.Version, &holding.CreatedAt, &holding.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning holding: %w", err)
		}
		holdings = append(holdings, holding)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating holdings: %w", err)
	}
	return holdings, nil
}

func (s *Store) CreateHolding(ctx context.Context, holding domain.Holding) (domain.Holding, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.ExecContext(ctx, `INSERT INTO holdings(id,fund_code,fund_name,shares_micros,cost_nav_micros,current_nav_micros,daily_change_bp,opened_on,version,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,1,?,?)`, holding.ID, holding.FundCode, holding.FundName, holding.Shares, holding.CostNAV, holding.CurrentNAV, holding.DailyChangeBP, holding.OpenedOn, now, now)
	if err != nil {
		return domain.Holding{}, fmt.Errorf("creating holding: %w", err)
	}
	holding.Version, holding.CreatedAt, holding.UpdatedAt = 1, now, now
	return holding, nil
}

func (s *Store) UpdateHolding(ctx context.Context, holding domain.Holding, expected int64) (domain.Holding, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := s.db.ExecContext(ctx, `UPDATE holdings SET shares_micros=?,cost_nav_micros=?,current_nav_micros=?,version=version+1,updated_at=? WHERE id=? AND version=? AND closed_at IS NULL`, holding.Shares, holding.CostNAV, holding.CurrentNAV, now, holding.ID, expected)
	if err != nil {
		return domain.Holding{}, fmt.Errorf("updating holding: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return domain.Holding{}, fmt.Errorf("checking holding update: %w", err)
	}
	if count != 1 {
		return domain.Holding{}, ErrConflict
	}
	holding.Version, holding.UpdatedAt = expected+1, now
	return holding, nil
}

func (s *Store) FindHolding(ctx context.Context, id string) (domain.Holding, error) {
	var holding domain.Holding
	err := s.db.QueryRowContext(ctx, `SELECT id,fund_code,fund_name,shares_micros,cost_nav_micros,current_nav_micros,daily_change_bp,opened_on,version,created_at,updated_at FROM holdings WHERE id=? AND closed_at IS NULL`, id).Scan(&holding.ID, &holding.FundCode, &holding.FundName, &holding.Shares, &holding.CostNAV, &holding.CurrentNAV, &holding.DailyChangeBP, &holding.OpenedOn, &holding.Version, &holding.CreatedAt, &holding.UpdatedAt)
	if err != nil {
		return domain.Holding{}, fmt.Errorf("finding holding: %w", err)
	}
	return holding, nil
}

func (s *Store) DeleteHolding(ctx context.Context, id string, expected int64) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM holdings WHERE id=? AND version=?`, id, expected)
	if err != nil {
		return fmt.Errorf("deleting holding: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking holding delete: %w", err)
	}
	if count != 1 { return ErrConflict }
	return nil
}
