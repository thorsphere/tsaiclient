// Copyright (c) 2026 thorsphere.
// All Rights Reserved. Use is governed by the Functional Source License v1.1
// (FSL-1.1-ALv2) that can be found in the LICENSE file.

package tsaiclient_test

// Import packages.
import (
	"context"       // Import the context package for request contexts.
	"encoding/json" // Import the encoding/json package for JSON handling.
	"fmt"           // Import the fmt package for formatted I/O.
	"io"            // Import the io package for I/O operations.
	"net/http"      // Import the net/http package for HTTP requests and responses.
	"strings"       // Import the strings package for string manipulation.
	"testing"       // Import the testing package for subtests.

	"github.com/thorsphere/tsaiclient" // Import the tsaiclient package for testing.
	"github.com/thorsphere/tserr"      // Import the tserr package for error handling.
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
				t.Fatal(tserr.EqualStr(&tserr.EqualStrArgs{
					Var:    fmt.Sprintf("IsValidCategory(%q)", tt.category),
					Actual: fmt.Sprintf("%t", got),
					Want:   fmt.Sprintf("%t", tt.want),
				}))
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
		wantErrSub string   // Substring expected in the error; empty means no error.
	}{
		{name: "no categories", categories: nil},
		{name: "one category", categories: []string{"open-router"}},
		{name: "two categories", categories: []string{"open-router", "developer-tools"}},
		{name: "three categories", categories: []string{"one", "two", "three"}, wantErrSub: "category count"},
		{name: "category at 30-character limit", categories: []string{strings.Repeat("a", 30)}},
		{name: "category over 30-character limit", categories: []string{strings.Repeat("a", 31)}, wantErrSub: "category length"},
		{name: "second category over limit", categories: []string{"valid", strings.Repeat("b", 31)}, wantErrSub: "category length"},
		{name: "repeated hyphen", categories: []string{"open--router"}, wantErrSub: "expected lowercase letters, digits, and hyphens"},
		{name: "empty category entry", categories: []string{""}, wantErrSub: "category"},
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
			if tt.wantErrSub != "" && err == nil {
				t.Fatal(tserr.NilFailed("OpenRouter.Validate"))
			}
			// Check whether an unexpected error was returned.
			if tt.wantErrSub == "" && err != nil {
				t.Fatal(tserr.NilExpected(&tserr.NilExpectedArgs{
					Op:  "OpenRouter.Validate",
					Err: err,
				}))
			}
			// Check whether the error carries the expected detail.
			if tt.wantErrSub != "" && !strings.Contains(err.Error(), tt.wantErrSub) {
				t.Fatal(tserr.EqualStr(&tserr.EqualStrArgs{
					Var:    "OpenRouter.Validate error",
					Actual: err.Error(),
					Want:   tt.wantErrSub,
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
		{
			name: "title with embedded newline",
			setup: func() *tsaiclient.OpenRouter {
				return tsaiclient.NewOpenRouter().
					WithReferer("https://example.com").
					WithTitle("My\nApp")
			},
			wantErr: true,
		},
		{
			name: "title with non-ASCII characters",
			setup: func() *tsaiclient.OpenRouter {
				return tsaiclient.NewOpenRouter().
					WithReferer("https://example.com").
					WithTitle("Café")
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

// TestOpenRouterValidateAPIKey tests rejection of missing and malformed API keys.
func TestOpenRouterValidateAPIKey(t *testing.T) {
	// Define test cases.
	tests := []struct {
		name   string // The name of the test case.
		apiKey string // The API key value under test.
	}{
		{name: "empty API key", apiKey: ""},
		{name: "whitespace-only API key", apiKey: " \t\n"},
		{name: "embedded newline", apiKey: "bad\nkey"},
		{name: "embedded carriage return", apiKey: "bad\rkey"},
		{name: "embedded tab", apiKey: "bad\tkey"},
		{name: "non-ASCII characters", apiKey: "café"},
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
					t.Error(tserr.EqualStr(&tserr.EqualStrArgs{
						Var:    "OpenRouterProviderError error",
						Actual: err.Error(),
						Want:   want,
					}))
				}
			}
			// Check for every forbidden error substring.
			for _, notWant := range tt.notWantSub {
				// Fail if a forbidden substring is present.
				if strings.Contains(err.Error(), notWant) {
					t.Error(tserr.NotEqual(&tserr.NotEqualArgs{
						X: "OpenRouterProviderError error",
						Y: notWant,
					}))
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
				t.Fatal(tserr.EqualStr(&tserr.EqualStrArgs{
					Var:    "OpenRouter.ParseHTTPResponse error",
					Actual: err.Error(),
					Want:   tt.wantSub,
				}))
			}
			// Fail if an error path returned completion data.
			if result != (tsaiclient.Result{}) {
				t.Fatal(tserr.EqualStr(&tserr.EqualStrArgs{
					Var:    "OpenRouter.ParseHTTPResponse result",
					Actual: fmt.Sprintf("%+v", result),
					Want:   fmt.Sprintf("%+v", tsaiclient.Result{}),
				}))
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
					t.Fatal(tserr.EqualStr(&tserr.EqualStrArgs{
						Var:    "OpenRouter.ParseHTTPResponse error",
						Actual: err.Error(),
						Want:   tt.wantErrSub,
					}))
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
				t.Fatal(tserr.EqualStr(&tserr.EqualStrArgs{
					Var:    "OpenRouter.ParseHTTPResponse content",
					Actual: result.Content,
					Want:   tt.wantContent,
				}))
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
		t.Fatal(tserr.EqualStr(&tserr.EqualStrArgs{
			Var:    "OpenRouter.ParseHTTPResponse result",
			Actual: fmt.Sprintf("%+v", result),
			Want:   fmt.Sprintf("%+v", tsaiclient.Result{}),
		}))
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
				t.Fatal(tserr.EqualStr(&tserr.EqualStrArgs{
					Var:    "OpenRouter.ParseHTTPResponse result",
					Actual: fmt.Sprintf("%+v", got),
					Want:   fmt.Sprintf("%+v", tt.want),
				}))
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
				t.Fatal(tserr.EqualStr(&tserr.EqualStrArgs{
					Var:    "OpenRouter.ParseHTTPResponse error",
					Actual: err.Error(),
					Want:   tt.wantSub,
				}))
			}
			// Fail if an error path returned completion data.
			if result != (tsaiclient.Result{}) {
				t.Fatal(tserr.EqualStr(&tserr.EqualStrArgs{
					Var:    "OpenRouter.ParseHTTPResponse result",
					Actual: fmt.Sprintf("%+v", result),
					Want:   fmt.Sprintf("%+v", tsaiclient.Result{}),
				}))
			}
		})
	}
}

// TestOpenRouterBuildHTTPRequestValidation tests defensive request-builder guards.
func TestOpenRouterBuildHTTPRequestValidation(t *testing.T) {
	// Install a valid API key for cases that reach payload construction.
	t.Setenv("OPENROUTER_API_KEY", "test-key")
	// Construct a valid configuration.
	validCfg, err := tsaiclient.NewConfig("openai/gpt-5")
	// Fail if constructing the valid configuration fails.
	if err != nil {
		t.Fatal(tserr.NilExpected(&tserr.NilExpectedArgs{Op: "NewConfig", Err: err}))
	}
	// Construct a valid request.
	validReq, err := tsaiclient.NewRequest("Hello")
	// Fail if constructing the valid request fails.
	if err != nil {
		t.Fatal(tserr.NilExpected(&tserr.NilExpectedArgs{Op: "NewRequest", Err: err}))
	}
	// Define test cases.
	tests := []struct {
		name     string                 // The name of the test case.
		provider *tsaiclient.OpenRouter // The provider under test.
		ctx      context.Context        // The request context under test.
		cfg      *tsaiclient.Config     // The configuration under test.
		req      *tsaiclient.Request    // The request under test.
	}{
		{
			name: "nil provider",
			ctx:  context.Background(),
			cfg:  validCfg,
			req:  validReq,
		},
		{
			name:     "nil context",
			provider: tsaiclient.NewOpenRouter(),
			cfg:      validCfg,
			req:      validReq,
		},
		{
			name:     "invalid provider",
			provider: tsaiclient.NewOpenRouter().WithTitle("missing referer"),
			ctx:      context.Background(),
			cfg:      validCfg,
			req:      validReq,
		},
		{
			name:     "nil config",
			provider: tsaiclient.NewOpenRouter(),
			ctx:      context.Background(),
			req:      validReq,
		},
		{
			name:     "invalid config",
			provider: tsaiclient.NewOpenRouter(),
			ctx:      context.Background(),
			cfg:      &tsaiclient.Config{},
			req:      validReq,
		},
		{
			name:     "nil request",
			provider: tsaiclient.NewOpenRouter(),
			ctx:      context.Background(),
			cfg:      validCfg,
		},
		{
			name:     "invalid request",
			provider: tsaiclient.NewOpenRouter(),
			ctx:      context.Background(),
			cfg:      validCfg,
			req:      &tsaiclient.Request{},
		},
	}
	// Iterate over test cases.
	for _, tt := range tests {
		// Run each test case as a subtest.
		t.Run(tt.name, func(t *testing.T) {
			// Attempt to build the HTTP request.
			httpReq, err := tt.provider.BuildHTTPRequest(tt.ctx, tt.cfg, tt.req)
			// Fail if no error was returned.
			if err == nil {
				t.Fatal(tserr.NilFailed("OpenRouter.BuildHTTPRequest"))
			}
			// Fail if an HTTP request was returned with the error.
			if httpReq != nil {
				t.Fatal(tserr.EqualStr(&tserr.EqualStrArgs{
					Var:    "OpenRouter.BuildHTTPRequest request",
					Actual: fmt.Sprintf("%+v", httpReq),
					Want:   fmt.Sprintf("%+v", (*http.Request)(nil)),
				}))
			}
		})
	}
}

// TestOpenRouterBuildHTTPRequest tests full OpenRouter request construction.
func TestOpenRouterBuildHTTPRequest(t *testing.T) {
	// Install the API key expected in the authorization header.
	t.Setenv("OPENROUTER_API_KEY", "test-key")
	// Define a context key type for context propagation.
	type contextKey string
	// Create a context carrying a test value.
	ctx := context.WithValue(context.Background(), contextKey("request-id"), "request-123")
	// Configure all supported request-generation fields.
	cfg, err := tsaiclient.NewConfig("openai/gpt-5")
	// Fail if configuration construction fails.
	if err != nil {
		t.Fatal(tserr.NilExpected(&tserr.NilExpectedArgs{Op: "NewConfig", Err: err}))
	}
	// Apply generation options.
	cfg.WithMaxCompletionTokens(128).
		WithTemperature(0.7).
		WithReasoningEffort("high")
	// Configure system instructions and a stable session ID.
	req, err := tsaiclient.NewRequest("User prompt")
	// Fail if request construction fails.
	if err != nil {
		t.Fatal(tserr.NilExpected(&tserr.NilExpectedArgs{Op: "NewRequest", Err: err}))
	}
	// Preserve surrounding whitespace in the nonblank system prompt.
	req.WithSystemPrompt("  System prompt  ").WithSessionID("session-123")
	// Configure all attribution fields.
	provider := tsaiclient.NewOpenRouter().
		WithReferer("https://example.com/app").
		WithTitle("Example App").
		WithCategories([]string{"developer-tools", "productivity"}).
		WithHidden(true)
	// Enable strict privacy routing.
	if err := provider.EnablePrivacy(); err != nil {
		t.Fatal(tserr.NilExpected(&tserr.NilExpectedArgs{Op: "EnablePrivacy", Err: err}))
	}
	// Build the OpenRouter HTTP request.
	httpReq, err := provider.BuildHTTPRequest(ctx, cfg, req)
	// Fail if request construction fails.
	if err != nil {
		t.Fatal(tserr.NilExpected(&tserr.NilExpectedArgs{Op: "OpenRouter.BuildHTTPRequest", Err: err}))
	}
	// Verify the HTTP method.
	if httpReq.Method != http.MethodPost {
		t.Fatal(tserr.EqualStr(&tserr.EqualStrArgs{
			Var:    "OpenRouter.BuildHTTPRequest method",
			Actual: httpReq.Method,
			Want:   http.MethodPost,
		}))
	}
	// Verify the fixed OpenRouter endpoint.
	if got, want := httpReq.URL.String(), "https://openrouter.ai/api/v1/chat/completions"; got != want {
		t.Fatal(tserr.EqualStr(&tserr.EqualStrArgs{
			Var:    "OpenRouter.BuildHTTPRequest URL",
			Actual: got,
			Want:   want,
		}))
	}
	// Record the propagated context value as text.
	gotCtx := fmt.Sprintf("%v", httpReq.Context().Value(contextKey("request-id")))
	// Verify that the original context was propagated.
	if gotCtx != "request-123" {
		t.Fatal(tserr.EqualStr(&tserr.EqualStrArgs{
			Var:    "OpenRouter.BuildHTTPRequest context value",
			Actual: gotCtx,
			Want:   "request-123",
		}))
	}
	// Define expected headers.
	wantHeaders := map[string]string{
		"Authorization":               "Bearer test-key",
		"Content-Type":                "application/json",
		"HTTP-Referer":                "https://example.com/app",
		"X-OpenRouter-Title":          "Example App",
		"X-OpenRouter-Categories":     "developer-tools,productivity",
		"X-OpenRouter-App-Visibility": "hidden",
	}
	// Check every expected header.
	for key, want := range wantHeaders {
		// Fail if the header value differs.
		if got := httpReq.Header.Get(key); got != want {
			t.Error(tserr.EqualStr(&tserr.EqualStrArgs{
				Var:    fmt.Sprintf("OpenRouter.BuildHTTPRequest header %q", key),
				Actual: got,
				Want:   want,
			}))
		}
	}
	// Read the serialized request body.
	body, err := io.ReadAll(httpReq.Body)
	// Fail if reading the body fails.
	if err != nil {
		t.Fatal(tserr.NilExpected(&tserr.NilExpectedArgs{Op: "read request body", Err: err}))
	}
	// Decode the payload into a typed test representation.
	var payload struct {
		Model    string `json:"model"`
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
		MaxCompletionTokens int     `json:"max_completion_tokens"`
		Temperature         float64 `json:"temperature"`
		ReasoningEffort     string  `json:"reasoning_effort"`
		SessionID           string  `json:"session_id"`
		Provider            struct {
			DataCollection string `json:"data_collection"`
			ZDR            bool   `json:"zdr"`
		} `json:"provider"`
	}
	// Fail if decoding the request payload fails.
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatal(tserr.NilExpected(&tserr.NilExpectedArgs{Op: "decode request payload", Err: err}))
	}
	// Verify the model payload field.
	if payload.Model != "openai/gpt-5" {
		t.Fatal(tserr.EqualStr(&tserr.EqualStrArgs{
			Var:    "OpenRouter.BuildHTTPRequest payload model",
			Actual: payload.Model,
			Want:   "openai/gpt-5",
		}))
	}
	// Verify the token cap payload field.
	if payload.MaxCompletionTokens != 128 {
		t.Fatal(tserr.EqualInt(&tserr.EqualIntArgs{
			Var:    "OpenRouter.BuildHTTPRequest payload max_completion_tokens",
			Actual: int64(payload.MaxCompletionTokens),
			Want:   128,
		}))
	}
	// Verify the temperature payload field.
	if payload.Temperature != 0.7 {
		t.Fatal(tserr.Equalf(&tserr.EqualfArgs{
			Var:    "OpenRouter.BuildHTTPRequest payload temperature",
			Actual: payload.Temperature,
			Want:   0.7,
		}))
	}
	// Verify the reasoning effort payload field.
	if payload.ReasoningEffort != "high" {
		t.Fatal(tserr.EqualStr(&tserr.EqualStrArgs{
			Var:    "OpenRouter.BuildHTTPRequest payload reasoning_effort",
			Actual: payload.ReasoningEffort,
			Want:   "high",
		}))
	}
	// Verify the session ID payload field.
	if payload.SessionID != "session-123" {
		t.Fatal(tserr.EqualStr(&tserr.EqualStrArgs{
			Var:    "OpenRouter.BuildHTTPRequest payload session_id",
			Actual: payload.SessionID,
			Want:   "session-123",
		}))
	}
	// Verify the number and ordering of conversation messages.
	if len(payload.Messages) != 2 {
		t.Fatal(tserr.EqualInt(&tserr.EqualIntArgs{
			Var:    "OpenRouter.BuildHTTPRequest message count",
			Actual: int64(len(payload.Messages)),
			Want:   2,
		}))
	}
	// Verify the system message role.
	if payload.Messages[0].Role != "system" {
		t.Fatal(tserr.EqualStr(&tserr.EqualStrArgs{
			Var:    "OpenRouter.BuildHTTPRequest system message role",
			Actual: payload.Messages[0].Role,
			Want:   "system",
		}))
	}
	// Verify the system message content.
	if payload.Messages[0].Content != "  System prompt  " {
		t.Fatal(tserr.EqualStr(&tserr.EqualStrArgs{
			Var:    "OpenRouter.BuildHTTPRequest system message content",
			Actual: payload.Messages[0].Content,
			Want:   "  System prompt  ",
		}))
	}
	// Verify the user message role.
	if payload.Messages[1].Role != "user" {
		t.Fatal(tserr.EqualStr(&tserr.EqualStrArgs{
			Var:    "OpenRouter.BuildHTTPRequest user message role",
			Actual: payload.Messages[1].Role,
			Want:   "user",
		}))
	}
	// Verify the user message content.
	if payload.Messages[1].Content != "User prompt" {
		t.Fatal(tserr.EqualStr(&tserr.EqualStrArgs{
			Var:    "OpenRouter.BuildHTTPRequest user message content",
			Actual: payload.Messages[1].Content,
			Want:   "User prompt",
		}))
	}
	// Verify the data collection privacy preference.
	if payload.Provider.DataCollection != "deny" {
		t.Fatal(tserr.EqualStr(&tserr.EqualStrArgs{
			Var:    "OpenRouter.BuildHTTPRequest provider data_collection",
			Actual: payload.Provider.DataCollection,
			Want:   "deny",
		}))
	}
	// Verify the zero data retention privacy preference.
	if !payload.Provider.ZDR {
		t.Fatal(tserr.EqualStr(&tserr.EqualStrArgs{
			Var:    "OpenRouter.BuildHTTPRequest provider zdr",
			Actual: fmt.Sprintf("%t", payload.Provider.ZDR),
			Want:   "true",
		}))
	}
}

// TestOpenRouterBuildHTTPRequestDefaults tests omitted optional fields and headers.
func TestOpenRouterBuildHTTPRequestDefaults(t *testing.T) {
	// Install a valid API key.
	t.Setenv("OPENROUTER_API_KEY", "test-key")
	// Construct the default configuration.
	cfg, err := tsaiclient.NewConfig("openai/gpt-5")
	// Fail if configuration construction fails.
	if err != nil {
		t.Fatal(tserr.NilExpected(&tserr.NilExpectedArgs{Op: "NewConfig", Err: err}))
	}
	// Construct a request without a system prompt.
	req, err := tsaiclient.NewRequest("Hello")
	// Fail if request construction fails.
	if err != nil {
		t.Fatal(tserr.NilExpected(&tserr.NilExpectedArgs{Op: "NewRequest", Err: err}))
	}
	// Replace the generated session ID and mark the system prompt blank-only.
	req.WithSystemPrompt(" \t\n").WithSessionID("")
	// Build a request with the zero-value provider configuration.
	httpReq, err := tsaiclient.NewOpenRouter().BuildHTTPRequest(context.Background(), cfg, req)
	// Fail if request construction fails.
	if err != nil {
		t.Fatal(tserr.NilExpected(&tserr.NilExpectedArgs{Op: "OpenRouter.BuildHTTPRequest", Err: err}))
	}
	// Read the request body.
	body, err := io.ReadAll(httpReq.Body)
	// Fail if reading the body fails.
	if err != nil {
		t.Fatal(tserr.NilExpected(&tserr.NilExpectedArgs{Op: "read request body", Err: err}))
	}
	// Decode the payload generically to check key omission.
	var payload map[string]any
	// Fail if decoding fails.
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatal(tserr.NilExpected(&tserr.NilExpectedArgs{Op: "decode request payload", Err: err}))
	}
	// Check that zero-value optional fields are omitted.
	for _, key := range []string{
		"max_completion_tokens",
		"temperature",
		"reasoning_effort",
		"session_id",
		"provider",
	} {
		// Fail if an omitted field is present.
		if _, present := payload[key]; present {
			t.Error(tserr.NotEqual(&tserr.NotEqualArgs{
				X: fmt.Sprintf("OpenRouter.BuildHTTPRequest payload key %q", key),
				Y: "present",
			}))
		}
	}
	// Decode the messages separately.
	var messages []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	// Re-marshal only the message value for typed decoding.
	messageJSON, err := json.Marshal(payload["messages"])
	// Fail if preparing the messages fails.
	if err != nil {
		t.Fatal(tserr.NilExpected(&tserr.NilExpectedArgs{Op: "encode messages", Err: err}))
	}
	// Fail if decoding the messages fails.
	if err := json.Unmarshal(messageJSON, &messages); err != nil {
		t.Fatal(tserr.NilExpected(&tserr.NilExpectedArgs{Op: "decode messages", Err: err}))
	}
	// Verify that only the user message is present.
	if len(messages) != 1 {
		t.Fatal(tserr.EqualInt(&tserr.EqualIntArgs{
			Var:    "OpenRouter.BuildHTTPRequest message count",
			Actual: int64(len(messages)),
			Want:   1,
		}))
	}
	// Verify the user message role.
	if messages[0].Role != "user" {
		t.Fatal(tserr.EqualStr(&tserr.EqualStrArgs{
			Var:    "OpenRouter.BuildHTTPRequest user message role",
			Actual: messages[0].Role,
			Want:   "user",
		}))
	}
	// Verify the user message content.
	if messages[0].Content != "Hello" {
		t.Fatal(tserr.EqualStr(&tserr.EqualStrArgs{
			Var:    "OpenRouter.BuildHTTPRequest user message content",
			Actual: messages[0].Content,
			Want:   "Hello",
		}))
	}
	// Verify that no attribution headers are emitted.
	for _, key := range []string{
		"HTTP-Referer",
		"X-OpenRouter-Title",
		"X-OpenRouter-Categories",
		"X-OpenRouter-App-Visibility",
	} {
		// Fail if an attribution header is present.
		if _, present := httpReq.Header[http.CanonicalHeaderKey(key)]; present {
			t.Error(tserr.NotEqual(&tserr.NotEqualArgs{
				X: fmt.Sprintf("OpenRouter.BuildHTTPRequest header %q", key),
				Y: "present",
			}))
		}
	}
}

// TestEnablePrivacyNil tests that enabling privacy on a nil provider returns an error.
func TestEnablePrivacyNil(t *testing.T) {
	// Run the test in parallel.
	t.Parallel()
	// Declare a typed-nil provider receiver.
	var provider *tsaiclient.OpenRouter
	// Attempt to enable privacy on the nil receiver.
	err := provider.EnablePrivacy()
	// Fail if no error was returned.
	if err == nil {
		t.Fatal(tserr.NilFailed("OpenRouter.EnablePrivacy"))
	}
}

// TestIsValidHeaderValue tests the HTTP header field value constraints of isValidHeaderValue.
func TestIsValidHeaderValue(t *testing.T) {
	// Run subtests in parallel.
	t.Parallel()
	// Define test cases.
	tests := []struct {
		name  string // The name of the test case.
		value string // The header field value under test.
		want  bool   // Whether the value is expected to be valid.
	}{
		{name: "text with spaces", value: "Example App", want: true},
		{name: "printable symbols", value: "sk-or-v1-abc_123-XYZ", want: true},
		{name: "single space", value: " ", want: true},
		{name: "empty value", value: "", want: false},
		{name: "embedded newline", value: "My\nApp", want: false},
		{name: "embedded carriage return", value: "My\rApp", want: false},
		{name: "embedded tab", value: "My\tApp", want: false},
		{name: "NUL byte", value: "My\x00App", want: false},
		{name: "DEL byte", value: "My\x7fApp", want: false},
		{name: "non-ASCII letters", value: "Café", want: false},
		{name: "invalid UTF-8", value: "My\xffApp", want: false},
	}
	// Iterate over test cases.
	for _, tt := range tests {
		// Run each test case as a subtest.
		t.Run(tt.name, func(t *testing.T) {
			// Run the subtest in parallel.
			t.Parallel()
			// Validate the header field value under test.
			got := tsaiclient.IsValidHeaderValue(tt.value)
			// Report a mismatch between actual and expected validity.
			if got != tt.want {
				t.Fatal(tserr.EqualStr(&tserr.EqualStrArgs{
					Var:    fmt.Sprintf("IsValidHeaderValue(%q)", tt.value),
					Actual: fmt.Sprintf("%t", got),
					Want:   fmt.Sprintf("%t", tt.want),
				}))
			}
		})
	}
}
