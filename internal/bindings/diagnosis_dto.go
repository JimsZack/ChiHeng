package bindings

type FundDiagnosisRequest struct { InstrumentID string `json:"instrumentId"` }
type AIFundDiagnosisRequest struct { InstrumentID string `json:"instrumentId"`; Consent bool `json:"consent"` }
type AIPortfolioDiagnosisRequest struct { Consent bool `json:"consent"`; IncludeAmounts bool `json:"includeAmounts"` }
type DiagnosisData struct { ReportID string `json:"reportId"`; Kind string `json:"kind"`; Score int `json:"score"`; Conclusion string `json:"conclusion"`; Metrics []MetricData `json:"metrics"`; Sections []ReportSection `json:"sections"`; CreatedAt string `json:"createdAt"` }
type MetricData struct { Name string `json:"name"`; Value string `json:"value"` }
type ReportSection struct { Title string `json:"title"`; Body string `json:"body"` }
type DiagnosisResponse struct { Meta ResponseMeta `json:"meta"`; Data *DiagnosisData `json:"data"`; Error *APIError `json:"error"` }
