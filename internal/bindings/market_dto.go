package bindings

type InstrumentData struct { ID string `json:"id"`; Code string `json:"code"`; Name string `json:"name"`; AssetType string `json:"assetType"`; Market string `json:"market"`; ProviderID string `json:"providerId"` }
type QuoteData struct { Instrument InstrumentData `json:"instrument"`; Price string `json:"price"`; Change string `json:"change"`; Open string `json:"open"`; High string `json:"high"`; Low string `json:"low"`; PreviousClose string `json:"previousClose"`; Volume string `json:"volume"`; Amount string `json:"amount"` }
type SeriesPoint struct { Time string `json:"time"`; Open string `json:"open,omitempty"`; High string `json:"high,omitempty"`; Low string `json:"low,omitempty"`; Close string `json:"close"`; Volume string `json:"volume,omitempty"` }
type OrderLevel struct { Side string `json:"side"`; Level int `json:"level"`; Price string `json:"price"`; Volume string `json:"volume"` }
type NewsItem struct { ID string `json:"id"`; Title string `json:"title"`; PublishedAt string `json:"publishedAt"`; Source string `json:"source"`; URL string `json:"url"`; Tag string `json:"tag,omitempty"` }
type SearchRequest struct { Keyword string `json:"keyword"`; AssetType string `json:"assetType,omitempty"`; Limit int `json:"limit,omitempty"` }
type InstrumentRequest struct { InstrumentID string `json:"instrumentId"`; Period string `json:"period,omitempty"`; Limit int `json:"limit,omitempty"` }
type InstrumentsRequest struct { InstrumentIDs []string `json:"instrumentIds"` }
type InstrumentsResponse struct { Meta ResponseMeta `json:"meta"`; Data []InstrumentData `json:"data"`; Error *APIError `json:"error"` }
type QuoteResponse struct { Meta ResponseMeta `json:"meta"`; Data *QuoteData `json:"data"`; Error *APIError `json:"error"` }
type QuotesResponse struct { Meta ResponseMeta `json:"meta"`; Data []QuoteData `json:"data"`; Error *APIError `json:"error"` }
type SeriesResponse struct { Meta ResponseMeta `json:"meta"`; Data []SeriesPoint `json:"data"`; Error *APIError `json:"error"` }
type OrderBookResponse struct { Meta ResponseMeta `json:"meta"`; Data []OrderLevel `json:"data"`; Error *APIError `json:"error"` }
type NewsResponse struct { Meta ResponseMeta `json:"meta"`; Data []NewsItem `json:"data"`; Error *APIError `json:"error"` }
