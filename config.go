// Copyright (c) 2026 thorsphere.
// All Rights Reserved. Use is governed by the Functional Source License v1.1
// (FSL-1.1-ALv2) that can be found in the LICENSE file.

package tsaiclient

// Import packages.
import (
	"strings" // Import the strings package for string manipulation.

	"github.com/thorsphere/tserr" // Import the tserr package for error handling.
)

// reasoningEffortList is a list of valid reasoning effort levels.
var reasoningEffortList = []string{"none", "minimal", "low", "medium", "high", "xhigh"}

// Defaults for provider request settings.
const (
	defaultModel               = "tencent/hy4-preview" // The default model identifier.
	defaultMaxCompletionTokens = 0                     // The default cap on generated tokens. Zero means no cap.
	defaultTemperature         = 1.0                   // The default sampling temperature.
	defaultReasoningEffort     = "medium"              // The default reasoning effort level.
)

// finishReasonLength is the provider finish reason indicating the generated
// message was truncated by the token limit.
const finishReasonLength = "length"

// finishReasonContentFilter is the provider finish reason indicating the
// response was blocked by the provider's content filter.
const finishReasonContentFilter = "content_filter"

// finishReasonError is the provider finish reason indicating generation
// failed on the provider side.
const finishReasonError = "error"

// statusEdgeTimeout is the non-standard status code edge networks (e.g.
// Cloudflare) return when the upstream provider timed out.
const statusEdgeTimeout = 524

// statusProviderOverloaded is the non-standard status code the provider
// returns when its upstream is temporarily overloaded.
const statusProviderOverloaded = 529

// maxResponseBytes is the maximum number of bytes to read from the response body.
const maxResponseBytes = 1024 * 1024 // 1 MiB

// Config carries the settings for an AI completion client.
// The zero value is not usable.
type Config struct {
	model               string  // The model identifier passed to the provider.
	maxCompletionTokens int     // The default cap on generated tokens per call. Zero means no cap.
	temperature         float64 // The default sampling temperature.
	reasoningEffort     string  // The default reasoning effort level (e.g. "low", "medium", "high").
}

// withModel sets the model identifier and returns cfg.
func (cfg *Config) withModel(model string) *Config {
	cfg.model = model
	return cfg
}

// withMaxCompletionTokens sets the generation token cap and returns cfg.
func (cfg *Config) withMaxCompletionTokens(maxCompletionTokens int) *Config {
	cfg.maxCompletionTokens = maxCompletionTokens
	return cfg
}

// withTemperature sets the sampling temperature and returns cfg.
func (cfg *Config) withTemperature(temperature float64) *Config {
	cfg.temperature = temperature
	return cfg
}

// WithReasoningEffort sets the reasoning effort level and returns cfg.
func (cfg *Config) WithReasoningEffort(effort string) *Config {
	cfg.reasoningEffort = effort
	return cfg
}

// Validate validates the configuration. Returns an error if the configuration
// is invalid, or nil if it is valid.
func (cfg *Config) Validate() error {
	// Return an error if the configuration is nil.
	if cfg == nil {
		return tserr.Empty("config")
	}
	// model must be non-empty, otherwise return an error.
	if cfg.model == "" {
		return tserr.Empty("model")
	}
	// max completion tokens must be greater than or equal to 0, otherwise return an error.
	if cfg.maxCompletionTokens < 0 {
		return tserr.Higher(&tserr.HigherArgs{
			Var:        "maximum completion tokens",
			LowerBound: 0,
			Actual:     int64(cfg.maxCompletionTokens)})
	}
	// "reasoning effort" must be one of the allowed values, otherwise return an error.
	found := false
	for _, effort := range reasoningEffortList {
		if cfg.reasoningEffort == effort {
			found = true
			break
		}
	}
	// If reasoning effort is not one of the allowed values, return an error.
	if !found {
		return tserr.InvalidFormat(&tserr.InvalidFormatArgs{
			F:      "reasoning effort",
			Value:  cfg.reasoningEffort,
			Detail: "expected one of: " + strings.Join(reasoningEffortList, ", ")})
	}
	// Temperature must be between 0 and 2, otherwise return an error.
	if cfg.temperature < 0 {
		return tserr.Higherf(&tserr.HigherfArgs{Var: "temperature", LowerBound: 0, Actual: cfg.temperature})
	}
	// Temperature must be below 2, otherwise return an error.
	if cfg.temperature > 2 {
		return tserr.Lowerf(&tserr.LowerfArgs{Var: "temperature", HigherBound: 2, Actual: cfg.temperature})
	}
	// Return no error to indicate success.
	return nil
}
