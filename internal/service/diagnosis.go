package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/chiheng-app/chiheng/internal/domain"
)

// Diagnosis 基金诊断域：本地量化评分 + 规则报告 + 可选 AI 深度分析。
type Diagnosis struct {
	chat ChatClient // 可选，nil 时 AI 功能不可用
}

// ChatClient AI 聊天接口（由 deepseek 适配器实现）。
type ChatClient interface {
	Chat(ctx context.Context, system, user string) (string, error)
}

func NewDiagnosis(chat ChatClient) *Diagnosis {
	return &Diagnosis{chat: chat}
}

// Score 由历史净值序列计算量化指标：收益、最大回撤、夏普（bp 单位）。
type Score struct {
	ReturnBP    int64
	MaxDrawdownBP int64
	Sharpe      float64
	Total       int
}

// ComputeScore 计算近一年（约 250 个交易日）指标并输出 1–5 星评分。
func ComputeScore(series []domain.SeriesPoint) Score {
	if len(series) < 2 {
		return Score{Total: 1}
	}
	start := series[0].Close
	end := series[len(series)-1].Close
	if start == 0 {
		return Score{Total: 1}
	}
	retBP := (end - start) * 10000 / start

	// 最大回撤
	peak := series[0].Close
	maxDD := int64(0)
	for _, p := range series {
		if p.Close > peak {
			peak = p.Close
		}
		dd := int64(0)
		if peak > 0 {
			dd = (p.Close - peak) * 10000 / peak
		}
		if dd < maxDD {
			maxDD = dd
		}
	}
	// 夏普比率：基于日收益率（百分比收益）均值/标准差年化
	var sum, sumSq float64
	count := 0
	for i := 1; i < len(series); i++ {
		prev, cur := float64(series[i-1].Close), float64(series[i].Close)
		if prev <= 0 {
			continue
		}
		ret := (cur - prev) / prev * 100
		sum += ret
		sumSq += ret * ret
		count++
	}
	sharpe := 0.0
	if count > 1 {
		mean := sum / float64(count)
		variance := (sumSq - float64(count)*mean*mean) / float64(count-1)
		if variance > 0 {
			std := sqrtFloat(variance)
			sharpe = mean / std * sqrtFloat(float64(count))
		} else if mean > 0 {
			// 收益恒定无波动：视为极佳风险调整后表现
			sharpe = 10.0
		}
	}

	total := 3
	switch {
	case retBP >= 2000 && maxDD >= -1500 && sharpe >= 1.0:
		total = 5
	case retBP >= 1000 && maxDD >= -2000 && sharpe >= 0.5:
		total = 4
	case retBP <= -1500 || maxDD <= -3000 || sharpe < -0.5:
		total = 1
	case retBP <= -500 || maxDD <= -2500 || sharpe < 0:
		total = 2
	}
	return Score{ReturnBP: retBP, MaxDrawdownBP: maxDD, Sharpe: sharpe, Total: total}
}

func sqrtFloat(v float64) float64 {
	if v <= 0 {
		return 0
	}
	x := v
	for i := 0; i < 20; i++ {
		x = (x + v/x) / 2
	}
	return x
}

// DiagnoseLocal 生成本地规则诊断报告。
func (d *Diagnosis) DiagnoseLocal(fundName, fundCode string, series []domain.SeriesPoint) domain.DiagnosisResult {
	score := ComputeScore(series)
	conclusion := "综合表现中性"
	switch score.Total {
	case 5:
		conclusion = "综合表现优秀，具备较好的风险收益比"
	case 4:
		conclusion = "综合表现良好，整体优于同类平均水平"
	case 2:
		conclusion = "综合表现偏弱，需关注回撤与波动风险"
	case 1:
		conclusion = "综合表现较弱，建议谨慎评估后再配置"
	}
	metrics := []domain.DiagnosisMetric{
		{Key: "return", Label: "近一年收益", Value: formatBP(score.ReturnBP)},
		{Key: "drawdown", Label: "最大回撤", Value: formatBP(score.MaxDrawdownBP)},
		{Key: "sharpe", Label: "夏普比率", Value: fmt.Sprintf("%.2f", score.Sharpe)},
		{Key: "score", Label: "综合评分", Value: fmt.Sprintf("%d/5", score.Total)},
	}
	sections := []domain.ReportSection{
		{Title: "业绩表现", Content: fmt.Sprintf("%s 近一年累计收益 %s，最大回撤 %s。", fundName, formatBP(score.ReturnBP), formatBP(score.MaxDrawdownBP))},
		{Title: "风险评估", Content: fmt.Sprintf("夏普比率 %.2f，波动水平%s。", score.Sharpe, volWord(score.Sharpe))},
		{Title: "投资建议", Content: conclusion + "。定投可平滑择时风险，建议结合自身风险承受能力配置。"},
		{Title: "适合人群", Content: "适合认可该基金投资逻辑、能承受相应波动的中长期投资者。"},
	}
	return domain.DiagnosisResult{
		ReportID:   newID("diag"),
		Kind:       "local",
		Score:      score.Total,
		Conclusion: conclusion,
		Metrics:    metrics,
		Sections:   sections,
		CreatedAt:  time.Now().UTC().Format(time.RFC3339),
	}
}

// DiagnoseAI 生成 AI 深度诊断（仅当用户明确 consent 且已配置密钥时）。
func (d *Diagnosis) DiagnoseAI(ctx context.Context, fundName, fundCode string, series []domain.SeriesPoint, includeAmounts bool) (domain.DiagnosisResult, error) {
	if d.chat == nil {
		return domain.DiagnosisResult{}, ErrAINotConfigured
	}
	local := d.DiagnoseLocal(fundName, fundCode, series)
	var b strings.Builder
	b.WriteString("请以金融分析师视角，从业绩表现、风险评估、投资建议、适合人群四个维度分析该基金，输出约 500 字中文报告。\n")
	b.WriteString(fmt.Sprintf("基金：%s（%s）\n", fundName, fundCode))
	for _, m := range local.Metrics {
		b.WriteString(fmt.Sprintf("%s：%s\n", m.Label, m.Value))
	}
	b.WriteString("结论参考：" + local.Conclusion + "\n")
	if !includeAmounts {
		b.WriteString("注：以下为脱敏分析，不包含具体持仓金额。\n")
	}
	text, err := d.chat.Chat(ctx, "你是一位谨慎、专业的金融投资分析师。回答需客观、克制，不承诺收益。", b.String())
	if err != nil {
		return domain.DiagnosisResult{}, err
	}
	return domain.DiagnosisResult{
		ReportID:   newID("diag"),
		Kind:       "ai",
		Score:      local.Score,
		Conclusion: local.Conclusion,
		Metrics:    local.Metrics,
		Sections: []domain.ReportSection{
			{Title: "AI 深度分析", Content: text},
		},
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}, nil
}

// ErrAINotConfigured 表示未配置 AI 密钥。
var ErrAINotConfigured = fmt.Errorf("AI 未配置")

func formatBP(bp int64) string {
	sign := ""
	if bp > 0 {
		sign = "+"
	}
	return fmt.Sprintf("%s%.2f%%", sign, float64(bp)/100)
}

func volWord(sharpe float64) string {
	switch {
	case sharpe >= 1.0:
		return "较为平稳"
	case sharpe >= 0:
		return "波动适中"
	default:
		return "波动较大"
	}
}
