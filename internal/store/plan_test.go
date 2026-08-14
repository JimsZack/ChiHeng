package store

import (
	"context"
	"errors"
	"testing"

	"github.com/chiheng-app/chiheng/internal/domain"
	"github.com/chiheng-app/chiheng/internal/schema"
)

func openStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(context.Background(), t.TempDir()+"/test.db")
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	if _, err := schema.EnsureSchema(context.Background(), s.DB()); err != nil {
		t.Fatalf("EnsureSchema() error = %v", err)
	}
	return s
}

func TestPlanCRUD(t *testing.T) {
	t.Parallel()
	s := openStore(t)
	ctx := context.Background()

	plan := domain.Plan{ID: "plan-1", FundCode: "000001", FundName: "测试基金", Amount: 100_00, Frequency: "monthly", ExecutionDay: 15, StartDate: "2026-08-01", Status: "active"}
	created, err := s.CreatePlan(ctx, plan)
	if err != nil {
		t.Fatalf("CreatePlan() error = %v", err)
	}
	if created.Version != 1 {
		t.Fatalf("CreatePlan() version = %d, want 1", created.Version)
	}

	found, err := s.FindPlan(ctx, "plan-1")
	if err != nil {
		t.Fatalf("FindPlan() error = %v", err)
	}
	if found.FundCode != "000001" || found.Amount != 100_00 {
		t.Fatalf("FindPlan() = %+v", found)
	}

	updated, err := s.UpdatePlanStatus(ctx, "plan-1", "paused", 1)
	if err != nil {
		t.Fatalf("UpdatePlanStatus() error = %v", err)
	}
	if updated.Status != "paused" || updated.Version != 2 {
		t.Fatalf("UpdatePlanStatus() = %+v", updated)
	}

	// 版本冲突应返回 ErrConflict
	if _, err := s.UpdatePlanStatus(ctx, "plan-1", "active", 1); !errors.Is(err, ErrConflict) {
		t.Fatalf("UpdatePlanStatus() stale error = %v, want ErrConflict", err)
	}

	if err := s.DeletePlan(ctx, "plan-1", 2); err != nil {
		t.Fatalf("DeletePlan() error = %v", err)
	}
	if _, err := s.FindPlan(ctx, "plan-1"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("FindPlan() after delete = %v, want ErrNotFound", err)
	}
}

func TestRecordExecution_whenIdempotent(t *testing.T) {
	t.Parallel()
	s := openStore(t)
	ctx := context.Background()

	// 先建计划以满足外键约束（plan_executions.plan_id -> investment_plans.id）
	plan := domain.Plan{ID: "plan-1", FundCode: "000001", FundName: "测试基金", Amount: 100_00, Frequency: "monthly", ExecutionDay: 15, StartDate: "2026-08-01", Status: "active"}
	if _, err := s.CreatePlan(ctx, plan); err != nil {
		t.Fatalf("CreatePlan() error = %v", err)
	}

	first, err := s.RecordExecution(ctx, "plan-1", "2026-08-13", "running", 100_00, 1_184_200)
	if err != nil {
		t.Fatalf("RecordExecution() error = %v", err)
	}
	second, err := s.RecordExecution(ctx, "plan-1", "2026-08-13", "running", 100_00, 1_184_200)
	if err != nil {
		t.Fatalf("RecordExecution() second error = %v", err)
	}
	_ = second

	exists, err := s.ExecutionExists(ctx, "plan-1", "2026-08-13")
	if err != nil || !exists {
		t.Fatalf("ExecutionExists() = %v, err = %v; want true", exists, err)
	}

	// 同日同计划只应有一条账本记录（幂等）
	executions, err := s.ListExecutions(ctx, "plan-1", 10)
	if err != nil {
		t.Fatalf("ListExecutions() error = %v", err)
	}
	if len(executions) != 1 {
		t.Fatalf("ListExecutions() count = %d, want 1 (idempotent)", len(executions))
	}
	if executions[0].ID != first {
		t.Fatalf("ListExecutions() id = %q, want %q", executions[0].ID, first)
	}
}

func TestSettingsStore(t *testing.T) {
	t.Parallel()
	s := openStore(t)
	ctx := context.Background()

	if err := s.SetSetting(ctx, "pref.theme", "dark"); err != nil {
		t.Fatalf("SetSetting() error = %v", err)
	}
	value, err := s.GetSetting(ctx, "pref.theme")
	if err != nil || value != "dark" {
		t.Fatalf("GetSetting() = %q, err = %v", value, err)
	}
	if err := s.SetSetting(ctx, "pref.theme", "light"); err != nil {
		t.Fatalf("SetSetting() update error = %v", err)
	}
	value, _ = s.GetSetting(ctx, "pref.theme")
	if value != "light" {
		t.Fatalf("GetSetting() after update = %q, want light", value)
	}
	all, err := s.ListSettings(ctx)
	if err != nil || all["pref.theme"] != "light" {
		t.Fatalf("ListSettings() = %v, err = %v", all, err)
	}
}

func TestUserIndices(t *testing.T) {
	t.Parallel()
	s := openStore(t)
	ctx := context.Background()

	if err := s.AddUserIndex(ctx, "sh600519", "贵州茅台", "cn"); err != nil {
		t.Fatalf("AddUserIndex() error = %v", err)
	}
	// 重复添加应幂等
	if err := s.AddUserIndex(ctx, "sh600519", "贵州茅台", "cn"); err != nil {
		t.Fatalf("AddUserIndex() duplicate error = %v", err)
	}
	list, err := s.ListUserIndices(ctx)
	if err != nil || len(list) != 1 {
		t.Fatalf("ListUserIndices() = %d items, err = %v; want 1", len(list), err)
	}
	if err := s.RemoveUserIndex(ctx, "sh600519"); err != nil {
		t.Fatalf("RemoveUserIndex() error = %v", err)
	}
	list, _ = s.ListUserIndices(ctx)
	if len(list) != 0 {
		t.Fatalf("ListUserIndices() after remove = %d, want 0", len(list))
	}
}

func TestSearchHistory_whenLimited(t *testing.T) {
	t.Parallel()
	s := openStore(t)
	ctx := context.Background()

	for i := 0; i < 15; i++ {
		if err := s.AddSearchHistory(ctx, "keyword"+string(rune('a'+i)), "fund"); err != nil {
			t.Fatalf("AddSearchHistory() error = %v", err)
		}
	}
	records, err := s.ListSearchHistory(ctx, 10)
	if err != nil {
		t.Fatalf("ListSearchHistory() error = %v", err)
	}
	if len(records) > 10 {
		t.Fatalf("ListSearchHistory() = %d, want <= 10", len(records))
	}
}

func TestAssetHistory(t *testing.T) {
	t.Parallel()
	s := openStore(t)
	ctx := context.Background()

	if err := s.UpsertAssetHistory(ctx, "2026-08-13", 286420360, 267778190, 1284200); err != nil {
		t.Fatalf("UpsertAssetHistory() error = %v", err)
	}
	points, err := s.ListAssetHistory(ctx)
	if err != nil || len(points) != 1 {
		t.Fatalf("ListAssetHistory() = %d, err = %v; want 1", len(points), err)
	}
	if points[0].TotalMarketValue != 286420360 {
		t.Fatalf("ListAssetHistory()[0] = %+v", points[0])
	}
}
