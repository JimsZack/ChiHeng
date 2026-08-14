package domain

type Holding struct {
	ID            string
	FundCode      string
	FundName      string
	Shares        int64
	CostNAV       int64
	CurrentNAV    int64
	DailyChangeBP int64
	OpenedOn      string
	Version       int64
	CreatedAt     string
	UpdatedAt     string
}

type Transaction struct {
	ID          string
	HoldingID   string
	Kind        string
	Shares      int64
	NAV         int64
	Amount      int64
	RealizedPnL int64
	OccurredAt  string
}

type Plan struct {
	ID           string
	FundCode     string
	FundName     string
	Amount       int64
	Frequency    string
	ExecutionDay int
	StartDate    string
	Status       string
	Version      int64
	CreatedAt    string
	UpdatedAt    string
}

type Instrument struct {
	ID         string
	Code       string
	Name       string
	AssetType  string
	Market     string
	ProviderID string
}

type Quote struct {
	Instrument Instrument
	Price      int64
	ChangeBP   int64
	Open       int64
	High       int64
	Low        int64
	Previous   int64
	Volume     int64
	Amount     int64
	Source     string
	SourceAt   string
	Stale      bool
}

type Execution struct {
	ID            string
	PlanID        string
	ScheduledDate string
	Status        string
	Amount        int64
	NAV           int64
	ErrorCode     string
	CreatedAt     string
}

type SearchRecord struct {
	Keyword   string
	AssetType string
	SearchedAt string
}

type AssetPoint struct {
	Date             string
	TotalMarketValue int64
	TotalCost        int64
	DayProfit        int64
}

type Preference struct {
	Key   string
	Value string
}

type NewsItem struct {
	ID          string
	Title       string
	PublishedAt string
	Source      string
	URL         string
	Tag         string
}

type SeriesPoint struct {
	Time   string
	Open   int64
	High   int64
	Low    int64
	Close  int64
	Volume int64
}

type OrderLevel struct {
	Side   string
	Level  int
	Price  int64
	Volume int64
}

type DiagnosisMetric struct {
	Key   string
	Label string
	Value string
}

type DiagnosisResult struct {
	ReportID   string
	Kind       string
	Score      int
	Conclusion string
	Metrics    []DiagnosisMetric
	Sections   []ReportSection
	CreatedAt  string
}

type ReportSection struct {
	Title   string
	Content string
}

type BacktestResult struct {
	Invested     int64
	CurrentValue int64
	YieldRate    int64 // bp
	Neutral      []AssetPoint
	Optimistic   []AssetPoint
	Pessimistic  []AssetPoint
}
