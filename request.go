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

// maxSessionIDLen is the maximum length in bytes of a request session ID.
const maxSessionIDLen = 256

// Request represents a provider-agnostic completion request.
// The zero value is not usable; construct instances with NewRequest.
type Request struct {
	systemPrompt string // Optional system instruction guiding the model's behavior. An empty or whitespace-only string means no system prompt.
	prompt       string // The user prompt to complete.
	sessionID    string // Optional sticky routing key for prompt-cache affinity.
}

// NewRequest returns a new Request with the given prompt. It returns an error if prompt is empty or whitespace-only.
// The session ID is generated automatically.
func NewRequest(prompt string) (*Request, error) {
	// Initialize a new request.
	req := &Request{}
	// Set the prompt.
	req.prompt = prompt
	// Generate a new session ID.
	if err := req.NewSessionID(); err != nil {
		// Return an error if generating a new session ID fails.
		return nil, tserr.Op(&tserr.OpArgs{Op: "new session ID", Fn: "request.NewSessionID", Err: err})
	}
	// Validate the request.
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
		req.systemPrompt = systemPrompt
	}
	// Return the updated request.
	return req
}

// WithSessionID sets the session ID and returns req.
// It does not validate; an invalid value fails later in Validate.
func (req *Request) WithSessionID(sessionID string) *Request {
	// Set the session ID when the request is non-nil.
	if req != nil {
		req.sessionID = sessionID
	}
	// Return the updated request.
	return req
}

// NewSessionID generates a new session ID for the request. It returns an error
// if req is nil.
func (req *Request) NewSessionID() error {
	// Return an error if the request is nil.
	if req == nil {
		return tserr.NilPtr()
	}
	// Assign a cryptographically random text identifier.
	req.sessionID = rand.Text()
	// Return no error to indicate success.
	return nil
}

// Validate returns an error if req is nil, has an empty or whitespace-only
// prompt, or has a session ID that exceeds 256 bytes or contains non-printable
// ASCII or whitespace characters.
func (req *Request) Validate() error {
	// Return an error if the request is nil.
	if req == nil {
		return tserr.NilPtr()
	}
	// Session ID must be less than or equal to 256 bytes, otherwise return an error.
	if len(req.sessionID) > maxSessionIDLen {
		return tserr.Lower(&tserr.LowerArgs{
			Var:         "session ID length",
			HigherBound: maxSessionIDLen,
			Actual:      int64(len(req.sessionID)),
		})
	}
	// Return an error if a non-empty session ID contains non-printable ASCII or whitespace.
	if req.sessionID != "" && !isValidSessionID(req.sessionID) {
		return tserr.InvalidFormat(&tserr.InvalidFormatArgs{
			F:      "session ID",
			Value:  req.sessionID,
			Detail: "expected printable ASCII characters without whitespace",
		})
	}
	// If the prompt is empty, return an error.
	if strings.TrimSpace(req.prompt) == "" {
		return tserr.Empty("prompt")
	}
	// Return no error to indicate success.
	return nil
}

// isValidSessionID reports whether s is non-empty and consists exclusively of
// printable ASCII characters excluding whitespace (0x21 through 0x7E).
func isValidSessionID(s string) bool {
	// Inspect each byte in s.
	for i := 0; i < len(s); i++ {
		// Reject whitespace, control characters, and non-ASCII bytes.
		if s[i] <= ' ' || s[i] > '~' {
			return false
		}
	}
	// Return true when all characters are valid.
	return len(s) > 0
}
