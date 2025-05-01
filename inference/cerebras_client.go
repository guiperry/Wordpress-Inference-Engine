// /home/gperry/Documents/GitHub/cloud-equities/FIG_Inference/inference/cerebras_client.go
package inference

import (
	"bytes"
	"context" // Import context
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	// "os" // No longer needed for API key here
	// "sync" // No longer needed for internal state mutex
)

const (
	CerebrasAPIURL = "https://api.cerebras.ai/v1/chat/completions"
)

// CerebrasClient represents a client for the Cerebras API
type CerebrasClient struct {
	// No internal state like apiKey, model, isRunning needed here anymore.
	// The http.Client is the main state.
	client *http.Client
}

// Message struct remains the same
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}




// NewCerebrasClient creates a new instance of CerebrasClient
func NewCerebrasClient() *CerebrasClient {
	// Only initialize the http client
	return &CerebrasClient{
		client: &http.Client{},
	}
}
// convertToCerebrasMessages converts a slice of Message to CerebrasMessage
func convertToCerebrasMessages(messages []Message) []CerebrasMessage {
	cerebrasMessages := make([]CerebrasMessage, len(messages))
	for i, msg := range messages {
		cerebrasMessages[i] = CerebrasMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}
	return cerebrasMessages
}

// MakeChatCompletionRequest performs the actual API call to Cerebras.
// It takes configuration parameters for each request.
func (c *CerebrasClient) MakeChatCompletionRequest(ctx context.Context, apiKey, model string, messages []Message, maxTokens int) (string, error) {
	if apiKey == "" {
		return "", errors.New("Cerebras API key is required")
	}
	if model == "" {
		return "", errors.New("Cerebras model is required")
	}
	if len(messages) == 0 {
		return "", errors.New("messages cannot be empty")
	}

	// Convert messages to CerebrasMessage format
	cerebrasMessages := convertToCerebrasMessages(messages)

	// Create the request body
	requestBody := ChatCompletionRequest{
		Model:     model,
		Messages:  cerebrasMessages,
		MaxTokens: maxTokens,
		// Stream: false, // Assuming non-streaming
	}

	// Convert the request body to JSON
	requestJSON, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create the HTTP request with context
	req, err := http.NewRequestWithContext(ctx, "POST", CerebrasAPIURL, bytes.NewBuffer(requestJSON))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set the headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("User-Agent", "FIG-Inference/1.0 (via Gollm Provider)") // Identify source

	// Send the request using the client's http.Client
	resp, err := c.client.Do(req)
	if err != nil {
		// Check for context cancellation
		if errors.Is(err, context.Canceled) {
			log.Println("Cerebras request cancelled.")
			return "", err
		}
		if errors.Is(err, context.DeadlineExceeded) {
			log.Println("Cerebras request timed out.")
			return "", err
		}
		return "", fmt.Errorf("failed to send request to Cerebras API: %w", err)
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read Cerebras response body: %w", err)
	}

	// Check for non-OK status code
	if resp.StatusCode != http.StatusOK {
		log.Printf("Cerebras API Error Response Body: %s", string(body))
		return "", fmt.Errorf("Cerebras API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse the response
	var response ChatCompletionResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to unmarshal Cerebras response: %w", err)
	}

	// Check if there are any choices
	if len(response.Choices) == 0 || response.Choices[0].Message.Content == "" {
		log.Printf("Cerebras response body with no choices: %s", string(body))
		return "", errors.New("no response choices or empty content returned from Cerebras")
	}

	// Return the content of the first choice
	return response.Choices[0].Message.Content, nil
}

