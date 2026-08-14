package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/chiheng-app/chiheng/internal/domain"
)

func TestComputeScore_whenStrongUptrend(t *testing.T) {
	t.Parallel()
	series := make([]domain.SeriesPoint, 250)
	for i := range series {
		series[i] = domain.SeriesPoint{Time: "2026-01-01", Close: 1_000_000 + int64(i)*8_000}
	}
	score := ComputeScore(series)
	if score.Total < 4 {
		t.Fatalf("ComputeScore() total = %d, want >= 4 for strong uptrend", score.Total)
	}
	if score.ReturnBP <= 0 {
		t.Fatalf("ComputeScore() return = %d, want positive", score.ReturnBP)
	}
}

func TestComputeScore_whenDowntrend(t *testing.T) {
	t.Parallel()
	series := make([]domain.SeriesPoint, 250)
	for i := range series {
		series[i] = domain.SeriesPoint{Time: "2026-01-01", Close: 1_000_000 - int64(i)*8_000}
	}
	score := ComputeScore(series)
	if score.Total > 2 {
		t.Fatalf("ComputeScore() total = %d, want <= 2 for downtrend", score.Total)
	}
}

func TestComputeScore_whenTooFewPoints(t *testing.T) {
	t.Parallel()
	score := ComputeScore([]domain.SeriesPoint{{Time: "2026-01-01", Close: 1_000_000}})
	if score.Total < 1 {
		t.Fatalf("ComputeScore() total = %d, want >= 1", score.Total)
	}
}

func TestDiagnoseLocal_whenNoSeries(t *testing.T) {
	t.Parallel()
	d := NewDiagnosis(nil)
	result := d.DiagnoseLocal("测试基金", "000001", nil)
	if result.ReportID == "" || result.Kind != "local" {
		t.Fatalf("DiagnoseLocal() = %+v, want report with local kind", result)
	}
	if len(result.Metrics) != 4 || len(result.Sections) != 4 {
		t.Fatalf("DiagnoseLocal() metrics=%d sections=%d, want 4/4",
			len(result.Metrics), len(result.Sections))
	}
}

func TestDiagnoseAI_whenNotConfigured(t *testing.T) {
	t.Parallel()
	d := NewDiagnosis(nil)
	_, err := d.DiagnoseAI(context.Background(), "基金", "000001", nil, false)
	if !errors.Is(err, ErrAINotConfigured) {
		t.Fatalf("DiagnoseAI() error = %v, want ErrAINotConfigured", err)
	}
}

type fakeChat struct {
	text string
}

func (f *fakeChat) Chat(_ context.Context, _, _ string) (string, error) {
	return f.text, nil
}

func TestDiagnoseAI_whenConfigured(t *testing.T) {
	t.Parallel()
	d := NewDiagnosis(&fakeChat{text: "AI 分析报告内容"})
	result, err := d.DiagnoseAI(context.Background(), "基金", "000001", nil, false)
	if err != nil {
		t.Fatalf("DiagnoseAI() error = %v", err)
	}
	if result.Kind != "ai" || len(result.Sections) != 1 {
		t.Fatalf("DiagnoseAI() = %+v, want ai kind with one section", result)
	}
	if !strings.Contains(result.Sections[0].Content, "AI 分析报告内容") {
		t.Fatalf("DiagnoseAI() section missing AI content: %+v", result.Sections)
	}
}
