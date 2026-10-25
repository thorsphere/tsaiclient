// Copyright (c) 2026 thorsphere.
// All Rights Reserved. Use is governed by the Functional Source License v1.1
// (FSL-1.1-ALv2) that can be found in the LICENSE file.

package tsaiclient

// Import packages.
import (
	"context"  // Import the context package for cancellation.
	"fmt"      // Import the fmt package for formatting.
	"net"      // Import the net package for IP address parsing.
	"net/http" // Import the http package for HTTP requests.
	"net/url"  // Import the url package for URL parsing.
	"strconv"  // Import the strconv package for numeric port validation.
	"strings"  // Import the strings package for string operations.
	"unicode"  // Import the unicode package for Unicode character classification.

	"github.com/thorsphere/tserr" // Import the tserr package for error handling.
)

// Provider HTTP status codes, TCP port bounds, and DNS name length limits.
const (
	statusEdgeTimeout        = 524   // The non-standard status code edge networks (e.g. Cloudflare) return when the upstream provider timed out.
	statusProviderOverloaded = 529   // The non-standard status code the provider returns when its upstream is temporarily overloaded.
	minPort                  = 1     // The minimum valid TCP port number.
	maxPort                  = 65535 // The maximum valid TCP port number.
	maxDNSLabelLength        = 63    // The maximum length of a single DNS label in bytes.
	maxDNSNameLength         = 253   // The maximum length of a fully qualified DNS name in bytes.
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
// port, or contains user credentials, a query string, or a fragment. The
// parameter field names the URL setting in returned errors. Returned errors
// redact user credentials, password, query string, and fragment, but echo the
// scheme, host, port, and path; rawURL must therefore not carry secrets in
// its path.
func validateHTTPSURL(field, rawURL string) error {
	// Return an error if the URL is empty or whitespace-only.
	if strings.TrimSpace(rawURL) == "" {
		return tserr.Empty(field)
	}
	// Return an error if the URL contains any whitespace.
	if strings.ContainsFunc(rawURL, unicode.IsSpace) {
		return tserr.InvalidFormat(&tserr.InvalidFormatArgs{
			F:      field,
			Value:  "<redacted>",
			Detail: "unexpected whitespace",
		})
	}
	// Parse the URL to inspect its components.
	u, err := url.Parse(rawURL)
	// Return an error if the URL cannot be parsed.
	if err != nil {
		return tserr.InvalidFormat(&tserr.InvalidFormatArgs{
			F:      field,
			Value:  "<redacted>",
			Detail: "malformed URL",
		})
	}
	// Record whether the URL carried a query string before stripping it for safe error reporting.
	hasQuery := u.RawQuery != "" || u.ForceQuery
	// Record whether the URL carried a fragment before stripping it for safe error reporting.
	hasFragment := u.Fragment != "" || strings.Contains(rawURL, "#")
	// Record whether the URL carried user credentials before stripping them for safe error reporting.
	hasUser := u.User != nil
	// Strip query, fragment, and user credentials so they are never echoed in error values.
	u.RawQuery = ""
	// Clear the bare question-mark flag before formatting the redacted URL.
	u.ForceQuery = false
	// Clear the fragment before formatting the redacted URL.
	u.Fragment = ""
	// Drop user credentials before formatting the redacted URL.
	u.User = nil
	// Format the stripped URL for safe error reporting.
	safeURL := u.Redacted()
	// Return an error if the URL is opaque rather than hierarchical.
	if u.Opaque != "" {
		return tserr.InvalidFormat(&tserr.InvalidFormatArgs{
			F:      field,
			Value:  "<redacted>",
			Detail: "unexpected opaque URL",
		})
	}
	// Return an error if the URL does not use HTTPS.
	if u.Scheme != "https" {
		return tserr.InvalidFormat(&tserr.InvalidFormatArgs{
			F:      field,
			Value:  safeURL,
			Detail: "expected scheme https",
		})
	}
	// Return an error if the URL contained user credentials.
	if hasUser {
		return tserr.InvalidFormat(&tserr.InvalidFormatArgs{
			F:      field,
			Value:  "<redacted>",
			Detail: "unexpected user credentials",
		})
	}
	// Return an error if the URL has no host.
	if u.Hostname() == "" {
		return tserr.InvalidFormat(&tserr.InvalidFormatArgs{
			F:      field,
			Value:  safeURL,
			Detail: "expected a host",
		})
	}
	// Return an error if the host is neither an IP literal nor a valid DNS name.
	if !validHostname(u.Hostname()) {
		return tserr.InvalidFormat(&tserr.InvalidFormatArgs{
			F:      field,
			Value:  safeURL,
			Detail: "invalid host",
		})
	}
	// Return an error if the host has a trailing colon with an empty port.
	if strings.HasSuffix(u.Host, ":") {
		return tserr.InvalidFormat(&tserr.InvalidFormatArgs{
			F:      field,
			Value:  safeURL,
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
				Value:  safeURL,
				Detail: "invalid port",
			})
		}
	}
	// Return an error if the URL contains a query string.
	if hasQuery {
		return tserr.InvalidFormat(&tserr.InvalidFormatArgs{
			F:      field,
			Value:  safeURL,
			Detail: "unexpected query string",
		})
	}
	// Return an error if the URL contains a fragment.
	if hasFragment {
		return tserr.InvalidFormat(&tserr.InvalidFormatArgs{
			F:      field,
			Value:  safeURL,
			Detail: "unexpected fragment",
		})
	}
	// Return no error to indicate success.
	return nil
}

// validHostname reports whether host is an IP literal with an optional IPv6
// zone identifier with a non-empty name, or a DNS name of dot-separated
// labels matching [A-Za-z0-9_-] with no leading or trailing hyphen, each
// label at most maxDNSLabelLength bytes and totalling at most
// maxDNSNameLength bytes, optionally ending in a single trailing dot.
// A zone identifier is accepted only directly after an IPv6 literal.
// Underscores are accepted because internal proxy and container DNS names
// use them.
func validHostname(host string) bool {
	// Record the host without any zone identifier for IP literal validation.
	ipHost := host
	// Strip the IPv6 zone identifier when the host carries one.
	if zone := strings.IndexByte(host, '%'); zone != -1 {
		// Reject an empty zone identifier or a zone identifier on a non-IPv6 host.
		if zone == len(host)-1 || !strings.Contains(host[:zone], ":") {
			return false
		}
		ipHost = host[:zone]
	}
	// Return true to indicate the host is an IP literal.
	if net.ParseIP(ipHost) != nil {
		return true
	}
	// Strip a single trailing dot, the root label of a fully qualified DNS name.
	host = strings.TrimSuffix(host, ".")
	// Reject a host that exceeds the DNS name length limit.
	if len(host) > maxDNSNameLength {
		return false
	}
	// Inspect each dot-separated DNS label of the host.
	for _, label := range strings.Split(host, ".") {
		// Reject an empty label.
		if label == "" {
			return false
		}
		// Reject a label that exceeds the DNS label length limit.
		if len(label) > maxDNSLabelLength {
			return false
		}
		// Reject a label that begins or ends with a hyphen.
		if strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return false
		}
		// Inspect each byte of the label.
		for i := 0; i < len(label); i++ {
			// Record the byte under inspection.
			c := label[i]
			// Reject a byte outside letters, digits, hyphens, and underscores.
			if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') && (c < '0' || c > '9') && c != '-' && c != '_' {
				return false
			}
		}
	}
	// Return true to indicate the host is a valid DNS name.
	return true
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
		// Return guidance when the requested model or endpoint does not exist or is not available. Note that
		// a valid model can also have no Zero Data Retention / data-policy-compliant upstream endpoints available.
		return "check the configured model name and privacy or routing settings"
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

// statusError returns an error for provider name when a response with status
// carried no usable error message. The returned error advises the caller how
// to recover when guidance applies to status and otherwise reports the status
// as unexpected.
func statusError(name string, status int) error {
	// Return an error carrying the advice when the status has one.
	if advice := statusAdvice(status); advice != "" {
		return tserr.NotAvailable(&tserr.NotAvailableArgs{
			S:   name,
			Err: fmt.Errorf("HTTP %d: %s", status, advice),
		})
	}
	// Return a generic, actionable error for non-2xx statuses without specific advice.
	return tserr.NotAvailable(&tserr.NotAvailableArgs{
		S:   name,
		Err: fmt.Errorf("HTTP %d: unexpected response from the provider; retry later or report the status code", status),
	})
}
