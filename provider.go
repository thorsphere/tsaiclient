// Copyright (c) 2026 thorsphere.
// All Rights Reserved. Use is governed by the Functional Source License v1.1
// (FSL-1.1-ALv2) that can be found in the LICENSE file.

package tsaiclient

// Import packages.
import (
	"context"  // Import the context package for cancellation.
	"fmt"      // Import the fmt package for formatting.
	"net/http" // Import the http package for HTTP requests.
	"net/url"  // Import the url package for URL parsing.
	"strings"  // Import the strings package for string operations.

	"github.com/thorsphere/tserr" // Import the tserr package for error handling.
)

// statusEdgeTimeout is the non-standard status code edge networks (e.g.
// Cloudflare) return when the upstream provider timed out.
const statusEdgeTimeout = 524

// statusProviderOverloaded is the non-standard status code the provider
// returns when its upstream is temporarily overloaded.
const statusProviderOverloaded = 529

// Provider describes how to format requests for and parse responses from an AI backend and
// how to handle privacy-related settings.
type Provider interface {
	Name() string                                                                           // The provider name used in error contexts.
	EnablePrivacy() error                                                                   // Enable strict privacy mode for the provider.
	Validate() error                                                                        // Validate the provider configuration.
	BuildHTTPRequest(ctx context.Context, cfg *Config, req *Request) (*http.Request, error) // Build an HTTP request for the provider.
	ParseHTTPResponse(statusCode int, body []byte) (Result, error)                          // Parse an HTTP response from the provider.
}

// validateEndpointURL returns an error if rawURL is empty, cannot be parsed,
// does not use the https scheme, or has no host. It returns no error when
// rawURL is an absolute HTTPS URL with a host.
func validateEndpointURL(rawURL string) error {
	// Trim whitespace from rawURL.
	rawURL = strings.TrimSpace(rawURL)
	// Return an error if the endpoint URL is empty.
	if rawURL == "" {
		return tserr.Empty("endpoint URL")
	}
	// Parse the endpoint URL to inspect its components.
	u, err := url.Parse(rawURL)
	// Return an error if the endpoint URL cannot be parsed.
	if err != nil {
		return tserr.InvalidFormat(&tserr.InvalidFormatArgs{
			F:      "endpoint URL",
			Value:  rawURL,
			Detail: err.Error(),
		})
	}
	// Return an error if the endpoint URL does not use HTTPS.
	if u.Scheme != "https" {
		return tserr.InvalidFormat(&tserr.InvalidFormatArgs{
			F:      "endpoint URL",
			Value:  rawURL,
			Detail: "expected scheme https",
		})
	}
	// Return an error if the endpoint URL has no host.
	if u.Hostname() == "" {
		return tserr.InvalidFormat(&tserr.InvalidFormatArgs{
			F:      "endpoint URL",
			Value:  rawURL,
			Detail: "expected a host",
		})
	}
	// Return no error to indicate success.
	return nil
}

// statusAdvice returns actionable, human-readable guidance for an HTTP status
// returned by the provider. It returns an empty string when no specific
// guidance applies.
func statusAdvice(status int) string {
	// Select guidance matching the status code.
	switch status {
	case http.StatusBadRequest, http.StatusUnprocessableEntity:
		return "check the configured model and provider settings"
	case http.StatusUnauthorized:
		return "check that the API key is set correctly"
	case http.StatusPaymentRequired:
		return "insufficient credits; add credits to the provider account"
	case http.StatusForbidden:
		return "check the API key permissions for this model"
	case http.StatusNotFound:
		return "check the configured model name"
	case http.StatusRequestEntityTooLarge:
		return "reduce the prompt or request payload size"
	case http.StatusTooManyRequests:
		return "rate limited; retry later"
	case http.StatusRequestTimeout, http.StatusGatewayTimeout, statusEdgeTimeout:
		return "the provider timed out; retry later"
	case http.StatusInternalServerError, http.StatusBadGateway,
		http.StatusServiceUnavailable, statusProviderOverloaded:
		return "the provider is temporarily unavailable; retry later"
	}
	// Return no guidance for other status codes.
	return ""
}

// statusError returns an error for a non-2xx provider response that carried no
// usable error message. It prefers actionable advice for the status and falls
// back to a status mismatch error.
func statusError(name string, status int) error {
	// Prefer actionable advice when the status has one.
	if advice := statusAdvice(status); advice != "" {
		return tserr.Op(&tserr.OpArgs{
			Op:  "chat completion",
			Fn:  name,
			Err: fmt.Errorf("HTTP %d: %s", status, advice),
		})
	}
	// Return the status mismatch error when no advice applies.
	return tserr.StatusNotMatching(&tserr.StatusNotMatchingArgs{
		Expected: http.StatusOK, Actual: status,
	})
}
