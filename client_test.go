// Copyright (c) 2026 thorsphere.
// All Rights Reserved. Use is governed by the Functional Source License v1.1
// (FSL-1.1-ALv2) that can be found in the LICENSE file.

package tsaiclient_test

// Import packages.
import (
	"bytes"          // Import the bytes package for byte slices.
	"context"        // Import the context package for request cancellation.
	"errors"         // Import the errors package for error handling.
	"io"             // Import the io package for reading response bodies.
	"math"           // Import the math package for signed zero checks.
	"net/http"       // Import the http package for HTTP requests.
	"strings"        // Import the strings package for string operations.
	"testing"        // Import the testing package for subtests.
	"testing/iotest" // Import the iotest package for testing I/O.
	"time"           // Import the time package for timeouts.

	"github.com/thorsphere/lpstats"    // Import the lpstats package for floating-point pointers.
	"github.com/thorsphere/tsaiclient" // Import the tsaiclient package for testing.
	"github.com/thorsphere/tserr"      // Import the tserr package for error handling.
)

// TestNewClient tests Client construction via NewClient.
func TestNewClient(t *testing.T) {
	// Run subtests in parallel.
	t.Parallel()
	// Create a valid base configuration.
	validCfg, err := tsaiclient.NewConfig("open/for-5")
	// Fail the test if creating the valid configuration fails.
	if err != nil {
		t.Fatal(err)
	}
	// Create an invalid configuration with an empty model.
	invalidCfg, err := tsaiclient.NewConfig("open/for-5")
	// Fail the test if creating the configuration fails.
	if err != nil {
		t.Fatal(err)
	}
	// Invalidate the configuration by clearing the model.
	invalidCfg.WithModel("")
	// Declare a typed-nil provider to test interface nil handling.
	var typedNilProvider *testProvider
	// Define test cases.
	tests := []struct {
		name     string              // The name of the test case.
		cfg      *tsaiclient.Config  // The configuration under test.
		provider tsaiclient.Provider // The provider under test.
		wantErr  bool                // Whether an error is expected.
	}{
		{
			name:     "valid config and provider",
			cfg:      validCfg,
			provider: &testProvider{},
			wantErr:  false,
		},
		{
			name:     "nil config",
			cfg:      nil,
			provider: &testProvider{},
			wantErr:  true,
		},
		{
			name:     "invalid config",
			cfg:      invalidCfg,
			provider: &testProvider{},
			wantErr:  true,
		},
		{
			name:     "nil provider",
			cfg:      validCfg,
			provider: nil,
			wantErr:  true,
		},
		{
			name:     "typed-nil provider",
			cfg:      validCfg,
			provider: typedNilProvider,
			wantErr:  true,
		},
		{
			name:     "invalid provider",
			cfg:      validCfg,
			provider: &testProvider{validateErr: tserr.Empty("test provider API key")},
			wantErr:  true,
		},
	}
	// Iterate over test cases.
	for _, tt := range tests {
		// Run each test case as a subtest.
		t.Run(tt.name, func(t *testing.T) {
			// Run subtest in parallel.
			t.Parallel()
			// Create a new client and check whether the error matches expectations.
			client, err := tsaiclient.NewClient(tt.cfg, tt.provider)
			// Check whether an expected error was missing.
			if tt.wantErr && err == nil {
				t.Fatal(tserr.NilFailed("NewClient"))
			}
			// Check whether an unexpected error was returned.
			if !tt.wantErr && err != nil {
				t.Fatal(tserr.NilExpected(&tserr.NilExpectedArgs{Op: "NewClient", Err: err}))
			}
			// Verify the returned client is non-nil when no error is expected.
			if !tt.wantErr && client == nil {
				t.Fatal(tserr.NilFailed("NewClient"))
			}
			// Verify the returned client is nil when an error is expected.
			if tt.wantErr && client != nil {
				t.Fatal(tserr.NilExpected(&tserr.NilExpectedArgs{Op: "NewClient"}))
			}
		})
	}
}

// TestNewClientTemperatureSnapshot tests that NewClient captures an independent
// optional temperature despite later caller and provider mutations.
func TestNewClientTemperatureSnapshot(t *testing.T) {
	// Run cases in parallel because they use no shared seams or environment changes.
	t.Parallel()
	// Create a sentinel error that stops completion before any HTTP request.
	stopBuild := errors.New("stop after capturing client configuration")
	// Define the initial values and subsequent caller mutations under test.
	tests := []struct {
		name        string   // The name of the test case.
		initial     *float64 // The initial configured temperature; nil leaves it unset.
		mutateTo    *float64 // The replacement temperature; nil clears the setting.
		want        *float64 // The temperature expected in the client snapshot.
		wantCurrent *float64 // The caller temperature expected after the mutation.
	}{
		{name: "unset stays unset after setting", mutateTo: lpstats.PtrFloat(0.7), wantCurrent: lpstats.PtrFloat(0.7)},
		{name: "unset stays unset after clearing"},
		{name: "configured value survives replacement", initial: lpstats.PtrFloat(0.7), mutateTo: lpstats.PtrFloat(1.2), want: lpstats.PtrFloat(0.7), wantCurrent: lpstats.PtrFloat(1.2)},
		{name: "configured value survives clearing", initial: lpstats.PtrFloat(0.7), want: lpstats.PtrFloat(0.7)},
		{name: "explicit zero survives clearing", initial: lpstats.PtrFloat(0.0), want: lpstats.PtrFloat(0.0)},
		{name: "negative zero is normalized", initial: lpstats.PtrFloat(math.Copysign(0, -1)), mutateTo: lpstats.PtrFloat(0.7), want: lpstats.PtrFloat(0.0), wantCurrent: lpstats.PtrFloat(0.7)},
	}
	// Iterate over the snapshot cases.
	for _, tt := range tests {
		// Run each case as an independent parallel subtest.
		t.Run(tt.name, func(t *testing.T) {
			// Allow this case to run alongside the other independent cases.
			t.Parallel()
			// Construct a valid caller-owned configuration.
			cfg, err := tsaiclient.NewConfig("open/for-5")
			// Fail if configuration construction fails.
			if err != nil {
				// Report the unexpected configuration construction error.
				t.Fatal(tserr.NilExpected(&tserr.NilExpectedArgs{Op: "NewConfig", Err: err}))
			}
			// Configure the initial temperature when provided.
			if tt.initial != nil {
				// Set the temperature before constructing the client.
				cfg.WithTemperature(*tt.initial)
			}
			// Record the original pointer so its underlying value can be mutated later.
			originalTemperature := cfg.GetTemperature()
			// Capture the configuration actually passed to the provider.
			var capturedCfg *tsaiclient.Config
			// Configure a provider that stops before the HTTP seam is reached.
			provider := &testProvider{
				buildFn: func(_ context.Context, clientCfg *tsaiclient.Config, _ *tsaiclient.Request) (*http.Request, error) {
					// Save the client-owned configuration supplied to the provider.
					capturedCfg = clientCfg
					// Stop the call without executing an HTTP request.
					return nil, stopBuild
				},
			}
			// Construct the client with the caller-owned configuration.
			client, err := tsaiclient.NewClient(cfg, provider)
			// Fail if client construction fails.
			if err != nil {
				// Report the unexpected client construction error.
				t.Fatal(tserr.NilExpected(&tserr.NilExpectedArgs{Op: "NewClient", Err: err}))
			}
			// Change the caller configuration without changing the client's snapshot.
			if tt.mutateTo != nil {
				// Replace the caller's temperature through the public builder.
				cfg.WithTemperature(*tt.mutateTo)
			} else {
				// Clear the caller's temperature through the public builder.
				cfg.WithoutTemperature()
			}
			// Verify the requested caller-side builder mutation took effect.
			if !lpstats.NearEqualFloatPtr(cfg.GetTemperature(), tt.wantCurrent, 0) {
				// Report the unexpected caller temperature.
				t.Fatal(tserr.EqualStr(&tserr.EqualStrArgs{
					Var:    "caller temperature after builder mutation",
					Actual: lpstats.FmtFloatPtr(cfg.GetTemperature(), 3),
					Want:   lpstats.FmtFloatPtr(tt.wantCurrent, 3),
				}))
			}
			// Construct a valid completion request.
			req, err := tsaiclient.NewRequest("Hello")
			// Fail if request construction fails.
			if err != nil {
				// Report the unexpected request construction error.
				t.Fatal(tserr.NilExpected(&tserr.NilExpectedArgs{Op: "NewRequest", Err: err}))
			}
			// Invoke completion to observe the captured configuration at the provider.
			_, err = client.Complete(context.Background(), req)
			// Fail if the provider's sentinel error was not returned.
			if err == nil {
				// Report the missing provider error.
				t.Fatal(tserr.NilFailed("Client.Complete"))
			}
			// Verify the error originated from the provider's capture hook.
			if !errors.Is(err, stopBuild) {
				// Report the unexpected provider error.
				t.Fatal(tserr.EqualStr(&tserr.EqualStrArgs{Var: "Client.Complete error", Actual: err.Error(), Want: stopBuild.Error()}))
			}
			// Verify the provider received the copied configuration.
			if capturedCfg == nil {
				// Report the missing provider configuration.
				t.Fatal(tserr.Empty("provider configuration"))
			}
			// Read the optional temperature from the client's captured configuration.
			capturedTemperature := capturedCfg.GetTemperature()
			// Verify the snapshot preserves the original value and omission state.
			if !lpstats.NearEqualFloatPtr(capturedTemperature, tt.want, 0) {
				// Report the unexpected captured temperature.
				t.Fatal(tserr.EqualStr(&tserr.EqualStrArgs{
					Var:    "captured client temperature",
					Actual: lpstats.FmtFloatPtr(capturedTemperature, 3),
					Want:   lpstats.FmtFloatPtr(tt.want, 3),
				}))
			}
			// Verify zero was normalized before it entered the client snapshot.
			if capturedTemperature != nil && *capturedTemperature == 0 && math.Signbit(*capturedTemperature) {
				// Report an unexpectedly negative zero.
				t.Error(tserr.EqualStr(&tserr.EqualStrArgs{Var: "captured zero sign", Actual: "negative", Want: "positive"}))
			}
			// Test pointer independence when an original temperature was configured.
			if originalTemperature != nil {
				// Reject a shared pointer even if the builder replacement hid the alias.
				if capturedTemperature == originalTemperature {
					// Report that the caller and client share the same pointee.
					t.Error("client and caller share the same temperature pointer")
				}
				// Mutate the original pointee directly to detect shallow copying.
				*originalTemperature = 1.5
				// Verify the client still holds its original temperature.
				if !lpstats.NearEqualFloatPtr(capturedTemperature, tt.want, 0) {
					// Report the unexpected captured temperature after mutation.
					t.Error(tserr.EqualStr(&tserr.EqualStrArgs{
						Var:    "captured client temperature after caller pointee mutation",
						Actual: lpstats.FmtFloatPtr(capturedTemperature, 3),
						Want:   lpstats.FmtFloatPtr(tt.want, 3),
					}))
				}
				// Verify a client-side pointee mutation cannot affect the original pointer.
				if capturedTemperature != nil {
					// Change only the client-owned temperature value.
					*capturedTemperature = 1.8
					// Prepare the expected caller-owned pointee value.
					wantOriginal := lpstats.PtrFloat(1.5)
					// Confirm that the original pointee remains unchanged.
					if !lpstats.NearEqualFloatPtr(originalTemperature, wantOriginal, 0) {
						// Report an unexpected change to the caller's pointee.
						t.Error(tserr.EqualStr(&tserr.EqualStrArgs{
							Var:    "caller temperature after client pointee mutation",
							Actual: lpstats.FmtFloatPtr(originalTemperature, 3),
							Want:   lpstats.FmtFloatPtr(wantOriginal, 3),
						}))
					}
				}
			}
		})
	}
}

// TestClientComplete tests the completion workflow and defensive guards of Client.Complete.
func TestClientComplete(t *testing.T) {
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
	// Create a pre-canceled context for the canceled-context test case.
	canceledCtx, cancel := context.WithCancel(context.Background())
	// Cancel the context immediately.
	cancel()
	// Construct a valid HTTP request to return from the stub provider.
	validHTTPReq, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "https://api.example.com/v1/chat", nil)
	// Fail the test if creating the HTTP request fails.
	if err != nil {
		t.Fatal(err)
	}
	// Construct a valid provider that returns validHTTPReq and a parsed Result.
	validProvider := &testProvider{
		buildFn: func(_ context.Context, _ *tsaiclient.Config, _ *tsaiclient.Request) (*http.Request, error) {
			return validHTTPReq, nil
		},
		parseFn: func(statusCode int, body []byte) (tsaiclient.Result, error) {
			return tsaiclient.Result{
				Content:   string(body),
				Truncated: statusCode == http.StatusPartialContent,
				Usage:     tsaiclient.Usage{PromptTokens: 3, CompletionTokens: 5, TotalTokens: 8},
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
		name       string                                      // The name of the test case.
		client     *tsaiclient.Client                          // The client under test.
		ctx        context.Context                             // The context passed to Complete.
		req        *tsaiclient.Request                         // The request passed to Complete.
		httpDo     func(*http.Request) (*http.Response, error) // The HTTP execution seam for the test.
		wantResult tsaiclient.Result                           // The expected completion result.
		wantErr    bool                                        // Whether an error is expected.
	}{
		{
			name:   "successful completion",
			client: validClient,
			ctx:    context.Background(),
			req:    validReq,
			httpDo: func(_ *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("world")),
				}, nil
			},
			wantResult: tsaiclient.Result{
				Content:   "world",
				Truncated: false,
				Usage:     tsaiclient.Usage{PromptTokens: 3, CompletionTokens: 5, TotalTokens: 8},
			},
			wantErr: false,
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
		{
			name:    "nil provider on bare client",
			client:  tsaiclient.NewBareClient(validCfg, nil),
			ctx:     context.Background(),
			req:     validReq,
			wantErr: true,
		},
		{
			name:    "nil config on bare client",
			client:  tsaiclient.NewBareClient(nil, validProvider),
			ctx:     context.Background(),
			req:     validReq,
			wantErr: true,
		},
		{
			name:    "nil context",
			client:  validClient,
			ctx:     nil,
			req:     validReq,
			wantErr: true,
		},
		{
			name:    "canceled context",
			client:  validClient,
			ctx:     canceledCtx,
			req:     validReq,
			wantErr: true,
		},
		{
			name: "BuildHTTPRequest returns error",
			client: tsaiclient.NewBareClient(validCfg, &testProvider{
				buildFn: func(_ context.Context, _ *tsaiclient.Config, _ *tsaiclient.Request) (*http.Request, error) {
					return nil, errors.New("build failed")
				},
			}),
			ctx:     context.Background(),
			req:     validReq,
			wantErr: true,
		},
		{
			name:    "BuildHTTPRequest returns nil request without error",
			client:  tsaiclient.NewBareClient(validCfg, &testProvider{}),
			ctx:     context.Background(),
			req:     validReq,
			wantErr: true,
		},
		{
			name:   "httpDo returns transport error",
			client: validClient,
			ctx:    context.Background(),
			req:    validReq,
			httpDo: func(_ *http.Request) (*http.Response, error) {
				return nil, errors.New("network failure")
			},
			wantErr: true,
		},
		{
			name:   "httpDo returns nil response without error",
			client: validClient,
			ctx:    context.Background(),
			req:    validReq,
			httpDo: func(_ *http.Request) (*http.Response, error) {
				return nil, nil
			},
			wantErr: true,
		},
		{
			name:   "httpDo returns response with nil body",
			client: validClient,
			ctx:    context.Background(),
			req:    validReq,
			httpDo: func(_ *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: http.StatusOK, Body: nil}, nil
			},
			wantErr: true,
		},
		{
			name:   "response body at exact maxResponseBytes limit succeeds",
			client: validClient,
			ctx:    context.Background(),
			req:    validReq,
			httpDo: func(_ *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader(make([]byte, tsaiclient.MaxResponseBytes))),
				}, nil
			},
			wantResult: tsaiclient.Result{
				Content: string(make([]byte, tsaiclient.MaxResponseBytes)),
				Usage:   tsaiclient.Usage{PromptTokens: 3, CompletionTokens: 5, TotalTokens: 8},
			},
			wantErr: false,
		},
		{
			name:   "response body exceeds maxResponseBytes",
			client: validClient,
			ctx:    context.Background(),
			req:    validReq,
			httpDo: func(_ *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader(make([]byte, tsaiclient.MaxResponseBytes+1))),
				}, nil
			},
			wantErr: true,
		},
		{
			name: "ParseHTTPResponse returns error",
			client: tsaiclient.NewBareClient(validCfg, &testProvider{
				buildFn: func(_ context.Context, _ *tsaiclient.Config, _ *tsaiclient.Request) (*http.Request, error) {
					return validHTTPReq, nil
				},
				parseFn: func(_ int, _ []byte) (tsaiclient.Result, error) {
					return tsaiclient.Result{}, errors.New("parse failed")
				},
			}),
			ctx: context.Background(),
			req: validReq,
			httpDo: func(_ *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusBadGateway,
					Body:       io.NopCloser(strings.NewReader("bad gateway")),
				}, nil
			},
			wantErr: true,
		},
		{
			name:   "reading response body fails",
			client: validClient,
			ctx:    context.Background(),
			req:    validReq,
			httpDo: func(_ *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(iotest.ErrReader(errors.New("read failed"))),
				}, nil
			},
			wantErr: true,
		},
		{
			name:   "Content-Length declares oversized response body",
			client: validClient,
			ctx:    context.Background(),
			req:    validReq,
			httpDo: func(_ *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode:    http.StatusOK,
					ContentLength: tsaiclient.MaxResponseBytes + 1, // Declared size exceeds the limit.
					Body:          io.NopCloser(iotest.ErrReader(errors.New("body must not be read"))),
				}, nil
			},
			wantErr: true,
		},
		{
			name:    "invalid provider on bare client",
			client:  tsaiclient.NewBareClient(validCfg, &testProvider{validateErr: errors.New("invalid provider")}),
			ctx:     context.Background(),
			req:     validReq,
			wantErr: true,
		},
		{
			name:    "invalid config on bare client",
			client:  tsaiclient.NewBareClient(&tsaiclient.Config{}, validProvider),
			ctx:     context.Background(),
			req:     validReq,
			wantErr: true,
		},
	}
	// Iterate over test cases.
	for _, tt := range tests {
		// Run each test case as a serial subtest because httpDo is a package-level seam.
		t.Run(tt.name, func(t *testing.T) {
			// Install the test HTTP seam when provided.
			if tt.httpDo != nil {
				restore := tsaiclient.SetHTTPDo(tt.httpDo)
				defer restore()
			}
			// Execute Complete under test.
			got, err := tt.client.Complete(tt.ctx, tt.req)
			// Check whether an expected error was missing.
			if tt.wantErr && err == nil {
				t.Fatal(tserr.NilFailed("Complete"))
			}
			// Check whether an unexpected error was returned.
			if !tt.wantErr && err != nil {
				t.Fatal(tserr.NilExpected(&tserr.NilExpectedArgs{Op: "Complete", Err: err}))
			}
			// Verify the returned Result matches wantResult.
			if got != tt.wantResult {
				t.Fatal(tserr.EqualStr(&tserr.EqualStrArgs{
					Var:    "Complete result",
					Actual: got.Content,
					Want:   tt.wantResult.Content,
				}))
			}
		})
	}
}

// TestClientCompleteTimeout tests that Complete propagates the configured
// timeout as a deadline on the request context.
func TestClientCompleteTimeout(t *testing.T) {
	// The timeout under test.
	const testTimeout = 5 * time.Second
	// Create a valid configuration.
	cfg, err := tsaiclient.NewConfig("open/for-5")
	// Fail the test if creating the configuration fails.
	if err != nil {
		t.Fatal(err)
	}
	// Apply the timeout under test.
	cfg.WithTimeout(testTimeout)
	// Create a valid request.
	req, err := tsaiclient.NewRequest("Hello")
	// Fail the test if creating the request fails.
	if err != nil {
		t.Fatal(err)
	}
	// Build a provider that forwards the context so the deadline reaches the HTTP request.
	provider := &testProvider{
		buildFn: func(ctx context.Context, _ *tsaiclient.Config, _ *tsaiclient.Request) (*http.Request, error) {
			// Build the HTTP request with the context under test.
			return http.NewRequestWithContext(ctx, http.MethodPost, "https://api.example.com/v1/chat", nil)
		},
		parseFn: func(_ int, body []byte) (tsaiclient.Result, error) {
			// Return the body as the completion content.
			return tsaiclient.Result{Content: string(body)}, nil
		},
	}
	// Create the client under test.
	client, err := tsaiclient.NewClient(cfg, provider)
	// Fail the test if creating the client fails.
	if err != nil {
		t.Fatal(err)
	}
	// Install the HTTP seam that inspects the request context.
	restore := tsaiclient.SetHTTPDo(func(req *http.Request) (*http.Response, error) {
		// Read the deadline from the request context.
		deadline, ok := req.Context().Deadline()
		// Fail the test when the request carries no deadline.
		if !ok {
			t.Fatal("request context has no deadline")
		}
		// Compute how much time is left until the deadline.
		remaining := time.Until(deadline)
		// Fail the test when the remaining time is not approximately the configured timeout.
		if remaining > testTimeout || remaining < testTimeout-time.Second {
			t.Fatalf("deadline remaining %v, want %v within 1s slack", remaining, testTimeout)
		}
		// Return a minimal successful response.
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("world")),
		}, nil
	})
	// Restore the previous seam when the test finishes.
	defer restore()
	// Execute Complete under test.
	_, err = client.Complete(context.Background(), req)
	// Fail the test when Complete fails.
	if err != nil {
		t.Fatal(err)
	}
}
