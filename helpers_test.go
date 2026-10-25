// Copyright (c) 2026 thorsphere.
// All Rights Reserved. Use is governed by the Functional Source License v1.1
// (FSL-1.1-ALv2) that can be found in the LICENSE file.

package tsaiclient_test

// Import packages.
import (
	"context"  // Import the context package for request cancellation.
	"net/http" // Import the http package for HTTP requests.

	"github.com/thorsphere/tsaiclient" // Import the tsaiclient package for testing.
	"github.com/thorsphere/tserr"      // Import the tserr package for error handling.
)

// testProvider is a configurable Provider stub for Client unit tests.
type testProvider struct {
	validateErr error                                                                                             // The error returned by Validate.
	buildFn     func(ctx context.Context, cfg *tsaiclient.Config, req *tsaiclient.Request) (*http.Request, error) // Optional custom BuildHTTPRequest hook.
	parseFn     func(statusCode int, body []byte) (tsaiclient.Result, error)                                      // Optional custom ParseHTTPResponse hook.
}

// Name returns the stub provider name.
func (p *testProvider) Name() string {
	// Return the stub provider name.
	return "TestProvider"
}

// EnablePrivacy enables privacy mode on the stub provider.
func (p *testProvider) EnablePrivacy() error {
	// Return an error if the receiver is nil.
	if p == nil {
		return tserr.NilPtr()
	}
	// Return no error to indicate success.
	return nil
}

// Validate returns the configured validation error for the stub provider.
func (p *testProvider) Validate() error {
	// Return an error if the receiver is nil.
	if p == nil {
		return tserr.NilPtr()
	}
	// Return the configured validation error.
	return p.validateErr
}

// BuildHTTPRequest delegates to buildFn when set or returns a nil request.
func (p *testProvider) BuildHTTPRequest(ctx context.Context, cfg *tsaiclient.Config, req *tsaiclient.Request) (*http.Request, error) {
	// Return an error if the receiver is nil.
	if p == nil {
		return nil, tserr.NilPtr()
	}
	// Delegate to the custom build function when configured.
	if p.buildFn != nil {
		return p.buildFn(ctx, cfg, req)
	}
	// Return a nil request by default.
	return nil, nil
}

// ParseHTTPResponse delegates to parseFn when set or returns an empty Result.
func (p *testProvider) ParseHTTPResponse(statusCode int, body []byte) (tsaiclient.Result, error) {
	// Return an error if the receiver is nil.
	if p == nil {
		return tsaiclient.Result{}, tserr.NilPtr()
	}
	// Delegate to the custom parse function when configured.
	if p.parseFn != nil {
		return p.parseFn(statusCode, body)
	}
	// Return an empty result by default.
	return tsaiclient.Result{}, nil
}
