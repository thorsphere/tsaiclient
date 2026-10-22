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
	"strconv"  // Import the strconv package for numeric port validation.
	"strings"  // Import the strings package for string operations.

	"github.com/thorsphere/tserr" // Import the tserr package for error handling.
)

// Provider HTTP status codes and TCP port bounds.
const (
	statusEdgeTimeout        = 524   // The non-standard status code edge networks (e.g. Cloudflare) return when the upstream provider timed out.
	statusProviderOverloaded = 529   // The non-standard status code the provider returns when its upstream is temporarily overloaded.
	minPort                  = 1     // The minimum valid TCP port number.
	maxPort                  = 65535 // The maximum valid TCP port number.
)

// Provider describes how to format requests for and parse responses from an AI backend and
// how to handle privacy-related settings.
type Provider interface {
	Name() string                                                                           // The provider name used in error contexts.
	EnablePrivacy() error                                                                   // Enable strict privacy mode for the provider.
	Validate() error                                                                        // Validate the provider configuration.
	BuildHTTPRequest(ctx context.Context, cfg *Config, req *Request) (*http.Request, error) // Build an HTTP request for the provider.
	ParseHTTPResponse(statusCode int, body []byte) (Result, error)                          // Parse an HTTP response from the provider.
}

// validateHTTPSURL returns an error if rawURL is empty, contains whitespace,
// cannot be parsed as a hierarchical HTTPS URL with a valid host and optional
// port, or contains user credentials. Parameter field names the URL setting in
// returned errors.
func validateHTTPSURL(field, rawURL string) error {
	// Return an error if the URL is empty or whitespace-only.
	if strings.TrimSpace(rawURL) == "" {
		return tserr.Empty(field)
	}
	// Return an error if the URL contains any whitespace.
	if strings.ContainsAny(rawURL, " \t\r\n") {
		return tserr.InvalidFormat(&tserr.InvalidFormatArgs{
			F:      field,
			Value:  rawURL,
			Detail: "unexpected whitespace",
		})
	}
	// Parse the URL to inspect its components.
	u, err := url.Parse(rawURL)
	// Return an error if the URL cannot be parsed.
	if err != nil {
		return tserr.InvalidFormat(&tserr.InvalidFormatArgs{
			F:      field,
			Value:  rawURL,
			Detail: err.Error(),
		})
	}
	// Return an error if the URL is opaque rather than hierarchical.
	if u.Opaque != "" {
		return tserr.InvalidFormat(&tserr.InvalidFormatArgs{
			F:      field,
			Value:  rawURL,
			Detail: "unexpected opaque URL",
		})
	}
	// Return an error if the URL does not use HTTPS.
	if u.Scheme != "https" {
		return tserr.InvalidFormat(&tserr.InvalidFormatArgs{
			F:      field,
			Value:  rawURL,
			Detail: "expected scheme https",
		})
	}
	// Return an error if the URL contains user credentials.
	if u.User != nil {
		return tserr.InvalidFormat(&tserr.InvalidFormatArgs{
			F:      field,
			Value:  rawURL,
			Detail: "unexpected user credentials",
		})
	}
	// Return an error if the URL has no host.
	if u.Hostname() == "" {
		return tserr.InvalidFormat(&tserr.InvalidFormatArgs{
			F:      field,
			Value:  rawURL,
			Detail: "expected a host",
		})
	}
	// Return an error if the host has a trailing colon with an empty port.
	if strings.HasSuffix(u.Host, ":") {
		return tserr.InvalidFormat(&tserr.InvalidFormatArgs{
			F:      field,
			Value:  rawURL,
			Detail: "empty port",
		})
	}
	// Validate the port number when an explicit port is present.
	if portStr := u.Port(); portStr != "" {
		// Parse the port string as an integer.
		port, err := strconv.Atoi(portStr)
		// Return an error if the port overflows or is outside [1, 65535].
		if err != nil || port < minPort || port > maxPort {
			return tserr.InvalidFormat(&tserr.InvalidFormatArgs{
				F:      field,
				Value:  rawURL,
				Detail: "invalid port",
			})
		}
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
		// Return guidance for malformed or invalid request parameters.
		return "check the configured model and provider settings"
	case http.StatusUnauthorized:
		// Return guidance for missing or invalid authentication credentials.
		return "check that the API key is set correctly"
	case http.StatusPaymentRequired:
		// Return guidance when the provider account has insufficient funds.
		return "insufficient credits; add credits to the provider account"
	case http.StatusForbidden:
		// Return guidance when the API key lacks access to the requested resource.
		return "check the API key permissions for this model"
	case http.StatusNotFound:
		// Return guidance when the requested model or endpoint does not exist.
		return "check the configured model name"
	case http.StatusRequestEntityTooLarge:
		// Return guidance when the request body exceeds the provider's size limit.
		return "reduce the prompt or request payload size"
	case http.StatusTooManyRequests:
		// Return guidance when the caller has exceeded the provider's rate limit.
		return "rate limited; retry later"
	case http.StatusRequestTimeout, http.StatusGatewayTimeout, statusEdgeTimeout:
		// Return guidance when the request or upstream gateway timed out.
		return "the provider timed out; retry later"
	case http.StatusInternalServerError, http.StatusBadGateway,
		http.StatusServiceUnavailable, statusProviderOverloaded:
		// Return guidance when the provider experiences a transient server failure or overload.
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
