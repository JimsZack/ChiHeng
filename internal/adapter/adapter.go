// Package adapter 封装外部行情与 AI 数据源，所有 HTTP 调用集中于此。
// 每个子包实现一个数据源，service 层通过本包定义的接口消费，互不依赖。
package adapter

import (
	"context"
	"time"
)

// AssetType 资产类型标识。
const (
	TypeStock     = "stock"
	TypeFund      = "fund"
	TypeIndex     = "index"
	TypeForex     = "forex"
	TypeMetal     = "metal"
	TypeCommodity = "commodity"
)

// Instrument 外部数据源返回的标的（代码/名称/类型/市场）。
type Instrument struct {
	Code   string
	Name   string
	Type   string
	Market string
}

// Quote 实时报价（金额为微元整数，换算由 service 层完成）。
type Quote struct {
	Code          string
	Name          string
	Type          string
	Price         int64
	ChangeBP      int64
	Open          int64
	High          int64
	Low           int64
	PreviousClose int64
	Volume        int64
	Amount        int64
	Source        string
	SourceAt      time.Time
	Stale         bool
}

// Point 分时/K 线点。
type Point struct {
	Time   string
	Open   int64
	High   int64
	Low    int64
	Close  int64
	Volume int64
}

// Level 盘口档位。
type Level struct {
	Side   string
	Level  int
	Price  int64
	Volume int64
}

// NewsItem 快讯条目。
type NewsItem struct {
	ID          string
	Title       string
	PublishedAt string
	Source      string
	URL         string
	Tag         string
}

// MarketProvider 行情数据源接口。各子包实现后由 market service 组装。
type MarketProvider interface {
	// Search 按关键字搜索股票/指数（type 限定资产类型，空为全部）。
	Search(ctx context.Context, keyword string, limit int) ([]Instrument, error)
	// Quote 查询实时报价。
	Quote(ctx context.Context, code string) (Quote, error)
	// Intraday 查询当日分时（time 升序）。
	Intraday(ctx context.Context, code string) ([]Point, error)
	// Kline 查询 K 线（period: day/week/month）。
	Kline(ctx context.Context, code, period string, limit int) ([]Point, error)
	// OrderBook 查询五档盘口（A 股）。
	OrderBook(ctx context.Context, code string) ([]Level, error)
	// News 查询财经快讯。
	News(ctx context.Context, limit int) ([]NewsItem, error)
}
