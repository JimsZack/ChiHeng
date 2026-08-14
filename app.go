package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/chiheng-app/chiheng/internal/adapter"
	"github.com/chiheng-app/chiheng/internal/adapter/deepseek"
	"github.com/chiheng-app/chiheng/internal/bindings"
	"github.com/chiheng-app/chiheng/internal/config"
	"github.com/chiheng-app/chiheng/internal/domain"
	"github.com/chiheng-app/chiheng/internal/scheduler"
	"github.com/chiheng-app/chiheng/internal/service"
	"github.com/chiheng-app/chiheng/internal/store"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App 是 Wails 绑定对象：前端经 window.go.main.App.<Method> 调用。
// 方法仅做 DTO 转换与错误映射，核心业务全部委托给 service 层。
type App struct {
	ctx        context.Context
	store      *store.Store
	secrets    *service.FileSecretStore
	portfolio  *service.Portfolio
	investment *service.Investment
	market     *service.Market
	diagnosis  *service.Diagnosis
	settings   *service.Settings
	scheduler  *scheduler.Scheduler
}

// NewApp 组装应用依赖。
func NewApp(db *store.Store, secrets *service.FileSecretStore) *App {
	portfolio := service.NewPortfolio(db)
	market := service.NewMarket(db)
	investment := service.NewInvestment(db, market, portfolio)
	chat := newChatClient(secrets)
	diagnosis := service.NewDiagnosis(chat)
	settings := service.NewSettings(db)

	app := &App{
		store:      db,
		secrets:    secrets,
		portfolio:  portfolio,
		investment: investment,
		market:     market,
		diagnosis:  diagnosis,
		settings:   settings,
	}
	app.scheduler = scheduler.New(&dueRunner{investment: investment}, 60*time.Second)
	return app
}

// dueRunner 将 service.Investment 适配为 scheduler.Runner，避免 App 绑定对象承担调度入口。
type dueRunner struct {
	investment *service.Investment
}

func (r *dueRunner) ExecuteDue(ctx context.Context) (int, error) {
	return r.investment.ExecuteDue(ctx)
}

// newChatClient 尝试用已存密钥构建 DeepSeek 客户端（未配置则返回 nil，AI 功能不可用）。
func newChatClient(secrets *service.FileSecretStore) service.ChatClient {
	if secrets == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	key, err := secrets.Get(ctx, "ai.api_key")
	if err != nil || key == "" {
		return nil
	}
	model := "deepseek-chat"
	if v, err := secrets.Get(ctx, "ai.model"); err == nil && v != "" {
		model = v
	}
	return deepseek.New(key, model)
}

// onStartup 在窗口创建前执行：保存上下文并启动定投调度器。
func (a *App) onStartup(ctx context.Context) {
	a.ctx = ctx
	slog.Info("应用启动", "version", config.Version)
	a.scheduler.Start()
}

// onShutdown 在应用退出时执行：优雅关闭调度器与数据库。
func (a *App) onShutdown(_ context.Context) {
	a.scheduler.Stop()
	slog.Info("应用退出")
	if err := a.store.Close(); err != nil {
		slog.Error("关闭数据库失败", "error", err)
	}
}

// appMenu 构建应用菜单：显示主窗口 / 退出。
// Wails v2 不提供系统托盘 API，采用「关闭即隐藏 + 菜单恢复」的等效方案。
func (a *App) appMenu() *menu.Menu {
	m := menu.NewMenu()
	appMenu := m.AddSubmenu("持衡 ChiHeng")
	appMenu.AddText("显示主窗口", nil, func(_ *menu.CallbackData) {
		runtime.WindowShow(a.ctx)
		runtime.WindowCenter(a.ctx)
	})
	appMenu.AddSeparator()
	appMenu.AddText("退出", nil, func(_ *menu.CallbackData) {
		runtime.Quit(a.ctx)
	})
	return m
}

// GetAppInfo 返回应用标识信息。
func (a *App) GetAppInfo() map[string]string {
	return map[string]string{
		"name":    config.AppDisplayName,
		"version": config.Version,
		"tagline": config.Tagline,
	}
}

// ---- 持仓域 Portfolio ----

// ListHoldings 返回持仓列表。
func (a *App) ListHoldings(meta bindings.RequestMeta) bindings.HoldingsResponse {
	holdings, err := a.portfolio.List(a.ctx)
	if err != nil {
		return bindings.HoldingsResponse{Meta: responseMeta(meta), Data: nil, Error: toAPIError(err)}
	}
	data := make([]bindings.HoldingData, 0, len(holdings))
	for _, h := range holdings {
		data = append(data, toHoldingData(h))
	}
	return bindings.HoldingsResponse{Meta: responseMeta(meta), Data: data, Error: nil}
}

// CreateHolding 新建持仓。
func (a *App) CreateHolding(meta bindings.RequestMeta, req bindings.HoldingRequest) bindings.HoldingResponse {
	shares, err := domain.ParseDecimal(req.Shares)
	if err != nil {
		return bindings.HoldingResponse{Meta: responseMeta(meta), Error: validation("份额格式无效", "shares")}
	}
	cost, err := domain.ParseDecimal(req.CostNAV)
	if err != nil {
		return bindings.HoldingResponse{Meta: responseMeta(meta), Error: validation("成本净值格式无效", "costNav")}
	}
	holding, err := a.portfolio.Create(a.ctx, domain.Holding{
		FundCode: req.FundCode,
		FundName: req.FundName,
		Shares:   shares,
		CostNAV:  cost,
		OpenedOn: req.OpenedOn,
	})
	if err != nil {
		return bindings.HoldingResponse{Meta: responseMeta(meta), Error: toAPIError(err)}
	}
	data := toHoldingData(holding)
	return bindings.HoldingResponse{Meta: responseMeta(meta), Data: &data, Error: nil}
}

// BuyHolding 加仓（加权平均成本）。
func (a *App) BuyHolding(meta bindings.RequestMeta, req bindings.TradeHoldingRequest) bindings.HoldingResponse {
	shares, err := domain.ParseDecimal(req.Shares)
	if err != nil {
		return bindings.HoldingResponse{Meta: responseMeta(meta), Error: validation("份额格式无效", "shares")}
	}
	nav, err := domain.ParseDecimal(req.NAV)
	if err != nil {
		return bindings.HoldingResponse{Meta: responseMeta(meta), Error: validation("净值格式无效", "nav")}
	}
	holding, err := a.portfolio.Buy(a.ctx, req.ID, shares, nav, req.ExpectedVersion)
	if err != nil {
		return bindings.HoldingResponse{Meta: responseMeta(meta), Error: toAPIError(err)}
	}
	data := toHoldingData(holding)
	return bindings.HoldingResponse{Meta: responseMeta(meta), Data: &data, Error: nil}
}

// SellHolding 减仓/清仓。
func (a *App) SellHolding(meta bindings.RequestMeta, req bindings.TradeHoldingRequest) bindings.HoldingResponse {
	shares, err := domain.ParseDecimal(req.Shares)
	if err != nil {
		return bindings.HoldingResponse{Meta: responseMeta(meta), Error: validation("份额格式无效", "shares")}
	}
	nav, err := domain.ParseDecimal(req.NAV)
	if err != nil {
		return bindings.HoldingResponse{Meta: responseMeta(meta), Error: validation("净值格式无效", "nav")}
	}
	holding, err := a.portfolio.Sell(a.ctx, req.ID, shares, nav, req.ExpectedVersion)
	if err != nil {
		return bindings.HoldingResponse{Meta: responseMeta(meta), Error: toAPIError(err)}
	}
	data := toHoldingData(holding)
	return bindings.HoldingResponse{Meta: responseMeta(meta), Data: &data, Error: nil}
}

// CorrectHolding 手动修正持仓。
func (a *App) CorrectHolding(meta bindings.RequestMeta, req bindings.CorrectHoldingRequest) bindings.HoldingResponse {
	shares, err := domain.ParseDecimal(req.Shares)
	if err != nil {
		return bindings.HoldingResponse{Meta: responseMeta(meta), Error: validation("份额格式无效", "shares")}
	}
	cost, err := domain.ParseDecimal(req.CostNAV)
	if err != nil {
		return bindings.HoldingResponse{Meta: responseMeta(meta), Error: validation("成本净值格式无效", "costNav")}
	}
	holding, err := a.portfolio.Correct(a.ctx, domain.Holding{ID: req.ID, Shares: shares, CostNAV: cost}, req.ExpectedVersion)
	if err != nil {
		return bindings.HoldingResponse{Meta: responseMeta(meta), Error: toAPIError(err)}
	}
	data := toHoldingData(holding)
	return bindings.HoldingResponse{Meta: responseMeta(meta), Data: &data, Error: nil}
}

// DeleteHolding 删除持仓。
func (a *App) DeleteHolding(meta bindings.RequestMeta, req bindings.IDRequest) bindings.EmptyResponse {
	if err := a.portfolio.Delete(a.ctx, req.ID, req.ExpectedVersion); err != nil {
		return bindings.EmptyResponse{Meta: responseMeta(meta), Error: toAPIError(err)}
	}
	return bindings.EmptyResponse{Meta: responseMeta(meta), Data: &bindings.EmptyData{}, Error: nil}
}

// PortfolioOverview 返回持仓总览（市值/盈亏/分布）。
func (a *App) PortfolioOverview(meta bindings.RequestMeta) bindings.PortfolioResponse {
	holdings, err := a.portfolio.List(a.ctx)
	if err != nil {
		return bindings.PortfolioResponse{Meta: responseMeta(meta), Error: toAPIError(err)}
	}
	overview := bindings.PortfolioOverviewData{
		TotalMarketValue: "0",
		TotalCost:        "0",
		DailyPnL:         "0",
		CumulativePnL:    "0",
		CumulativeRate:   "0",
		Allocations:      []bindings.Allocation{},
		Holdings:         []bindings.HoldingData{},
	}
	var totalValue, totalCost int64
	for _, h := range holdings {
		value := h.Shares * h.CurrentNAV / domain.Scale
		cost := h.Shares * h.CostNAV / domain.Scale
		totalValue += value
		totalCost += cost
		overview.Holdings = append(overview.Holdings, toHoldingData(h))
	}
	overview.TotalMarketValue = domain.FormatDecimal(totalValue)
	overview.TotalCost = domain.FormatDecimal(totalCost)
	if totalCost > 0 {
		overview.CumulativeRate = domain.FormatDecimal((totalValue-totalCost)*10000/totalCost)
	}
	overview.CumulativePnL = domain.FormatDecimal(totalValue - totalCost)
	return bindings.PortfolioResponse{Meta: responseMeta(meta), Data: &overview, Error: nil}
}

// AssetHistory 资产历史曲线。
func (a *App) AssetHistory(meta bindings.RequestMeta) bindings.HistoryResponse {
	points, err := a.market.AssetHistory(a.ctx)
	if err != nil {
		return bindings.HistoryResponse{Meta: responseMeta(meta), Error: toAPIError(err)}
	}
	data := make([]bindings.HistoryPoint, 0, len(points))
	for _, p := range points {
		data = append(data, bindings.HistoryPoint{Date: p.Date, Value: domain.FormatDecimal(p.TotalMarketValue)})
	}
	return bindings.HistoryResponse{Meta: responseMeta(meta), Data: data, Error: nil}
}

// ExportHoldingsCSV 将持仓导出为 CSV：弹出保存对话框，写入用户选择路径。
func (a *App) ExportHoldingsCSV(meta bindings.RequestMeta) bindings.TaskResponse {
	holdings, err := a.portfolio.List(a.ctx)
	if err != nil {
		return bindings.TaskResponse{Meta: responseMeta(meta), Error: toAPIError(err)}
	}
	content, err := service.ExportHoldingsCSV(holdings)
	if err != nil {
		return bindings.TaskResponse{Meta: responseMeta(meta), Error: toAPIError(err)}
	}
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		DefaultFilename: "holdings.csv",
		Filters: []runtime.FileFilter{
			{DisplayName: "CSV 文件", Pattern: "*.csv"},
		},
	})
	if err != nil || path == "" {
		return bindings.TaskResponse{Meta: responseMeta(meta), Error: &bindings.APIError{Code: "CANCELLED", Message: "已取消导出", Retryable: false}}
	}
	if err := os.WriteFile(path, []byte("\uFEFF"+content), 0o644); err != nil {
		return bindings.TaskResponse{Meta: responseMeta(meta), Error: toAPIError(err)}
	}
	data := bindings.TaskData{TaskID: "export-csv", Kind: "csv", Status: "done", Progress: 100, Phase: "written"}
	return bindings.TaskResponse{Meta: responseMeta(meta), Data: &data, Error: nil}
}

// ImportHoldingsCSV 批量导入持仓：弹出文件选择对话框，解析 CSV 后逐行创建。
func (a *App) ImportHoldingsCSV(meta bindings.RequestMeta) bindings.TaskResponse {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Filters: []runtime.FileFilter{
			{DisplayName: "CSV 文件", Pattern: "*.csv"},
		},
	})
	if err != nil || path == "" {
		return bindings.TaskResponse{Meta: responseMeta(meta), Error: &bindings.APIError{Code: "CANCELLED", Message: "已取消导入", Retryable: false}}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return bindings.TaskResponse{Meta: responseMeta(meta), Error: toAPIError(err)}
	}
	// 去除 UTF-8 BOM（导出时写入，兼容 Excel 打开）
	content := strings.TrimPrefix(string(data), "\uFEFF")
	result, err := a.portfolio.ImportHoldingsCSV(a.ctx, content)
	if err != nil {
		return bindings.TaskResponse{Meta: responseMeta(meta), Error: toAPIError(err)}
	}
	phase := fmt.Sprintf("导入 %d 条，忽略 %d 条", result.Imported, result.Ignored)
	data2 := bindings.TaskData{TaskID: "import-csv", Kind: "csv", Status: "done", Progress: 100, Phase: phase}
	return bindings.TaskResponse{Meta: responseMeta(meta), Data: &data2, Error: nil}
}

// ---- 定投域 Investment ----

// CreatePlan 创建定投计划。
func (a *App) CreatePlan(meta bindings.RequestMeta, req bindings.PlanRequest) bindings.PlanResponse {
	amount, err := domain.ParseDecimal(req.Amount)
	if err != nil {
		return bindings.PlanResponse{Meta: responseMeta(meta), Error: validation("金额格式无效", "amount")}
	}
	// 定投金额单位为分
	plan, err := a.investment.CreatePlan(a.ctx, domain.Plan{
		FundCode:     req.FundCode,
		FundName:     req.FundName,
		Amount:       amount,
		Frequency:    req.Frequency,
		ExecutionDay: req.ExecutionDay,
		StartDate:    req.StartDate,
	})
	if err != nil {
		return bindings.PlanResponse{Meta: responseMeta(meta), Error: toAPIError(err)}
	}
	data := toPlanData(plan)
	return bindings.PlanResponse{Meta: responseMeta(meta), Data: &data, Error: nil}
}

// ListPlans 定投计划列表。
func (a *App) ListPlans(meta bindings.RequestMeta) bindings.PlansResponse {
	plans, err := a.investment.ListPlans(a.ctx)
	if err != nil {
		return bindings.PlansResponse{Meta: responseMeta(meta), Error: toAPIError(err)}
	}
	data := make([]bindings.PlanData, 0, len(plans))
	for _, p := range plans {
		data = append(data, toPlanData(p))
	}
	return bindings.PlansResponse{Meta: responseMeta(meta), Data: data, Error: nil}
}

// PausePlan 暂停计划。
func (a *App) PausePlan(meta bindings.RequestMeta, req bindings.IDRequest) bindings.PlanResponse {
	plan, err := a.investment.PausePlan(a.ctx, req.ID, req.ExpectedVersion)
	if err != nil {
		return bindings.PlanResponse{Meta: responseMeta(meta), Error: toAPIError(err)}
	}
	data := toPlanData(plan)
	return bindings.PlanResponse{Meta: responseMeta(meta), Data: &data, Error: nil}
}

// ResumePlan 恢复计划。
func (a *App) ResumePlan(meta bindings.RequestMeta, req bindings.IDRequest) bindings.PlanResponse {
	plan, err := a.investment.ResumePlan(a.ctx, req.ID, req.ExpectedVersion)
	if err != nil {
		return bindings.PlanResponse{Meta: responseMeta(meta), Error: toAPIError(err)}
	}
	data := toPlanData(plan)
	return bindings.PlanResponse{Meta: responseMeta(meta), Data: &data, Error: nil}
}

// DeletePlan 删除计划。
func (a *App) DeletePlan(meta bindings.RequestMeta, req bindings.IDRequest) bindings.EmptyResponse {
	if err := a.investment.DeletePlan(a.ctx, req.ID, req.ExpectedVersion); err != nil {
		return bindings.EmptyResponse{Meta: responseMeta(meta), Error: toAPIError(err)}
	}
	return bindings.EmptyResponse{Meta: responseMeta(meta), Data: &bindings.EmptyData{}, Error: nil}
}

// ListExecutions 执行记录。
func (a *App) ListExecutions(meta bindings.RequestMeta, req bindings.IDRequest) bindings.ExecutionsResponse {
	executions, err := a.investment.ListExecutions(a.ctx, req.ID, 50)
	if err != nil {
		return bindings.ExecutionsResponse{Meta: responseMeta(meta), Error: toAPIError(err)}
	}
	data := make([]bindings.ExecutionData, 0, len(executions))
	for _, e := range executions {
		data = append(data, toExecutionData(e))
	}
	return bindings.ExecutionsResponse{Meta: responseMeta(meta), Data: data, Error: nil}
}

// ExecuteDue 立即执行到期计划（供手动触发与调度器共用）。
func (a *App) ExecuteDue(meta bindings.RequestMeta) bindings.TaskResponse {
	count, err := a.investment.ExecuteDue(a.ctx)
	if err != nil {
		return bindings.TaskResponse{Meta: responseMeta(meta), Error: toAPIError(err)}
	}
	data := bindings.TaskData{TaskID: "execute-due", Kind: "investment", Status: "done", Progress: 100, Phase: "executed"}
	_ = count
	return bindings.TaskResponse{Meta: responseMeta(meta), Data: &data, Error: nil}
}

// Backtest 定投历史回测：基于基金历史净值序列测算中性/乐观/悲观三曲线。
func (a *App) Backtest(meta bindings.RequestMeta, req bindings.BacktestRequest) bindings.BacktestResponse {
	amount, err := domain.ParseDecimal(req.Amount)
	if err != nil {
		return bindings.BacktestResponse{Meta: responseMeta(meta), Error: validation("金额格式无效", "amount")}
	}
	years := req.Years
	if years <= 0 {
		years = 3
	}
	// 用日 K 历史净值近似回测序列（取 years*250 个交易日）
	series, err := a.market.Kline(a.ctx, req.FundCode, "day", years*250)
	if err != nil || len(series) == 0 {
		return bindings.BacktestResponse{Meta: responseMeta(meta), Error: unavailable("NO_DATA", "暂无该基金的历史净值数据，无法回测")}
	}
	result := a.investment.Backtest(a.ctx, series, amount, years)
	data := toBacktestData(result)
	return bindings.BacktestResponse{Meta: responseMeta(meta), Data: &data, Error: nil}
}

// ---- 行情域 Market ----

// SearchInstruments 搜索股票/指数/基金。
func (a *App) SearchInstruments(meta bindings.RequestMeta, req bindings.SearchRequest) bindings.InstrumentsResponse {
	results, err := a.market.Search(a.ctx, req.Keyword, req.AssetType, req.Limit)
	if err != nil {
		return bindings.InstrumentsResponse{Meta: responseMeta(meta), Error: toAPIError(err)}
	}
	_ = a.market.AddSearchHistory(a.ctx, req.Keyword, req.AssetType)
	data := make([]bindings.InstrumentData, 0, len(results))
	for _, in := range results {
		data = append(data, toInstrumentData(in))
	}
	return bindings.InstrumentsResponse{Meta: responseMeta(meta), Data: data, Error: nil}
}

// GetQuote 实时报价。
func (a *App) GetQuote(meta bindings.RequestMeta, req bindings.InstrumentRequest) bindings.QuoteResponse {
	quote, err := a.market.Quote(a.ctx, req.InstrumentID)
	if err != nil {
		return bindings.QuoteResponse{Meta: responseMeta(meta), Error: toAPIError(err)}
	}
	data := toQuoteData(quote)
	return bindings.QuoteResponse{Meta: responseMeta(meta), Data: &data, Error: nil}
}

// GetSeries 分时或 K 线。
func (a *App) GetSeries(meta bindings.RequestMeta, req bindings.InstrumentRequest) bindings.SeriesResponse {
	var points []domain.SeriesPoint
	var err error
	if req.Period == "" || req.Period == "intraday" {
		points, err = a.market.Intraday(a.ctx, req.InstrumentID)
	} else {
		points, err = a.market.Kline(a.ctx, req.InstrumentID, req.Period, req.Limit)
	}
	if err != nil {
		return bindings.SeriesResponse{Meta: responseMeta(meta), Error: toAPIError(err)}
	}
	data := make([]bindings.SeriesPoint, 0, len(points))
	for _, p := range points {
		data = append(data, bindings.SeriesPoint{Time: p.Time, Open: domain.FormatDecimal(p.Open), High: domain.FormatDecimal(p.High), Low: domain.FormatDecimal(p.Low), Close: domain.FormatDecimal(p.Close), Volume: fmtVolume(p.Volume)})
	}
	return bindings.SeriesResponse{Meta: responseMeta(meta), Data: data, Error: nil}
}

// GetOrderBook 五档盘口。
func (a *App) GetOrderBook(meta bindings.RequestMeta, req bindings.InstrumentRequest) bindings.OrderBookResponse {
	levels, err := a.market.OrderBook(a.ctx, req.InstrumentID)
	if err != nil {
		return bindings.OrderBookResponse{Meta: responseMeta(meta), Error: toAPIError(err)}
	}
	data := make([]bindings.OrderLevel, 0, len(levels))
	for _, l := range levels {
		data = append(data, bindings.OrderLevel{Side: l.Side, Level: l.Level, Price: domain.FormatDecimal(l.Price), Volume: fmtVolume(l.Volume)})
	}
	return bindings.OrderBookResponse{Meta: responseMeta(meta), Data: data, Error: nil}
}

// GetNews 财经快讯。
func (a *App) GetNews(meta bindings.RequestMeta) bindings.NewsResponse {
	items, err := a.market.News(a.ctx, 20)
	if err != nil {
		return bindings.NewsResponse{Meta: responseMeta(meta), Error: toAPIError(err)}
	}
	data := make([]bindings.NewsItem, 0, len(items))
	for _, n := range items {
		data = append(data, bindings.NewsItem{ID: n.ID, Title: n.Title, PublishedAt: n.PublishedAt, Source: n.Source, URL: n.URL, Tag: n.Tag})
	}
	return bindings.NewsResponse{Meta: responseMeta(meta), Data: data, Error: nil}
}

// GetGlobalIndices 全球指数。
func (a *App) GetGlobalIndices(meta bindings.RequestMeta) bindings.QuotesResponse {
	quotes, err := a.market.GlobalIndices(a.ctx)
	if err != nil {
		return bindings.QuotesResponse{Meta: responseMeta(meta), Error: toAPIError(err)}
	}
	data := make([]bindings.QuoteData, 0, len(quotes))
	for _, q := range quotes {
		data = append(data, toQuoteData(q))
	}
	return bindings.QuotesResponse{Meta: responseMeta(meta), Data: data, Error: nil}
}

// GetCurrencyRates 货币汇率。
func (a *App) GetCurrencyRates(meta bindings.RequestMeta) bindings.QuotesResponse {
	quotes, err := a.market.CurrencyRates(a.ctx)
	if err != nil {
		return bindings.QuotesResponse{Meta: responseMeta(meta), Error: toAPIError(err)}
	}
	data := make([]bindings.QuoteData, 0, len(quotes))
	for _, q := range quotes {
		data = append(data, toQuoteData(q))
	}
	return bindings.QuotesResponse{Meta: responseMeta(meta), Data: data, Error: nil}
}

// GetPreciousMetals 贵金属行情。
func (a *App) GetPreciousMetals(meta bindings.RequestMeta) bindings.QuotesResponse {
	quotes, err := a.market.PreciousMetals(a.ctx)
	if err != nil {
		return bindings.QuotesResponse{Meta: responseMeta(meta), Error: toAPIError(err)}
	}
	data := make([]bindings.QuoteData, 0, len(quotes))
	for _, q := range quotes {
		data = append(data, toQuoteData(q))
	}
	return bindings.QuotesResponse{Meta: responseMeta(meta), Data: data, Error: nil}
}

// GetCommodityPrices 全球商品行情。
func (a *App) GetCommodityPrices(meta bindings.RequestMeta) bindings.QuotesResponse {
	quotes, err := a.market.CommodityPrices(a.ctx)
	if err != nil {
		return bindings.QuotesResponse{Meta: responseMeta(meta), Error: toAPIError(err)}
	}
	data := make([]bindings.QuoteData, 0, len(quotes))
	for _, q := range quotes {
		data = append(data, toQuoteData(q))
	}
	return bindings.QuotesResponse{Meta: responseMeta(meta), Data: data, Error: nil}
}

// GetUserIndices 自选行情列表。
func (a *App) GetUserIndices(meta bindings.RequestMeta) bindings.InstrumentsResponse {
	instruments, err := a.market.ListUserIndices(a.ctx)
	if err != nil {
		return bindings.InstrumentsResponse{Meta: responseMeta(meta), Error: toAPIError(err)}
	}
	data := make([]bindings.InstrumentData, 0, len(instruments))
	for _, in := range instruments {
		data = append(data, bindings.InstrumentData{
			ID: in.Code, Code: in.Code, Name: in.Name,
			AssetType: in.AssetType, Market: in.Market, ProviderID: in.Code,
		})
	}
	return bindings.InstrumentsResponse{Meta: responseMeta(meta), Data: data, Error: nil}
}

// AddUserIndex 添加自选。
func (a *App) AddUserIndex(meta bindings.RequestMeta, req bindings.InstrumentRequest) bindings.EmptyResponse {
	if err := a.market.AddUserIndex(a.ctx, req.InstrumentID, ""); err != nil {
		return bindings.EmptyResponse{Meta: responseMeta(meta), Error: toAPIError(err)}
	}
	return bindings.EmptyResponse{Meta: responseMeta(meta), Data: &bindings.EmptyData{}, Error: nil}
}

// RemoveUserIndex 移除自选。
func (a *App) RemoveUserIndex(meta bindings.RequestMeta, req bindings.InstrumentRequest) bindings.EmptyResponse {
	if err := a.market.RemoveUserIndex(a.ctx, req.InstrumentID); err != nil {
		return bindings.EmptyResponse{Meta: responseMeta(meta), Error: toAPIError(err)}
	}
	return bindings.EmptyResponse{Meta: responseMeta(meta), Data: &bindings.EmptyData{}, Error: nil}
}

// GetSearchHistory 搜索历史。
func (a *App) GetSearchHistory(meta bindings.RequestMeta) bindings.InstrumentsResponse {
	records, err := a.market.ListSearchHistory(a.ctx)
	if err != nil {
		return bindings.InstrumentsResponse{Meta: responseMeta(meta), Error: toAPIError(err)}
	}
	data := make([]bindings.InstrumentData, 0, len(records))
	for _, r := range records {
		data = append(data, bindings.InstrumentData{Code: r.Keyword, Name: r.Keyword, AssetType: r.AssetType})
	}
	return bindings.InstrumentsResponse{Meta: responseMeta(meta), Data: data, Error: nil}
}

// ---- 诊断域 Diagnosis ----

// DiagnoseFund 本地量化诊断。
func (a *App) DiagnoseFund(meta bindings.RequestMeta, req bindings.FundDiagnosisRequest) bindings.DiagnosisResponse {
	series, err := a.market.Kline(a.ctx, req.InstrumentID, "day", 250)
	if err != nil || len(series) == 0 {
		return bindings.DiagnosisResponse{Meta: responseMeta(meta), Error: unavailable("NO_DATA", "暂无该基金的历史净值数据")}
	}
	result := a.diagnosis.DiagnoseLocal(req.InstrumentID, req.InstrumentID, series)
	data := toDiagnosisData(result)
	return bindings.DiagnosisResponse{Meta: responseMeta(meta), Data: &data, Error: nil}
}

// DiagnoseFundAI AI 深度诊断（需 consent）。
func (a *App) DiagnoseFundAI(meta bindings.RequestMeta, req bindings.AIFundDiagnosisRequest) bindings.DiagnosisResponse {
	if !req.Consent {
		return bindings.DiagnosisResponse{Meta: responseMeta(meta), Error: validation("需要明确同意后才能进行 AI 分析", "consent")}
	}
	series, err := a.market.Kline(a.ctx, req.InstrumentID, "day", 250)
	if err != nil || len(series) == 0 {
		return bindings.DiagnosisResponse{Meta: responseMeta(meta), Error: unavailable("NO_DATA", "暂无该基金的历史净值数据")}
	}
	result, err := a.diagnosis.DiagnoseAI(a.ctx, req.InstrumentID, req.InstrumentID, series, false)
	if err != nil {
		return bindings.DiagnosisResponse{Meta: responseMeta(meta), Error: toAPIError(err)}
	}
	data := toDiagnosisData(result)
	return bindings.DiagnosisResponse{Meta: responseMeta(meta), Data: &data, Error: nil}
}

// ---- 设置域 Settings ----

// GetPreferences 读取偏好。
func (a *App) GetPreferences(meta bindings.RequestMeta) bindings.PreferencesResponse {
	p, err := a.settings.GetPreferences(a.ctx)
	if err != nil {
		return bindings.PreferencesResponse{Meta: responseMeta(meta), Error: toAPIError(err)}
	}
	data := bindings.PreferencesData{
		Locale:               p.Locale,
		Theme:                p.Theme,
		RefreshIntervalSeconds: p.RefreshInterval,
		DisclaimerVersion:    p.DisclaimerVersion,
		LogLevel:             p.LogLevel,
	}
	return bindings.PreferencesResponse{Meta: responseMeta(meta), Data: &data, Error: nil}
}

// SavePreferences 保存偏好。
func (a *App) SavePreferences(meta bindings.RequestMeta, req bindings.PreferencesRequest) bindings.PreferencesResponse {
	p := service.Preferences{
		Locale:            req.Locale,
		Theme:             req.Theme,
		RefreshInterval:   req.RefreshIntervalSeconds,
		DisclaimerVersion: req.DisclaimerVersion,
		LogLevel:          req.LogLevel,
	}
	if err := a.settings.SavePreferences(a.ctx, p); err != nil {
		return bindings.PreferencesResponse{Meta: responseMeta(meta), Error: toAPIError(err)}
	}
	return a.GetPreferences(meta)
}

// GetAIProfile 读取 AI 配置状态。
func (a *App) GetAIProfile(meta bindings.RequestMeta) bindings.AIProfileResponse {
	profile, err := a.settings.GetAIProfile(a.ctx, a.secrets)
	if err != nil {
		return bindings.AIProfileResponse{Meta: responseMeta(meta), Error: toAPIError(err)}
	}
	data := bindings.AIProfileData{
		Provider:          profile.Provider,
		Model:             profile.Model,
		HasSecret:         profile.HasSecret,
		Available:         profile.Available,
		UnavailableReason: profile.UnavailableReason,
	}
	return bindings.AIProfileResponse{Meta: responseMeta(meta), Data: &data, Error: nil}
}

// SaveAIProfile 保存 AI 配置（密钥写入密钥存储）。
func (a *App) SaveAIProfile(meta bindings.RequestMeta, req bindings.AIProfileRequest) bindings.AIProfileResponse {
	if err := a.settings.SaveAIProfile(a.ctx, req.Provider, req.Model, req.APIKey, a.secrets); err != nil {
		return bindings.AIProfileResponse{Meta: responseMeta(meta), Error: toAPIError(err)}
	}
	return a.GetAIProfile(meta)
}

// ---- 系统域 System ----

// OpenExternal 在系统浏览器打开外部链接。
func (a *App) OpenExternal(meta bindings.RequestMeta, req bindings.URLRequest) bindings.EmptyResponse {
	if !strings.HasPrefix(req.URL, "https://") && !strings.HasPrefix(req.URL, "http://") {
		return bindings.EmptyResponse{Meta: responseMeta(meta), Error: validation("仅允许 http(s) 链接", "url")}
	}
	runtime.BrowserOpenURL(a.ctx, req.URL)
	return bindings.EmptyResponse{Meta: responseMeta(meta), Data: &bindings.EmptyData{}, Error: nil}
}

// ---- DTO 转换 ----

func responseMeta(meta bindings.RequestMeta) bindings.ResponseMeta {
	return bindings.ResponseMeta{RequestID: meta.RequestID, ServedAt: time.Now().UTC().Format(time.RFC3339), Stale: false}
}

func validation(message, field string) *bindings.APIError {
	return &bindings.APIError{Code: "VALIDATION", Message: message, Field: field, Retryable: false}
}

func unavailable(code, message string) *bindings.APIError {
	return &bindings.APIError{Code: code, Message: message, Retryable: false}
}

func toAPIError(err error) *bindings.APIError {
	switch {
	case err == nil:
		return nil
	case err == service.ErrAINotConfigured:
		return &bindings.APIError{Code: "AUTH_REQUIRED", Message: "请先在设置中配置 AI 服务", Retryable: false}
	case err == service.ErrNotFound || err == store.ErrNotFound:
		return &bindings.APIError{Code: "NOT_FOUND", Message: "未找到对应记录", Retryable: false}
	case err == store.ErrConflict || service.IsConflict(err):
		return &bindings.APIError{Code: "CONFLICT", Message: "数据已被其他操作修改，请刷新后重试", Retryable: true}
	case err == domain.ErrInvalidDecimal:
		return &bindings.APIError{Code: "VALIDATION", Message: "金额或份额格式无效", Retryable: false}
	default:
		return &bindings.APIError{Code: "INTERNAL", Message: err.Error(), Retryable: true}
	}
}

func toHoldingData(h domain.Holding) bindings.HoldingData {
	return bindings.HoldingData{
		ID:            h.ID,
		FundCode:      h.FundCode,
		FundName:      h.FundName,
		Shares:        domain.FormatDecimal(h.Shares),
		CostNAV:       domain.FormatDecimal(h.CostNAV),
		CurrentNAV:    domain.FormatDecimal(h.CurrentNAV),
		MarketValue:   domain.FormatDecimal(h.Shares * h.CurrentNAV / domain.Scale),
		DailyChange:   formatBP(h.DailyChangeBP),
		DailyPnL:      domain.FormatDecimal(h.Shares * h.DailyChangeBP * h.CurrentNAV / domain.Scale / 10000),
		CumulativePnL: domain.FormatDecimal((h.CurrentNAV-h.CostNAV)*h.Shares/domain.Scale),
		OpenedOn:      h.OpenedOn,
		Version:       h.Version,
	}
}

func toPlanData(p domain.Plan) bindings.PlanData {
	return bindings.PlanData{
		ID:           p.ID,
		FundCode:     p.FundCode,
		FundName:     p.FundName,
		Amount:       fmtAmount(p.Amount),
		Frequency:    p.Frequency,
		ExecutionDay: p.ExecutionDay,
		StartDate:    p.StartDate,
		Status:       p.Status,
		Version:      p.Version,
	}
}

func toExecutionData(e domain.Execution) bindings.ExecutionData {
	return bindings.ExecutionData{
		ID:            e.ID,
		PlanID:        e.PlanID,
		ScheduledDate: e.ScheduledDate,
		Status:        e.Status,
		Amount:        fmtAmount(e.Amount),
		NAV:           domain.FormatDecimal(e.NAV),
		ErrorCode:     e.ErrorCode,
	}
}

func toBacktestData(r domain.BacktestResult) bindings.BacktestData {
	return bindings.BacktestData{
		Invested:     domain.FormatDecimal(r.Invested),
		CurrentValue: domain.FormatDecimal(r.CurrentValue),
		YieldRate:    formatBP(r.YieldRate),
		Neutral:      toHistoryPoints(r.Neutral),
		Optimistic:   toHistoryPoints(r.Optimistic),
		Pessimistic:  toHistoryPoints(r.Pessimistic),
	}
}

func toHistoryPoints(points []domain.AssetPoint) []bindings.HistoryPoint {
	result := make([]bindings.HistoryPoint, 0, len(points))
	for _, p := range points {
		result = append(result, bindings.HistoryPoint{Date: p.Date, Value: domain.FormatDecimal(p.TotalMarketValue)})
	}
	return result
}

func toInstrumentData(in adapter.Instrument) bindings.InstrumentData {
	return bindings.InstrumentData{
		ID:         in.Code,
		Code:       in.Code,
		Name:       in.Name,
		AssetType:  in.Type,
		Market:     in.Market,
		ProviderID: in.Code,
	}
}

func toQuoteData(q adapter.Quote) bindings.QuoteData {
	change := int64(0)
	if q.PreviousClose > 0 {
		change = (q.Price - q.PreviousClose) * 10000 / q.PreviousClose
	}
	if q.ChangeBP != 0 {
		change = q.ChangeBP
	}
	return bindings.QuoteData{
		Instrument:     toInstrumentData(adapter.Instrument{Code: q.Code, Name: q.Name, Type: q.Type}),
		Price:          domain.FormatDecimal(q.Price),
		Change:         formatBP(change),
		Open:           domain.FormatDecimal(q.Open),
		High:           domain.FormatDecimal(q.High),
		Low:            domain.FormatDecimal(q.Low),
		PreviousClose:  domain.FormatDecimal(q.PreviousClose),
		Volume:         fmtVolume(q.Volume),
		Amount:         domain.FormatDecimal(q.Amount),
	}
}

func toDiagnosisData(r domain.DiagnosisResult) bindings.DiagnosisData {
	metrics := make([]bindings.MetricData, 0, len(r.Metrics))
	for _, m := range r.Metrics {
		metrics = append(metrics, bindings.MetricData{Name: m.Label, Value: m.Value})
	}
	sections := make([]bindings.ReportSection, 0, len(r.Sections))
	for _, s := range r.Sections {
		sections = append(sections, bindings.ReportSection{Title: s.Title, Body: s.Content})
	}
	return bindings.DiagnosisData{
		ReportID:   r.ReportID,
		Kind:       r.Kind,
		Score:      r.Score,
		Conclusion: r.Conclusion,
		Metrics:    metrics,
		Sections:   sections,
		CreatedAt:  r.CreatedAt,
	}
}

func formatBP(bp int64) string {
	sign := ""
	if bp > 0 {
		sign = "+"
	}
	return sign + strings.TrimRight(strings.TrimRight(domain.FormatDecimal(bp*100), "0"), ".") + "%"
}

func fmtAmount(cents int64) string {
	return domain.FormatDecimal(cents * 100)
}

func fmtVolume(volume int64) string {
	return domain.FormatDecimal(volume)
}
