package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/chiheng-app/chiheng/internal/domain"
	"github.com/chiheng-app/chiheng/internal/store"
)

type Portfolio struct { store *store.Store }

func NewPortfolio(repository *store.Store) *Portfolio { return &Portfolio{store: repository} }

func (s *Portfolio) List(ctx context.Context) ([]domain.Holding, error) { return s.store.ListHoldings(ctx) }

func (s *Portfolio) Create(ctx context.Context, holding domain.Holding) (domain.Holding, error) {
	if holding.Shares <= 0 || holding.CostNAV <= 0 { return domain.Holding{}, domain.ErrInvalidDecimal }
	holding.ID = newID("holding")
	if holding.CurrentNAV == 0 { holding.CurrentNAV = holding.CostNAV }
	return s.store.CreateHolding(ctx, holding)
}

func (s *Portfolio) Buy(ctx context.Context, id string, shares, nav, expected int64) (domain.Holding, error) {
	if shares <= 0 || nav <= 0 { return domain.Holding{}, domain.ErrInvalidDecimal }
	holding, err := s.store.FindHolding(ctx, id)
	if err != nil { return domain.Holding{}, err }
	oldCost, err := domain.Multiply(holding.Shares, holding.CostNAV)
	if err != nil { return domain.Holding{}, fmt.Errorf("calculating current cost: %w", err) }
	newCost, err := domain.Multiply(shares, nav)
	if err != nil { return domain.Holding{}, fmt.Errorf("calculating purchase cost: %w", err) }
	holding.Shares += shares
	holding.CostNAV, err = domain.Divide(oldCost+newCost, holding.Shares)
	if err != nil { return domain.Holding{}, fmt.Errorf("calculating weighted cost: %w", err) }
	holding.CurrentNAV = nav
	return s.store.UpdateHolding(ctx, holding, expected)
}

func (s *Portfolio) Sell(ctx context.Context, id string, shares, nav, expected int64) (domain.Holding, error) {
	if shares <= 0 || nav <= 0 { return domain.Holding{}, domain.ErrInvalidDecimal }
	holding, err := s.store.FindHolding(ctx, id)
	if err != nil { return domain.Holding{}, err }
	if shares > holding.Shares { return domain.Holding{}, domain.ErrInvalidDecimal }
	holding.Shares -= shares
	holding.CurrentNAV = nav
	if holding.Shares == 0 {
		if err := s.store.DeleteHolding(ctx, id, expected); err != nil { return domain.Holding{}, err }
		return holding, nil
	}
	return s.store.UpdateHolding(ctx, holding, expected)
}

func (s *Portfolio) Correct(ctx context.Context, holding domain.Holding, expected int64) (domain.Holding, error) {
	if holding.Shares <= 0 || holding.CostNAV <= 0 { return domain.Holding{}, domain.ErrInvalidDecimal }
	current, err := s.store.FindHolding(ctx, holding.ID)
	if err != nil { return domain.Holding{}, err }
	current.Shares, current.CostNAV = holding.Shares, holding.CostNAV
	return s.store.UpdateHolding(ctx, current, expected)
}

func (s *Portfolio) Delete(ctx context.Context, id string, expected int64) error { return s.store.DeleteHolding(ctx, id, expected) }

func newID(prefix string) string {
	value := make([]byte, 12)
	if _, err := rand.Read(value); err != nil { return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano()) }
	return prefix + "-" + hex.EncodeToString(value)
}

func IsConflict(err error) bool { return errors.Is(err, store.ErrConflict) }
