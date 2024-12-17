package yandexai

import (
	"time"
)

type CompletionRequest struct {
	ModelURI          string            `json:"modelUri"`
	CompletionOptions CompletionOptions `json:"completionOptions"`
	Messages          []Message         `json:"messages"`
}

type CompletionOptions struct {
	Stream      bool    `json:"stream"`
	Temperature float64 `json:"temperature"`
	MaxTokens   int     `json:"maxTokens"`
}

type Message struct {
	Role string `json:"role"`
	Text string `json:"text"`
}

type PendingRequests struct {
	OperationID string
}

type RequestInfo struct {
	OperationID string
	Attempts    int
	CreatedAt   time.Time
}
