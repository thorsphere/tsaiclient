// Copyright (c) 2026 thorsphere.
// All Rights Reserved. Use is governed by the Functional Source License v1.1
// (FSL-1.1-ALv2) that can be found in the LICENSE file.

package tsaiclient_test

// Import packages.
import (
	"testing" // Import the testing package for testing.

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
