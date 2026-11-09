// Copyright (c) 2026 thorsphere.
// All Rights Reserved. Use is governed by the Functional Source License v1.1
// (FSL-1.1-ALv2) that can be found in the LICENSE file.

package tsaiclient

// Import packages.
import (
	"context"       // Import the context package for request cancellation.
	"encoding/json" // Import the json package for decoding envelope JSON.
	"strings"       // Import the strings package for code fence stripping.

	"github.com/thorsphere/tserr" // Import the tserr package for error handling.
)

// Envelope is the standard response contract for structured completions: the
// model delivers its findings first, then reports its confidence. The zero
// value is an envelope with no findings and zero confidence.
type Envelope struct {
	Findings   []string `json:"findings"`   // The deliverables produced for the prompt.
	Confidence float64  `json:"confidence"` // The model's self-reported confidence; the scale is defined by the consumer's prompt contract.
}

// JSONResult holds a decoded envelope alongside the underlying completion
// metadata. The zero value means no completion was decoded.
type JSONResult struct {
	Envelope  Envelope // The decoded structured response.
	Truncated bool     // True when generation hit the token limit (the JSON may be incomplete).
	Usage     Usage    // Token counts and cost for logging/metering.
}

// CompleteJSON returns the completion of req with its response content
// decoded into the standard structured Envelope via json.Unmarshal. The model
// is expected to reply with a JSON object carrying the keys "findings" (an
// array of strings) and "confidence" (a number); extra keys are ignored and
// omitted keys keep their zero values. A wrapping markdown code fence of three
// backticks (with an optional language tag such as json) is tolerated and
// stripped before decoding. The prompt must state this contract explicitly
// because the library cannot enforce it at generation time; the package
// README shows a prompt template for it. CompleteJSON performs no semantic
// validation of the decoded envelope; the meaning and scale of Findings and
// Confidence are defined by the consumer's prompt contract. It returns an
// error if the receiver, ctx, or req is nil or invalid, if the completion
// fails, or if the response content does not decode as an envelope. The
// returned JSONResult carries the completion metadata in both the success
// and the error case.
func (c *Client) CompleteJSON(ctx context.Context, req *Request) (JSONResult, error) {
	// Complete the request to obtain the normalized completion result.
	res, err := c.Complete(ctx, req)
	// Preserve the completion metadata for the returned result.
	jsonRes := JSONResult{Truncated: res.Truncated, Usage: res.Usage}
	// Return the metadata and the error when the completion call failed.
	if err != nil {
		return jsonRes, err
	}
	// Strip a wrapping markdown code fence when the model added one.
	payload := stripCodeFences(res.Content)
	// Decode the payload into the standard envelope.
	var env Envelope
	// Return the metadata and an error when the payload does not decode.
	if err := json.Unmarshal([]byte(payload), &env); err != nil {
		return jsonRes, tserr.Op(&tserr.OpArgs{Op: "decode envelope", Fn: c.provider.Name(), Err: err})
	}
	// Attach the decoded envelope to the result.
	jsonRes.Envelope = env
	// Return the decoded result with no error to indicate success.
	return jsonRes, nil
}

// stripCodeFences returns content with surrounding whitespace and any
// wrapping markdown code fence removed. When content opens with a fence line
// of three backticks, optionally followed by a language tag such as json, the
// payload up to a trailing closing fence is returned; content without an
// opening fence is returned trimmed.
func stripCodeFences(content string) string {
	// Trim surrounding whitespace so a leading newline cannot hide the fence.
	content = strings.TrimSpace(content)
	// Return the trimmed content when it does not open with a fence line.
	if !strings.HasPrefix(content, "```") {
		return content
	}
	// Find the end of the opening fence line.
	lineEnd := strings.IndexByte(content, '\n')
	// Return an empty payload when the content carries only fence syntax.
	if lineEnd == -1 {
		return ""
	}
	// Keep the payload after the opening fence line.
	rest := content[lineEnd+1:]
	// Remove the closing fence when one terminates the content.
	rest = strings.TrimSuffix(rest, "```")
	// Return the trimmed unfenced payload.
	return strings.TrimSpace(rest)
}
