package yandexai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

// APIClient represents the client for interacting with the Yandex LLM API
type APIClient struct {
	BaseURL                string
	ModelURI               string
	HTTPClient             *http.Client
	APIKey                 string
	PendingRequestsChannel chan PendingRequests
	log                    *slog.Logger
}

const (
	modelYandexGPT     = "yandexgpt/latest"
	modelYandexGPTLite = "yandexgpt-lite"

	modelLlama = "llama/latest"
)

// NewAPIClient creates a new API client with the given base URL and API key
func NewAPIClient(baseURL, apiKey, folderID string, checkChannel chan PendingRequests, log *slog.Logger) *APIClient {
	return &APIClient{
		BaseURL:                baseURL,
		ModelURI:               fmt.Sprintf("gpt://%s/%s", folderID, modelYandexGPT),
		HTTPClient:             &http.Client{Timeout: 10 * time.Second},
		APIKey:                 apiKey,
		PendingRequestsChannel: checkChannel,
		log:                    log,
	}
}

func (c *APIClient) MakeRequest(data string) error {
	prompt, instr, err := c.constructPrompt(data)
	if err != nil {
		return err
	}

	c.log.With("prompt", prompt, "instr", instr).Info("Prompt created")

	completionReq := &CompletionRequest{
		ModelURI: c.ModelURI,
		CompletionOptions: CompletionOptions{
			Stream:      false,
			Temperature: 0.2,
			MaxTokens:   5000,
		},
		Messages: []Message{
			{Role: "system", Text: instr},
			{Role: "user", Text: prompt},
		},
	}

	operationID, err := c.createCompletion(completionReq)
	if err != nil {
		return err
	}

	c.log.With("operation_id", operationID).Info("Request sent. OperationID received")
	c.PendingRequestsChannel <- PendingRequests{OperationID: operationID}

	return nil
}

func (c *APIClient) CheckCompletionStatus(operationID string) (*CompletionResponse, error) {
	url := fmt.Sprintf("%s/operations/%s", c.BaseURL, operationID)

	httpReq, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Authorization", fmt.Sprintf("Api-Key %s", c.APIKey))

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, errors.Wrap(err, "completion status: cant check completion status")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("failed to get completion status: " + resp.Status)
	}

	var response CompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, errors.Wrap(err, "completion status: cant decode response")
	}

	return &response, nil
}

func (c *APIClient) ParseResponse(jsonStr string) (*Response, error) {
	var response Response
	err := json.Unmarshal([]byte(jsonStr), &response)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

// CompletionResponse represents the response from the completion API
type CompletionResponse struct {
	ID          string          `json:"id"`
	Description string          `json:"description"`
	CreatedAt   time.Time       `json:"createdAt"`
	CreatedBy   string          `json:"createdBy"`
	ModifiedAt  time.Time       `json:"modifiedAt"`
	Done        bool            `json:"done"`
	Metadata    json.RawMessage `json:"metadata"`
	Error       *APIError       `json:"error,omitempty"`
	Response    json.RawMessage `json:"response,omitempty"`
}

type Response struct {
	Alternatives []struct {
		Message struct {
			Role string `json:"role"`
			Text string `json:"text"`
		} `json:"message"`
		Status string `json:"status"`
	} `json:"alternatives"`
	Usage struct {
		InputTextTokens  string `json:"inputTextTokens"`
		CompletionTokens string `json:"completionTokens"`
		TotalTokens      string `json:"totalTokens"`
	} `json:"usage"`
	ModelVersion string `json:"modelVersion"`
}

// APIError represents an error response from the API
type APIError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Details json.RawMessage `json:"details"`
}

// сreateCompletion sends a request to the completion API and returns the operation Name
func (c *APIClient) createCompletion(req *CompletionRequest) (string, error) {
	url := c.BaseURL + "/foundationModels/v1/completionAsync"
	jsonData, err := json.Marshal(req)
	if err != nil {
		return "", err
	}

	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Api-Key %s", c.APIKey))

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", errors.New("failed to create completion: " + resp.Status)
	}

	var response CompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", err
	}

	return response.ID, nil
}
