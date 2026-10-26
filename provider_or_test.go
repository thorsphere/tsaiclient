// Copyright (c) 2026 thorsphere.
// All Rights Reserved. Use is governed by the Functional Source License v1.1
// (FSL-1.1-ALv2) that can be found in the LICENSE file.

package tsaiclient_test

// Import packages.
import (
	"net/http"
	"strings"
	"testing" // Import the testing package for subtests.

	"github.com/thorsphere/tsaiclient" // Import the tsaiclient package for testing.
	"github.com/thorsphere/tserr"
)

// TestIsValidCategory tests the category validation logic of isValidCategory.
func TestIsValidCategory(t *testing.T) {
	// Run subtests in parallel.
	t.Parallel()
	// Define test cases.
	tests := []struct {
		name     string // The name of the test case.
		category string // The category string under test.
		want     bool   // Whether the category is expected to be valid.
	}{
		{name: "single lowercase letter", category: "a", want: true},
		{name: "single digit", category: "7", want: true},
		{name: "letters digits and interior hyphens", category: "open-router-2", want: true},
		{name: "empty category", category: "", want: false},
		{name: "hyphen only", category: "-", want: false},
		{name: "leading hyphen", category: "-open-router", want: false},
		{name: "trailing hyphen", category: "open-router-", want: false},
		{name: "uppercase ASCII", category: "Open-Router", want: false},
		{name: "underscore", category: "open_router", want: false},
		{name: "space", category: "open router", want: false},
		{name: "comma delimiter", category: "open,router", want: false},
		{name: "newline", category: "open\nrouter", want: false},
		{name: "NUL byte", category: "open\x00router", want: false},
		{name: "non-ASCII lowercase letter", category: "café", want: false},
		{name: "invalid UTF-8", category: "open\xffrouter", want: false},
		{name: "consecutive interior hyphens", category: "open--router", want: false},
	}
	// Iterate over test cases.
	for _, tt := range tests {
		// Run each test case as a subtest.
		t.Run(tt.name, func(t *testing.T) {
			// Run the subtest in parallel.
			t.Parallel()
			// Validate the category under test.
			got := tsaiclient.IsValidCategory(tt.category)
			// Report a mismatch between actual and expected validity.
			if got != tt.want {
				t.Errorf("IsValidCategory(%q) = %t, want %t", tt.category, got, tt.want)
			}
		})
	}
}

// TestOpenRouterValidateCategories tests the category collection constraints of OpenRouter.Validate.
func TestOpenRouterValidateCategories(t *testing.T) {
	// Set the API key required before category validation.
	t.Setenv("OPENROUTER_API_KEY", "test-key")
	// Define test cases.
	tests := []struct {
		name       string   // The name of the test case.
		categories []string // The categories under test.
		wantErr    bool     // Whether validation is expected to fail.
	}{
		{name: "no categories", categories: nil},
		{name: "one category", categories: []string{"open-router"}},
		{name: "two categories", categories: []string{"open-router", "developer-tools"}},
		{name: "three categories", categories: []string{"one", "two", "three"}, wantErr: true},
		{name: "category at 30-character limit", categories: []string{strings.Repeat("a", 30)}},
		{name: "category over 30-character limit", categories: []string{strings.Repeat("a", 31)}, wantErr: true},
		{name: "second category over limit", categories: []string{"valid", strings.Repeat("b", 31)}, wantErr: true},
		{name: "repeated hyphen", categories: []string{"open--router"}, wantErr: true},
		{name: "empty category entry", categories: []string{""}, wantErr: true},
	}
	// Iterate over test cases.
	for _, tt := range tests {
		// Run each test case as a subtest.
		t.Run(tt.name, func(t *testing.T) {
			// Construct a provider with otherwise-valid attribution settings.
			provider := tsaiclient.NewOpenRouter().
				WithReferer("https://example.com").
				WithCategories(tt.categories)
			// Validate the provider configuration.
			err := provider.Validate()
			// Check whether an expected error was missing.
			if tt.wantErr && err == nil {
				t.Fatal(tserr.NilFailed("OpenRouter.Validate"))
			}
			// Check whether an unexpected error was returned.
			if !tt.wantErr && err != nil {
				t.Fatal(tserr.NilExpected(&tserr.NilExpectedArgs{
					Op:  "OpenRouter.Validate",
					Err: err,
				}))
			}
		})
	}
}

// TestOpenRouterValidateWithoutAttribution tests that ordinary requests do not require a referer.
func TestOpenRouterValidateWithoutAttribution(t *testing.T) {
	// Set the API key required for provider validation.
	t.Setenv("OPENROUTER_API_KEY", "test-key")
	// Construct a provider without app attribution.
	provider := tsaiclient.NewOpenRouter()
	// Validate the unattributed configuration.
	err := provider.Validate()
	// Report an unexpected validation error.
	if err != nil {
		t.Fatal(tserr.NilExpected(&tserr.NilExpectedArgs{
			Op:  "OpenRouter.Validate",
			Err: err,
		}))
	}
}

// TestOpenRouterValidateAttribution tests the OpenRouter attribution contract.
func TestOpenRouterValidateAttribution(t *testing.T) {
	// Set the API key required before attribution validation.
	t.Setenv("OPENROUTER_API_KEY", "test-key")
	// Define test cases.
	tests := []struct {
		name    string                        // The name of the test case.
		setup   func() *tsaiclient.OpenRouter // The provider configuration under test.
		wantErr bool                          // Whether validation is expected to fail.
	}{
		{
			name: "nil receiver",
			setup: func() *tsaiclient.OpenRouter {
				return nil
			},
			wantErr: true,
		},
		{
			name: "zero value without attribution",
			setup: func() *tsaiclient.OpenRouter {
				return new(tsaiclient.OpenRouter)
			},
		},
		{
			name: "constructor without attribution",
			setup: func() *tsaiclient.OpenRouter {
				return tsaiclient.NewOpenRouter()
			},
		},
		{
			name: "referer only",
			setup: func() *tsaiclient.OpenRouter {
				return tsaiclient.NewOpenRouter().WithReferer("https://example.com")
			},
		},
		{
			name: "referer and title",
			setup: func() *tsaiclient.OpenRouter {
				return tsaiclient.NewOpenRouter().
					WithReferer("https://example.com").
					WithTitle("Example")
			},
		},
		{
			name: "referer and categories",
			setup: func() *tsaiclient.OpenRouter {
				return tsaiclient.NewOpenRouter().
					WithReferer("https://example.com").
					WithCategories([]string{"developer-tools"})
			},
		},
		{
			name: "referer and hidden",
			setup: func() *tsaiclient.OpenRouter {
				return tsaiclient.NewOpenRouter().
					WithReferer("https://example.com").
					WithHidden(true)
			},
		},
		{
			name: "all attribution fields",
			setup: func() *tsaiclient.OpenRouter {
				return tsaiclient.NewOpenRouter().
					WithReferer("https://example.com").
					WithTitle("Example").
					WithCategories([]string{"developer-tools", "productivity"}).
					WithHidden(true)
			},
		},
		{
			name: "missing referer with title",
			setup: func() *tsaiclient.OpenRouter {
				return tsaiclient.NewOpenRouter().WithTitle("Example")
			},
			wantErr: true,
		},
		{
			name: "blank referer with title",
			setup: func() *tsaiclient.OpenRouter {
				return tsaiclient.NewOpenRouter().
					WithReferer(" \t\n").
					WithTitle("Example")
			},
			wantErr: true,
		},
		{
			name: "blank referer with categories",
			setup: func() *tsaiclient.OpenRouter {
				return tsaiclient.NewOpenRouter().
					WithReferer(" \t\n").
					WithCategories([]string{"developer-tools"})
			},
			wantErr: true,
		},
		{
			name: "blank referer with hidden",
			setup: func() *tsaiclient.OpenRouter {
				return tsaiclient.NewOpenRouter().
					WithReferer(" \t\n").
					WithHidden(true)
			},
			wantErr: true,
		},
		{
			name: "blank referer without secondary attribution",
			setup: func() *tsaiclient.OpenRouter {
				return tsaiclient.NewOpenRouter().WithReferer(" \t\n")
			},
		},
		{
			name: "malformed referer",
			setup: func() *tsaiclient.OpenRouter {
				return tsaiclient.NewOpenRouter().WithReferer("://bad-url")
			},
			wantErr: true,
		},
		{
			name: "non-HTTPS referer",
			setup: func() *tsaiclient.OpenRouter {
				return tsaiclient.NewOpenRouter().WithReferer("http://example.com")
			},
			wantErr: true,
		},
	}
	// Iterate over test cases.
	for _, tt := range tests {
		// Run each test case as a subtest.
		t.Run(tt.name, func(t *testing.T) {
			// Construct the provider under test.
			provider := tt.setup()
			// Validate its attribution configuration.
			err := provider.Validate()
			// Check whether an expected error was missing.
			if tt.wantErr && err == nil {
				t.Fatal(tserr.NilFailed("OpenRouter.Validate"))
			}
			// Check whether an unexpected error was returned.
			if !tt.wantErr && err != nil {
				t.Fatal(tserr.NilExpected(&tserr.NilExpectedArgs{
					Op:  "OpenRouter.Validate",
					Err: err,
				}))
			}
		})
	}
}

// TestOpenRouterValidateAPIKey tests rejection of empty and whitespace-only API keys.
func TestOpenRouterValidateAPIKey(t *testing.T) {
	// Define test cases.
	tests := []struct {
		name   string // The name of the test case.
		apiKey string // The API key value under test.
	}{
		{name: "empty API key", apiKey: ""},
		{name: "whitespace-only API key", apiKey: " \t\n"},
	}
	// Iterate over test cases.
	for _, tt := range tests {
		// Run each test case as a subtest.
		t.Run(tt.name, func(t *testing.T) {
			// Install the API key value under test.
			t.Setenv("OPENROUTER_API_KEY", tt.apiKey)
			// Validate an otherwise valid unattributed provider.
			err := tsaiclient.NewOpenRouter().Validate()
			// Fail if validation unexpectedly accepts the missing API key.
			if err == nil {
				t.Fatal(tserr.NilFailed("OpenRouter.Validate"))
			}
		})
	}
}

// TestOpenRouterProviderError tests provider error formatting and status advice selection.
func TestOpenRouterProviderError(t *testing.T) {
	// Run subtests in parallel.
	t.Parallel()
	// Define test cases.
	tests := []struct {
		name       string                     // The name of the test case.
		status     int                        // The HTTP status accompanying the provider error.
		apiErr     tsaiclient.OpenRouterError // The provider error under test.
		wantSub    []string                   // Substrings required in the returned error.
		notWantSub []string                   // Substrings forbidden in the returned error.
	}{
		{
			name:   "message and provider name",
			status: http.StatusTeapot,
			apiErr: tsaiclient.OpenRouterError{Message: "completion failed"},
			wantSub: []string{
				"OpenRouter",
				"completion failed",
			},
			notWantSub: []string{"418"},
		},
		{
			name:   "numeric error code and embedded advice",
			status: http.StatusBadGateway,
			apiErr: tsaiclient.OpenRouterError{
				Code:    http.StatusTooManyRequests,
				Message: "slow down",
			},
			wantSub: []string{
				"429: slow down",
				"rate limited; retry later",
			},
			notWantSub: []string{"502", "temporarily unavailable"},
		},
		{
			name:   "HTTP status advice fallback",
			status: http.StatusBadGateway,
			apiErr: tsaiclient.OpenRouterError{
				Code:    499,
				Message: "upstream failed",
			},
			wantSub: []string{
				"499: upstream failed",
				"the provider is temporarily unavailable; retry later",
			},
		},
		{
			name:   "all structured metadata",
			status: http.StatusBadGateway,
			apiErr: tsaiclient.OpenRouterError{
				Code:    http.StatusForbidden,
				Message: "input rejected",
				Metadata: &tsaiclient.OpenRouterErrorMetadata{
					ErrorType:    "content_policy_violation",
					ProviderName: "upstream-a",
					ProviderCode: "moderation-17",
					Reasons:      []string{"violence", "self-harm"},
				},
			},
			wantSub: []string{
				"403: input rejected",
				"(type: content_policy_violation)",
				"(provider: upstream-a)",
				"(provider code: moderation-17)",
				"[reasons: violence, self-harm]",
				"check the API key permissions for this model",
			},
		},
		{
			name:   "empty metadata still receives advice",
			status: http.StatusTooManyRequests,
			apiErr: tsaiclient.OpenRouterError{
				Message:  "slow down",
				Metadata: &tsaiclient.OpenRouterErrorMetadata{},
			},
			wantSub: []string{"rate limited; retry later"},
		},
		{
			name:   "remedy hint suppresses generic advice",
			status: http.StatusBadGateway,
			apiErr: tsaiclient.OpenRouterError{
				Code:    http.StatusPaymentRequired,
				Message: "credit limit reached",
				Metadata: &tsaiclient.OpenRouterErrorMetadata{
					RemedyHint: "raise the account credit limit",
				},
			},
			wantSub: []string{
				"402: credit limit reached",
				"; raise the account credit limit",
			},
			notWantSub: []string{
				"insufficient credits",
				"temporarily unavailable",
			},
		},
		{
			name:   "unmapped codes add no advice",
			status: http.StatusTeapot,
			apiErr: tsaiclient.OpenRouterError{
				Code:    499,
				Message: "client closed request",
			},
			wantSub: []string{"499: client closed request"},
		},
	}
	// Iterate over test cases.
	for _, tt := range tests {
		// Run each test case as a subtest.
		t.Run(tt.name, func(t *testing.T) {
			// Run the subtest in parallel.
			t.Parallel()
			// Construct the provider error under test.
			err := tsaiclient.OpenRouterProviderError("OpenRouter", tt.status, tt.apiErr)
			// Fail if no error was returned.
			if err == nil {
				t.Fatal(tserr.NilFailed("OpenRouterProviderError"))
			}
			// Check for every required error substring.
			for _, want := range tt.wantSub {
				// Fail if a required substring is missing.
				if !strings.Contains(err.Error(), want) {
					t.Errorf("OpenRouterProviderError() error = %q, want substring %q", err, want)
				}
			}
			// Check for every forbidden error substring.
			for _, notWant := range tt.notWantSub {
				// Fail if a forbidden substring is present.
				if strings.Contains(err.Error(), notWant) {
					t.Errorf("OpenRouterProviderError() error = %q, must not contain %q", err, notWant)
				}
			}
		})
	}
}

// TestOpenRouterParseHTTPResponseErrorFallback tests structured and blank provider error handling.
func TestOpenRouterParseHTTPResponseErrorFallback(t *testing.T) {
	// Run subtests in parallel.
	t.Parallel()
	// Define test cases.
	tests := []struct {
		name    string // The name of the test case.
		status  int    // The HTTP response status.
		body    string // The response body under test.
		wantSub string // The expected error substring.
	}{
		{
			name:    "non-2xx blank message uses embedded code",
			status:  http.StatusInternalServerError,
			body:    `{"error":{"code":503,"message":" \t\n"}}`,
			wantSub: "HTTP 503: the provider is temporarily unavailable; retry later",
		},
		{
			name:    "non-2xx blank message falls back to HTTP status",
			status:  http.StatusTooManyRequests,
			body:    `{"error":{"message":""}}`,
			wantSub: "HTTP 429: rate limited; retry later",
		},
		{
			name:    "non-2xx nonblank message remains structured",
			status:  http.StatusInternalServerError,
			body:    `{"error":{"code":503,"message":"upstream unavailable"}}`,
			wantSub: "503: upstream unavailable",
		},
		{
			name:    "top-level blank message uses embedded code",
			status:  http.StatusOK,
			body:    `{"error":{"code":503,"message":"   "}}`,
			wantSub: "HTTP 503: the provider is temporarily unavailable; retry later",
		},
		{
			name:    "top-level blank message without code uses bad gateway",
			status:  http.StatusOK,
			body:    `{"error":{"message":""}}`,
			wantSub: "HTTP 502: the provider is temporarily unavailable; retry later",
		},
		{
			name:   "choice blank message cannot be hidden by content",
			status: http.StatusOK,
			body: `{
                "choices":[{
                    "message":{"role":"assistant","content":"must not be returned"},
                    "finish_reason":"stop",
                    "error":{"code":503,"message":" "}
                }]
            }`,
			wantSub: "HTTP 503: the provider is temporarily unavailable; retry later",
		},
		{
			name:   "choice blank message without code uses bad gateway",
			status: http.StatusOK,
			body: `{
                "choices":[{
                    "message":{"role":"assistant","content":"must not be returned"},
                    "finish_reason":"stop",
                    "error":{"message":""}
                }]
            }`,
			wantSub: "HTTP 502: the provider is temporarily unavailable; retry later",
		},
		{
			name:   "choice nonblank message remains structured",
			status: http.StatusOK,
			body: `{
                "choices":[{
                    "message":{"role":"assistant","content":""},
                    "finish_reason":"error",
                    "error":{"code":503,"message":"generation interrupted"}
                }]
            }`,
			wantSub: "503: generation interrupted",
		},
	}
	// Iterate over test cases.
	for _, tt := range tests {
		// Run each test case as a subtest.
		t.Run(tt.name, func(t *testing.T) {
			// Run the subtest in parallel.
			t.Parallel()
			// Parse the response under test.
			result, err := tsaiclient.NewOpenRouter().ParseHTTPResponse(tt.status, []byte(tt.body))
			// Fail if no error was returned.
			if err == nil {
				t.Fatal(tserr.NilFailed("OpenRouter.ParseHTTPResponse"))
			}
			// Fail if the expected error detail is missing.
			if !strings.Contains(err.Error(), tt.wantSub) {
				t.Errorf("ParseHTTPResponse() error = %q, want substring %q", err, tt.wantSub)
			}
			// Fail if an error path returned completion data.
			if result != (tsaiclient.Result{}) {
				t.Errorf("ParseHTTPResponse() result = %+v, want zero result", result)
			}
		})
	}
}

// TestOpenRouterParseHTTPResponseFirstChoicePolicy tests that only the first completion choice determines the result.
func TestOpenRouterParseHTTPResponseFirstChoicePolicy(t *testing.T) {
	// Run subtests in parallel.
	t.Parallel()
	// Define test cases.
	tests := []struct {
		name        string // The name of the test case.
		body        string // The response body under test.
		wantContent string // The expected content for a successful response.
		wantErrSub  string // The expected error substring for a failed response.
	}{
		{
			name: "valid first choice ignores failed alternative",
			body: `{
                "choices": [
                    {
                        "message": {"role": "assistant", "content": "first"},
                        "finish_reason": "stop"
                    },
                    {
                        "message": {"role": "assistant", "content": ""},
                        "finish_reason": "error",
                        "error": {"code": 500, "message": "second failed"}
                    }
                ]
            }`,
			wantContent: "first",
		},
		{
			name: "failed first choice is not bypassed",
			body: `{
                "choices": [
                    {
                        "message": {"role": "assistant", "content": ""},
                        "finish_reason": "error"
                    },
                    {
                        "message": {"role": "assistant", "content": "second"},
                        "finish_reason": "stop"
                    }
                ]
            }`,
			wantErrSub: "provider reported a generation error",
		},
		{
			name: "reasoning exhaustion in first choice is not bypassed",
			body: `{
                "choices": [
                    {
                        "message": {"role": "assistant", "content": ""},
                        "finish_reason": "length"
                    },
                    {
                        "message": {"role": "assistant", "content": "second"},
                        "finish_reason": "stop"
                    }
                ],
                "usage": {
                    "completion_tokens_details": {"reasoning_tokens": 12}
                }
            }`,
			wantErrSub: "reasoning tokens exhausted max_completion_tokens before content was generated",
		},
	}
	// Iterate over test cases.
	for _, tt := range tests {
		// Run each test case as a subtest.
		t.Run(tt.name, func(t *testing.T) {
			// Run the subtest in parallel.
			t.Parallel()
			// Parse the response under test.
			result, err := tsaiclient.NewOpenRouter().ParseHTTPResponse(http.StatusOK, []byte(tt.body))
			// Check the expected-error path.
			if tt.wantErrSub != "" {
				// Fail if no error was returned.
				if err == nil {
					t.Fatal(tserr.NilFailed("OpenRouter.ParseHTTPResponse"))
				}
				// Fail if the expected error detail is missing.
				if !strings.Contains(err.Error(), tt.wantErrSub) {
					t.Errorf("ParseHTTPResponse() error = %q, want substring %q", err, tt.wantErrSub)
				}
				// Stop after checking the expected error.
				return
			}
			// Fail if an unexpected error was returned.
			if err != nil {
				t.Fatal(tserr.NilExpected(&tserr.NilExpectedArgs{
					Op:  "OpenRouter.ParseHTTPResponse",
					Err: err,
				}))
			}
			// Fail if the first choice was not selected.
			if result.Content != tt.wantContent {
				t.Errorf("ParseHTTPResponse() content = %q, want %q", result.Content, tt.wantContent)
			}
		})
	}
}

// TestOpenRouterParseHTTPResponseNilReceiver tests rejection of a nil provider receiver.
func TestOpenRouterParseHTTPResponseNilReceiver(t *testing.T) {
	// Run the test in parallel.
	t.Parallel()
	// Declare a typed-nil provider receiver.
	var provider *tsaiclient.OpenRouter
	// Attempt to parse a response with the nil receiver.
	result, err := provider.ParseHTTPResponse(http.StatusOK, []byte(`{}`))
	// Fail if no error was returned.
	if err == nil {
		t.Fatal(tserr.NilFailed("OpenRouter.ParseHTTPResponse"))
	}
	// Fail if a nil-receiver error returned completion data.
	if result != (tsaiclient.Result{}) {
		t.Errorf("ParseHTTPResponse() result = %+v, want zero result", result)
	}
}

// TestOpenRouterParseHTTPResponseSuccess tests successful completion normalization.
func TestOpenRouterParseHTTPResponseSuccess(t *testing.T) {
	// Run subtests in parallel.
	t.Parallel()
	// Define test cases.
	tests := []struct {
		name string            // The name of the test case.
		body string            // The successful response body under test.
		want tsaiclient.Result // The expected normalized result.
	}{
		{
			name: "completed response with usage",
			body: `{
                "choices":[{
                    "message":{"role":"assistant","content":"answer"},
                    "finish_reason":"stop"
                }],
                "usage":{
                    "prompt_tokens":10,
                    "completion_tokens":8,
                    "total_tokens":18,
                    "cost":0.00125,
                    "completion_tokens_details":{"reasoning_tokens":3}
                }
            }`,
			want: tsaiclient.Result{
				Content: "answer",
				Usage: tsaiclient.Usage{
					PromptTokens:     10,
					CompletionTokens: 8,
					ReasoningTokens:  3,
					TotalTokens:      18,
					Cost:             0.00125,
				},
			},
		},
		{
			name: "truncated response preserves partial content",
			body: `{
                "choices":[{
                    "message":{"role":"assistant","content":"partial"},
                    "finish_reason":"length"
                }],
                "usage":{
                    "prompt_tokens":4,
                    "completion_tokens":5,
                    "total_tokens":9,
                    "completion_tokens_details":{"reasoning_tokens":2}
                }
            }`,
			want: tsaiclient.Result{
				Content:   "partial",
				Truncated: true,
				Usage: tsaiclient.Usage{
					PromptTokens:     4,
					CompletionTokens: 5,
					ReasoningTokens:  2,
					TotalTokens:      9,
				},
			},
		},
	}
	// Iterate over test cases.
	for _, tt := range tests {
		// Run each test case as a subtest.
		t.Run(tt.name, func(t *testing.T) {
			// Run the subtest in parallel.
			t.Parallel()
			// Parse the successful response.
			got, err := tsaiclient.NewOpenRouter().ParseHTTPResponse(http.StatusOK, []byte(tt.body))
			// Fail if an unexpected error was returned.
			if err != nil {
				t.Fatal(tserr.NilExpected(&tserr.NilExpectedArgs{
					Op:  "OpenRouter.ParseHTTPResponse",
					Err: err,
				}))
			}
			// Compare the complete normalized result.
			if got != tt.want {
				t.Errorf("ParseHTTPResponse() result = %+v, want %+v", got, tt.want)
			}
		})
	}
}

// TestOpenRouterParseHTTPResponseErrors tests the remaining response error branches.
func TestOpenRouterParseHTTPResponseErrors(t *testing.T) {
	// Run subtests in parallel.
	t.Parallel()
	// Define test cases.
	tests := []struct {
		name    string // The name of the test case.
		status  int    // The HTTP response status.
		body    string // The response body under test.
		wantSub string // The expected error substring.
	}{
		{
			name:    "status below 200",
			status:  199,
			body:    `not-json`,
			wantSub: "HTTP 199: unexpected response from the provider",
		},
		{
			name:    "redirect status",
			status:  http.StatusMultipleChoices,
			body:    `not-json`,
			wantSub: "HTTP 300: unexpected response from the provider",
		},
		{
			name:    "non-2xx malformed body",
			status:  http.StatusBadGateway,
			body:    `not-json`,
			wantSub: "HTTP 502: the provider is temporarily unavailable; retry later",
		},
		{
			name:    "2xx malformed body",
			status:  http.StatusOK,
			body:    `{"choices":`,
			wantSub: "unmarshal response",
		},
		{
			name:    "missing choices",
			status:  http.StatusOK,
			body:    `{}`,
			wantSub: "response choices",
		},
		{
			name:    "explicit null choices",
			status:  http.StatusOK,
			body:    `{"choices":null}`,
			wantSub: "response choices",
		},
		{
			name:    "empty choices",
			status:  http.StatusOK,
			body:    `{"choices":[]}`,
			wantSub: "response choices",
		},
		{
			name:   "content filter",
			status: http.StatusOK,
			body: `{
                "choices":[{
                    "message":{"role":"assistant","content":"blocked content"},
                    "finish_reason":"content_filter"
                }]
            }`,
			wantSub: "response blocked by content filter",
		},
		{
			name:   "provider generation failure",
			status: http.StatusOK,
			body: `{
                "choices":[{
                    "message":{"role":"assistant","content":""},
                    "finish_reason":"error"
                }]
            }`,
			wantSub: "provider reported a generation error",
		},
		{
			name:   "reasoning tokens exhausted",
			status: http.StatusOK,
			body: `{
                "choices":[{
                    "message":{"role":"assistant","content":""},
                    "finish_reason":"length"
                }],
                "usage":{"completion_tokens_details":{"reasoning_tokens":12}}
            }`,
			wantSub: "reasoning tokens exhausted max_completion_tokens before content was generated",
		},
		{
			name:   "empty stopped completion",
			status: http.StatusOK,
			body: `{
                "choices":[{
                    "message":{"role":"assistant","content":""},
                    "finish_reason":"stop"
                }]
            }`,
			wantSub: "completion content",
		},
		{
			name:   "empty truncated completion without reasoning tokens",
			status: http.StatusOK,
			body: `{
                "choices":[{
                    "message":{"role":"assistant","content":""},
                    "finish_reason":"length"
                }]
            }`,
			wantSub: "completion content",
		},
	}
	// Iterate over test cases.
	for _, tt := range tests {
		// Run each test case as a subtest.
		t.Run(tt.name, func(t *testing.T) {
			// Run the subtest in parallel.
			t.Parallel()
			// Parse the response under test.
			result, err := tsaiclient.NewOpenRouter().ParseHTTPResponse(tt.status, []byte(tt.body))
			// Fail if no error was returned.
			if err == nil {
				t.Fatal(tserr.NilFailed("OpenRouter.ParseHTTPResponse"))
			}
			// Fail if the expected error detail is missing.
			if !strings.Contains(err.Error(), tt.wantSub) {
				t.Errorf("ParseHTTPResponse() error = %q, want substring %q", err, tt.wantSub)
			}
			// Fail if an error path returned completion data.
			if result != (tsaiclient.Result{}) {
				t.Errorf("ParseHTTPResponse() result = %+v, want zero result", result)
			}
		})
	}
}
