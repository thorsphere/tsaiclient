// Copyright (c) 2026 thorsphere.
// All Rights Reserved. Use is governed by the Functional Source License v1.1
// (FSL-1.1-ALv2) that can be found in the LICENSE file.

package tsaiclient

// Import packages.
import (
	"bytes"         // Import the bytes package for byte slices.
	"context"       // Import the context package for contexts.
	"encoding/json" // Import the json package for JSON marshaling.
	"net/http"      // Import the http package for HTTP requests.
	"os"            // Import the os package for environment variables.
	"slices"        // Import the slices package for slice operations.
	"strings"       // Import the strings package for string operations.

	"github.com/thorsphere/tserr" // Import the tserr package for error handling.
)

const (
	openRouterAPIKeyEnv           = "OPENROUTER_API_KEY"                            // The environment variable name for the OpenRouter API key.
	openRouterBaseURL             = "https://openrouter.ai/api/v1/chat/completions" // The default OpenRouter base URL.
	openRouterAuthHeader          = "Authorization"                                 // The authorization header name.
	openRouterAuthValue           = "Bearer "                                       // The authorization header value prefix.
	openRouterRefererHeader       = "HTTP-Referer"                                  // The header identifying the calling application URL.
	openRouterTitleHeader         = "X-OpenRouter-Title"                            // The header identifying the calling application name.
	openRouterCategoriesHeader    = "X-OpenRouter-Categories"                       // The header listing comma-separated OpenRouter app categories.
	openRouterAppVisibilityHeader = "X-OpenRouter-App-Visibility"                   // The header controlling whether a new app is listed publicly.
	openRouterAppVisibilityHidden = "hidden"                                        // The visibility header value that excludes a new app from public rankings.
	openRouterMaxCategoryLen      = 30                                              // The maximum length in bytes of a single OpenRouter category slug.
	openRouterDataCollectionDeny  = "deny"                                          // openRouterDataCollectionDeny is the OpenRouter data collection policy valuethat forbids upstream providers from retaining or training on prompts.
)

// openRouterProviderPref configures OpenRouter upstream routing and
// privacy constraints in the chat request body.
type openRouterProviderPref struct {
	DataCollection string `json:"data_collection,omitempty"` // Set to "deny" in privacy mode to forbid upstream data collection.
	ZDR            bool   `json:"zdr,omitempty"`             // Set to true in privacy mode to restrict routing to Zero Data Retention endpoints.
}

// requestPayload is the JSON body sent to the provider chat endpoint.
type openRouterRequestPayload struct {
	Model               string                  `json:"model"`                           // The model identifier for the request.
	Messages            []message               `json:"messages"`                        // The conversation messages, oldest first.
	MaxCompletionTokens int                     `json:"max_completion_tokens,omitempty"` // The maximum number of tokens the model may generate; omitted when zero.
	Temperature         float64                 `json:"temperature,omitempty"`           // The sampling temperature; omitted when zero.
	SessionID           string                  `json:"session_id,omitempty"`            // The sticky routing key for prompt-cache affinity; omitted when empty.
	Provider            *openRouterProviderPref `json:"provider,omitempty"`              // Upstream routing and privacy preferences; omitted when nil.
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

func (p *OpenRouter) BuildHTTPRequest(ctx context.Context, cfg *Config, req *Request) (*http.Request, error) {
	// OpenRouter URL, headers, extraBody, and JSON marshaling directly here

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
	var messages []message
	// Prepend the system message when a system prompt is provided.
	if strings.TrimSpace(req.SystemPrompt) != "" {
		messages = append(messages, message{Role: "system", Content: req.SystemPrompt})
	}
	// Append the user prompt message.
	messages = append(messages, message{Role: "user", Content: req.Prompt})
	// Build payload
	payload := openRouterRequestPayload{
		Model:               cfg.model,
		Messages:            messages,
		MaxCompletionTokens: cfg.maxCompletionTokens,
		Temperature:         cfg.temperature,
		SessionID:           req.SessionID,
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

func (p *OpenRouter) ParseHTTPResponse(statusCode int, body []byte) (Result, error) {
	// OpenRouter response parsing directly here
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
		if err := validateEndpointURL(p.referer); err != nil {
			return err
		}
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

// isValidCategory reports whether s consists exclusively of lowercase ASCII
// letters, digits, and interior hyphens.
func isValidCategory(s string) bool {
	// Reject leading or trailing hyphens.
	if strings.HasPrefix(s, "-") || strings.HasSuffix(s, "-") {
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
	// Return true when all characters are valid.
	return len(s) > 0
}
