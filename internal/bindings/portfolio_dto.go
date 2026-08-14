package bindings

type HoldingData struct {
	ID            string `json:"id"`
	FundCode      string `json:"fundCode"`
	FundName      string `json:"fundName"`
	Shares        string `json:"shares"`
	CostNAV       string `json:"costNav"`
	CurrentNAV    string `json:"currentNav"`
	MarketValue   string `json:"marketValue"`
	DailyChange   string `json:"dailyChange"`
	DailyPnL      string `json:"dailyPnl"`
	CumulativePnL string `json:"cumulativePnl"`
	OpenedOn      string `json:"openedOn"`
	Version       int64  `json:"version"`
}

type PortfolioOverviewData struct {
	TotalMarketValue string        `json:"totalMarketValue"`
	TotalCost        string        `json:"totalCost"`
	DailyPnL         string        `json:"dailyPnl"`
	CumulativePnL    string        `json:"cumulativePnl"`
	CumulativeRate   string        `json:"cumulativeRate"`
	Allocations      []Allocation  `json:"allocations"`
	Holdings         []HoldingData `json:"holdings"`
}

type Allocation struct {
	HoldingID string `json:"holdingId"`
	Name      string `json:"name"`
	Value     string `json:"value"`
}

type HoldingRequest struct {
	FundCode       string `json:"fundCode"`
	FundName       string `json:"fundName"`
	Shares         string `json:"shares"`
	CostNAV        string `json:"costNav"`
	OpenedOn       string `json:"openedOn"`
	ExpectedVersion int64 `json:"expectedVersion,omitempty"`
}

type TradeHoldingRequest struct {
	ID              string `json:"id"`
	Shares          string `json:"shares"`
	NAV             string `json:"nav"`
	ExpectedVersion int64  `json:"expectedVersion"`
}

type CorrectHoldingRequest struct {
	ID              string `json:"id"`
	Shares          string `json:"shares"`
	CostNAV         string `json:"costNav"`
	Reason          string `json:"reason"`
	Confirmed       bool   `json:"confirmed"`
	ExpectedVersion int64  `json:"expectedVersion"`
}

type HistoryPoint struct {
	Date  string `json:"date"`
	Value string `json:"value"`
}

type AssetHistoryRequest struct {
	Days int `json:"days"`
}

type PortfolioResponse struct { Meta ResponseMeta `json:"meta"`; Data *PortfolioOverviewData `json:"data"`; Error *APIError `json:"error"` }
type HoldingsResponse struct { Meta ResponseMeta `json:"meta"`; Data []HoldingData `json:"data"`; Error *APIError `json:"error"` }
type HoldingResponse struct { Meta ResponseMeta `json:"meta"`; Data *HoldingData `json:"data"`; Error *APIError `json:"error"` }
type HistoryResponse struct { Meta ResponseMeta `json:"meta"`; Data []HistoryPoint `json:"data"`; Error *APIError `json:"error"` }
type EmptyResponse struct { Meta ResponseMeta `json:"meta"`; Data *EmptyData `json:"data"`; Error *APIError `json:"error"` }
