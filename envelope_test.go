// Copyright (c) 2026 thorsphere.
// All Rights Reserved. Use is governed by the Functional Source License v1.1
// (FSL-1.1-ALv2) that can be found in the LICENSE file.

package tsaiclient_test

// Import packages.
import (
	"context"  // Import the context package for request cancellation.
	"io"       // Import the io package for response bodies.
	"net/http" // Import the http package for HTTP responses.
	"slices"   // Import the slices package for finding slice comparison.
	"strings"  // Import the strings package for response bodies.
	"testing"  // Import the testing package for subtests.

	"github.com/thorsphere/tsaiclient" // Import the tsaiclient package for testing.
	"github.com/thorsphere/tserr"      // Import the tserr package for error handling.
)

// TestStripCodeFences tests the code fence stripping helper.
func TestStripCodeFences(t *testing.T) {
	// Run subtests in parallel.
	t.Parallel()
	// Define test cases.
	tests := []struct {
		name  string // The name of the test case.
		input string // The raw model content under test.
		want  string // The expected stripped payload.
	}{
		{
			name:  "plain JSON passes through",
			input: `{"findings":["a"]}`,
			want:  `{"findings":["a"]}`,
		},
		{
			name:  "surrounding whitespace is trimmed",
			input: "  \n{\"findings\":[]} \n",
			want:  `{"findings":[]}`,
		},
		{
			name:  "json fence is stripped",
			input: "```json\n{\"findings\":[]}\n```",
			want:  `{"findings":[]}`,
		},
		{
			name:  "bare fence is stripped",
			input: "```\n{\"findings\":[]}\n```",
			want:  `{"findings":[]}`,
		},
		{
			name:  "leading newline before fence is tolerated",
			input: "\n```json\n{}\n```",
			want:  "{}",
		},
		{
			name:  "embedded backticks inside fenced JSON are preserved",
			input: "```json\n{\"findings\":[\"```go\\nfmt.Println()\\n```\"]}\n```",
			want:  `{"findings":["` + "```go\\nfmt.Println()\\n```" + `"]}`,
		},
		{
			name:  "opening fence without closing fence returns payload",
			input: "```json\n{\"findings\":[]}",
			want:  `{"findings":[]}`,
		},
		{
			name:  "fence line without newline returns empty payload",
			input: "```json",
			want:  "",
		},
		{
			name:  "empty content returns empty payload",
			input: "",
			want:  "",
		},
	}
	// Iterate over test cases.
	for _, tt := range tests {
		// Run each test case as a parallel subtest.
		t.Run(tt.name, func(t *testing.T) {
			// Run subtest in parallel.
			t.Parallel()
			// Verify the stripped payload matches want.
			if got := tsaiclient.StripCodeFences(tt.input); got != tt.want {
				t.Fatal(tserr.EqualStr(&tserr.EqualStrArgs{
					Var:    "StripCodeFences",
					Actual: got,
					Want:   tt.want,
				}))
			}
		})
	}
}

// TestClientCompleteJSON tests the envelope decoding workflow of
// Client.CompleteJSON.
func TestClientCompleteJSON(t *testing.T) {
	// Create a valid configuration for tests.
	validCfg, err := tsaiclient.NewConfig("open/for-5")
	// Fail the test if creating the configuration fails.
	if err != nil {
		t.Fatal(err)
	}
	// Create a valid request for tests.
	validReq, err := tsaiclient.NewRequest("Hello")
	// Fail the test if creating the request fails.
	if err != nil {
		t.Fatal(err)
	}
	// Create an invalid request with an empty prompt.
	invalidReq := &tsaiclient.Request{}
	// Construct a valid HTTP request to return from the stub provider.
	validHTTPReq, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "https://api.example.com/v1/chat", nil)
	// Fail the test if creating the HTTP request fails.
	if err != nil {
		t.Fatal(err)
	}
	// Record the usage reported by the stub provider for assertions.
	wantUsage := tsaiclient.Usage{PromptTokens: 3, CompletionTokens: 5, TotalTokens: 8}
	// Construct a provider that returns validHTTPReq and a parsed Result carrying the response body as content.
	validProvider := &testProvider{
		buildFn: func(_ context.Context, _ *tsaiclient.Config, _ *tsaiclient.Request) (*http.Request, error) {
			return validHTTPReq, nil
		},
		parseFn: func(statusCode int, body []byte) (tsaiclient.Result, error) {
			return tsaiclient.Result{
				Content:   string(body),
				Truncated: statusCode == http.StatusPartialContent,
				Usage:     wantUsage,
			}, nil
		},
	}
	// Create a valid client for the standard test cases.
	validClient, err := tsaiclient.NewClient(validCfg, validProvider)
	// Fail the test if creating the client fails.
	if err != nil {
		t.Fatal(err)
	}
	// Define test cases.
	tests := []struct {
		name        string                // The name of the test case.
		client      *tsaiclient.Client    // The client under test.
		ctx         context.Context       // The context passed to CompleteJSON.
		req         *tsaiclient.Request   // The request passed to CompleteJSON.
		statusCode  int                   // The HTTP status code returned by the stub.
		body        string                // The response body returned by the stub.
		wantJSONRes tsaiclient.JSONResult // The expected result.
		wantErr     bool                  // Whether an error is expected.
	}{
		{
			name:       "valid envelope JSON",
			client:     validClient,
			ctx:        context.Background(),
			req:        validReq,
			statusCode: http.StatusOK,
			body:       `{"findings":["a","b"],"confidence":0.5}`,
			wantJSONRes: tsaiclient.JSONResult{
				Envelope: tsaiclient.Envelope{Findings: []string{"a", "b"}, Confidence: 0.5},
				Usage:    wantUsage,
			},
			wantErr: false,
		},
		{
			name:       "fence-wrapped envelope JSON",
			client:     validClient,
			ctx:        context.Background(),
			req:        validReq,
			statusCode: http.StatusOK,
			body:       "```json\n{\"findings\":[\"a\"],\"confidence\":0.25}\n```",
			wantJSONRes: tsaiclient.JSONResult{
				Envelope: tsaiclient.Envelope{Findings: []string{"a"}, Confidence: 0.25},
				Usage:    wantUsage,
			},
			wantErr: false,
		},
		{
			name:       "bare fence-wrapped envelope JSON with boundary confidence",
			client:     validClient,
			ctx:        context.Background(),
			req:        validReq,
			statusCode: http.StatusOK,
			body:       "```\n{\"findings\":[\"a\"],\"confidence\":1.0}\n```",
			wantJSONRes: tsaiclient.JSONResult{
				Envelope: tsaiclient.Envelope{Findings: []string{"a"}, Confidence: 1.0},
				Usage:    wantUsage,
			},
			wantErr: false,
		},
		{
			name:       "extra envelope fields are ignored",
			client:     validClient,
			ctx:        context.Background(),
			req:        validReq,
			statusCode: http.StatusOK,
			body:       `{"findings":["a"],"confidence":0.5,"note":"extra"}`,
			wantJSONRes: tsaiclient.JSONResult{
				Envelope: tsaiclient.Envelope{Findings: []string{"a"}, Confidence: 0.5},
				Usage:    wantUsage,
			},
			wantErr: false,
		},
		{
			name:       "envelope with missing findings decodes without validation",
			client:     validClient,
			ctx:        context.Background(),
			req:        validReq,
			statusCode: http.StatusOK,
			body:       `{"confidence":0.5}`,
			wantJSONRes: tsaiclient.JSONResult{
				Envelope: tsaiclient.Envelope{Confidence: 0.5},
				Usage:    wantUsage,
			},
			wantErr: false,
		},
		{
			name:       "malformed JSON returns an error",
			client:     validClient,
			ctx:        context.Background(),
			req:        validReq,
			statusCode: http.StatusOK,
			body:       "sorry, I cannot comply",
			wantJSONRes: tsaiclient.JSONResult{
				Usage: wantUsage,
			},
			wantErr: true,
		},
		{
			name:       "truncated envelope JSON returns an error with metadata",
			client:     validClient,
			ctx:        context.Background(),
			req:        validReq,
			statusCode: http.StatusPartialContent,
			body:       `{"findings":["a`,
			wantJSONRes: tsaiclient.JSONResult{
				Truncated: true,
				Usage:     wantUsage,
			},
			wantErr: true,
		},
		{
			name:    "nil client",
			client:  nil,
			ctx:     context.Background(),
			req:     validReq,
			wantErr: true,
		},
		{
			name:    "nil request",
			client:  validClient,
			ctx:     context.Background(),
			req:     nil,
			wantErr: true,
		},
		{
			name:    "invalid request",
			client:  validClient,
			ctx:     context.Background(),
			req:     invalidReq,
			wantErr: true,
		},
	}
	// Iterate over test cases.
	for _, tt := range tests {
		// Run each test case as a serial subtest because httpDo is a package-level seam.
		t.Run(tt.name, func(t *testing.T) {
			// Install the test HTTP seam returning the case response.
			restore := tsaiclient.SetHTTPDo(func(_ *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: tt.statusCode,
					Body:       io.NopCloser(strings.NewReader(tt.body)),
				}, nil
			})
			// Restore the previous HTTP execution function when the subtest ends.
			defer restore()
			// Execute CompleteJSON under test.
			got, err := tt.client.CompleteJSON(tt.ctx, tt.req)
			// Check whether an expected error was missing.
			if tt.wantErr && err == nil {
				t.Fatal(tserr.NilFailed("CompleteJSON"))
			}
			// Check whether an unexpected error was returned.
			if !tt.wantErr && err != nil {
				t.Fatal(tserr.NilExpected(&tserr.NilExpectedArgs{Op: "CompleteJSON", Err: err}))
			}
			// Verify the decoded findings match wantJSONRes.
			if !slices.Equal(got.Envelope.Findings, tt.wantJSONRes.Envelope.Findings) {
				t.Errorf("CompleteJSON findings = %v, want %v", got.Envelope.Findings, tt.wantJSONRes.Envelope.Findings)
			}
			// Verify the decoded confidence matches wantJSONRes.
			if got.Envelope.Confidence != tt.wantJSONRes.Envelope.Confidence {
				t.Errorf("CompleteJSON confidence = %v, want %v", got.Envelope.Confidence, tt.wantJSONRes.Envelope.Confidence)
			}
			// Verify the truncation status matches wantJSONRes.
			if got.Truncated != tt.wantJSONRes.Truncated {
				t.Errorf("CompleteJSON truncated = %v, want %v", got.Truncated, tt.wantJSONRes.Truncated)
			}
			// Verify the usage accounting matches wantJSONRes.
			if got.Usage != tt.wantJSONRes.Usage {
				t.Errorf("CompleteJSON usage = %+v, want %+v", got.Usage, tt.wantJSONRes.Usage)
			}
		})
	}
}

// TestCompleteJSONModel tests that CompleteJSON reports the model returned by
// the provider on the JSONResult, and reports an empty model when the provider
// does not report one.
func TestCompleteJSONModel(t *testing.T) {
	// Create a valid configuration.
	validCfg, err := tsaiclient.NewConfig("open/for-5")
	// Fail the test if creating the configuration fails.
	if err != nil {
		t.Fatal(err)
	}
	// Create a valid request for tests.
	validReq, err := tsaiclient.NewRequest("Hello")
	// Fail the test if creating the request fails.
	if err != nil {
		t.Fatal(err)
	}
	// Construct a valid HTTP request to return from the stub provider.
	validHTTPReq, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "https://api.example.com/v1/chat", nil)
	// Fail the test if creating the HTTP request fails.
	if err != nil {
		t.Fatal(err)
	}
	// Define test cases.
	tests := []struct {
		name      string // The name of the test case.
		model     string // The model reported by the stub provider.
		wantModel string // The model expected on the returned JSONResult.
	}{
		{
			name:      "provider reports the used model",
			model:     "closedai/ghj-5",
			wantModel: "closedai/ghj-5",
		},
		{
			name:      "provider reports no model",
			model:     "",
			wantModel: "",
		},
	}
	// Iterate over test cases.
	for _, tt := range tests {
		// Run each test case as a serial subtest because httpDo is a package-level seam.
		t.Run(tt.name, func(t *testing.T) {
			// Construct a provider that reports the case model.
			provider := &testProvider{
				buildFn: func(_ context.Context, _ *tsaiclient.Config, _ *tsaiclient.Request) (*http.Request, error) {
					return validHTTPReq, nil
				},
				parseFn: func(_ int, body []byte) (tsaiclient.Result, error) {
					return tsaiclient.Result{Content: string(body), Model: tt.model}, nil
				},
			}
			// Create a client with the stub provider.
			client, err := tsaiclient.NewClient(validCfg, provider)
			// Fail the test if creating the client fails.
			if err != nil {
				t.Fatal(err)
			}
			// Install the test HTTP seam returning a valid envelope body.
			restore := tsaiclient.SetHTTPDo(func(_ *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"findings":["a"],"confidence":0.5}`)),
				}, nil
			})
			// Restore the previous HTTP execution function when the subtest ends.
			defer restore()
			// Execute CompleteJSON under test.
			got, err := client.CompleteJSON(context.Background(), validReq)
			// Fail the test if an unexpected error was returned.
			if err != nil {
				t.Fatal(tserr.NilExpected(&tserr.NilExpectedArgs{Op: "CompleteJSON", Err: err}))
			}
			// Verify the reported model matches the expectation.
			if got.Model != tt.wantModel {
				t.Errorf("CompleteJSON model = %q, want %q", got.Model, tt.wantModel)
			}
		})
	}
}
