package yandexai

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

const (
	maxCheckAttempts     = 15
	checkerTickerSeconds = 20
)

type Checker struct {
	AIClient       *APIClient
	CheckChannel   chan PendingRequests
	ResultChannel  chan string
	RequestsBuffer map[string]*RequestInfo
	Mu             sync.Mutex
	log            *slog.Logger
}

// NewChecker initializes a new Checker
func NewChecker(aiClient *APIClient, checkChannel chan PendingRequests, res chan string, log *slog.Logger) *Checker {
	return &Checker{
		AIClient:       aiClient,
		CheckChannel:   checkChannel,
		ResultChannel:  res,
		RequestsBuffer: make(map[string]*RequestInfo),
		log:            log,
	}
}

// Start initiates the checker process
func (c *Checker) Start(ctx context.Context) {
	// Initialize the RequestsBuffer if not already initialized
	if c.RequestsBuffer == nil {
		c.RequestsBuffer = make(map[string]*RequestInfo)
	}

	go c.registerRequests(ctx)
	go c.checkStatuses(ctx)
}

func (c *Checker) registerRequests(ctx context.Context) {
	c.log.Info("registerRequests started")
	for {
		select {
		case req := <-c.CheckChannel:
			c.Mu.Lock()
			c.RequestsBuffer[req.OperationID] = &RequestInfo{
				OperationID: req.OperationID,
				Attempts:    0,
				CreatedAt:   time.Now(),
			}
			c.Mu.Unlock()
			c.log.With("operation_id", req.OperationID).Info("checker: Operation registered")

		case <-ctx.Done():
			c.log.Info("register requests stopped")
			return
		}
	}
}

func (c *Checker) checkStatuses(ctx context.Context) {
	ticker := time.NewTicker(time.Second * checkerTickerSeconds)
	defer ticker.Stop()

	c.log.Info("check statuses started")
	for {
		select {
		case <-ticker.C:
			c.log.Debug("check statuses ticker")
			c.processRequests()

		case <-ctx.Done():
			c.log.Info("check statuses stopped")
			return
		}
	}
}

func (c *Checker) processRequests() {
	c.Mu.Lock()
	for _, reqInfo := range c.RequestsBuffer {
		reqInfo.Attempts++

		if reqInfo.Attempts >= maxCheckAttempts {
			c.handleExceededAttempts(reqInfo)
			continue
		}

		c.checkCompletionStatus(reqInfo)
	}
	c.Mu.Unlock()
}

func (c *Checker) handleExceededAttempts(reqInfo *RequestInfo) {
	c.log.With("operation_id", reqInfo.OperationID).Warn("Attempts count exceeded")
	delete(c.RequestsBuffer, reqInfo.OperationID)
}

func (c *Checker) checkCompletionStatus(reqInfo *RequestInfo) {
	c.log.With(
		"operation_id", reqInfo.OperationID,
		"time", reqInfo.CreatedAt.Format(time.DateTime),
		"buffer_size", len(c.RequestsBuffer)).
		Info("Checking completion status")

	statusResp, err := c.AIClient.CheckCompletionStatus(reqInfo.OperationID)
	if err != nil {
		c.log.With("operation_id", reqInfo.OperationID, "error", err.Error()).Error("check attempt")
		return
	}

	if statusResp.Error != nil && statusResp.Error.Message != "" {
		c.log.With("operation_id", reqInfo.OperationID, "error", statusResp.Error.Message).Error("API error")
	}

	if statusResp.Done {
		c.handleCompletion(reqInfo, statusResp)
	}
}

func (c *Checker) handleCompletion(reqInfo *RequestInfo, statusResp *CompletionResponse) {
	resp := string(statusResp.Response)
	parsed, err := c.AIClient.ParseResponse(resp)
	if err != nil {
		c.log.With("operation_id", reqInfo.OperationID, "error", err.Error()).Error("cant parse response from completion")
		return
	}

	if len(parsed.Alternatives) > 0 {
		c.log.With(
			"operation_id", reqInfo.OperationID,
			"InputTextTokens", parsed.Usage.InputTextTokens,
			"TotalTokens", parsed.Usage.TotalTokens,
			"CompletionTokens", parsed.Usage.CompletionTokens,
			"ModelVersion", parsed.ModelVersion).
			Info("received completion result")

		text := parsed.Alternatives[0].Message.Text

		c.log.With("operation_id", reqInfo.OperationID, "message", text).Info("Status done")
		c.ResultChannel <- text
	}

	delete(c.RequestsBuffer, reqInfo.OperationID)

}
