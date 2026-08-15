package service

import (
	"context"
	"fmt"

	"github.com/JimsZack/ChiHeng/internal/adapter"
	"github.com/JimsZack/ChiHeng/internal/adapter/eastmoney"
	"github.com/JimsZack/ChiHeng/internal/adapter/fundgz"
	"github.com/JimsZack/ChiHeng/internal/adapter/sina"
	"github.com/JimsZack/ChiHeng/internal/domain"
	"github.com/JimsZack/ChiHeng/internal/store"
)

// Market 行情域：聚合新浪/东方财富/天天基金数据源，提供统一查询接口。
type Market struct {
	store  *store.Store
	sina   *sina.Provider
	east   adapter.MarketProvider
	fundgz *fundgz.Client
}

func NewMarket(repository *store.Store) *Market {
	return &Market{
		store:  repository,
		sina:   sina.New(),
		east:   eastmoney.New(),
		fundgz: fundgz.New(),
	}
}

// Search 搜索股票/指数/外汇商品（先东方财富，失败回退新浪）。
func (m *Market) Search(ctx context.Context, keyword string, assetType string, limit int) ([]adapter.Instrument, error) {
	if limit <= 0 {
		limit = 20
	}
	results, err := m.east.Search(ctx, keyword, limit)
	if err == nil && len(results) > 0 {
		return results, nil
	}
	return m.sina.Search(ctx, keyword, limit)
}

// Quote 查询实时报价（优先新浪 A 股/指数，东方财富兜底）。
func (m *Market) Quote(ctx context.Context, code string) (adapter.Quote, error) {
	quote, err := m.sina.Quote(ctx, code)
	if err == nil {
		return quote, nil
	}
	return m.east.Quote(ctx, code)
}

// FundEstimate 查询基金实时估值（天天基金）。
func (m *Market) FundEstimate(ctx context.Context, code string) (adapter.Quote, error) {
	return m.fundgz.GetEstimate(ctx, code)
}

// FundNAV 返回基金实时估值净值（微元），供定投执行使用。
func (m *Market) FundNAV(ctx context.Context, code string) (int64, error) {
	quote, err := m.fundgz.GetEstimate(ctx, code)
	if err != nil {
		return 0, err
	}
	if quote.Price <= 0 {
		return 0, fmt.Errorf("无效估值")
	}
	return quote.Price, nil
}

// Intraday 分时走势（股票走新浪，基金走本地 ticks 缓存）。
func (m *Market) Intraday(ctx context.Context, code string) ([]domain.SeriesPoint, error) {
	points, err := m.sina.Intraday(ctx, code)
	if err != nil {
		points, err = m.east.Intraday(ctx, code)
	}
	if err != nil {
		return nil, err
	}
	return toDomainPoints(points), nil
}

// Kline K 线（东方财富）。
func (m *Market) Kline(ctx context.Context, code, period string, limit int) ([]domain.SeriesPoint, error) {
	if limit <= 0 {
		limit = 250
	}
	points, err := m.east.Kline(ctx, code, period, limit)
	if err != nil {
		return nil, err
	}
	return toDomainPoints(points), nil
}

// OrderBook 五档盘口（新浪 A 股）。
func (m *Market) OrderBook(ctx context.Context, code string) ([]domain.OrderLevel, error) {
	levels, err := m.sina.OrderBook(ctx, code)
	if err != nil {
		return nil, err
	}
	result := make([]domain.OrderLevel, 0, len(levels))
	for _, l := range levels {
		result = append(result, domain.OrderLevel{Side: l.Side, Level: l.Level, Price: l.Price, Volume: l.Volume})
	}
	return result, nil
}

// News 财经快讯（东方财富）。
func (m *Market) News(ctx context.Context, limit int) ([]domain.NewsItem, error) {
	if limit <= 0 {
		limit = 20
	}
	items, err := m.east.News(ctx, limit)
	if err != nil {
		return nil, err
	}
	result := make([]domain.NewsItem, 0, len(items))
	for _, n := range items {
		result = append(result, domain.NewsItem{ID: n.ID, Title: n.Title, PublishedAt: n.PublishedAt, Source: n.Source, URL: n.URL, Tag: n.Tag})
	}
	return result, nil
}

// GlobalIndices 全球指数（新浪）。
func (m *Market) GlobalIndices(ctx context.Context) ([]adapter.Quote, error) {
	return m.sina.GlobalIndices(ctx)
}

// CurrencyRates 货币汇率（新浪）。
func (m *Market) CurrencyRates(ctx context.Context) ([]adapter.Quote, error) {
	return m.sina.CurrencyRates(ctx)
}

// PreciousMetals 贵金属（新浪/上金所）。
func (m *Market) PreciousMetals(ctx context.Context) ([]adapter.Quote, error) {
	return m.sina.PreciousMetals(ctx)
}

// CommodityPrices 全球商品（新浪）。
func (m *Market) CommodityPrices(ctx context.Context) ([]adapter.Quote, error) {
	return m.sina.CommodityPrices(ctx)
}

// AddUserIndex / RemoveUserIndex / ListUserIndices 自选行情。
func (m *Market) AddUserIndex(ctx context.Context, symbol, name string) error {
	quote, err := m.Quote(ctx, symbol)
	if err == nil && name == "" {
		name = quote.Name
	}
	return m.store.AddUserIndex(ctx, symbol, name, marketOf(symbol))
}

func (m *Market) RemoveUserIndex(ctx context.Context, symbol string) error {
	return m.store.RemoveUserIndex(ctx, symbol)
}

func (m *Market) ListUserIndices(ctx context.Context) ([]domain.Instrument, error) {
	return m.store.ListUserIndices(ctx)
}

// AddSearchHistory / ListSearchHistory 搜索历史。
func (m *Market) AddSearchHistory(ctx context.Context, keyword, assetType string) error {
	return m.store.AddSearchHistory(ctx, keyword, assetType)
}

func (m *Market) ListSearchHistory(ctx context.Context) ([]domain.SearchRecord, error) {
	return m.store.ListSearchHistory(ctx, 10)
}

// SnapshotAssetHistory 记录资产日快照（供总览曲线）。
func (m *Market) SnapshotAssetHistory(ctx context.Context, date string, marketValue, cost, dayProfit int64) error {
	return m.store.UpsertAssetHistory(ctx, date, marketValue, cost, dayProfit)
}

func (m *Market) AssetHistory(ctx context.Context) ([]domain.AssetPoint, error) {
	return m.store.ListAssetHistory(ctx)
}

func marketOf(symbol string) string {
	switch {
	case len(symbol) >= 2 && (symbol[:2] == "sh" || symbol[:2] == "sz"):
		return "cn"
	case len(symbol) >= 3 && symbol[:3] == "int":
		return "global"
	case len(symbol) >= 3 && symbol[:3] == "fx_":
		return "forex"
	default:
		return "global"
	}
}

func toDomainPoints(points []adapter.Point) []domain.SeriesPoint {
	result := make([]domain.SeriesPoint, 0, len(points))
	for _, p := range points {
		result = append(result, domain.SeriesPoint{Time: p.Time, Open: p.Open, High: p.High, Low: p.Low, Close: p.Close, Volume: p.Volume})
	}
	return result
}
