// /home/gperry/Documents/GitHub/Inc-Line/Wordpress-Inference-Engine/inference/gemini_provider.go
package inference

import (
	"context"
	"fmt"
	"log"
	"sync"

	// Import Google's Gemini client library
	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"

	"github.com/teilomillet/gollm/config"
	"github.com/teilomillet/gollm/providers"
	"github.com/teilomillet/gollm/types"
	"github.com/teilomillet/gollm/utils"
	
)

// GeminiProvider implements the provider interface for Google Gemini.
type GeminiProvider struct {
	apiKey       string
	model        string
	maxTokens    int
	temperature  *float32
	topP         *float32
	topK         *int32
	geminiClient *genai.Client
	extraHeaders map[string]string
	logger       utils.Logger
	mutex        sync.Mutex
	
	
}

// init registers the Gemini provider with the gollm registry.
// This function runs automatically when the package is imported.
func init() {
	log.Println("Registering Gemini provider constructor with gollm registry")
	providers.GetDefaultRegistry().Register("gemini", NewGeminiProvider)
}

// NewGeminiProvider creates an instance of the Gemini provider.
// It's called by gollm when gollm.NewLLM is used with provider "gemini".
func NewGeminiProvider(apiKey, model string, extraHeaders map[string]string) providers.Provider {
	log.Printf("[DEBUG] NewGeminiProvider called! apiKey: %t, model: %s", apiKey != "", model)

	// Initialize with arguments and defaults
	provider := &GeminiProvider{
		apiKey:       apiKey,
		model:        model,
		maxTokens:    1024,
		extraHeaders: make(map[string]string),
		logger:       utils.NewLogger(utils.LogLevelInfo),
	}

	// Set default model if provided one is empty
	if provider.model == "" {
		provider.model = "gemini-2.0-flash"
		log.Printf("Gemini model defaulting to %s", provider.model)
	}

	// Copy provided extraHeaders
	if extraHeaders != nil {
		for k, v := range extraHeaders {
			provider.extraHeaders[k] = v
		}
	}

	// Initialize the Gemini client
	ctx := context.Background()
	clientOptions := []option.ClientOption{
		option.WithAPIKey(apiKey),
		// --- Add Endpoint Override ---
		// Uncomment the line below to explicitly target the v1beta endpoint
		//option.WithEndpoint("generativelanguage.googleapis.com:443"), // Base endpoint, library adds path
		// Or potentially the full path if needed, check genai docs:
		option.WithEndpoint("https://generativelanguage.googleapis.com/v1beta/"),
	}
	client, err := genai.NewClient(ctx, clientOptions...)
	// Check if the client was created successfully
	if err != nil {
		log.Printf("Error creating Gemini client: %v", err)
		// We'll return the provider anyway and let the actual API calls fail if needed
	} else {
		provider.geminiClient = client
		log.Println("Gemini client created successfully.")
	}

	log.Printf("NewGeminiProvider created: model=%s", provider.model)
	return provider
}

// --- Implement the providers.Provider interface methods ---

// Name returns the name of the provider.
func (p *GeminiProvider) Name() string {
	return "gemini"
}

// Endpoint returns the API endpoint URL.
func (p *GeminiProvider) Endpoint() string {
	// Gemini uses the Google client library which handles the endpoint internally
	return "https://generativelanguage.googleapis.com/"
}

// Headers returns the necessary HTTP headers.
func (p *GeminiProvider) Headers() map[string]string {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	headers := map[string]string{
		"Content-Type": "application/json",
		"User-Agent":   "Wordpress-Inference-Engine/1.0",
	}

	// Add API key header if needed (though the client handles this)
	if p.apiKey != "" {
		headers["x-goog-api-key"] = p.apiKey
	}

	// Add any extra headers
	for k, v := range p.extraHeaders {
		headers[k] = v
	}

	return headers
}

// PrepareRequest creates the request body for a standard API call.
func (p *GeminiProvider) PrepareRequest(prompt string, options map[string]interface{}) ([]byte, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	log.Printf("GeminiProvider: Preparing request for model %s", p.model)

	// We don't need to return actual bytes since we're using the client library
	// This is just a placeholder to satisfy the interface
	return []byte(prompt), nil
}

// PrepareRequestWithSchema creates a request with JSON schema validation.
func (p *GeminiProvider) PrepareRequestWithSchema(prompt string, options map[string]interface{}, schema interface{}) ([]byte, error) {
	// Gemini supports structured output, but we'll implement this in a basic way for now
	return p.PrepareRequest(prompt, options)
}

// PrepareRequestWithMessages handles messages for conversation.
func (p *GeminiProvider) PrepareRequestWithMessages(messages []types.MemoryMessage, options map[string]interface{}) ([]byte, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	log.Printf("GeminiProvider: Preparing request with messages for model %s", p.model)

	// We don't need to return actual bytes since we're using the client library
	// This is just a placeholder to satisfy the interface
	return []byte("messages"), nil
}

// ParseResponse extracts the generated text from the API response.
func (p *GeminiProvider) ParseResponse(body []byte) (string, error) {
	// This method won't be used directly since we're using the client library
	// But we need to implement it to satisfy the interface
	return string(body), nil
}

// SetExtraHeaders configures additional HTTP headers.
func (p *GeminiProvider) SetExtraHeaders(extraHeaders map[string]string) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// Replace existing headers
	p.extraHeaders = make(map[string]string)
	for k, v := range extraHeaders {
		p.extraHeaders[k] = v
	}
}

// HandleFunctionCalls processes function calling capabilities.
func (p *GeminiProvider) HandleFunctionCalls(body []byte) ([]byte, error) {
	// Gemini supports function calling, but we'll implement this in a basic way for now
	return body, nil
}

// SupportsJSONSchema indicates whether the provider supports native JSON schema validation.
func (p *GeminiProvider) SupportsJSONSchema() bool {
	return true // Gemini supports structured output
}

// SetDefaultOptions configures provider-specific defaults.
func (p *GeminiProvider) SetDefaultOptions(cfg *config.Config) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// --- Use cfg directly ---
	if cfg == nil {
		p.logger.Warn("SetDefaultOptions called with nil config")
		return
	}

	// Get Gemini provider-specific API key if available
	providerAPIKey := ""
	if cfg.APIKeys != nil {
		if apiKey, ok := cfg.APIKeys[p.Name()]; ok { // Use p.Name() which is "gemini"
			providerAPIKey = apiKey
			p.logger.Debug("Found provider-specific API key for Gemini")
		}
	}

	// Get provider-specific model if available (assuming gollm config supports this structure)
	// Note: gollm's base config might only have a single cfg.Model.
	// If provider-specific models aren't directly in cfg, we might need to adjust.
	// For now, let's prioritize the provider key and then the global model.
	providerModel := ""
	// if cfg.ProviderModels != nil { // Assuming a hypothetical structure
	// 	if model, ok := cfg.ProviderModels[p.Name()]; ok {
	// 		providerModel = model
	// 	}
	// }

	// --- Apply settings ---

	// Set API key if provided and not already set
	if providerAPIKey != "" && p.apiKey == "" {
		p.apiKey = providerAPIKey
		p.logger.Info("Applied default API key for Gemini")
		// Reinitialize client if API key was just set and client is nil
		if p.geminiClient == nil {
			ctx := context.Background()
			client, err := genai.NewClient(ctx, option.WithAPIKey(p.apiKey))
			if err != nil {
				p.logger.Error("Error creating Gemini client after setting default API key", "error", err)
			} else {
				p.geminiClient = client
				p.logger.Info("Gemini client re-initialized with default API key")
			}
		}
	} else if p.apiKey == "" {
		p.logger.Warn("No default or specific API key found/set for Gemini")
	}

	// Set model: Prioritize provider-specific, then global, then keep existing default
	if providerModel != "" && (p.model == "" || p.model == "gemini-2.0-flash") {
		p.model = providerModel
		p.logger.Info("Applied provider-specific default model", "model", p.model)
	} else if cfg.Model != "" && (p.model == "" || p.model == "gemini-2.0-flash") {
		p.model = cfg.Model // Fallback to global default model
		p.logger.Info("Applied global default model", "model", p.model)
	}

	// Set max tokens: Prioritize global, then keep existing default
	if cfg.MaxTokens > 0 && (p.maxTokens == 0 || p.maxTokens == 1024) {
		p.maxTokens = cfg.MaxTokens
		p.logger.Info("Applied global default max tokens", "maxTokens", p.maxTokens)
	}

	// Set temperature if not already set
	if p.temperature == nil && cfg.Temperature > 0 {
		tempFloat32 := float32(cfg.Temperature)
		p.temperature = &tempFloat32
		p.logger.Info("Applied global default temperature", "temperature", *p.temperature)
	}

	// Set TopP if not already set
	if p.topP == nil && cfg.TopP > 0 {
		topPFloat32 := float32(cfg.TopP)
		p.topP = &topPFloat32
		p.logger.Info("Applied global default TopP", "topP", *p.topP)
	}

	// Set TopK if not already set (assuming cfg has TopK)
	// if p.topK == nil && cfg.TopK > 0 {
	// 	topKInt32 := int32(cfg.TopK)
	// 	p.topK = &topKInt32
	// 	p.logger.Info("Applied global default TopK", "topK", *p.topK)
	// }

	


	p.logger.Info("Default options processing complete for Gemini", "final_model", p.model, "final_maxTokens", p.maxTokens)
}

// SetOption sets a specific option for the provider.
func (p *GeminiProvider) SetOption(key string, value interface{}) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	switch key {
	case "model":
		if modelStr, ok := value.(string); ok {
			p.model = modelStr
		}
	case "max_tokens":
		if maxTokens, ok := value.(int); ok {
			p.maxTokens = maxTokens
		}
	case "temperature":
		if temp, ok := value.(float64); ok {
			tempFloat32 := float32(temp)
			p.temperature = &tempFloat32
		} else if temp, ok := value.(float32); ok {
			p.temperature = &temp
		}
	case "top_p":
		if topP, ok := value.(float64); ok {
			topPFloat32 := float32(topP)
			p.topP = &topPFloat32
		} else if topP, ok := value.(float32); ok {
			p.topP = &topP
		}
	case "top_k":
		if topK, ok := value.(int); ok {
			topKInt32 := int32(topK)
			p.topK = &topKInt32
		} else if topK, ok := value.(int32); ok {
			p.topK = &topK
		}
	}
}

// SetLogger configures the logger for the provider.
func (p *GeminiProvider) SetLogger(logger utils.Logger) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.logger = logger
}

// SupportsStreaming indicates if the provider supports streaming responses.
func (p *GeminiProvider) SupportsStreaming() bool {
	return true // Gemini supports streaming
}

// PrepareStreamRequest creates a request body for streaming API calls.
func (p *GeminiProvider) PrepareStreamRequest(prompt string, options map[string]interface{}) ([]byte, error) {
	// Similar to PrepareRequest, but for streaming
	return p.PrepareRequest(prompt, options)
}

// ParseStreamResponse processes a single chunk from a streaming response.
func (p *GeminiProvider) ParseStreamResponse(chunk []byte) (string, error) {
	// This method won't be used directly since we're using the client library
	// But we need to implement it to satisfy the interface
	return string(chunk), nil
}

// --- Helper methods for actual implementation ---

// GenerateContent sends a request to the Gemini API and returns the response.
func (p *GeminiProvider) GenerateContent(ctx context.Context, prompt string) (string, error) {
	p.mutex.Lock()
	client := p.geminiClient
	model := p.model
	p.mutex.Unlock()

	if client == nil {
		p.logger.Error("GeminiProvider: GenerateContent called but client is nil")
		return "", fmt.Errorf("gemini client not initialized")
	}

	// Create a model instance
	genModel := client.GenerativeModel(model)

	// Configure generation settings
	p.mutex.Lock()
	if p.temperature != nil {
		genModel.SetTemperature(*p.temperature)
	}
	if p.topP != nil {
		genModel.SetTopP(*p.topP)
	}
	if p.topK != nil {
		genModel.SetTopK(*p.topK)
	}
	genModel.SetMaxOutputTokens(int32(p.maxTokens))
	p.mutex.Unlock()

	// --- Add Debug Logging ---
	p.logger.Debug("GeminiProvider: Attempting GenerateContent", "model", model, "prompt_length", len(prompt))
	if len(prompt) > 100 {
		p.logger.Debug("GeminiProvider: Prompt prefix", "prefix", prompt[:100]+"...")
	} else {
		p.logger.Debug("GeminiProvider: Prompt", "prompt", prompt)
	}
	// --- End Debug Logging ---


	// Generate content
	resp, err := genModel.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		// Log the specific error from the client library
		p.logger.Error("GeminiProvider: genModel.GenerateContent call failed", "error", err)
		return "", fmt.Errorf("gemini API call failed: %w", err)
	}

	// Extract the generated text
	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("empty response from Gemini API")
	}

	// Extract text from the response
	var result string
	for _, part := range resp.Candidates[0].Content.Parts {
		if textPart, ok := part.(genai.Text); ok {
			result += string(textPart)
		}
	}
	
	p.logger.Debug("GeminiProvider: GenerateContent successful")
	return result, nil
}

// GenerateContentFromMessages sends a conversation to the Gemini API and returns the response.
func (p *GeminiProvider) GenerateContentFromMessages(ctx context.Context, messages []types.MemoryMessage) (string, error) {
	p.mutex.Lock()
	client := p.geminiClient
	model := p.model
	p.mutex.Unlock()

	if client == nil {
		return "", fmt.Errorf("gemini client not initialized")
	}

	// Create a model instance
	genModel := client.GenerativeModel(model)

	// Configure generation settings
	p.mutex.Lock()
	if p.temperature != nil {
		genModel.SetTemperature(*p.temperature)
	}
	if p.topP != nil {
		genModel.SetTopP(*p.topP)
	}
	if p.topK != nil {
		genModel.SetTopK(*p.topK)
	}
	genModel.SetMaxOutputTokens(int32(p.maxTokens))
	p.mutex.Unlock()

	// Convert messages to Gemini format
	var chat []*genai.Content // Use pointer slice
	for _, msg := range messages {
		role := msg.Role
		if role == "assistant" {
			role = "model"
		}
		// Ensure role is either "user" or "model"
		if role != "user" && role != "model" {
			p.logger.Warn("Invalid role for Gemini, skipping message", "role", role)
			continue
		}

		content := &genai.Content{ // Create pointer
			Role:  role,
			Parts: []genai.Part{genai.Text(msg.Content)},
		}
		chat = append(chat, content)
	}

	// Start chat session and send messages
	session := genModel.StartChat()
	session.History = chat // Assign history

	// Send an empty message to get the next response based on history
	// Or, if the last message was 'user', use that as the prompt
	var resp *genai.GenerateContentResponse
	var sendErr error
	if len(chat) > 0 && chat[len(chat)-1].Role == "user" {
		// If the last message is user, treat it as the current prompt
		// Remove it from history before sending
		lastUserContent := chat[len(chat)-1]
		session.History = chat[:len(chat)-1]
		resp, sendErr = session.SendMessage(ctx, lastUserContent.Parts...)
	} else {
		// If history ends with model or is empty, send an empty prompt to continue
		resp, sendErr = session.SendMessage(ctx /* empty parts */)
	}


    // ... Generate content ...
    if sendErr != nil {
		return "", fmt.Errorf("gemini API call failed: %w", sendErr)
	}

    // ... Extract text ...
	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("empty response from Gemini API")
	}
	var result string
	for _, part := range resp.Candidates[0].Content.Parts {
		if textPart, ok := part.(genai.Text); ok {
			result += string(textPart)
		}
	}
	return result, nil
}

// StreamContent streams a response from the Gemini API.
func (p *GeminiProvider) StreamContent(ctx context.Context, prompt string) (chan string, chan error) {
	textChan := make(chan string)
	errChan := make(chan error, 1)

	go func() {
		defer close(textChan)
		defer close(errChan)

		p.mutex.Lock()
		client := p.geminiClient
		model := p.model
		p.mutex.Unlock()

		if client == nil {
			errChan <- fmt.Errorf("gemini client not initialized")
			return
		}

		// Create a model instance
		genModel := client.GenerativeModel(model)

		// Configure generation settings
		p.mutex.Lock()
		if p.temperature != nil {
			genModel.SetTemperature(*p.temperature)
		}
		if p.topP != nil {
			genModel.SetTopP(*p.topP)
		}
		if p.topK != nil {
			genModel.SetTopK(*p.topK)
		}
		genModel.SetMaxOutputTokens(int32(p.maxTokens))
		p.mutex.Unlock()

		// Stream content
		iter := genModel.GenerateContentStream(ctx, genai.Text(prompt))
		for {
			resp, err := iter.Next()
			if err != nil {
				if err.Error() == "iterator done" {
					break
				}
				errChan <- fmt.Errorf("gemini API streaming error: %w", err)
				return
			}

			// Extract text from the response
			if len(resp.Candidates) > 0 && len(resp.Candidates[0].Content.Parts) > 0 {
				for _, part := range resp.Candidates[0].Content.Parts {
					if textPart, ok := part.(genai.Text); ok {
						textChan <- string(textPart)
					}
				}
			}
		}
	}()

	return textChan, errChan
}
