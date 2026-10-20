// Copyright (c) 2026 thorsphere.
// All Rights Reserved. Use is governed by the Functional Source License v1.1
// (FSL-1.1-ALv2) that can be found in the LICENSE file.

package tsaiclient

// OpenRouterAPIKeyEnv is the environment variable name for the OpenRouter API key.
const OpenRouterAPIKeyEnv = openRouterAPIKeyEnv

var (
	ValidateEndpointURL = validateEndpointURL
)

// GetSessionID returns the session ID of the request.
func (req *Request) GetSessionID() string {
	// Set the session ID when the request is non-nil.
	if req != nil {
		return req.sessionID
	}
	// Return an empty string when the request is nil.
	return ""
}
