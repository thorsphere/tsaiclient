// Copyright (c) 2026 thorsphere.
// All Rights Reserved. Use is governed by the Functional Source License v1.1
// (FSL-1.1-ALv2) that can be found in the LICENSE file.

package tsaiclient

// Import packages.
import (
	"context"  // Import the context package for cancellation.
	"io"       // Import the io package for reading response bodies.
	"net/http" // Import the http package for HTTP requests.

	"github.com/thorsphere/tserr" // Import the tserr package for error handling.
)

// Client holds a configured Provider and its configuration.
// The zero value is not usable; construct instances with NewClient.
type Client struct {
	provider Provider // The AI backend implementation.
	cfg      *Config  // The validated client configuration.
}

// Result holds a normalized completion response.
type Result struct {
	Content   string // The raw text returned by the model.
	Truncated bool   // True when generation hit MaxCompletionTokens (finish_reason == "length").
	Usage     Usage  // Token counts and cost for logging/metering.
}

// Usage is the normalized token and cost accounting reported by a provider for
// one chat completion. The zero value means the provider reported no usage.
type Usage struct {
	PromptTokens     int     // The number of tokens in the prompt.
	CompletionTokens int     // The number of tokens in the completion.
	ReasoningTokens  int     // The reasoning tokens generated; zero means not reported.
	TotalTokens      int     // The total number of tokens billed for the call.
	Cost             float64 // The completion cost in USD; zero means not reported.
}

// defaultHTTPClient is the HTTP client used for provider chat requests.
// Per-call deadlines are enforced through the request context, not the client.
var defaultHTTPClient = &http.Client{}

// httpDo is the package-level seam for executing HTTP requests.
var httpDo = defaultHTTPClient.Do

// NewClient returns a Client configured with cfg and provider. It returns an error
// if cfg or provider is nil or invalid.
func NewClient(cfg *Config, provider Provider) (*Client, error) {
	// Return an error if the configuration is nil.
	if cfg == nil {
		return nil, tserr.Empty("config")
	}
	// Return an error if the configuration is invalid.
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	// Return an error if the provider is not set.
	if provider == nil {
		return nil, tserr.Empty("provider")
	}
	// Return an error if the provider is invalid.
	if err := provider.Validate(); err != nil {
		return nil, err
	}
	// Copy the configuration to avoid mutating the original.
	cfgCopy := *cfg
	// Return the initialized client.
	return &Client{cfg: &cfgCopy, provider: provider}, nil
}

// Complete sends req to the configured provider using ctx and returns the
// normalized Result. It returns an error if the receiver, ctx, or req is nil
// or invalid, or if the request fails or the response is unusable.
func (c *Client) Complete(ctx context.Context, req *Request) (Result, error) {
	// Return an error if the client is nil.
	if c == nil {
		return Result{}, tserr.NilPtr()
	}
	// Return an error if the request is nil.
	if req == nil {
		return Result{}, tserr.NilPtr()
	}
	// Return an error if the request is invalid.
	if err := req.Validate(); err != nil {
		return Result{}, err
	}
	// Return an error if the provider is nil.
	if c.provider == nil {
		return Result{}, tserr.Empty("provider")
	}
	// Return an error if the provider is invalid.
	if err := c.provider.Validate(); err != nil {
		return Result{}, err
	}
	// Return an error if the configuration is invalid.
	if err := c.cfg.Validate(); err != nil {
		return Result{}, err
	}
	// Return an error if the context is nil.
	if ctx == nil {
		return Result{}, tserr.Empty("context")
	}
	// Return an error if the context is already canceled or expired.
	if err := ctx.Err(); err != nil {
		return Result{}, tserr.ContextDone(err)
	}
	// Resolve the effective per-call timeout from the configuration.
	timeout := c.cfg.timeout
	// Fall back to the default timeout when none is configured.
	if timeout == 0 {
		timeout = defaultTimeout
	}
	// Bound the call with a deadline and release it when Complete returns.
	ctx, cancel := context.WithTimeout(ctx, timeout)
	// Release the timer resources when the call finishes.
	defer cancel()
	// Build the HTTP request for the configured provider.
	httpReq, err := c.provider.BuildHTTPRequest(ctx, c.cfg, req)
	// Return an empty result and an error if building request fails.
	if err != nil {
		return Result{}, err
	}
	// Return an error if the HTTP request is nil.
	if httpReq == nil {
		return Result{}, tserr.Empty("http request")
	}
	// Send the HTTP request.
	resp, err := httpDo(httpReq)
	// Return an empty result and an error if sending request fails.
	if err != nil {
		return Result{}, tserr.Op(&tserr.OpArgs{Op: "send request", Fn: c.provider.Name(), Err: err})
	}
	// Return an error if the HTTP response or its body is nil.
	if resp == nil || resp.Body == nil {
		return Result{}, tserr.Empty("http response")
	}
	// Close the response body after reading.
	defer resp.Body.Close()
	// Return an error early when Content-Length already declares an oversized body.
	if resp.ContentLength > maxResponseBytes {
		return Result{}, tserr.TooLarge(&tserr.TooLargeArgs{
			F:        "response body",
			Actual:   resp.ContentLength, // The exact size declared by the provider.
			MaxBytes: maxResponseBytes,
		})
	}
	// Read the response body, limiting it to maxResponseBytes.
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	// Return an empty result and an error if reading the response body fails.
	if err != nil {
		return Result{}, tserr.Op(&tserr.OpArgs{Op: "read body", Fn: c.provider.Name(), Err: err})
	}
	// Return an error if the response body exceeds the maximum allowed size.
	if len(respBody) > maxResponseBytes {
		return Result{}, tserr.TooLarge(&tserr.TooLargeArgs{
			F:        "response body",
			Actual:   int64(len(respBody)), // The bytes read before the cutoff, a lower bound on the true size.
			MaxBytes: maxResponseBytes,
		})
	}
	// Parse the HTTP response body into a normalized Result.
	return c.provider.ParseHTTPResponse(resp.StatusCode, respBody)
}
