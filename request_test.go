// Copyright (c) 2026 thorsphere.
// All Rights Reserved. Use is governed by the Functional Source License v1.1
// (FSL-1.1-ALv2) that can be found in the LICENSE file.

package tsaiclient_test

// Import packages.
import (
	"fmt"     // Import the fmt package for formatting.
	"strings" // Import the strings package for string operations.
	"testing" // Import the testing package for subtests.

	"github.com/thorsphere/tsaiclient" // Import the tsaiclient package for testing.
	"github.com/thorsphere/tserr"      // Import the tserr package for error handling.
)

// TestNewRequest tests Request construction via NewRequest.
func TestNewRequest(t *testing.T) {
	// Run subtests in parallel.
	t.Parallel()
	// Define test cases.
	tests := []struct {
		name    string // The name of the test case.
		prompt  string // The prompt under test.
		wantErr bool   // Whether an error is expected.
	}{
		{
			name:    "valid prompt",
			prompt:  "Hello",
			wantErr: false,
		},
		{
			name:    "empty prompt",
			prompt:  "",
			wantErr: true,
		},
		{
			name:    "whitespace-only prompt",
			prompt:  "   \t\n  ",
			wantErr: true,
		},
	}
	// Iterate over test cases.
	for _, tt := range tests {
		// Run each test case as a subtest.
		t.Run(tt.name, func(t *testing.T) {
			// Run subtest in parallel.
			t.Parallel()
			// Create a new request and check whether the error matches expectations.
			_, err := tsaiclient.NewRequest(tt.prompt)
			if (err != nil) != tt.wantErr {
				// Fail the test if the error presence does not match wantErr.
				t.Fatal(tserr.EqualStr(&tserr.EqualStrArgs{
					Var:    "error",
					Actual: fmt.Sprintf("%v", err),
					Want:   fmt.Sprintf("wantErr=%v", tt.wantErr),
				}))
			}
		})
	}
}

// TestRequestValidate tests the validation logic of Request.Validate.
func TestRequestValidate(t *testing.T) {
	// Run subtests in parallel.
	t.Parallel()
	// Define test cases.
	tests := []struct {
		name         string // The name of the test case.
		prompt       string // The prompt under test.
		systemPrompt string // The system prompt under test.
		sessionID    string // The session ID under test.
		wantErr      bool   // Whether an error is expected.
	}{
		{
			name:         "valid request",
			prompt:       "Hello",
			systemPrompt: "Be concise.",
			sessionID:    "session-123",
			wantErr:      false,
		},
		{
			name:      "valid request with session ID at max length (256)",
			prompt:    "Hello",
			sessionID: strings.Repeat("a", 256),
			wantErr:   false,
		},
		{
			name:      "session ID exceeds max length (257)",
			prompt:    "Hello",
			sessionID: strings.Repeat("a", 257),
			wantErr:   true,
		},
	}
	// Iterate over test cases.
	for _, tt := range tests {
		// Run each test case as a subtest.
		t.Run(tt.name, func(t *testing.T) {
			// Run subtest in parallel.
			t.Parallel()
			// Create a new request.
			req, err := tsaiclient.NewRequest(tt.prompt)
			// Fail the test if an error is returned.
			if err != nil {
				t.Fatal(err)
			}
			// Set the system prompt and session ID.
			req.WithSystemPrompt(tt.systemPrompt).WithSessionID(tt.sessionID)
			// Validate the request and check whether the error matches expectations.
			if err := req.Validate(); (err != nil) != tt.wantErr {
				// Fail the test if the error presence does not match wantErr.
				t.Fatal(tserr.EqualStr(&tserr.EqualStrArgs{
					Var:    "error",
					Actual: fmt.Sprintf("%v", err),
					Want:   fmt.Sprintf("wantErr=%v", tt.wantErr),
				}))
			}
		})
	}
}

// TestRequestValidateNil tests the validation logic of Request.Validate when the request is nil.
func TestRequestValidateNil(t *testing.T) {
	// Create a nil request.
	var req *tsaiclient.Request
	// Validate the request and check whether an error is returned.
	if err := req.Validate(); err == nil {
		// Fail the test if the error is nil.
		t.Fatal(tserr.NilFailed("Validate"))
	}
}

// TestRequestValidateEmpty tests the validation logic of Request.Validate when the request is empty.
func TestRequestValidateEmpty(t *testing.T) {
	// Create an empty request.
	req := &tsaiclient.Request{}
	// Validate the request and check whether an error is returned.
	if err := req.Validate(); err == nil {
		// Fail the test if the error is nil.
		t.Fatal(tserr.NilFailed("Validate"))
	}
}

// TestRequestNewSessionIDNil tests the session ID generation logic of Request.NewSessionID
// when the request is nil.
func TestRequestNewSessionIDNil(t *testing.T) {
	// Create a nil request.
	var req *tsaiclient.Request
	// Generate a new session ID and check whether an error is returned.
	if err := req.NewSessionID(); err == nil {
		// Fail the test if the error is nil.
		t.Fatal(tserr.NilFailed("NewSessionID"))
	}
}

// TestRequestNewSessionID tests the session ID generation logic of Request.NewSessionID.
func TestRequestNewSessionID(t *testing.T) {
	// Run subtests in parallel.
	t.Parallel()
	// Define test cases.
	tests := []struct {
		name      string // The name of the test case.
		prompt    string // The prompt under test.
		sessionID string // The session ID under test.
	}{
		{
			name:   "empty session ID",
			prompt: "Hello",
		},
		{
			name:      "overwrites existing session ID",
			prompt:    "Hello",
			sessionID: "initial-session-id",
		},
	}
	// Iterate over test cases.
	for _, tt := range tests {
		// Run each test case as a subtest.
		t.Run(tt.name, func(t *testing.T) {
			// Run subtest in parallel.
			t.Parallel()
			// Create a new request.
			req, err := tsaiclient.NewRequest(tt.prompt)
			// Fail the test if an error is returned.
			if err != nil {
				t.Fatal(err)
			}
			// Set the session ID.
			req.WithSessionID(tt.sessionID)
			// Record the previous session ID when the request is non-nil.
			var prevID string
			if req != nil {
				prevID = req.GetSessionID()
			}
			// Generate a new session ID and check whether the error matches expectations.
			if err := req.NewSessionID(); err != nil {
				// Fail the test if the error presence does not match wantErr.
				t.Fatal(tserr.NilExpected(&tserr.NilExpectedArgs{Op: "NewSessionID", Err: err}))
			}
			// Fail the test if the generated session ID is empty.
			if req.GetSessionID() == "" {
				t.Fatal(tserr.Empty("session ID"))
			}
			// Fail the test if the generated session ID did not replace the previous value.
			if req.GetSessionID() == prevID {
				t.Fatal(tserr.EqualStr(&tserr.EqualStrArgs{
					Var:    "session ID",
					Actual: req.GetSessionID(),
					Want:   "distinct from " + prevID,
				}))
			}
		})
	}
}
