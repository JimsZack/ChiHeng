package service

import (
	"testing"
	"time"
)

func TestEffectiveTradingDate_whenBefore15(t *testing.T) {
	t.Parallel()
	// 2026-08-13 是周四，14:30 应归属当日
	now := time.Date(2026, 8, 13, 14, 30, 0, 0, time.Local)
	got := EffectiveTradingDate(now)
	want := time.Date(2026, 8, 13, 0, 0, 0, 0, time.Local)
	if !got.Equal(want) {
		t.Fatalf("EffectiveTradingDate() = %v, want %v", got, want)
	}
}

func TestEffectiveTradingDate_whenAfter15(t *testing.T) {
	t.Parallel()
	// 2026-08-13 周四 15:00 后 → 归属周五 08-14
	now := time.Date(2026, 8, 13, 15, 30, 0, 0, time.Local)
	got := EffectiveTradingDate(now)
	want := time.Date(2026, 8, 14, 0, 0, 0, 0, time.Local)
	if !got.Equal(want) {
		t.Fatalf("EffectiveTradingDate() = %v, want %v", got, want)
	}
}

func TestEffectiveTradingDate_whenFridayAfter15(t *testing.T) {
	t.Parallel()
	// 2026-08-14 周五 16:00 → 跳过周末 → 下周一 08-17
	now := time.Date(2026, 8, 14, 16, 0, 0, 0, time.Local)
	got := EffectiveTradingDate(now)
	want := time.Date(2026, 8, 17, 0, 0, 0, 0, time.Local)
	if !got.Equal(want) {
		t.Fatalf("EffectiveTradingDate() = %v, want %v", got, want)
	}
}

func TestEffectiveTradingDate_whenWeekend(t *testing.T) {
	t.Parallel()
	// 2026-08-15 周六 → 下周一 08-17
	now := time.Date(2026, 8, 15, 10, 0, 0, 0, time.Local)
	got := EffectiveTradingDate(now)
	want := time.Date(2026, 8, 17, 0, 0, 0, 0, time.Local)
	if !got.Equal(want) {
		t.Fatalf("EffectiveTradingDate() = %v, want %v", got, want)
	}
}

func TestIsTradingDay(t *testing.T) {
	t.Parallel()
	if !IsTradingDay(time.Date(2026, 8, 13, 0, 0, 0, 0, time.Local)) {
		t.Fatal("2026-08-13 应为交易日")
	}
	if IsTradingDay(time.Date(2026, 8, 15, 0, 0, 0, 0, time.Local)) {
		t.Fatal("2026-08-15 周六不应为交易日")
	}
}
