// Package fundgz 实现天天基金实时估值数据源（fundgz.1234567.com.cn）。
//
// 实现情况：
//   - GetEstimate: http://fundgz.1234567.com.cn/js/<code>.js 返回 jsonpgz(...) JSONP，
//     解析后取 gsz（估算值）、gszzl（估算涨跌幅 %）、gztime（估值时间）字段。
//
// 说明：
//   - 本包仅提供实时估值查询（返回 adapter.Quote 形式），不实现 MarketProvider 接口。
//   - 金额换算为微元 int64（1e6），涨跌幅换算为 bp。
//   - 请求受 ctx 控制，客户端整体超时 10s。
package fundgz

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/JimsZack/ChiHeng/internal/adapter"
)

// microScale 金额微元换算系数（1 元 = 1e6 微元）。
const microScale = 1_000_000

// jsonpPrefix JSONP 包装前缀。
const jsonpPrefix = "jsonpgz("

// Client 天天基金实时估值客户端。
type Client struct {
	httpClient *http.Client
}

// New 创建天天基金实时估值客户端。
func New() *Client {
	return &Client{httpClient: &http.Client{Timeout: 10 * time.Second}}
}

// estimate fundgz JSONP 载荷。
type estimate struct {
	FundCode string `json:"fundcode"` // 基金代码
	Name     string `json:"name"`     // 基金名称
	DWJZ     string `json:"dwjz"`     // 单位净值（前一交易日）
	GSZ      string `json:"gsz"`      // 估算值
	GSZZL    string `json:"gszzl"`    // 估算涨跌幅（%）
	GZTime   string `json:"gztime"`   // 估值时间（如 2024-01-01 15:00）
}

// GetEstimate 查询基金实时估值，返回 adapter.Quote 形式结果。
func (c *Client) GetEstimate(ctx context.Context, code string) (adapter.Quote, error) {
	if code == "" {
		return adapter.Quote{}, fmt.Errorf("fundgz estimate: empty code")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://fundgz.1234567.com.cn/js/"+code+".js", nil)
	if err != nil {
		return adapter.Quote{}, fmt.Errorf("fundgz estimate %s: %w", code, err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return adapter.Quote{}, fmt.Errorf("fundgz estimate %s: %w", code, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return adapter.Quote{}, fmt.Errorf("fundgz estimate %s: unexpected status %s", code, resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return adapter.Quote{}, fmt.Errorf("fundgz estimate %s: %w", code, err)
	}
	raw, err := extractJSONP(body)
	if err != nil {
		return adapter.Quote{}, fmt.Errorf("fundgz estimate %s: %w", code, err)
	}
	var est estimate
	if err := json.Unmarshal(raw, &est); err != nil {
		return adapter.Quote{}, fmt.Errorf("fundgz estimate %s: %w", code, err)
	}
	if est.FundCode == "" {
		return adapter.Quote{}, fmt.Errorf("fundgz estimate %s: empty payload", code)
	}
	gsz, _ := strconv.ParseFloat(est.GSZ, 64)
	zzl, _ := strconv.ParseFloat(est.GSZZL, 64)
	dwjz, _ := strconv.ParseFloat(est.DWJZ, 64)
	q := adapter.Quote{
		Code:          code,
		Name:          est.Name,
		Type:          adapter.TypeFund,
		Price:         toMicro(gsz),
		ChangeBP:      int64(math.Round(zzl * 100)),
		PreviousClose: toMicro(dwjz),
		Source:        "天天基金估值",
		SourceAt:      time.Now(),
	}
	if t, err := time.Parse("2006-01-02 15:04", est.GZTime); err == nil {
		q.SourceAt = t
	}
	return q, nil
}

// extractJSONP 从 jsonpgz({...}) 响应中提取花括号包裹的 JSON。
func extractJSONP(body []byte) ([]byte, error) {
	start := bytes.Index(body, []byte(jsonpPrefix))
	if start < 0 {
		return nil, fmt.Errorf("invalid jsonpgz response")
	}
	open := start + len(jsonpPrefix)
	end := bytes.LastIndex(body[open:], []byte(")"))
	if end < 0 {
		return nil, fmt.Errorf("invalid jsonpgz response")
	}
	return body[open : open+end], nil
}

// toMicro 金额（元）转微元 int64。
func toMicro(v float64) int64 {
	return int64(math.Round(v * microScale))
}
