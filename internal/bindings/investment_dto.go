package bindings

type PlanData struct { ID string `json:"id"`; FundCode string `json:"fundCode"`; FundName string `json:"fundName"`; Amount string `json:"amount"`; Frequency string `json:"frequency"`; ExecutionDay int `json:"executionDay"`; StartDate string `json:"startDate"`; Status string `json:"status"`; Version int64 `json:"version"` }
type PlanRequest struct { ID string `json:"id,omitempty"`; FundCode string `json:"fundCode"`; FundName string `json:"fundName"`; Amount string `json:"amount"`; Frequency string `json:"frequency"`; ExecutionDay int `json:"executionDay"`; StartDate string `json:"startDate"`; ExpectedVersion int64 `json:"expectedVersion,omitempty"` }
type BacktestRequest struct { FundCode string `json:"fundCode"`; Amount string `json:"amount"`; Frequency string `json:"frequency"`; ExecutionDay int `json:"executionDay"`; Years int `json:"years"` }
type BacktestData struct { Invested string `json:"invested"`; CurrentValue string `json:"currentValue"`; YieldRate string `json:"yieldRate"`; Neutral []HistoryPoint `json:"neutral"`; Optimistic []HistoryPoint `json:"optimistic"`; Pessimistic []HistoryPoint `json:"pessimistic"` }
type ExecutionData struct { ID string `json:"id"`; PlanID string `json:"planId"`; ScheduledDate string `json:"scheduledDate"`; Status string `json:"status"`; Amount string `json:"amount"`; NAV string `json:"nav"`; CatchUp bool `json:"catchUp"`; ErrorCode string `json:"errorCode,omitempty"` }
type PlansResponse struct { Meta ResponseMeta `json:"meta"`; Data []PlanData `json:"data"`; Error *APIError `json:"error"` }
type PlanResponse struct { Meta ResponseMeta `json:"meta"`; Data *PlanData `json:"data"`; Error *APIError `json:"error"` }
type BacktestResponse struct { Meta ResponseMeta `json:"meta"`; Data *BacktestData `json:"data"`; Error *APIError `json:"error"` }
type ExecutionsResponse struct { Meta ResponseMeta `json:"meta"`; Data []ExecutionData `json:"data"`; Error *APIError `json:"error"` }
type TaskResponse struct { Meta ResponseMeta `json:"meta"`; Data *TaskData `json:"data"`; Error *APIError `json:"error"` }
