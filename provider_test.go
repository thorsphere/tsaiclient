// Copyright (c) 2026 thorsphere.
// All Rights Reserved. Use is governed by the Functional Source License v1.1
// (FSL-1.1-ALv2) that can be found in the LICENSE file.

package tsaiclient_test

// Import packages.
import (
	"net/http" // Import the http package for HTTP status codes.
	"strconv"  // Import the strconv package for numeric port validation.
	"strings"  // Import the strings package for string operations.
	"testing"  // Import the testing package for testing.

	"github.com/thorsphere/tsaiclient" // Import the tsaiclient package for testing.
	"github.com/thorsphere/tserr"      // Import the tserr package for error handling.
)

// TestValidateHTTPSURL tests the HTTPS URL validation logic of validateHTTPSURL.
func TestValidateHTTPSURL(t *testing.T) {
	// Run subtests in parallel.
	t.Parallel()
	// Define test cases.
	tests := []struct {
		name    string // The name of the test case.
		rawURL  string // The URL string under test.
		wantErr bool   // Whether an error is expected.
	}{
		{name: "valid URL", rawURL: "https://endpoint.ai/api/v1/chat/completions", wantErr: false},
		{name: "valid URL with explicit port", rawURL: "https://endpoint.ai:8443/api/v1/chat/completions", wantErr: false},
		{name: "empty URL", rawURL: "", wantErr: true},
		{name: "whitespace-only URL", rawURL: "   \t\n  ", wantErr: true},
		{name: "interior whitespace", rawURL: "https://endpoint.ai/chat completions", wantErr: true},
		{name: "control character in URL", rawURL: "https://endpoint.ai/\x7f", wantErr: true},
		{name: "opaque URL", rawURL: "https:opaque@endpoint.ai", wantErr: true},
		{name: "relative URL", rawURL: "/api/v1/chat/completions", wantErr: true},
		{name: "non-https URL", rawURL: "http://endpoint.ai/api/v1/chat/completions", wantErr: true},
		{name: "missing host", rawURL: "https://", wantErr: true},
		{name: "missing host with path", rawURL: "https:///api/v1", wantErr: true},
		{name: "empty port after colon", rawURL: "https://endpoint.ai:/api/v1", wantErr: true},
		{name: "non-numeric port", rawURL: "https://endpoint.ai:abc/api/v1", wantErr: true},
		{name: "port below minimum", rawURL: "https://endpoint.ai:0/api/v1", wantErr: true},
		{name: "port above maximum", rawURL: "https://endpoint.ai:65536/api/v1", wantErr: true},
		{name: "embedded user credentials", rawURL: "https://user:pass@endpoint.ai/api/v1", wantErr: true},
		{name: "surrounding whitespace", rawURL: "  https://endpoint.ai/api/v1/chat/completions  ", wantErr: true},
		{name: "port integer overflow", rawURL: "https://endpoint.ai:999999999999999999999/api/v1", wantErr: true},
		{name: "URL with query string", rawURL: "https://endpoint.ai/api/v1?token=secret", wantErr: true},
		{name: "URL with bare question mark", rawURL: "https://endpoint.ai/api/v1?", wantErr: true},
		{name: "URL with fragment", rawURL: "https://endpoint.ai/api/v1#section", wantErr: true},
		{name: "URL with bare hash", rawURL: "https://endpoint.ai/api/v1#", wantErr: true},
		{name: "bracketed IPv6 literal", rawURL: "https://[2001:db8::1]:8443/api/v1", wantErr: false},
		{name: "unbracketed IPv6 literal in host", rawURL: "https://::1/", wantErr: true},
		{name: "IPv6 literal with an empty zone", rawURL: "https://[fe80::1%25]/", wantErr: true},
		{name: "zone identifier on an IPv4 literal", rawURL: "https://192.0.2.1%25eth0/", wantErr: true},
	}
	// Iterate over test cases.
	for _, tt := range tests {
		// Run each test case as a subtest.
		t.Run(tt.name, func(t *testing.T) {
			// Run subtest in parallel.
			t.Parallel()
			// Validate the URL under test.
			err := tsaiclient.ValidateHTTPSURL("test URL", tt.rawURL)
			// Check whether an expected error was missing.
			if tt.wantErr && err == nil {
				t.Fatal(tserr.NilFailed("ValidateHTTPSURL"))
			}
			// Check whether an unexpected error was returned.
			if !tt.wantErr && err != nil {
				t.Fatal(tserr.NilExpected(&tserr.NilExpectedArgs{Op: "ValidateHTTPSURL", Err: err}))
			}
		})
	}
}

// TestStatusAdvice tests the HTTP status guidance mapping of statusAdvice.
func TestStatusAdvice(t *testing.T) {
	// Run subtests in parallel.
	t.Parallel()
	// Define test cases.
	tests := []struct {
		name   string // The name of the test case.
		status int    // The HTTP status code under test.
		want   string // The expected advice string.
	}{
		{
			name:   "bad request (400)",
			status: http.StatusBadRequest,
			want:   "check the configured model and provider settings",
		},
		{
			name:   "unprocessable entity (422)",
			status: http.StatusUnprocessableEntity,
			want:   "check the configured model and provider settings",
		},
		{
			name:   "unauthorized (401)",
			status: http.StatusUnauthorized,
			want:   "check that the API key is set correctly",
		},
		{
			name:   "payment required (402)",
			status: http.StatusPaymentRequired,
			want:   "insufficient credits; add credits to the provider account",
		},
		{
			name:   "forbidden (403)",
			status: http.StatusForbidden,
			want:   "check the API key permissions for this model",
		},
		{
			name:   "not found (404)",
			status: http.StatusNotFound,
			want:   "check the configured model name and privacy or routing settings",
		},
		{
			name:   "request entity too large (413)",
			status: http.StatusRequestEntityTooLarge,
			want:   "reduce the prompt or request payload size",
		},
		{
			name:   "too many requests (429)",
			status: http.StatusTooManyRequests,
			want:   "rate limited; retry later",
		},
		{
			name:   "request timeout (408)",
			status: http.StatusRequestTimeout,
			want:   "the provider timed out; retry later",
		},
		{
			name:   "gateway timeout (504)",
			status: http.StatusGatewayTimeout,
			want:   "the provider timed out; retry later",
		},
		{
			name:   "edge timeout (524)",
			status: 524,
			want:   "the provider timed out; retry later",
		},
		{
			name:   "internal server error (500)",
			status: http.StatusInternalServerError,
			want:   "the provider is temporarily unavailable; retry later",
		},
		{
			name:   "bad gateway (502)",
			status: http.StatusBadGateway,
			want:   "the provider is temporarily unavailable; retry later",
		},
		{
			name:   "service unavailable (503)",
			status: http.StatusServiceUnavailable,
			want:   "the provider is temporarily unavailable; retry later",
		},
		{
			name:   "provider overloaded (529)",
			status: 529,
			want:   "the provider is temporarily unavailable; retry later",
		},
		{
			name:   "ok (200) returns empty",
			status: http.StatusOK,
			want:   "",
		},
		{
			name:   "unmapped status (418) returns empty",
			status: http.StatusTeapot,
			want:   "",
		},
	}
	// Iterate over test cases.
	for _, tt := range tests {
		// Run each test case as a subtest.
		t.Run(tt.name, func(t *testing.T) {
			// Run subtest in parallel.
			t.Parallel()
			// Retrieve the status advice under test.
			got := tsaiclient.StatusAdvice(tt.status)
			// Fail the test if the returned advice does not match want.
			if got != tt.want {
				t.Fatal(tserr.EqualStr(&tserr.EqualStrArgs{
					Var:    "statusAdvice",
					Actual: got,
					Want:   tt.want,
				}))
			}
		})
	}
}

// TestStatusError tests the fallback HTTP status error construction of statusError.
func TestStatusError(t *testing.T) {
	// Run subtests in parallel.
	t.Parallel()
	// Define test cases.
	tests := []struct {
		name     string // The name of the test case.
		provider string // The provider name under test.
		status   int    // The HTTP status code under test.
		wantSub  string // The substring expected in the returned error message.
	}{
		{
			name:     "status with actionable advice (429)",
			provider: "OpenRouter",
			status:   http.StatusTooManyRequests,
			wantSub:  "HTTP 429: rate limited; retry later",
		},
		{
			name:     "status with actionable advice (502)",
			provider: "OpenRouter",
			status:   http.StatusBadGateway,
			wantSub:  "HTTP 502: the provider is temporarily unavailable; retry later",
		},
		{
			name:     "status without actionable advice falls back to status mismatch (418)",
			provider: "OpenRouter",
			status:   http.StatusTeapot,
			wantSub:  "418",
		},
	}
	// Iterate over test cases.
	for _, tt := range tests {
		// Run each test case as a subtest.
		t.Run(tt.name, func(t *testing.T) {
			// Run subtest in parallel.
			t.Parallel()
			// Construct the status error under test.
			err := tsaiclient.StatusError(tt.provider, tt.status)
			// Fail the test if no error was returned.
			if err == nil {
				t.Fatal(tserr.NilFailed("StatusError"))
			}
			// Fail the test if the error does not include the provider name.
			if !strings.Contains(err.Error(), tt.provider) {
				t.Fatal(tserr.EqualStr(&tserr.EqualStrArgs{
					Var:    "statusError provider",
					Actual: err.Error(),
					Want:   tt.provider,
				}))
			}
			// Fail the test if the error does not include the expected detail substring.
			if !strings.Contains(err.Error(), tt.wantSub) {
				t.Fatal(tserr.EqualStr(&tserr.EqualStrArgs{
					Var:    "statusError detail",
					Actual: err.Error(),
					Want:   tt.wantSub,
				}))
			}
		})
	}
}

// TestValidateHTTPSURLRedaction verifies that validateHTTPSURL returns errors
// that never echo credentials or other secret URL components.
func TestValidateHTTPSURLRedaction(t *testing.T) {
	// Run subtests in parallel.
	t.Parallel()
	// Define the cases covering acceptance and every rejection branch.
	tests := []struct {
		name    string   // The name of the test case.
		rawURL  string   // The raw URL passed to validateHTTPSURL.
		wantErr bool     // Whether validateHTTPSURL must return an error.
		secrets []string // Substrings that must never appear in the returned error text.
	}{
		{
			name:   "accepts a hierarchical HTTPS URL",
			rawURL: "https://api.example.com",
		},
		{
			name:   "accepts a path and an explicit port",
			rawURL: "https://api.example.com:8443/v1/chat",
		},
		{
			name:    "rejects a whitespace-only URL",
			rawURL:  "   ",
			wantErr: true,
		},
		{
			name:    "rejects embedded whitespace without echoing secrets",
			rawURL:  "https://alice-internal:s3cr3t-pw@example.com/pa th?token=tok-998877",
			wantErr: true,
			secrets: []string{"alice-internal", "s3cr3t-pw", "tok-998877"},
		},
		{
			name:    "rejects an unparseable URL without echoing secrets",
			rawURL:  "https://alice-internal:s3cr3t-pw@example.com/%zz?token=tok-998877",
			wantErr: true,
			secrets: []string{"alice-internal", "s3cr3t-pw", "tok-998877"},
		},
		{
			name:    "rejects an opaque URL without echoing the opaque body",
			rawURL:  "https:sk-live-9f8e7d6c",
			wantErr: true,
			secrets: []string{"sk-live-9f8e7d6c"},
		},
		{
			name:    "rejects a non-HTTPS scheme without echoing credentials",
			rawURL:  "http://alice-internal:s3cr3t-pw@example.com/?token=tok-998877",
			wantErr: true,
			secrets: []string{"alice-internal", "s3cr3t-pw", "tok-998877"},
		},
		{
			name:    "rejects user credentials without echoing the username",
			rawURL:  "https://alice-internal:s3cr3t-pw@example.com/",
			wantErr: true,
			secrets: []string{"alice-internal", "s3cr3t-pw"},
		},
		{
			name:    "rejects a missing host",
			rawURL:  "https:/only-a-path",
			wantErr: true,
		},
		{
			name:    "rejects an empty port",
			rawURL:  "https://api.example.com:/",
			wantErr: true,
		},
		{
			name:    "rejects an out-of-range port",
			rawURL:  "https://api.example.com:70000/",
			wantErr: true,
		},
		{
			name:    "rejects a query string without echoing it",
			rawURL:  "https://api.example.com/?token=tok-998877",
			wantErr: true,
			secrets: []string{"tok-998877"},
		},
		{
			name:    "rejects a fragment without echoing it",
			rawURL:  "https://api.example.com/#frag-secret-42",
			wantErr: true,
			secrets: []string{"frag-secret-42"},
		},
	}
	// Iterate over test cases.
	for _, tc := range tests {
		// Run each test case as a subtest.
		t.Run(tc.name, func(t *testing.T) {
			// Run subtest in parallel.
			t.Parallel()
			// Validate the raw URL under test.
			err := tsaiclient.ValidateHTTPSURL("base URL", tc.rawURL)
			// Check whether an expected error was missing.
			if tc.wantErr && err == nil {
				t.Fatal(tserr.NilFailed("ValidateHTTPSURL"))
			}
			// Check whether an unexpected error was returned.
			if !tc.wantErr && err != nil {
				t.Fatal(tserr.NilExpected(&tserr.NilExpectedArgs{Op: "ValidateHTTPSURL", Err: err}))
			}
			// Skip the secret assertions when no error is returned.
			if err == nil {
				// Return to continue with the next case.
				return
			}
			// Assert that no secret substring leaked into the error text.
			for _, secret := range tc.secrets {
				// Fail the test if the error text echoes the secret.
				if strings.Contains(err.Error(), secret) {
					t.Fatal(tserr.EqualStr(&tserr.EqualStrArgs{
						Var:    "redacted error text",
						Actual: err.Error(),
						Want:   "an error text without " + secret,
					}))
				}
			}
		})
	}
}

// TestValidHostname tests the host classification logic of validHostname.
func TestValidHostname(t *testing.T) {
	// Run subtests in parallel.
	t.Parallel()
	// Define test cases.
	tests := []struct {
		name string // The name of the test case.
		host string // The host string under test.
		want bool   // Whether the host must be accepted.
	}{
		{name: "DNS name", host: "api.example.com", want: true},
		{name: "single-label DNS name", host: "localhost", want: true},
		{name: "uppercase DNS name", host: "API.Example.COM", want: true},
		{name: "punycode DNS name", host: "xn--bcher-kva.example", want: true},
		{name: "IPv4 literal", host: "192.0.2.1", want: true},
		{name: "IPv6 literal", host: "2001:db8::1", want: true},
		{name: "zone-scoped IPv6 literal", host: "fe80::1%eth0", want: true},
		{name: "IPv6 literal with an empty zone", host: "fe80::1%", want: false},
		{name: "zone identifier on an IPv4 literal", host: "192.0.2.1%eth0", want: false},
		{name: "zone identifier on a DNS name", host: "my_service.local%eth0", want: false},
		{name: "zone identifier before the address", host: "%eth0::1", want: false},
		{name: "label at the DNS label limit", host: strings.Repeat("a", 63), want: true},
		{
			name: "name at the DNS name limit",
			host: strings.Repeat("a", 63) + "." + strings.Repeat("b", 63) + "." + strings.Repeat("c", 63) + "." + strings.Repeat("d", 61),
			want: true,
		},
		{name: "empty host", host: "", want: false},
		{name: "colon of an unbracketed IPv6 literal", host: ":", want: false},
		{name: "host carrying a port", host: "api.example.com:8443", want: false},
		{name: "empty label", host: "api..example.com", want: false},
		{name: "label with a leading hyphen", host: "-api.example.com", want: false},
		{name: "label with a trailing hyphen", host: "api-.example.com", want: false},
		{name: "embedded whitespace", host: "exa mple.com", want: false},
		{name: "label beyond the DNS label limit", host: strings.Repeat("a", 64), want: false},
		{
			name: "name beyond the DNS name limit",
			host: strings.Repeat("a", 63) + "." + strings.Repeat("b", 63) + "." + strings.Repeat("c", 63) + "." + strings.Repeat("d", 62),
			want: false,
		},
		{name: "internal DNS name with underscores", host: "my_service.local", want: true},
		{name: "fully qualified name with trailing dot", host: "api.example.com.", want: true},
		{name: "double trailing dot", host: "api.example.com..", want: false},
	}
	// Iterate over test cases.
	for _, tt := range tests {
		// Run each test case as a subtest.
		t.Run(tt.name, func(t *testing.T) {
			// Run subtest in parallel.
			t.Parallel()
			// Classify the host under test.
			got := tsaiclient.ValidHostname(tt.host)
			// Fail the test if the classification does not match want.
			if got != tt.want {
				t.Fatal(tserr.EqualStr(&tserr.EqualStrArgs{
					Var:    "validHostname",
					Actual: strconv.FormatBool(got),
					Want:   strconv.FormatBool(tt.want),
				}))
			}
		})
	}
}
