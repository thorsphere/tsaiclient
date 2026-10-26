// Copyright (c) 2026 thorsphere.
// All Rights Reserved. Use is governed by the Functional Source License v1.1
// (FSL-1.1-ALv2) that can be found in the LICENSE file.

package tsaiclient

// Import packages.
import (
	"bytes"         // Import the bytes package for byte slices.
	"context"       // Import the context package for contexts.
	"encoding/json" // Import the json package for JSON marshaling.
	"errors"        // Import the errors package for error handling.
	"fmt"           // Import the fmt package for formatting.
	"net/http"      // Import the http package for HTTP requests.
	"os"            // Import the os package for environment variables.
	"slices"        // Import the slices package for slice operations.
	"strings"       // Import the strings package for string operations.

	"github.com/thorsphere/tserr" // Import the tserr package for error handling.
)

// Set of OpenRouter-specific constants.
const (
	openRouterAPIKeyEnv                 = "OPENROUTER_API_KEY"                            // The environment variable name for the OpenRouter API key.
	openRouterBaseURL                   = "https://openrouter.ai/api/v1/chat/completions" // The default OpenRouter base URL.
	openRouterAuthHeader                = "Authorization"                                 // The authorization header name.
	openRouterAuthValue                 = "Bearer "                                       // The authorization header value prefix.
	openRouterRefererHeader             = "HTTP-Referer"                                  // The header identifying the calling application URL.
	openRouterTitleHeader               = "X-OpenRouter-Title"                            // The header identifying the calling application name.
	openRouterCategoriesHeader          = "X-OpenRouter-Categories"                       // The header listing comma-separated OpenRouter app categories.
	openRouterAppVisibilityHeader       = "X-OpenRouter-App-Visibility"                   // The header controlling whether a new app is listed publicly.
	openRouterAppVisibilityHidden       = "hidden"                                        // The visibility header value that excludes a new app from public rankings.
	openRouterMaxCategoryLen            = 30                                              // The maximum length in bytes of a single OpenRouter category slug.
	openRouterMaxCategories             = 2                                               // The maximum number of OpenRouter categories per request.
	openRouterDataCollectionDeny        = "deny"                                          // openRouterDataCollectionDeny is the OpenRouter data collection policy valuethat forbids upstream providers from retaining or training on prompts.
	openRouterFinishReasonLength        = "length"                                        // The provider finish reason indicating the generated message was truncated by the token limit.
	openRouterFinishReasonContentFilter = "content_filter"                                // The provider finish reason indicating the response was blocked by the provider's content filter.
	openRouterFinishReasonError         = "error"                                         // The provider finish reason indicating generation failed on the provider side.
)

// openRouterProviderPref configures OpenRouter upstream routing and
// privacy constraints in the chat request body.
type openRouterProviderPref struct {
	DataCollection string `json:"data_collection,omitempty"` // Set to "deny" in privacy mode to forbid upstream data collection.
	ZDR            bool   `json:"zdr,omitempty"`             // Set to true in privacy mode to restrict routing to Zero Data Retention endpoints.
}

// openRouterRequestPayload is the JSON body sent to the provider chat endpoint.
type openRouterRequestPayload struct {
	Model               string                  `json:"model"`                           // The model identifier for the request.
	Messages            []openRouterMessage     `json:"messages"`                        // The conversation messages, oldest first.
	MaxCompletionTokens int                     `json:"max_completion_tokens,omitempty"` // The maximum number of tokens the model may generate; omitted when zero.
	Temperature         float64                 `json:"temperature,omitempty"`           // The sampling temperature; omitted when zero.
	ReasoningEffort     string                  `json:"reasoning_effort,omitempty"`      // The reasoning effort level; omitted when empty.
	SessionID           string                  `json:"session_id,omitempty"`            // The sticky routing key for prompt-cache affinity; omitted when empty.
	Provider            *openRouterProviderPref `json:"provider,omitempty"`              // Upstream routing and privacy preferences; omitted when nil.
}

// openRouterMessage is a single chat message exchanged with the provider.
type openRouterMessage struct {
	Role    string `json:"role"`    // The conversational role, e.g. "user" or "assistant".
	Content string `json:"content"` // The message text.
}

// openRouterResponsePayload is the provider chat completion response body.
// The zero value means no completion was returned.
type openRouterResponsePayload struct {
	Choices []struct {
		Message      openRouterMessage `json:"message"`         // The generated message for this choice.
		FinishReason string            `json:"finish_reason"`   // Why generation stopped, e.g. "stop" or "length".
		Error        *openRouterError  `json:"error,omitempty"` // Mid-generation error embedded in the choice on 200 OK.
	} `json:"choices"` // The completion choices returned by the model.
	Usage openRouterUsage  `json:"usage"`           // The token accounting for the completion.
	Error *openRouterError `json:"error,omitempty"` // Top-level error returned when upstream fails after 200 OK headers.
}

// openRouterErrorMetadata holds structured diagnostic fields attached to an
// OpenRouter error object.
type openRouterErrorMetadata struct {
	ErrorType    string   `json:"error_type,omitempty"`    // Canonical OpenRouter error category (e.g. "rate_limit_exceeded").
	ProviderName string   `json:"provider_name,omitempty"` // Name of the upstream provider that reported the error.
	ProviderCode string   `json:"provider_code,omitempty"` // Original upstream provider error code.
	RemedyHint   string   `json:"remedy_hint,omitempty"`   // Actionable next step for budget limit errors.
	Reasons      []string `json:"reasons,omitempty"`       // Moderation reasons when input was flagged.
}

// openRouterError is the error detail object returned by OpenRouter at the
// top level or inside a completion choice.
type openRouterError struct {
	Code     int                      `json:"code"`               // The HTTP status code associated with the error.
	Message  string                   `json:"message"`            // The human-readable error message.
	Metadata *openRouterErrorMetadata `json:"metadata,omitempty"` // Optional structured error metadata.
}

// openRouterErrorResponse is the provider error body returned for non-2xx responses.
type openRouterErrorResponse struct {
	Error openRouterError `json:"error"` // The error details.
}

// openRouterUsage is the token accounting object in the OpenRouter response body.
type openRouterUsage struct {
	PromptTokens     int     `json:"prompt_tokens"`     // The number of tokens in the prompt.
	CompletionTokens int     `json:"completion_tokens"` // The number of tokens in the completion.
	TotalTokens      int     `json:"total_tokens"`      // The total number of tokens billed for the call.
	Cost             float64 `json:"cost"`              // The completion cost in USD.
	Details          struct {
		ReasoningTokens int `json:"reasoning_tokens"` // The reasoning tokens generated.
	} `json:"completion_tokens_details"` // The breakdown of completion tokens.
}

// OpenRouter implements Provider for the OpenRouter Chat Completions API.
// The zero value is ready to use without app attribution.
type OpenRouter struct {
	referer    string   // Optional app URL sent as HTTP-Referer (required when any attribution field is set).
	title      string   // Optional app display name sent as X-OpenRouter-Title.
	categories []string // Optional app categories sent as X-OpenRouter-Categories.
	hidden     bool     // When true, sends X-OpenRouter-App-Visibility: hidden.
	privacy    bool     // When true, sends data_collection: deny and zdr: true.
}

// NewOpenRouter returns a new OpenRouter provider instance.
func NewOpenRouter() *OpenRouter {
	// Return a new OpenRouter provider.
	return &OpenRouter{}
}

// EnablePrivacy enables strict privacy mode for OpenRouter requests by setting
// provider.zdr to true (routing exclusively to Zero Data Retention endpoints)
// and provider.data_collection to "deny" (forbidding upstream providers from
// retaining or training on user prompts and completions). Note that OpenRouter
// does not support geographic routing restrictions (such as EU-only residency).
// It returns an error if the receiver is nil.
func (p *OpenRouter) EnablePrivacy() error {
	// Return an error if the receiver is nil.
	if p == nil {
		return tserr.NilPtr()
	}
	// Enable strict privacy mode on the provider.
	p.privacy = true
	// Return no error to indicate success.
	return nil
}

// WithReferer sets the HTTP-Referer attribution URL and returns p.
func (p *OpenRouter) WithReferer(referer string) *OpenRouter {
	// Set the referer when the receiver is non-nil.
	if p != nil {
		p.referer = strings.TrimSpace(referer)
	}
	// Return the updated provider.
	return p
}

// WithTitle sets the X-OpenRouter-Title attribution header and returns p.
func (p *OpenRouter) WithTitle(title string) *OpenRouter {
	// Set the title when the receiver is non-nil.
	if p != nil {
		p.title = strings.TrimSpace(title)
	}
	// Return the updated provider.
	return p
}

// WithCategories sets the X-OpenRouter-Categories header values and returns p.
func (p *OpenRouter) WithCategories(categories []string) *OpenRouter {
	// Set the categories when the receiver is non-nil.
	if p != nil {
		p.categories = slices.Clone(categories)
	}
	// Return the updated provider.
	return p
}

// WithHidden controls whether X-OpenRouter-App-Visibility is set to "hidden"
// when registering a new app with OpenRouter, and returns p.
func (p *OpenRouter) WithHidden(hidden bool) *OpenRouter {
	// Set the hidden flag when the receiver is non-nil.
	if p != nil {
		p.hidden = hidden
	}
	// Return the updated provider.
	return p
}

// Name returns the provider name.
func (p *OpenRouter) Name() string {
	// Return the provider name.
	return "OpenRouter"
}

// BuildHTTPRequest returns an authenticated HTTP POST request bound to ctx
// for the OpenRouter chat completions endpoint using cfg and req. It returns
// an error if p, cfg, or req is nil or invalid, or if marshaling the payload
// or constructing the HTTP request fails.
func (p *OpenRouter) BuildHTTPRequest(ctx context.Context, cfg *Config, req *Request) (*http.Request, error) {
	// Return an error if the provider is nil.
	if p == nil {
		return nil, tserr.NilPtr()
	}
	// Return an error if the provider is invalid.
	if err := p.Validate(); err != nil {
		return nil, err
	}
	// Return an error if the configuration is nil.
	if cfg == nil {
		return nil, tserr.Empty("config")
	}
	// Return an error if the configuration is invalid.
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	// Return an error if the request is nil.
	if req == nil {
		return nil, tserr.Empty("request")
	}
	// Return an error if the request is invalid.
	if err := req.Validate(); err != nil {
		return nil, err
	}
	// Initialize the conversation message list.
	var messages []openRouterMessage
	// Prepend the system message when a system prompt is provided.
	if strings.TrimSpace(req.systemPrompt) != "" {
		messages = append(messages, openRouterMessage{Role: "system", Content: req.systemPrompt})
	}
	// Append the user prompt message.
	messages = append(messages, openRouterMessage{Role: "user", Content: req.prompt})
	// Build payload
	payload := openRouterRequestPayload{
		Model:               cfg.model,
		Messages:            messages,
		MaxCompletionTokens: cfg.maxCompletionTokens,
		Temperature:         cfg.temperature,
		ReasoningEffort:     cfg.reasoningEffort,
		SessionID:           req.sessionID,
	}
	// Attach strict privacy routing preferences when privacy mode is enabled.
	if p.privacy {
		payload.Provider = &openRouterProviderPref{
			DataCollection: openRouterDataCollectionDeny,
			ZDR:            true,
		}
	}
	// Marshal payload
	body, err := json.Marshal(payload)
	// Return the zero result and an error if marshalling payload fails.
	if err != nil {
		return nil, tserr.Op(&tserr.OpArgs{Op: "marshal payload", Fn: p.Name(), Err: err})
	}
	// Create request with body
	httpReq, err := http.NewRequestWithContext(ctx, "POST", openRouterBaseURL, bytes.NewBuffer(body))
	// Return nil and an error if creating request fails.
	if err != nil {
		return nil, tserr.Op(&tserr.OpArgs{Op: "new http request", Fn: p.Name(), Err: err})
	}
	// Set headers
	// Authorization header
	httpReq.Header.Set(openRouterAuthHeader, openRouterAuthValue+os.Getenv(openRouterAPIKeyEnv))
	// Content-Type header
	httpReq.Header.Set("Content-Type", "application/json")
	// Set the HTTP-Referer header when provided by the calling app.
	if p.referer != "" {
		httpReq.Header.Set(openRouterRefererHeader, p.referer)
	}
	// Set the X-OpenRouter-Title header when provided by the calling app.
	if p.title != "" {
		httpReq.Header.Set(openRouterTitleHeader, p.title)
	}
	// Set the X-OpenRouter-Categories header when provided by the calling app.
	if len(p.categories) > 0 {
		httpReq.Header.Set(openRouterCategoriesHeader, strings.Join(p.categories, ","))
	}
	// Set the X-OpenRouter-App-Visibility header when the app is marked hidden.
	if p.hidden {
		httpReq.Header.Set(openRouterAppVisibilityHeader, openRouterAppVisibilityHidden)
	}
	// Return the constructed HTTP request.
	return httpReq, nil
}

// ParseHTTPResponse parses the HTTP statusCode and response body from the
// OpenRouter chat completions endpoint and returns a normalized Result for
// the first completion choice. Later choices are ignored and cannot replace
// an empty or failed first choice. It returns an error if p is nil, the
// status code or payload indicates an upstream error, the response cannot be
// unmarshaled, or no usable completion content was produced.
func (p *OpenRouter) ParseHTTPResponse(statusCode int, body []byte) (Result, error) {
	// Return an error if the provider is nil.
	if p == nil {
		return Result{}, tserr.NilPtr()
	}
	// Handle non-2xx HTTP responses.
	if statusCode < http.StatusOK || statusCode >= http.StatusMultipleChoices {
		// Declare the error envelope to decode into.
		var errRes openRouterErrorResponse
		// Use any structured error details when the body can be decoded.
		if err := json.Unmarshal(body, &errRes); err == nil {
			return Result{}, openRouterResponseError(p.Name(), statusCode, errRes.Error)
		}
		// Fall back to the HTTP status when the body cannot be decoded.
		return Result{}, statusError(p.Name(), statusCode)
	}
	// Unmarshal the 2xx response body.
	var res openRouterResponsePayload
	if err := json.Unmarshal(body, &res); err != nil {
		// Return an error if unmarshaling the response payload fails.
		return Result{}, tserr.Op(&tserr.OpArgs{Op: "unmarshal response", Fn: p.Name(), Err: err})
	}
	// Check for a top-level error returned inside a 200 OK body when upstream failed after headers were sent.
	if res.Error != nil {
		// Return the provider error or a status fallback, using HTTP 502 when no code is available.
		return Result{}, openRouterResponseError(p.Name(), http.StatusBadGateway, *res.Error)
	}
	// Return an error if the response contains no completion choices.
	if len(res.Choices) == 0 {
		return Result{}, tserr.Empty(p.Name() + " response choices")
	}
	// Select the first completion choice.
	choice := res.Choices[0]
	// Check for a mid-generation error embedded inside the completion choice.
	if choice.Error != nil {
		// Return the provider error or a status fallback, using HTTP 502 when no code is available.
		return Result{}, openRouterResponseError(p.Name(), http.StatusBadGateway, *choice.Error)
	}
	// Reject responses flagged as blocked or failed by finish_reason.
	switch choice.FinishReason {
	case openRouterFinishReasonContentFilter:
		// Return an error when generation was stopped by a content filter.
		return Result{}, tserr.Op(&tserr.OpArgs{
			Op:  "generate completion",
			Fn:  p.Name(),
			Err: errors.New("response blocked by content filter"),
		})
	case openRouterFinishReasonError:
		// Return an error when the provider reported a generation error without a choice error body.
		return Result{}, tserr.Op(&tserr.OpArgs{
			Op:  "generate completion",
			Fn:  p.Name(),
			Err: errors.New("provider reported a generation error"),
		})
	}
	// Extract the generated text content.
	content := choice.Message.Content
	// Return an error when no content was generated.
	if content == "" {
		// Report when a reasoning model exhausted max_completion_tokens on reasoning before emitting content.
		if choice.FinishReason == openRouterFinishReasonLength && res.Usage.Details.ReasoningTokens > 0 {
			return Result{}, tserr.Op(&tserr.OpArgs{
				Op:  "generate completion",
				Fn:  p.Name(),
				Err: errors.New("reasoning tokens exhausted max_completion_tokens before content was generated"),
			})
		}
		// Return an empty-content error for cold starts or empty upstream responses.
		return Result{}, tserr.Empty(p.Name() + " completion content")
	}
	// Return the normalized completion result.
	return Result{
		Content:   content,
		Truncated: choice.FinishReason == openRouterFinishReasonLength,
		Usage: Usage{
			PromptTokens:     res.Usage.PromptTokens,
			CompletionTokens: res.Usage.CompletionTokens,
			ReasoningTokens:  res.Usage.Details.ReasoningTokens,
			TotalTokens:      res.Usage.TotalTokens,
			Cost:             res.Usage.Cost,
		},
	}, nil
}

// openRouterResponseError returns a structured provider error when apiErr has
// a nonblank message. Otherwise, it returns a status error using apiErr.Code
// when available, or status when no embedded code was reported.
func openRouterResponseError(name string, status int, apiErr openRouterError) error {
	// Preserve the structured provider error when it carries a usable message.
	if strings.TrimSpace(apiErr.Message) != "" {
		return openRouterProviderError(name, status, apiErr)
	}
	// Use the embedded error code for blank or missing messages when available.
	if apiErr.Code != 0 {
		return statusError(name, apiErr.Code)
	}
	// Fall back to the HTTP status when no embedded code was reported.
	return statusError(name, status)
}

// openRouterProviderError returns an error for an OpenRouter error payload,
// combining the error code, message, structured metadata, and status advice.
func openRouterProviderError(name string, status int, apiErr openRouterError) error {
	// Start with the human-readable error message.
	detail := apiErr.Message
	// Append metadata context when OpenRouter included structured metadata.
	if m := apiErr.Metadata; m != nil {
		// Include the canonical error type when present.
		if m.ErrorType != "" {
			detail += fmt.Sprintf(" (type: %s)", m.ErrorType)
		}
		// Include the upstream provider name when present.
		if m.ProviderName != "" {
			detail += fmt.Sprintf(" (provider: %s)", m.ProviderName)
		}
		// Include the upstream provider error code when present.
		if m.ProviderCode != "" {
			detail += fmt.Sprintf(" (provider code: %s)", m.ProviderCode)
		}
		// Include moderation reasons when input was flagged.
		if len(m.Reasons) > 0 {
			detail += fmt.Sprintf(" [reasons: %s]", strings.Join(m.Reasons, ", "))
		}
		// Include the budget remedy hint when present.
		if m.RemedyHint != "" {
			detail += "; " + m.RemedyHint
		}
	}
	// Prefix the numeric error code when reported.
	if apiErr.Code != 0 {
		detail = fmt.Sprintf("%d: %s", apiErr.Code, detail)
	}
	// Append actionable status advice when no specific remedy hint was already provided.
	if apiErr.Metadata == nil || apiErr.Metadata.RemedyHint == "" {
		// Prefer advice for the embedded error code.
		advice := statusAdvice(apiErr.Code)
		// Fall back to advice for the HTTP status code when the embedded code has none.
		if advice == "" {
			advice = statusAdvice(status)
		}
		// Append the advice when available.
		if advice != "" {
			detail += "; " + advice
		}
	}
	// Return the provider error wrapped with the provider name.
	return tserr.NotAvailable(&tserr.NotAvailableArgs{S: name, Err: errors.New(detail)})
}

// Validate validates the OpenRouter provider configuration and environment.
// It returns an error if the API key is missing or if attribution settings
// violate OpenRouter's header constraints.
func (p *OpenRouter) Validate() error {
	// Return an error if the provider is nil.
	if p == nil {
		return tserr.NilPtr()
	}
	// Return an error if the API key is not set.
	if strings.TrimSpace(os.Getenv(openRouterAPIKeyEnv)) == "" {
		return tserr.Empty("OpenRouter API key")
	}
	// Require HTTP-Referer whenever secondary attribution headers are configured.
	if p.referer == "" && (p.title != "" || len(p.categories) > 0 || p.hidden) {
		return tserr.Empty("OpenRouter HTTP-Referer")
	}
	// Validate the HTTP-Referer URL when provided.
	if p.referer != "" {
		if err := validateHTTPSURL("OpenRouter HTTP-Referer", p.referer); err != nil {
			return err
		}
	}
	// Return an error if more than two categories are configured.
	if len(p.categories) > openRouterMaxCategories {
		return tserr.Lower(&tserr.LowerArgs{
			Var:         "OpenRouter category count",
			HigherBound: openRouterMaxCategories,
			Actual:      int64(len(p.categories)),
		})
	}
	// Validate each category slug when categories are provided.
	for _, cat := range p.categories {
		// Return an error if a category entry is empty.
		if cat == "" {
			return tserr.Empty("OpenRouter category")
		}
		// Return an error if a category exceeds 30 characters.
		if len(cat) > openRouterMaxCategoryLen {
			return tserr.Lower(&tserr.LowerArgs{
				Var:         "OpenRouter category length",
				HigherBound: openRouterMaxCategoryLen,
				Actual:      int64(len(cat)),
			})
		}
		// Return an error if the category is not lowercase hyphen-separated ASCII.
		if !isValidCategory(cat) {
			return tserr.InvalidFormat(&tserr.InvalidFormatArgs{
				F:      "OpenRouter category",
				Value:  cat,
				Detail: "expected lowercase letters, digits, and hyphens",
			})
		}
	}
	// Return no error to indicate success.
	return nil
}

// isValidCategory reports whether s consists of non-empty groups of lowercase
// ASCII letters and digits joined by single hyphens.
func isValidCategory(s string) bool {
	// Reject leading, trailing, or repeated hyphens.
	if strings.HasPrefix(s, "-") || strings.HasSuffix(s, "-") || strings.Contains(s, "--") {
		return false
	}
	// Inspect each byte in s.
	for i := 0; i < len(s); i++ {
		c := s[i]
		// Allow lowercase letters, digits, and hyphens.
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' {
			continue
		}
		// Reject any other character.
		return false
	}
	// Return true when all characters are valid and s is non-empty.
	return len(s) > 0
}
