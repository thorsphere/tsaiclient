// Copyright (c) 2026 thorsphere.
// All Rights Reserved. Use is governed by the Functional Source License v1.1
// (FSL-1.1-ALv2) that can be found in the LICENSE file.

package tsaiclient

// Import packages.
import (
	"crypto/rand" // Import the rand package for random number generation.
	"strings"     // Import the strings package for string operations.

	"github.com/thorsphere/tserr" // Import the tserr package for error handling.
)

// Request represents a provider-agnostic completion request.
type Request struct {
	SystemPrompt string // Optional system instruction guiding the model's behavior. An empty string means no system prompt.
	Prompt       string // The user prompt to complete.
	SessionID    string // Optional sticky routing key for prompt-cache affinity.
}

// NewRequest returns a new Request with the given prompt. It returns an error
// if the prompt is empty. The session ID is generated automatically.
func NewRequest(prompt string) (*Request, error) {
	// Initialize a new request.
	req := &Request{}
	// Set the prompt.
	req.Prompt = prompt
	// Generate a new session ID if one is not provided.
	if err := req.NewSessionID(); err != nil {
		// Return an error if generating a new session ID fails.
		return nil, tserr.Op(&tserr.OpArgs{Op: "new session ID", Fn: "request.NewSessionID", Err: err})
	}
	// Return an error if validating the request fails.
	if err := req.Validate(); err != nil {
		// Return an error if validating the request fails.
		return nil, err
	}
	// Return the initialized request.
	return req, nil
}

// WithSystemPrompt sets the system instruction and returns req.
func (req *Request) WithSystemPrompt(systemPrompt string) *Request {
	// Set the system prompt when the request is non-nil.
	if req != nil {
		req.SystemPrompt = systemPrompt
	}
	// Return the updated request.
	return req
}

// NewSessionID generates a new session ID for the request. It returns an error
// if the request is nil.
func (req *Request) NewSessionID() error {
	// Return an error if the request is nil.
	if req == nil {
		return tserr.Empty("request")
	}
	// Return a cryptographically random text identifier.
	req.SessionID = rand.Text()
	return nil
}

// Validate validates the request. Returns an error if the request is invalid, or nil if it is valid.
func (req *Request) Validate() error {
	// Return an error if the request is nil.
	if req == nil {
		return tserr.Empty("request")
	}
	// Session ID must be less than or equal to 256 characters, otherwise return an error.
	if len(req.SessionID) > 256 {
		return tserr.Lower(&tserr.LowerArgs{
			Var:         "session ID",
			HigherBound: 256,
			Actual:      int64(len(req.SessionID))})
	}
	// If the prompt is empty, return an error.
	if strings.TrimSpace(req.Prompt) == "" {
		return tserr.Empty("prompt")
	}
	// Return no error to indicate success.
	return nil
}
