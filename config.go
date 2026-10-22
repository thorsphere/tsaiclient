// Copyright (c) 2026 thorsphere.
// All Rights Reserved. Use is governed by the Functional Source License v1.1
// (FSL-1.1-ALv2) that can be found in the LICENSE file.

package tsaiclient

// Import packages.
import (
	"fmt"     // Import the fmt package for formatting.
	"math"    // Import the math package for floating-point checks.
	"slices"  // Import the slices package for slice membership checks.
	"strings" // Import the strings package for string manipulation.

	"github.com/thorsphere/tserr" // Import the tserr package for error handling.
)

// Configuration and response validation bounds.
const (
	minTemperature   = 0.0         // The minimum allowed sampling temperature.
	maxTemperature   = 2.0         // The maximum allowed sampling temperature.
	maxResponseBytes = 1024 * 1024 // The maximum number of bytes to read from a response body (1 MiB).
)

// reasoningEffortList is a list of valid reasoning effort levels.
var reasoningEffortList = []string{"none", "minimal", "low", "medium", "high", "xhigh"}

// Config carries the settings for an AI completion client.
// The zero value is not usable; construct instances with NewConfig.
type Config struct {
	model               string  // The model identifier passed to the provider.
	maxCompletionTokens int     // The default cap on generated tokens per call. Zero means no cap.
	temperature         float64 // The default sampling temperature.
	reasoningEffort     string  // Optional reasoning effort level (e.g. "low", "medium", "high"). Empty means omitted so the model default applies.
}

// NewConfig returns a new Config for model with default settings. It returns
// an error if model is empty or whitespace-only.
func NewConfig(model string) (*Config, error) {
	// Initialize a new configuration.
	cfg := &Config{}
	// Set the model identifier.
	cfg.WithModel(model)
	// Validate the configuration.
	if err := cfg.Validate(); err != nil {
		// Return an error if validating the configuration fails.
		return nil, err
	}
	// Return the initialized configuration.
	return cfg, nil
}

// WithModel sets the model identifier and returns cfg. Parameter model is trimmed of leading and trailing whitespace.
func (cfg *Config) WithModel(model string) *Config {
	// Set the model identifier if cfg is not nil.
	if cfg != nil {
		cfg.model = strings.TrimSpace(model)
	}
	// Return the configuration.
	return cfg
}

// WithMaxCompletionTokens sets the generation token cap and returns cfg.
func (cfg *Config) WithMaxCompletionTokens(maxCompletionTokens int) *Config {
	// Set the generation token cap if cfg is not nil.
	if cfg != nil {
		cfg.maxCompletionTokens = maxCompletionTokens
	}
	// Return the configuration.
	return cfg
}

// WithTemperature sets the sampling temperature and returns cfg.
func (cfg *Config) WithTemperature(temperature float64) *Config {
	// Set the sampling temperature if cfg is not nil.
	if cfg != nil {
		cfg.temperature = temperature
	}
	// Return the configuration.
	return cfg
}

// WithReasoningEffort sets the reasoning effort level and returns cfg. Parameter effort is trimmed of leading and trailing whitespace.
func (cfg *Config) WithReasoningEffort(effort string) *Config {
	// Set the reasoning effort level if cfg is not nil.
	if cfg != nil {
		cfg.reasoningEffort = strings.TrimSpace(effort)
	}
	// Return the configuration.
	return cfg
}

// Validate returns an error if cfg is nil, has an empty model, a negative
// token cap, an unsupported reasoning effort, or a temperature outside [0, 2].
func (cfg *Config) Validate() error {
	// Return an error if the configuration is nil.
	if cfg == nil {
		return tserr.Empty("config")
	}
	// model must be non-empty, otherwise return an error.
	if strings.TrimSpace(cfg.model) == "" {
		return tserr.Empty("model")
	}
	// max completion tokens must be greater than or equal to 0, otherwise return an error.
	if cfg.maxCompletionTokens < 0 {
		return tserr.Higher(&tserr.HigherArgs{
			Var:        "maximum completion tokens",
			LowerBound: 0,
			Actual:     int64(cfg.maxCompletionTokens)})
	}
	// When reasoning effort is set, it must be one of the allowed values.
	if cfg.reasoningEffort != "" && !slices.Contains(reasoningEffortList, cfg.reasoningEffort) {
		return tserr.InvalidFormat(&tserr.InvalidFormatArgs{
			F:      "reasoning effort",
			Value:  cfg.reasoningEffort,
			Detail: "expected one of: " + strings.Join(reasoningEffortList, ", "),
		})
	}
	// Temperature must be a number, otherwise return an error.
	if math.IsNaN(cfg.temperature) {
		return tserr.InvalidFormat(&tserr.InvalidFormatArgs{
			F:      "configuration temperature",
			Value:  fmt.Sprintf("%v", math.NaN()),
			Detail: "expected a number between 0 and 2",
		})
	}
	// Temperature must be at least minTemperature, otherwise return an error.
	if cfg.temperature < minTemperature {
		return tserr.Higherf(&tserr.HigherfArgs{Var: "temperature", LowerBound: minTemperature, Actual: cfg.temperature})
	}
	// Temperature must be at most maxTemperature, otherwise return an error.
	if cfg.temperature > maxTemperature {
		return tserr.Lowerf(&tserr.LowerfArgs{Var: "temperature", HigherBound: maxTemperature, Actual: cfg.temperature})
	}
	// Return no error to indicate success.
	return nil
}
