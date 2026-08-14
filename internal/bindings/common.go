package bindings

import "time"

type RequestMeta struct {
	RequestID string `json:"requestId"`
	Locale    string `json:"locale"`
}

type ResponseMeta struct {
	RequestID string `json:"requestId"`
	ServedAt  string `json:"servedAt"`
	Source    string `json:"source,omitempty"`
	SourceAt  string `json:"sourceAt,omitempty"`
	Stale     bool   `json:"stale"`
}

type APIError struct {
	Code      string   `json:"code"`
	Message   string   `json:"message"`
	Field     string   `json:"field,omitempty"`
	Retryable bool     `json:"retryable"`
	Details   []string `json:"details,omitempty"`
}

func responseMeta(meta RequestMeta) ResponseMeta {
	return ResponseMeta{RequestID: meta.RequestID, ServedAt: time.Now().UTC().Format(time.RFC3339), Stale: false}
}

func validation(message, field string) *APIError {
	return &APIError{Code: "VALIDATION", Message: message, Field: field, Retryable: false, Details: []string{}}
}

func unavailable(code, message string) *APIError {
	return &APIError{Code: code, Message: message, Retryable: false, Details: []string{}}
}

type EmptyData struct{}

type IDRequest struct {
	ID              string `json:"id"`
	ExpectedVersion int64  `json:"expectedVersion,omitempty"`
}

type PathRequest struct {
	Path string `json:"path"`
}

type URLRequest struct {
	URL string `json:"url"`
}

type TaskData struct {
	TaskID    string `json:"taskId"`
	Kind      string `json:"kind"`
	Status    string `json:"status"`
	Progress  int    `json:"progress"`
	Phase     string `json:"phase"`
	ErrorCode string `json:"errorCode,omitempty"`
}
