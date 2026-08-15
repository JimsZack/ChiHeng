// Package sina 实现新浪财经行情数据源。
//
// 数据端点为 https://hq.sinajs.cn/list=... 与 https://suggest3.sinajs.cn/suggest，
// 响应为 GBK 编码（经 golang.org/x/text/encoding/simplifiedchinese 解码），请求必须携带
// Referer: https://finance.sina.com.cn/ 头，否则新浪拒绝返回。金额/价格统一解析为
// int64 微元（scale=1e6），涨跌幅换算为基点（bp = 涨跌幅% * 100）。
//
// 已实现 MarketProvider 接口方法：
//   - Search: 通过 suggest3.sinajs.cn 关键字搜索，返回股票/指数/基金等标的。
//   - Quote:  通过 hq.sinajs.cn 实时报价，支持 A 股（sh/sz）、指数（s_/int_）、
//     外汇（USDCNY 等）、贵金属（hf_X/gds_/nf_AU/AG）与商品（hf_/nf_）。
//
// 额外导出的数据源方法：
//   - MarketIndex:    沪深300 指数。
//   - GlobalIndices:  全球主要指数，道琼斯/纳斯达克/标普500。
//   - CurrencyRates:  主要货币兑人民币汇率。
//   - PreciousMetals: 贵金属行情。
//   - CommodityPrices: 主要商品行情。
//
// 未实现 MarketProvider 接口方法（新浪 hq 接口不提供，返回哨兵错误 sina: not supported）：
//   - Intraday（分时）
//   - Kline（K 线）
//   - OrderBook（盘口）
//   - News（快讯）
package sina

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"golang.org/x/text/encoding/simplifiedchinese"

	"github.com/JimsZack/ChiHeng/internal/adapter"
	"github.com/JimsZack/ChiHeng/internal/domain"
)

const (
	hqURL      = "https://hq.sinajs.cn"
	suggestURL = "https://suggest3.sinajs.cn/suggest"
	referer    = "https://finance.sina.com.cn/"
	userAgent  = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"
	// httpTimeout 单次 HTTP 调用的超时时间。
	httpTimeout = 10 * time.Second
)

// errNotSupported 新浪不提供分时/盘口/快讯等数据时的哨兵错误。
var errNotSupported = errors.New("sina: not supported")

// Provider 新浪行情数据源，无全局状态，每次调用均带 ctx 并受超时约束。
type Provider struct {
	client *http.Client
}

var _ adapter.MarketProvider = (*Provider)(nil)

// New 构造新浪数据源 Provider。
func New() *Provider {
	return &Provider{
		client: &http.Client{Timeout: httpTimeout},
	}
}

// Search 按关键字搜索股票/指数/基金等标的（type 限定资产类型，空为全部）。
func (p *Provider) Search(ctx context.Context, keyword string, limit int) ([]adapter.Instrument, error) {
	if limit <= 0 {
		limit = 20
	}
	text, err := p.fetchRaw(ctx, suggestURL+"/type=&key="+url.QueryEscape(keyword))
	if err != nil {
		return nil, err
	}
	// 响应形如 var suggestvalue="sh600519,贵州茅台,600519,sh600519,贵州茅台,,贵州茅台,99,1,...;...";
	start := strings.Index(text, "\"")
	end := strings.Index(text[start+1:], "\"")
	if start < 0 || end < 0 {
		return nil, fmt.Errorf("sina: malformed search response")
	}
	payload := text[start+1 : start+1+end]
	out := make([]adapter.Instrument, 0, limit)
	for _, item := range strings.Split(payload, ";") {
		parts := strings.Split(item, ",")
		if len(parts) < 5 {
			continue
		}
		code := parts[3]
		market := marketOf(code)
		if market == "" {
			continue
		}
		out = append(out, adapter.Instrument{
			Code:   code,
			Name:   parts[4],
			Type:   instrumentType(market, parts[2]),
			Market: market,
		})
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

// Quote 查询单个 code 的实时报价。
func (p *Provider) Quote(ctx context.Context, code string) (adapter.Quote, error) {
	switch {
	case strings.HasPrefix(code, "s_"), strings.HasPrefix(code, "int_"):
		return p.indexQuote(ctx, code)
	case strings.HasPrefix(code, "sh"), strings.HasPrefix(code, "sz"):
		if isIndexCode(code) {
			return p.indexQuote(ctx, code)
		}
		return p.stockQuote(ctx, code)
	case strings.HasPrefix(code, "hf_"), strings.HasPrefix(code, "gds_"), strings.HasPrefix(code, "nf_"):
		return p.hfQuote(ctx, code, code, quoteTypeOf(code))
	default:
		return p.forexQuote(ctx, code)
	}
}

// Intraday 新浪 hq 接口不提供分时数据。
func (p *Provider) Intraday(ctx context.Context, code string) ([]adapter.Point, error) {
	return nil, errNotSupported
}

// Kline 新浪 hq 接口不提供 K 线数据。
func (p *Provider) Kline(ctx context.Context, code, period string, limit int) ([]adapter.Point, error) {
	return nil, errNotSupported
}

// OrderBook 新浪 hq 接口不提供盘口数据。
func (p *Provider) OrderBook(ctx context.Context, code string) ([]adapter.Level, error) {
	return nil, errNotSupported
}

// News 新浪不提供财经快讯接口。
func (p *Provider) News(ctx context.Context, limit int) ([]adapter.NewsItem, error) {
	return nil, errNotSupported
}

// MarketIndex 查询沪深300指数实时行情。
func (p *Provider) MarketIndex(ctx context.Context) (adapter.Quote, error) {
	return p.indexQuote(ctx, "s_sh000300")
}

// GlobalIndices 查询全球主要指数。
func (p *Provider) GlobalIndices(ctx context.Context) ([]adapter.Quote, error) {
	codes := []string{"int_dji", "int_nasdaq", "int_sp500"}
	data, err := p.fetchHqBatch(ctx, codes)
	if err != nil {
		return nil, err
	}
	out := make([]adapter.Quote, 0, len(codes))
	for _, code := range codes {
		text, ok := data[code]
		if !ok {
			continue
		}
		if q, err := parseIndexQuote(text, code); err == nil {
			out = append(out, q)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("sina: no global index data")
	}
	return out, nil
}

// CurrencyRates 查询主要货币兑人民币汇率。
func (p *Provider) CurrencyRates(ctx context.Context) ([]adapter.Quote, error) {
	pairs := []struct{ code, name string }{
		{"USDCNY", "美元人民币"},
		{"USDCNH", "美元离岸人民币"},
		{"EURCNY", "欧元人民币"},
		{"GBPCNY", "英镑人民币"},
		{"HKDCNY", "港元人民币"},
		{"AUDCNY", "澳元人民币"},
		{"CADCNY", "加元人民币"},
		{"SGDCNY", "新加坡元人民币"},
		{"CHFCNY", "瑞士法郎人民币"},
		{"JPYCNY", "日元人民币"},
	}
	return p.quoteBatch(ctx, pairs, func(text, code string) (adapter.Quote, error) {
		return parseForexQuote(text, code)
	}, "currency")
}

// PreciousMetals 查询贵金属行情。
func (p *Provider) PreciousMetals(ctx context.Context) ([]adapter.Quote, error) {
	list := []struct{ code, name string }{
		{"hf_XAU", "黄金现货"},
		{"hf_XAG", "白银现货"},
		{"gds_AUTD", "黄金延期"},
		{"gds_AGTD", "白银延期"},
		{"nf_AU0", "黄金期货"},
		{"nf_AG0", "白银期货"},
	}
	return p.quoteBatch(ctx, list, func(text, code string) (adapter.Quote, error) {
		return parseHFQuote(text, code, code, adapter.TypeMetal)
	}, "metal")
}

// CommodityPrices 查询主要商品行情。
func (p *Provider) CommodityPrices(ctx context.Context) ([]adapter.Quote, error) {
	list := []struct{ code, name string }{
		{"hf_OIL", "原油"},
		{"hf_CU", "铜"},
		{"hf_AL", "铝"},
		{"hf_ZN", "锌"},
		{"hf_NI", "镍"},
		{"hf_PB", "铅"},
		{"hf_SN", "锡"},
		{"hf_RB", "螺纹钢"},
		{"hf_HC", "热轧卷板"},
		{"hf_FU", "燃料油"},
		{"hf_BU", "沥青"},
		{"hf_RU", "橡胶"},
		{"nf_SC0", "原油期货"},
		{"nf_FU0", "燃料油期货"},
		{"nf_BU0", "沥青期货"},
	}
	return p.quoteBatch(ctx, list, func(text, code string) (adapter.Quote, error) {
		return parseHFQuote(text, code, code, adapter.TypeCommodity)
	}, "commodity")
}

// quoteBatch 批量请求并解析行情，单条失败/为空时跳过。
func (p *Provider) quoteBatch(ctx context.Context, list []struct{ code, name string }, parse func(text, code string) (adapter.Quote, error), kind string) ([]adapter.Quote, error) {
	codes := make([]string, 0, len(list))
	for _, item := range list {
		codes = append(codes, item.code)
	}
	data, err := p.fetchHqBatch(ctx, codes)
	if err != nil {
		return nil, err
	}
	out := make([]adapter.Quote, 0, len(list))
	for _, item := range list {
		text, ok := data[item.code]
		if !ok {
			continue
		}
		q, err := parse(text, item.code)
		if err != nil {
			continue
		}
		if q.Name == "" || q.Name == item.code {
			q.Name = item.name
		}
		out = append(out, q)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("sina: no %s data", kind)
	}
	return out, nil
}

// stockQuote 查询并解析 A 股实时行情。
func (p *Provider) stockQuote(ctx context.Context, code string) (adapter.Quote, error) {
	data, err := p.fetchHq(ctx, code)
	if err != nil {
		return adapter.Quote{}, err
	}
	parts := strings.Split(data, ",")
	if len(parts) < 32 {
		return adapter.Quote{}, fmt.Errorf("sina: malformed stock quote %s", code)
	}
	// 格式: 名称,今开,昨收,最新价,最高,最低,买一,卖一,成交量,成交额,买1量,买1价,...,日期,时间
	price := parseMoney(parts[3])
	q := adapter.Quote{
		Code:          code,
		Name:          parts[0],
		Type:          adapter.TypeStock,
		Price:         price,
		Open:          parseMoney(parts[1]),
		High:          parseMoney(parts[4]),
		Low:           parseMoney(parts[5]),
		PreviousClose: parseMoney(parts[2]),
		Volume:        parseCount(parts[8]),
		Amount:        parseMoney(parts[9]),
		Source:        "sina",
		SourceAt:      time.Now(),
	}
	q.ChangeBP = changeBP(q.Price, q.PreviousClose)
	return q, nil
}

// indexQuote 查询并解析指数实时行情。
func (p *Provider) indexQuote(ctx context.Context, code string) (adapter.Quote, error) {
	data, err := p.fetchHq(ctx, code)
	if err != nil {
		return adapter.Quote{}, err
	}
	return parseIndexQuote(data, code)
}

// parseIndexQuote 解析指数行情（格式: 名称,最新价,涨跌额,涨跌幅[,成交量,成交额]）。
func parseIndexQuote(text, code string) (adapter.Quote, error) {
	parts := strings.Split(text, ",")
	if len(parts) < 4 {
		return adapter.Quote{}, fmt.Errorf("sina: malformed index quote %s", code)
	}
	price := parseMoney(parts[1])
	return adapter.Quote{
		Code:          code,
		Name:          parts[0],
		Type:          adapter.TypeIndex,
		Price:         price,
		PreviousClose: price - parseMoney(parts[2]),
		ChangeBP:      pctToBP(parseMoney(parts[3])),
		Source:        "sina",
		SourceAt:      time.Now(),
	}, nil
}

// forexQuote 查询并解析外汇实时行情。
func (p *Provider) forexQuote(ctx context.Context, code string) (adapter.Quote, error) {
	data, err := p.fetchHq(ctx, code)
	if err != nil {
		return adapter.Quote{}, err
	}
	return parseForexQuote(data, code)
}

// parseForexQuote 解析外汇行情（格式: 时间,今开,最高,最低,成交量,买价,卖价,昨收,最新价,名称,日期）。
func parseForexQuote(text, code string) (adapter.Quote, error) {
	parts := strings.Split(text, ",")
	if len(parts) < 9 {
		return adapter.Quote{}, fmt.Errorf("sina: malformed forex quote %s", code)
	}
	q := adapter.Quote{
		Code:          code,
		Name:          parts[9],
		Type:          adapter.TypeForex,
		Price:         parseMoney(parts[8]),
		Open:          parseMoney(parts[1]),
		High:          parseMoney(parts[2]),
		Low:           parseMoney(parts[3]),
		PreviousClose: parseMoney(parts[7]),
		Volume:        parseCount(parts[4]),
		Source:        "sina",
		SourceAt:      time.Now(),
	}
	if q.Name == "" {
		q.Name = code
	}
	q.ChangeBP = changeBP(q.Price, q.PreviousClose)
	return q, nil
}

// hfQuote 查询并解析贵金属/商品实时行情。
func (p *Provider) hfQuote(ctx context.Context, code, name, typ string) (adapter.Quote, error) {
	data, err := p.fetchHq(ctx, code)
	if err != nil {
		return adapter.Quote{}, err
	}
	q, err := parseHFQuote(data, code, name, typ)
	if err != nil {
		return adapter.Quote{}, err
	}
	return q, nil
}

// parseHFQuote 解析贵金属/商品行情。
//   - nf_ 期货: 名称,时间,今开,最高,最低,结算,买价,卖价,最新价,...,昨结,...,持仓量,成交量,...；
//   - hf_/gds_: 最新价,今开,最高,最低,买价,卖价,时间,昨收,...,日期,名称。
func parseHFQuote(text, code, name, typ string) (adapter.Quote, error) {
	parts := strings.Split(text, ",")
	q := adapter.Quote{
		Code:     code,
		Name:     name,
		Type:     typ,
		Source:   "sina",
		SourceAt: time.Now(),
	}
	switch {
	case strings.HasPrefix(code, "nf_") && len(parts) >= 18:
		q.Name = parts[0]
		q.Price = parseMoney(parts[8])
		q.Open = parseMoney(parts[2])
		q.High = parseMoney(parts[3])
		q.Low = parseMoney(parts[4])
		q.PreviousClose = parseMoney(parts[10])
		q.Volume = parseCount(parts[14])
	case len(parts) >= 14:
		q.Name = parts[13]
		q.Price = parseMoney(parts[0])
		q.Open = parseMoney(parts[1])
		q.High = parseMoney(parts[2])
		q.Low = parseMoney(parts[3])
		q.PreviousClose = firstNumeric(parts[6], parts[7], parts[0])
		q.Volume = parseCount(parts[9])
		q.Amount = parseMoney(parts[10])
	default:
		return adapter.Quote{}, fmt.Errorf("sina: malformed %s quote %s", typ, code)
	}
	q.ChangeBP = changeBP(q.Price, q.PreviousClose)
	return q, nil
}

// fetchHq 请求单个 code 的实时行情并返回解引号后的数据串。
func (p *Provider) fetchHq(ctx context.Context, code string) (string, error) {
	text, err := p.fetchRaw(ctx, hqURL+"/list="+code)
	if err != nil {
		return "", err
	}
	data, ok := extractHq(text, code)
	if !ok {
		return "", fmt.Errorf("sina: quote %s not found", code)
	}
	if data == "" {
		return "", fmt.Errorf("sina: quote %s empty", code)
	}
	return data, nil
}

// fetchHqBatch 批量请求多个 code 的实时行情，空数据的 code 自动剔除。
func (p *Provider) fetchHqBatch(ctx context.Context, codes []string) (map[string]string, error) {
	if len(codes) == 0 {
		return map[string]string{}, nil
	}
	text, err := p.fetchRaw(ctx, hqURL+"/list="+strings.Join(codes, ","))
	if err != nil {
		return nil, err
	}
	result := make(map[string]string, len(codes))
	for _, code := range codes {
		if data, ok := extractHq(text, code); ok && data != "" {
			result[code] = data
		}
	}
	return result, nil
}

// extractHq 从响应文本中提取指定 code 的行情数据串。
func extractHq(text, code string) (string, bool) {
	marker := "var hq_str_" + code + "=\""
	idx := strings.Index(text, marker)
	if idx < 0 {
		return "", false
	}
	rest := text[idx+len(marker):]
	end := strings.Index(rest, "\"")
	if end < 0 {
		return "", false
	}
	return rest[:end], true
}

// fetchRaw 发起 GET 请求并把 GBK 响应解码为 UTF-8 文本。
func (p *Provider) fetchRaw(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("sina: build request: %w", err)
	}
	req.Header.Set("Referer", referer)
	req.Header.Set("User-Agent", userAgent)
	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("sina: request %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("sina: request %s: status %d", url, resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return "", fmt.Errorf("sina: read %s: %w", url, err)
	}
	text, err := simplifiedchinese.GBK.NewDecoder().Bytes(body)
	if err != nil {
		return "", fmt.Errorf("sina: decode gbk %s: %w", url, err)
	}
	return string(text), nil
}

// parseMoney 把价格/金额字符串解析为 int64 微元（scale=1e6），非法或空返回 0。
func parseMoney(s string) int64 {
	v, err := domain.ParseDecimal(strings.TrimSpace(s))
	if err != nil {
		return 0
	}
	return v
}

// parseCount 把数量字符串解析为整数（成交量等），支持小数并按微元四舍五入。
func parseCount(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	if !strings.Contains(s, ".") {
		if v, err := strconv.ParseInt(s, 10, 64); err == nil {
			return v
		}
	}
	micro, err := domain.ParseDecimal(s)
	if err != nil {
		return 0
	}
	return (micro + 500_000) / 1_000_000
}

// firstNumeric 依序返回第一个可解析为正数金额的字段，全部失败返回 0。
func firstNumeric(fields ...string) int64 {
	for _, f := range fields {
		if v := parseMoney(f); v > 0 {
			return v
		}
	}
	return 0
}

// changeBP 由现价与昨收计算涨跌基点（bp = 涨跌幅% * 100）。
func changeBP(price, preClose int64) int64 {
	if preClose <= 0 {
		return 0
	}
	return (price - preClose) * 10000 / preClose
}

// pctToBP 把涨跌幅（百分数，微元表示，如 0.32% 为 320000）转基点。
func pctToBP(pct int64) int64 {
	return pct / 10000
}

// marketOf 由完整代码前缀判断市场（sh/sz/hk/us/gb/dk/fr），未知返回空串。
func marketOf(code string) string {
	if len(code) < 2 {
		return ""
	}
	switch code[:2] {
	case "sh", "sz", "hk", "us", "gb", "dk", "fr":
		return code[:2]
	}
	return ""
}

// instrumentType 由市场与代码推断资产类型（股票/指数/基金）。
func instrumentType(market, symbol string) string {
	switch market {
	case "sh":
		switch {
		case strings.HasPrefix(symbol, "000"), strings.HasPrefix(symbol, "880"):
			return adapter.TypeIndex
		case strings.HasPrefix(symbol, "5"):
			return adapter.TypeFund
		}
	case "sz":
		switch {
		case strings.HasPrefix(symbol, "399"):
			return adapter.TypeIndex
		case strings.HasPrefix(symbol, "1"):
			return adapter.TypeFund
		}
	}
	return adapter.TypeStock
}

// isIndexCode 判断 sh/sz 代码是否为指数（sh000xxx / sh880xxx / sz399xxx）。
func isIndexCode(code string) bool {
	switch {
	case strings.HasPrefix(code, "sh000"), strings.HasPrefix(code, "sh880"):
		return true
	case strings.HasPrefix(code, "sz399"):
		return true
	}
	return false
}

// quoteTypeOf 推断 hf_/gds_/nf_ 代码的资产类型（贵金属或商品）。
func quoteTypeOf(code string) string {
	switch {
	case strings.HasPrefix(code, "hf_X"), strings.HasPrefix(code, "gds_"),
		strings.HasPrefix(code, "nf_AU"), strings.HasPrefix(code, "nf_AG"):
		return adapter.TypeMetal
	default:
		return adapter.TypeCommodity
	}
}
