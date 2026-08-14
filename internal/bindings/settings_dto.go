package bindings

type PreferencesData struct { Locale string `json:"locale"`; Theme string `json:"theme"`; RefreshIntervalSeconds int `json:"refreshIntervalSeconds"`; DisclaimerVersion string `json:"disclaimerVersion"`; LogLevel string `json:"logLevel"`; Version int64 `json:"version"` }
type PreferencesRequest struct { Locale string `json:"locale"`; Theme string `json:"theme"`; RefreshIntervalSeconds int `json:"refreshIntervalSeconds"`; DisclaimerVersion string `json:"disclaimerVersion"`; LogLevel string `json:"logLevel"`; ExpectedVersion int64 `json:"expectedVersion"` }
type AIProfileData struct { Provider string `json:"provider"`; Model string `json:"model"`; HasSecret bool `json:"hasSecret"`; Available bool `json:"available"`; UnavailableReason string `json:"unavailableReason,omitempty"` }
type AIProfileRequest struct { Provider string `json:"provider"`; Model string `json:"model"`; APIKey string `json:"apiKey"` }
type PreferencesResponse struct { Meta ResponseMeta `json:"meta"`; Data *PreferencesData `json:"data"`; Error *APIError `json:"error"` }
type AIProfileResponse struct { Meta ResponseMeta `json:"meta"`; Data *AIProfileData `json:"data"`; Error *APIError `json:"error"` }
