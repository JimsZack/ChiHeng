package service

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/JimsZack/ChiHeng/internal/domain"
)

// 持仓 CSV 列顺序（导出与导入共用同一契约，保证往返一致）。
const (
	csvColCode      = 0 // fund_code
	csvColName      = 1 // fund_name
	csvColShares    = 2 // shares（十进制字符串）
	csvColCostNAV   = 3 // cost_nav（十进制字符串）
	csvColOpenedOn  = 4 // opened_on（YYYY-MM-DD）
	csvColCount     = 5
)

// ExportHoldingsCSV 将持仓序列化为 CSV 文本（UTF-8，带表头）。
func ExportHoldingsCSV(holdings []domain.Holding) (string, error) {
	var builder strings.Builder
	writer := csv.NewWriter(&builder)
	if err := writer.Write([]string{"fund_code", "fund_name", "shares", "cost_nav", "opened_on"}); err != nil {
		return "", fmt.Errorf("write csv header: %w", err)
	}
	for _, h := range holdings {
		record := []string{
			h.FundCode,
			h.FundName,
			domain.FormatDecimal(h.Shares),
			domain.FormatDecimal(h.CostNAV),
			h.OpenedOn,
		}
		if err := writer.Write(record); err != nil {
			return "", fmt.Errorf("write csv holding %s: %w", h.ID, err)
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return "", fmt.Errorf("flush csv: %w", err)
	}
	return builder.String(), nil
}

// ParseHoldingsCSV 解析持仓 CSV 文本并校验，返回解析后的持仓与错误行数。
// 跳过空行与表头；任一行字段缺失或金额非法即整体返回错误（保证数据一致）。
func ParseHoldingsCSV(content string) ([]domain.Holding, error) {
	reader := csv.NewReader(strings.NewReader(content))
	reader.FieldsPerRecord = csvColCount
	holdings := []domain.Holding{}
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("解析 CSV 失败: %w", err)
		}
		if isCSVHeader(record) {
			continue
		}
		holding, err := holdingFromCSV(record)
		if err != nil {
			return nil, err
		}
		holdings = append(holdings, holding)
	}
	return holdings, nil
}

func isCSVHeader(record []string) bool {
	if len(record) == 0 {
		return true
	}
	first := strings.ToLower(strings.TrimSpace(record[0]))
	return first == "fund_code" || first == "代码" || first == "基金代码"
}

func holdingFromCSV(record []string) (domain.Holding, error) {
	shares, err := domain.ParseDecimal(strings.TrimSpace(record[csvColShares]))
	if err != nil {
		return domain.Holding{}, fmt.Errorf("份额格式无效（行 %q）", strings.Join(record, ","))
	}
	cost, err := domain.ParseDecimal(strings.TrimSpace(record[csvColCostNAV]))
	if err != nil {
		return domain.Holding{}, fmt.Errorf("成本净值格式无效（行 %q）", strings.Join(record, ","))
	}
	code := strings.TrimSpace(record[csvColCode])
	if code == "" {
		return domain.Holding{}, fmt.Errorf("基金代码为空（行 %q）", strings.Join(record, ","))
	}
	openedOn := strings.TrimSpace(record[csvColOpenedOn])
	if openedOn == "" {
		openedOn = time.Now().UTC().Format("2006-01-02")
	}
	name := strings.TrimSpace(record[csvColName])
	if name == "" {
		name = code
	}
	return domain.Holding{
		FundCode: code,
		FundName: name,
		Shares:   shares,
		CostNAV:  cost,
		OpenedOn: openedOn,
	}, nil
}

// CSVImportResult 批量导入结果。
type CSVImportResult struct {
	Imported int
	Ignored  int
}

// ImportHoldingsCSV 批量导入持仓（宽松模式）：逐行解析，坏行跳过并计数，
// 好行经 Portfolio.Create 创建；与严格解析的 ParseHoldingsCSV 语义区分。
func (p *Portfolio) ImportHoldingsCSV(ctx context.Context, content string) (CSVImportResult, error) {
	reader := csv.NewReader(strings.NewReader(content))
	reader.FieldsPerRecord = csvColCount
	result := CSVImportResult{}
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			// 列数不符的行按结构错误跳过，不中断整体导入
			result.Ignored++
			continue
		}
		if isCSVHeader(record) {
			continue
		}
		holding, err := holdingFromCSV(record)
		if err != nil {
			result.Ignored++
			continue
		}
		if _, err := p.Create(ctx, holding); err != nil {
			if err == domain.ErrInvalidDecimal {
				result.Ignored++
				continue
			}
			return result, err
		}
		result.Imported++
	}
	return result, nil
}
