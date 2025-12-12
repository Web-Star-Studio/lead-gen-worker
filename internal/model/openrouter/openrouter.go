// Package openrouter provides an OpenRouter model implementation for Google ADK.
// OpenRouter exposes an OpenAI-compatible API that supports 100+ LLM models.
package openrouter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"net/http"
	"strings"
	"time"

	"google.golang.org/adk/model"
	"google.golang.org/genai"
)

const (
	// DefaultBaseURL is the OpenRouter API endpoint
	DefaultBaseURL = "https://openrouter.ai/api/v1"
	// DefaultTimeout for API requests
	DefaultTimeout = 120 * time.Second
)

// Config holds configuration for the OpenRouter model
type Config struct {
	// APIKey is the OpenRouter API key (required)
	APIKey string
	// BaseURL is the API base URL (defaults to OpenRouter)
	BaseURL string
	// HTTPClient allows custom HTTP client (optional)
	HTTPClient *http.Client
	// Timeout for requests (defaults to 120s)
	Timeout time.Duration
	// SiteName is sent as X-Title header for OpenRouter rankings (optional)
	SiteName string
	// SiteURL is sent as HTTP-Referer for OpenRouter rankings (optional)
	SiteURL string
}

// Model implements the ADK model.LLM interface for OpenRouter
type Model struct {
	name       string
	config     Config
	httpClient *http.Client
}

// NewModel creates a new OpenRouter model instance
func NewModel(ctx context.Context, modelName string, config *Config) (*Model, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}
	if config.APIKey == "" {
		return nil, fmt.Errorf("APIKey is required")
	}
	if modelName == "" {
		return nil, fmt.Errorf("modelName is required")
	}

	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}

	timeout := config.Timeout
	if timeout == 0 {
		timeout = DefaultTimeout
	}

	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: timeout}
	}

	return &Model{
		name: modelName,
		config: Config{
			APIKey:     config.APIKey,
			BaseURL:    baseURL,
			HTTPClient: httpClient,
			Timeout:    timeout,
			SiteName:   config.SiteName,
			SiteURL:    config.SiteURL,
		},
		httpClient: httpClient,
	}, nil
}

// Name returns the model name
func (m *Model) Name() string {
	return m.name
}

// GenerateContent implements the model.LLM interface
func (m *Model) GenerateContent(ctx context.Context, req *model.LLMRequest, stream bool) iter.Seq2[*model.LLMResponse, error] {
	return func(yield func(*model.LLMResponse, error) bool) {
		// Convert ADK request to OpenAI format
		openAIReq, err := m.convertRequest(req)
		if err != nil {
			yield(nil, fmt.Errorf("failed to convert request: %w", err))
			return
		}

		if stream {
			m.generateStreaming(ctx, openAIReq, yield)
		} else {
			m.generateNonStreaming(ctx, openAIReq, yield)
		}
	}
}

// generateNonStreaming handles non-streaming requests
func (m *Model) generateNonStreaming(ctx context.Context, req *openAIRequest, yield func(*model.LLMResponse, error) bool) {
	req.Stream = false

	respBody, err := m.doRequest(ctx, req)
	if err != nil {
		yield(nil, err)
		return
	}
	defer respBody.Close()

	var resp openAIResponse
	if err := json.NewDecoder(respBody).Decode(&resp); err != nil {
		yield(nil, fmt.Errorf("failed to decode response: %w", err))
		return
	}

	// Check for API errors
	if resp.Error != nil {
		yield(nil, fmt.Errorf("OpenRouter API error: %s (code: %v)", resp.Error.Message, resp.Error.Code))
		return
	}

	// Convert response
	llmResp := m.convertResponse(&resp)
	yield(llmResp, nil)
}

// generateStreaming handles streaming requests
func (m *Model) generateStreaming(ctx context.Context, req *openAIRequest, yield func(*model.LLMResponse, error) bool) {
	req.Stream = true

	respBody, err := m.doRequest(ctx, req)
	if err != nil {
		yield(nil, err)
		return
	}
	defer respBody.Close()

	// Read SSE stream
	var fullContent strings.Builder
	var lastResp *model.LLMResponse

	// Buffer for reading lines
	buf := make([]byte, 4096)
	var lineBuf bytes.Buffer

	for {
		n, readErr := respBody.Read(buf)
		if n > 0 {
			lineBuf.Write(buf[:n])

			// Process complete lines
			for {
				line, err := lineBuf.ReadString('\n')
				if err != nil {
					// Put back incomplete line
					lineBuf.WriteString(line)
					break
				}

				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}

				// Handle SSE format
				if strings.HasPrefix(line, "data: ") {
					data := strings.TrimPrefix(line, "data: ")
					if data == "[DONE]" {
						// Stream complete
						if lastResp != nil {
							lastResp.TurnComplete = true
							lastResp.Partial = false
							yield(lastResp, nil)
						}
						return
					}

					var chunk openAIStreamChunk
					if err := json.Unmarshal([]byte(data), &chunk); err != nil {
						continue // Skip malformed chunks
					}

					// Process chunk
					if len(chunk.Choices) > 0 {
						delta := chunk.Choices[0].Delta
						if delta.Content != "" {
							fullContent.WriteString(delta.Content)

							lastResp = &model.LLMResponse{
								Content: &genai.Content{
									Role: "model",
									Parts: []*genai.Part{
										{Text: delta.Content},
									},
								},
								Partial:      true,
								TurnComplete: false,
							}

							if chunk.Choices[0].FinishReason != "" {
								lastResp.FinishReason = convertFinishReason(chunk.Choices[0].FinishReason)
								lastResp.TurnComplete = true
								lastResp.Partial = false
							}

							if !yield(lastResp, nil) {
								return
							}
						}
					}
				}
			}
		}

		if readErr != nil {
			if readErr == io.EOF {
				// Ensure we send final response
				if lastResp != nil && !lastResp.TurnComplete {
					lastResp.TurnComplete = true
					lastResp.Partial = false
					yield(lastResp, nil)
				}
				return
			}
			yield(nil, fmt.Errorf("stream read error: %w", readErr))
			return
		}
	}
}

// doRequest performs the HTTP request
func (m *Model) doRequest(ctx context.Context, req *openAIRequest) (io.ReadCloser, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", m.config.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+m.config.APIKey)

	// OpenRouter-specific headers
	if m.config.SiteName != "" {
		httpReq.Header.Set("X-Title", m.config.SiteName)
	}
	if m.config.SiteURL != "" {
		httpReq.Header.Set("HTTP-Referer", m.config.SiteURL)
	}

	resp, err := m.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	return resp.Body, nil
}

// convertRequest converts ADK LLMRequest to OpenAI format
func (m *Model) convertRequest(req *model.LLMRequest) (*openAIRequest, error) {
	messages := make([]openAIMessage, 0, len(req.Contents))

	for _, content := range req.Contents {
		msg := openAIMessage{
			Role: convertRole(content.Role),
		}

		// Build content from parts
		var textParts []string
		for _, part := range content.Parts {
			if part.Text != "" {
				textParts = append(textParts, part.Text)
			}
			// Handle function calls
			if part.FunctionCall != nil {
				msg.ToolCalls = append(msg.ToolCalls, openAIToolCall{
					ID:   part.FunctionCall.ID,
					Type: "function",
					Function: openAIFunctionCall{
						Name:      part.FunctionCall.Name,
						Arguments: mapToJSON(part.FunctionCall.Args),
					},
				})
			}
			// Handle function responses
			if part.FunctionResponse != nil {
				msg.Role = "tool"
				msg.ToolCallID = part.FunctionResponse.ID
				respMap := part.FunctionResponse.Response
				if result, ok := respMap["result"]; ok {
					msg.Content = fmt.Sprintf("%v", result)
				} else {
					msg.Content = mapToJSON(respMap)
				}
			}
		}

		if len(textParts) > 0 {
			msg.Content = strings.Join(textParts, "\n")
		}

		messages = append(messages, msg)
	}

	openAIReq := &openAIRequest{
		Model:    m.name,
		Messages: messages,
	}

	// Apply generation config
	if req.Config != nil {
		if req.Config.Temperature != nil {
			openAIReq.Temperature = req.Config.Temperature
		}
		if req.Config.TopP != nil {
			openAIReq.TopP = req.Config.TopP
		}
		if req.Config.MaxOutputTokens != 0 {
			maxTokens := req.Config.MaxOutputTokens
			openAIReq.MaxTokens = &maxTokens
		}
		if req.Config.StopSequences != nil {
			openAIReq.Stop = req.Config.StopSequences
		}
	}

	// Convert tools if present
	if len(req.Tools) > 0 {
		openAIReq.Tools = convertTools(req.Tools)
	}

	return openAIReq, nil
}

// convertResponse converts OpenAI response to ADK LLMResponse
func (m *Model) convertResponse(resp *openAIResponse) *model.LLMResponse {
	llmResp := &model.LLMResponse{
		TurnComplete: true,
	}

	if len(resp.Choices) > 0 {
		choice := resp.Choices[0]

		content := &genai.Content{
			Role:  "model",
			Parts: []*genai.Part{},
		}

		// Add text content
		if choice.Message.Content != "" {
			content.Parts = append(content.Parts, &genai.Part{
				Text: choice.Message.Content,
			})
		}

		// Add tool calls
		for _, tc := range choice.Message.ToolCalls {
			content.Parts = append(content.Parts, &genai.Part{
				FunctionCall: &genai.FunctionCall{
					ID:   tc.ID,
					Name: tc.Function.Name,
					Args: jsonToMap(tc.Function.Arguments),
				},
			})
		}

		llmResp.Content = content
		llmResp.FinishReason = convertFinishReason(choice.FinishReason)
	}

	// Add usage metadata
	if resp.Usage != nil {
		llmResp.UsageMetadata = &genai.GenerateContentResponseUsageMetadata{
			PromptTokenCount:     int32(resp.Usage.PromptTokens),
			CandidatesTokenCount: int32(resp.Usage.CompletionTokens),
			TotalTokenCount:      int32(resp.Usage.TotalTokens),
		}
	}

	return llmResp
}

// Helper functions

func convertRole(role string) string {
	switch role {
	case "model":
		return "assistant"
	case "user":
		return "user"
	case "system":
		return "system"
	default:
		return role
	}
}

func convertFinishReason(reason string) genai.FinishReason {
	switch reason {
	case "stop":
		return genai.FinishReasonStop
	case "length":
		return genai.FinishReasonMaxTokens
	case "tool_calls", "function_call":
		return genai.FinishReasonStop
	case "content_filter":
		return genai.FinishReasonSafety
	default:
		return genai.FinishReasonUnspecified
	}
}

func convertTools(tools map[string]any) []openAITool {
	var result []openAITool

	// Tools come as FunctionDeclaration from ADK
	for name, toolDef := range tools {
		if fd, ok := toolDef.(*genai.FunctionDeclaration); ok {
			result = append(result, openAITool{
				Type: "function",
				Function: openAIFunction{
					Name:        fd.Name,
					Description: fd.Description,
					Parameters:  fd.Parameters,
				},
			})
		} else if fdMap, ok := toolDef.(map[string]any); ok {
			// Handle map-based tool definition
			fn := openAIFunction{Name: name}
			if desc, ok := fdMap["description"].(string); ok {
				fn.Description = desc
			}
			if params, ok := fdMap["parameters"]; ok {
				fn.Parameters = params
			}
			result = append(result, openAITool{
				Type:     "function",
				Function: fn,
			})
		}
	}

	return result
}

func mapToJSON(m map[string]any) string {
	if m == nil {
		return "{}"
	}
	b, err := json.Marshal(m)
	if err != nil {
		return "{}"
	}
	return string(b)
}

func jsonToMap(s string) map[string]any {
	var m map[string]any
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return map[string]any{}
	}
	return m
}
