package service

import (
	"context"
	"strings"
	"testing"

	"github.com/chiheng-app/chiheng/internal/domain"
)

func TestExportHoldingsCSV_whenEmpty(t *testing.T) {
	t.Parallel()
	content, err := ExportHoldingsCSV(nil)
	if err != nil {
		t.Fatalf("ExportHoldingsCSV() error = %v", err)
	}
	if !strings.HasPrefix(content, "fund_code,fund_name,shares,cost_nav,opened_on") {
		t.Fatalf("ExportHoldingsCSV() header = %q", content)
	}
}

func TestCSVRoundTrip(t *testing.T) {
	t.Parallel()
	holdings := []domain.Holding{
		{FundCode: "000001", FundName: "测试基金A", Shares: 100_000_000, CostNAV: 1_234_567, OpenedOn: "2026-08-13"},
		{FundCode: "000002", FundName: "测试基金B", Shares: 50_500_000, CostNAV: 2_000_000, OpenedOn: "2026-07-01"},
	}
	content, err := ExportHoldingsCSV(holdings)
	if err != nil {
		t.Fatalf("ExportHoldingsCSV() error = %v", err)
	}
	parsed, err := ParseHoldingsCSV(content)
	if err != nil {
		t.Fatalf("ParseHoldingsCSV() error = %v", err)
	}
	if len(parsed) != 2 {
		t.Fatalf("ParseHoldingsCSV() count = %d, want 2", len(parsed))
	}
	if parsed[0].FundCode != "000001" || parsed[0].Shares != 100_000_000 || parsed[0].CostNAV != 1_234_567 {
		t.Fatalf("ParseHoldingsCSV()[0] = %+v", parsed[0])
	}
	if parsed[1].FundName != "测试基金B" || parsed[1].OpenedOn != "2026-07-01" {
		t.Fatalf("ParseHoldingsCSV()[1] = %+v", parsed[1])
	}
}

func TestParseHoldingsCSV_whenHeaderSkipped(t *testing.T) {
	t.Parallel()
	content := "fund_code,fund_name,shares,cost_nav,opened_on\n000001,测试基金,100.00,1.2345,2026-08-13\n"
	parsed, err := ParseHoldingsCSV(content)
	if err != nil {
		t.Fatalf("ParseHoldingsCSV() error = %v", err)
	}
	if len(parsed) != 1 {
		t.Fatalf("ParseHoldingsCSV() count = %d, want 1 (header skipped)", len(parsed))
	}
}

func TestParseHoldingsCSV_whenInvalidShares(t *testing.T) {
	t.Parallel()
	content := "000001,测试基金,abc,1.2345,2026-08-13\n"
	if _, err := ParseHoldingsCSV(content); err == nil {
		t.Fatal("ParseHoldingsCSV() error = nil, want error for invalid shares")
	}
}

func TestImportHoldingsCSV(t *testing.T) {
	t.Parallel()
	db := openStore(t)
	portfolio := NewPortfolio(db)
	ctx := context.Background()

	content := "fund_code,fund_name,shares,cost_nav,opened_on\n" +
		"000001,测试基金A,100.00,1.2345,2026-08-13\n" +
		"000002,测试基金B,50.00,2.0000,2026-07-01\n"
	result, err := portfolio.ImportHoldingsCSV(ctx, content)
	if err != nil {
		t.Fatalf("ImportHoldingsCSV() error = %v", err)
	}
	if result.Imported != 2 || result.Ignored != 0 {
		t.Fatalf("ImportHoldingsCSV() = %+v, want 2 imported", result)
	}
	holdings, err := portfolio.List(ctx)
	if err != nil || len(holdings) != 2 {
		t.Fatalf("portfolio holdings = %d, err = %v; want 2", len(holdings), err)
	}
}

func TestImportHoldingsCSV_whenInvalidRowIgnored(t *testing.T) {
	t.Parallel()
	db := openStore(t)
	portfolio := NewPortfolio(db)
	ctx := context.Background()

	// 第二行份额非法 → 忽略；第一行应导入成功
	content := "000001,测试基金A,100.00,1.2345,2026-08-13\n000002,测试基金B,abc,2.0000,2026-07-01\n"
	result, err := portfolio.ImportHoldingsCSV(ctx, content)
	if err != nil {
		t.Fatalf("ImportHoldingsCSV() error = %v", err)
	}
	if result.Imported != 1 || result.Ignored != 1 {
		t.Fatalf("ImportHoldingsCSV() = %+v, want 1 imported / 1 ignored", result)
	}
}
