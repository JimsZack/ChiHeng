// Package eastmoney 实现东方财富行情数据源，适配 internal/adapter.MarketProvider 接口。
//
// 实现情况：
//   - Search    searchapi.eastmoney.com/api/suggest/get（type=14 股票/指数、type=15 外汇/商品）
//   - Quote     push2.eastmoney.com/api/qt/stock/get（含 A 股、港股、美股等境外市场）
//   - Intraday  push2.eastmoney.com/api/qt/stock/trends2/get 当日分时
//   - Kline     push2his.eastmoney.com/api/qt/stock/kline/get（day/week/month -> klt 101/102/103）
//   - OrderBook push2.eastmoney.com/api/qt/stock/get 五档盘口（A 股）
//   - News      np-listapi.eastmoney.com/comm/web/getNewsByColumns
//     （https://finance.eastmoney.com/a/cjdd.html 页面「长江电讯」快讯的 JSON 接口）
//
// 数据说明：
//   - 金额一律换算为微元 int64（1e6），涨跌幅换算为 bp。
//   - 分时/K 线点按时间升序返回。
//   - 所有请求均受 ctx 控制，客户端整体超时 10s。
//   - 快讯 PublishedAt 保留原始格式，Source 统一为「东方财富」。
package eastmoney

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/JimsZack/ChiHeng/internal/adapter"
)

// microScale 金额微元换算系数（1 元 = 1e6 微元）。
const microScale = 1_000_000

// 东方财富 secid 市场编号（A 股/境外市场）。
var emSecID = map[string]string{
	"sh": "1", "sz": "0", "bj": "0",
	"hk": "116", "us": "105", "gb": "106",
	"jp": "107", "kr": "108", "de": "109", "fr": "110",
}

// emSearchMarket suggest 接口 MarketType -> 市场缩写。
var emSearchMarket = map[string]string{
	"1": "sh", "2": "sz", "3": "bj",
	"4": "hk", "5": "hk",
	"6": "gb", "7": "us", "8": "jp", "9": "kr", "10": "de",
}

// Client 东方财富行情客户端，实现 adapter.MarketProvider。
type Client struct {
	httpClient *http.Client
}

// New 创建东方财富行情客户端。
func New() *Client {
	return &Client{httpClient: &http.Client{Timeout: 10 * time.Second}}
}

var _ adapter.MarketProvider = (*Client)(nil)

// Search 按关键字搜索股票/指数（type=14）与外汇/商品（type=15），按代码去重合并。
func (c *Client) Search(ctx context.Context, keyword string, limit int) ([]adapter.Instrument, error) {
	if limit <= 0 {
		limit = 20
	}
	stocks, stockErr := c.searchType(ctx, keyword, 14, limit)
	other, otherErr := c.searchType(ctx, keyword, 15, limit)
	if stockErr != nil && otherErr != nil {
		return nil, fmt.Errorf("eastmoney search: %v; %w", otherErr, stockErr)
	}
	if stockErr != nil {
		return other, nil
	}
	if otherErr == nil {
		stocks = mergeInstruments(stocks, other)
	}
	return stocks, nil
}

// Quote 查询实时报价（push2 stock/get，含境外市场）。
func (c *Client) Quote(ctx context.Context, code string) (adapter.Quote, error) {
	id, err := secID(code)
	if err != nil {
		return adapter.Quote{}, err
	}
	params := url.Values{}
	params.Set("ut", "fa5fd1943c7b386f172d6893dbfba10b")
	params.Set("invt", "2")
	params.Set("fltt", "2")
	params.Set("fields", "f43,f44,f45,f46,f47,f48,f50,f57,f58,f60,f170")
	params.Set("secid", id)
	var resp struct {
		Data *quoteData `json:"data"`
	}
	if err := c.getJSON(ctx, "https://push2.eastmoney.com/api/qt/stock/get", params, &resp); err != nil {
		return adapter.Quote{}, fmt.Errorf("eastmoney quote %s: %w", code, err)
	}
	if resp.Data == nil {
		return adapter.Quote{}, fmt.Errorf("eastmoney quote %s: empty data", code)
	}
	price, prev := resp.Data.F43.Val(), resp.Data.F44.Val()
	name := resp.Data.F58
	if name == "" {
		name = code
	}
	return adapter.Quote{
		Code:          code,
		Name:          name,
		Type:          adapter.TypeStock,
		Price:         toMicro(price),
		ChangeBP:      pctBP(price, prev),
		Open:          toMicro(resp.Data.F45.Val()),
		High:          toMicro(resp.Data.F46.Val()),
		Low:           toMicro(resp.Data.F47.Val()),
		PreviousClose: toMicro(prev),
		Volume:        int64(math.Round(resp.Data.F48.Val())),
		Amount:        toMicro(resp.Data.F50.Val()),
		Source:        "东方财富",
		SourceAt:      time.Now(),
	}, nil
}

// Intraday 查询当日分时（trends2/get，按时间升序）。
func (c *Client) Intraday(ctx context.Context, code string) ([]adapter.Point, error) {
	id, err := secID(code)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("secid", id)
	params.Set("fields1", "f1,f2,f3,f4,f5,f6,f7,f8")
	params.Set("fields2", "f51,f52,f53,f54,f55,f56")
	params.Set("iscr", "0")
	var resp struct {
		Data *struct {
			Trends []string `json:"trends"`
		} `json:"data"`
	}
	if err := c.getJSON(ctx, "https://push2.eastmoney.com/api/qt/stock/trends2/get", params, &resp); err != nil {
		return nil, fmt.Errorf("eastmoney intraday %s: %w", code, err)
	}
	if resp.Data == nil {
		return nil, fmt.Errorf("eastmoney intraday %s: empty data", code)
	}
	points := make([]adapter.Point, 0, len(resp.Data.Trends))
	for _, line := range resp.Data.Trends {
		parts := strings.Split(line, ",")
		if len(parts) < 6 {
			continue
		}
		points = append(points, adapter.Point{
			Time:   parts[0],
			Open:   toMicro(parseFloat(parts[1])),
			Close:  toMicro(parseFloat(parts[2])),
			High:   toMicro(parseFloat(parts[3])),
			Low:    toMicro(parseFloat(parts[4])),
			Volume: int64(math.Round(parseFloat(parts[5]))),
		})
	}
	sortPoints(points)
	return points, nil
}

// Kline 查询 K 线（push2his kline/get，period: day/week/month，按时间升序）。
func (c *Client) Kline(ctx context.Context, code, period string, limit int) ([]adapter.Point, error) {
	klt, ok := map[string]string{"day": "101", "week": "102", "month": "103"}[period]
	if !ok {
		return nil, fmt.Errorf("eastmoney kline %s: unsupported period %q", code, period)
	}
	id, err := secID(code)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 120
	}
	params := url.Values{}
	params.Set("secid", id)
	params.Set("fields1", "f1,f2,f3,f4,f5,f6,f7,f8")
	params.Set("fields2", "f51,f52,f53,f54,f55,f56,f57,f58")
	params.Set("klt", klt)
	params.Set("fqt", "1")
	params.Set("end", "20500101")
	params.Set("lmt", strconv.Itoa(limit))
	var resp struct {
		Data *struct {
			Klines []string `json:"klines"`
		} `json:"data"`
	}
	if err := c.getJSON(ctx, "https://push2his.eastmoney.com/api/qt/stock/kline/get", params, &resp); err != nil {
		return nil, fmt.Errorf("eastmoney kline %s: %w", code, err)
	}
	if resp.Data == nil {
		return nil, fmt.Errorf("eastmoney kline %s: empty data", code)
	}
	points := make([]adapter.Point, 0, len(resp.Data.Klines))
	for _, line := range resp.Data.Klines {
		parts := strings.Split(line, ",")
		if len(parts) < 6 {
			continue
		}
		points = append(points, adapter.Point{
			Time:   parts[0],
			Open:   toMicro(parseFloat(parts[1])),
			Close:  toMicro(parseFloat(parts[2])),
			High:   toMicro(parseFloat(parts[3])),
			Low:    toMicro(parseFloat(parts[4])),
			Volume: int64(math.Round(parseFloat(parts[5]))),
		})
	}
	sortPoints(points)
	return points, nil
}

// OrderBook 查询五档盘口（A 股，push2 stock/get 的买一~买五/卖一~卖五字段）。
func (c *Client) OrderBook(ctx context.Context, code string) ([]adapter.Level, error) {
	id, err := secID(code)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("ut", "fa5fd1943c7b386f172d6893dbfba10b")
	params.Set("invt", "2")
	params.Set("fltt", "2")
	params.Set("fields", "f19,f20,f21,f22,f23,f24,f25,f26,f27,f28,f29,f30,f31,f32,f33,f34,f35,f36,f37,f38")
	params.Set("secid", id)
	var resp struct {
		Data *orderBookData `json:"data"`
	}
	if err := c.getJSON(ctx, "https://push2.eastmoney.com/api/qt/stock/get", params, &resp); err != nil {
		return nil, fmt.Errorf("eastmoney orderbook %s: %w", code, err)
	}
	if resp.Data == nil {
		return nil, fmt.Errorf("eastmoney orderbook %s: empty data", code)
	}
	levels := make([]adapter.Level, 0, 10)
	for i := 0; i < 5; i++ {
		levels = append(levels, resp.Data.bid(i))
		levels = append(levels, resp.Data.ask(i))
	}
	return levels, nil
}

// News 查询财经快讯（finance.eastmoney.com/a/cjdd.html 页面的 JSON 接口）。
func (c *Client) News(ctx context.Context, limit int) ([]adapter.NewsItem, error) {
	if limit <= 0 {
		limit = 20
	}
	params := url.Values{}
	params.Set("client", "web")
	params.Set("biz", "web_724")
	params.Set("column", "102") // 长江电讯快讯栏目
	params.Set("order", "1")
	params.Set("needInteractData", "0")
	params.Set("page_index", "1")
	params.Set("page_size", strconv.Itoa(limit))
	var resp struct {
		Data struct {
			List []struct {
				Code      string `json:"code"`
				Title     string `json:"title"`
				ShowTime  string `json:"showTime"`
				URL       string `json:"url"`
				MediaName string `json:"mediaName"`
			} `json:"list"`
		} `json:"data"`
	}
	if err := c.getJSON(ctx, "https://np-listapi.eastmoney.com/comm/web/getNewsByColumns", params, &resp); err != nil {
		return nil, fmt.Errorf("eastmoney news: %w", err)
	}
	items := make([]adapter.NewsItem, 0, len(resp.Data.List))
	for _, it := range resp.Data.List {
		if it.Title == "" {
			continue
		}
		id := it.Code
		if id == "" {
			id = fmt.Sprintf("em-news-%d", len(items))
		}
		items = append(items, adapter.NewsItem{
			ID:          id,
			Title:       it.Title,
			PublishedAt: it.ShowTime, // 保留原始时间格式
			Source:      "东方财富",
			URL:         it.URL,
			Tag:         "东方财富快讯",
		})
	}
	return items, nil
}

// getJSON 发送 GET 请求并把 JSON 响应解码到 out。
func (c *Client) getJSON(ctx context.Context, baseURL string, params url.Values, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"?"+params.Encode(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Referer", "https://www.eastmoney.com/")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %s", resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// secID 把全代码（如 sh600519 / usAAPL / 600519）转为东方财富 secid（市场.代码）。
func secID(code string) (string, error) {
	if len(code) > 2 {
		if m, ok := emSecID[strings.ToLower(code[:2])]; ok {
			return m + "." + code[2:], nil
		}
	}
	if len(code) == 6 && isDigits(code) {
		market := "0"
		switch code[0] {
		case '5', '6', '9':
			market = "1"
		case '8', '4': // 北交所
			market = "0"
		}
		return market + "." + code, nil
	}
	return "", fmt.Errorf("eastmoney: unsupported code %q", code)
}

func isDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// searchType 调用 suggest 接口搜索指定类型（14 股票/指数、15 外汇/商品）。
func (c *Client) searchType(ctx context.Context, keyword string, typ, limit int) ([]adapter.Instrument, error) {
	params := url.Values{}
	params.Set("input", keyword)
	params.Set("type", strconv.Itoa(typ))
	params.Set("count", strconv.Itoa(limit))
	var resp struct {
		QuotationCodeTable struct {
			Data []struct {
				Code       string `json:"Code"`
				Name       string `json:"Name"`
				MarketType string `json:"MarketType"`
				Classify   string `json:"Classify"`
			} `json:"Data"`
		} `json:"QuotationCodeTable"`
	}
	if err := c.getJSON(ctx, "https://searchapi.eastmoney.com/api/suggest/get", params, &resp); err != nil {
		return nil, err
	}
	var items []adapter.Instrument
	for _, it := range resp.QuotationCodeTable.Data {
		if it.Code == "" || it.Name == "" {
			continue
		}
		switch typ {
		case 14:
			if it.Classify == "BK" || it.Classify == "Futures" || it.Classify == "HKDerivative" {
				continue
			}
			market, ok := emSearchMarket[it.MarketType]
			if !ok {
				continue
			}
			assetType := adapter.TypeStock
			if it.Classify == "Index" {
				assetType = adapter.TypeIndex
			}
			items = append(items, adapter.Instrument{
				Code: market + it.Code, Name: it.Name, Type: assetType, Market: market,
			})
		case 15:
			assetType, ok := forexCommodityType(it.Classify, it.Name, it.Code)
			if !ok {
				continue
			}
			items = append(items, adapter.Instrument{
				Code: it.Code, Name: it.Name, Type: assetType, Market: assetType,
			})
		}
	}
	return items, nil
}

// forexCommodityType 判定外汇/商品条目的资产类型；非相关条目返回 ok=false。
func forexCommodityType(classify, name, code string) (string, bool) {
	if classify != "Futures" && classify != "Forex" && classify != "Commodity" &&
		!strings.Contains(name, "外汇") && !strings.Contains(name, "期货") && !strings.Contains(name, "商品") {
		return "", false
	}
	upper := strings.ToUpper(code)
	switch {
	case strings.Contains(name, "黄金"), strings.Contains(name, "白银"),
		strings.HasPrefix(upper, "XAU"), strings.HasPrefix(upper, "XAG"):
		return adapter.TypeMetal, true
	case strings.Contains(name, "原油"), strings.Contains(name, "铜"), strings.Contains(name, "铝"),
		strings.HasPrefix(upper, "OIL"), strings.HasPrefix(upper, "CU"):
		return adapter.TypeCommodity, true
	case classify == "Futures":
		return adapter.TypeCommodity, true
	default:
		return adapter.TypeForex, true
	}
}

// mergeInstruments 按 Code 去重合并多份搜索结果。
func mergeInstruments(lists ...[]adapter.Instrument) []adapter.Instrument {
	seen := make(map[string]bool)
	var out []adapter.Instrument
	for _, list := range lists {
		for _, in := range list {
			if seen[in.Code] {
				continue
			}
			seen[in.Code] = true
			out = append(out, in)
		}
	}
	return out
}

// sortPoints 按时间升序排序（时间格式可字符串比较）。
func sortPoints(points []adapter.Point) {
	sort.SliceStable(points, func(i, j int) bool { return points[i].Time < points[j].Time })
}

// toMicro 金额（元）转微元 int64。
func toMicro(v float64) int64 {
	return int64(math.Round(v * microScale))
}

// pctBP 由价格与昨收计算涨跌幅（bp），昨收非正返回 0。
func pctBP(price, prev float64) int64 {
	if prev <= 0 {
		return 0
	}
	return int64(math.Round((price - prev) / prev * 10000))
}

// parseFloat 解析可能为 "-"、空串的字符串数值，失败按 0 处理。
func parseFloat(s string) float64 {
	v, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return v
}

// flexFloat 兼容东方财富把字段返回为数字或字符串（"-"）的情况。
type flexFloat struct {
	ok bool
	v  float64
}

func (f *flexFloat) UnmarshalJSON(data []byte) error {
	s := strings.TrimSpace(strings.Trim(string(data), `"`))
	if s == "" || s == "-" || s == "null" {
		return nil
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil
	}
	f.ok, f.v = true, v
	return nil
}

func (f *flexFloat) Val() float64 {
	if f == nil || !f.ok {
		return 0
	}
	return f.v
}

// quoteData push2 stock/get 报价字段（fltt=2 时返回真实小数）。
type quoteData struct {
	F43 flexFloat `json:"f43"` // 最新价
	F44 flexFloat `json:"f44"` // 昨收
	F45 flexFloat `json:"f45"` // 今开
	F46 flexFloat `json:"f46"` // 最高
	F47 flexFloat `json:"f47"` // 最低
	F48 flexFloat `json:"f48"` // 成交量（手）
	F50 flexFloat `json:"f50"` // 成交额（元）
	F57 string    `json:"f57"` // 代码
	F58 string    `json:"f58"` // 名称
	F60 flexFloat `json:"f60"` // 昨收（冗余）
}

// orderBookData push2 stock/get 五档盘口字段。
// 字段顺序：买一~买五价 f19,f21,f23,f25,f27；买一~买五量 f20,f22,f24,f26,f28；
// 卖一~卖五价 f29,f31,f33,f35,f37；卖一~卖五量 f30,f32,f34,f36,f38。
type orderBookData struct {
	BidPrice [5]flexFloat
	BidVol   [5]flexFloat
	AskPrice [5]flexFloat
	AskVol   [5]flexFloat
}

// UnmarshalJSON 把 f19~f38 归入五档数组。
func (d *orderBookData) UnmarshalJSON(data []byte) error {
	var raw struct {
		F19 flexFloat `json:"f19"`
		F20 flexFloat `json:"f20"`
		F21 flexFloat `json:"f21"`
		F22 flexFloat `json:"f22"`
		F23 flexFloat `json:"f23"`
		F24 flexFloat `json:"f24"`
		F25 flexFloat `json:"f25"`
		F26 flexFloat `json:"f26"`
		F27 flexFloat `json:"f27"`
		F28 flexFloat `json:"f28"`
		F29 flexFloat `json:"f29"`
		F30 flexFloat `json:"f30"`
		F31 flexFloat `json:"f31"`
		F32 flexFloat `json:"f32"`
		F33 flexFloat `json:"f33"`
		F34 flexFloat `json:"f34"`
		F35 flexFloat `json:"f35"`
		F36 flexFloat `json:"f36"`
		F37 flexFloat `json:"f37"`
		F38 flexFloat `json:"f38"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	d.BidPrice = [5]flexFloat{raw.F19, raw.F21, raw.F23, raw.F25, raw.F27}
	d.BidVol = [5]flexFloat{raw.F20, raw.F22, raw.F24, raw.F26, raw.F28}
	d.AskPrice = [5]flexFloat{raw.F29, raw.F31, raw.F33, raw.F35, raw.F37}
	d.AskVol = [5]flexFloat{raw.F30, raw.F32, raw.F34, raw.F36, raw.F38}
	return nil
}

// bid 返回第 i 档买单（i: 0~4）。
func (d *orderBookData) bid(i int) adapter.Level {
	return adapter.Level{
		Side: "bid", Level: i + 1,
		Price:  toMicro(d.BidPrice[i].Val()),
		Volume: int64(math.Round(d.BidVol[i].Val())),
	}
}

// ask 返回第 i 档卖单（i: 0~4）。
func (d *orderBookData) ask(i int) adapter.Level {
	return adapter.Level{
		Side: "ask", Level: i + 1,
		Price:  toMicro(d.AskPrice[i].Val()),
		Volume: int64(math.Round(d.AskVol[i].Val())),
	}
}
