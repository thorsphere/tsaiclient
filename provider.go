// Copyright (c) 2026 thorsphere.
// All Rights Reserved. Use is governed by the Functional Source License v1.1
// (FSL-1.1-ALv2) that can be found in the LICENSE file.

package tsaiclient

// Import packages.
import (
	"context"  // Import the context package for cancellation.
	"net/http" // Import the http package for HTTP requests.
	"net/url"  // Import the url package for URL parsing.
	"strings"  // Import the strings package for string operations.

	"github.com/thorsphere/tserr" // Import the tserr package for error handling.
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
