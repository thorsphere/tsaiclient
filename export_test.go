// Copyright (c) 2026 thorsphere.
// All Rights Reserved. Use is governed by the Functional Source License v1.1
// (FSL-1.1-ALv2) that can be found in the LICENSE file.

package tsaiclient

// Import packages.
import "net/http" // Import the http package for HTTP requests.

// Unexported functions exported for external tests.
var (
	ValidateHTTPSURL        = validateHTTPSURL        // Exports validateHTTPSURL for external tests.
	StatusAdvice            = statusAdvice            // Exports statusAdvice for external tests.
	StatusError             = statusError             // Exports statusError for external tests.
	ValidHostname           = validHostname           // Exports validHostname for external tests.
	IsValidCategory         = isValidCategory         // Exports isValidCategory for external tests.
	OpenRouterProviderError = openRouterProviderError // Exports openRouterProviderError for external tests.
	IsValidHeaderValue      = isValidHeaderValue      // Exports isValidHeaderValue for external tests.
)

type (
	OpenRouterErrorMetadata = openRouterErrorMetadata // Exports openRouterErrorMetadata for external tests.
	OpenRouterError         = openRouterError         // Exports openRouterError for external tests.
)

// GetSessionID returns the session ID of the request.
func (req *Request) GetSessionID() string {
	// Set the session ID when the request is non-nil.
	if req != nil {
		return req.sessionID
	}
	// Return an empty string when the request is nil.
	return ""
}

// MaxResponseBytes is the maximum number of bytes read from a response body.
const MaxResponseBytes = maxResponseBytes

// SetHTTPDo replaces the HTTP execution seam and returns a restore function.
func SetHTTPDo(fn func(*http.Request) (*http.Response, error)) func() {
	// Save the previous HTTP execution function.
	prev := httpDo
	// Install the replacement HTTP execution function.
	httpDo = fn
	// Return a function that restores the previous HTTP execution function.
	return func() {
		httpDo = prev
	}
}

// NewBareClient constructs a Client without validation for testing defensive guards.
func NewBareClient(cfg *Config, provider Provider) *Client {
	// Return a Client with the raw fields.
	return &Client{cfg: cfg, provider: provider}
}
