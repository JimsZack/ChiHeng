package service

import (
	"context"
	"testing"

	"github.com/JimsZack/ChiHeng/internal/domain"
	"github.com/JimsZack/ChiHeng/internal/store"
)

func Test_Buy_when_HoldingExists(t *testing.T) {
	// Given
	ctx := context.Background()
	repository, err := store.Open(ctx, t.TempDir()+"/test.db")
	if err != nil { t.Fatal(err) }
	defer repository.Close()
	if _, err := repository.DB().ExecContext(ctx, `CREATE TABLE holdings(id TEXT PRIMARY KEY,fund_code TEXT,fund_name TEXT,shares_micros INTEGER,cost_nav_micros INTEGER,current_nav_micros INTEGER,daily_change_bp INTEGER,opened_on TEXT,closed_at TEXT,version INTEGER,created_at TEXT,updated_at TEXT)`); err != nil { t.Fatal(err) }
	portfolio := NewPortfolio(repository)
	holding, err := portfolio.Create(ctx, domain.Holding{FundCode: "000001", FundName: "sample", Shares: 10*domain.Scale, CostNAV: domain.Scale, OpenedOn: "2026-01-01"})
	if err != nil { t.Fatal(err) }

	// When
	updated, err := portfolio.Buy(ctx, holding.ID, 10*domain.Scale, 2*domain.Scale, holding.Version)

	// Then
	if err != nil || updated.Shares != 20*domain.Scale || updated.CostNAV != 1_500_000 { t.Fatalf("updated=%+v err=%v", updated, err) }
}
