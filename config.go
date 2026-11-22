// Copyright (c) 2026 thorsphere.
// All Rights Reserved. Use is governed by the Functional Source License v1.1
// (FSL-1.1-ALv2) that can be found in the LICENSE file.

package tsaiclient

// Import packages.
import (
	"math"    // Import the math package for floating-point checks.
	"slices"  // Import the slices package for slice membership checks.
	"strings" // Import the strings package for string manipulation.
	"time"    // Import the time package for timeouts.

	"github.com/thorsphere/lpstats" // Import the lpstats package for floating-point pointers.
	"github.com/thorsphere/tserr"   // Import the tserr package for error handling.
)

// Configuration and response validation bounds.
const (
	minTemperature   = 0.0              // The minimum allowed sampling temperature.
	maxTemperature   = 2.0              // The maximum allowed sampling temperature.
	maxResponseBytes = 1024 * 1024      // The maximum number of bytes to read from a response body (1 MiB).
	defaultTimeout   = 60 * time.Second // The default per-call timeout applied when none is configured.
)

// reasoningEffortList is a list of valid reasoning effort levels.
var reasoningEffortList = []string{"none", "minimal", "low", "medium", "high", "xhigh"}

// Config carries the settings for an AI completion client. Temperature is
// optional and omitted when unset so the model or provider default applies.
// The zero value is not usable; construct instances with NewConfig.
type Config struct {
	model               string        // The model identifier passed to the provider.
	maxCompletionTokens int           // The default cap on generated tokens per call. Zero means no cap.
	temperature         *float64      // The optional sampling temperature; nil means omitted.
	reasoningEffort     string        // Optional reasoning effort level (e.g. "low", "medium", "high"). Empty means omitted so the model default applies.
	timeout             time.Duration // The per-call deadline for one completion. Zero means defaultTimeout.
}

// NewConfig returns a new Config for model with default settings, including an
// unset temperature. It returns an error if model is empty or whitespace-only.
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

// WithTemperature sets the sampling temperature to temperature and returns
// cfg. A configured zero is included in requests; an unset temperature is
// omitted instead. Validate checks that temperature is within [0, 2]. It
// safely returns nil for a nil receiver.
func (cfg *Config) WithTemperature(temperature float64) *Config {
	// Set the sampling temperature if cfg is not nil.
	if cfg != nil {
		// Normalize negative zero so an explicitly configured zero is sent as 0.
		if temperature == 0 {
			// Replace either signed zero with positive zero.
			temperature = 0
		}
		// Store the configured temperature independently of the parameter.
		cfg.temperature = lpstats.PtrFloat(temperature)
	}
	// Return the configuration.
	return cfg
}

// WithoutTemperature clears the sampling temperature so it is omitted from
// requests and returns cfg. It safely returns nil for a nil receiver.
func (cfg *Config) WithoutTemperature() *Config {
	// Clear the configured temperature when cfg is not nil.
	if cfg != nil {
		// Allow the model or provider default to apply.
		cfg.temperature = nil
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

// WithTimeout sets the per-call completion deadline and returns cfg.
func (cfg *Config) WithTimeout(timeout time.Duration) *Config {
	// Set the timeout when cfg is non-nil.
	if cfg != nil {
		cfg.timeout = timeout
	}
	// Return the configuration.
	return cfg
}

// Validate returns an error if cfg is nil, has an empty model, a negative
// timeout, a negative token cap, an unsupported reasoning effort, or a
// configured temperature that is NaN or outside [0, 2]. Unset temperature
// is valid and omitted from requests.
func (cfg *Config) Validate() error {
	// Return an error if the configuration is nil.
	if cfg == nil {
		return tserr.Empty("config")
	}
	// model must be non-empty, otherwise return an error.
	if strings.TrimSpace(cfg.model) == "" {
		return tserr.Empty("model")
	}
	// The timeout must be greater than or equal to 0, otherwise return an error.
	if cfg.timeout < 0 {
		return tserr.Higher(&tserr.HigherArgs{
			Var:        "timeout",
			LowerBound: 0,
			Actual:     int64(cfg.timeout)})
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
	// Validate the sampling temperature only when it is configured.
	if cfg.temperature != nil {
		// Temperature must be a number, otherwise return an error.
		if math.IsNaN(*cfg.temperature) {
			return tserr.InvalidFormat(&tserr.InvalidFormatArgs{
				F:      "configuration temperature",
				Value:  lpstats.FmtFloatPtr(cfg.temperature, 0),
				Detail: "expected a number between 0 and 2",
			})
		}
		// Temperature must be at least minTemperature, otherwise return an error.
		if *cfg.temperature < minTemperature {
			return tserr.Higherf(&tserr.HigherfArgs{Var: "temperature", LowerBound: minTemperature, Actual: *cfg.temperature})
		}
		// Temperature must be at most maxTemperature, otherwise return an error.
		if *cfg.temperature > maxTemperature {
			return tserr.Lowerf(&tserr.LowerfArgs{Var: "temperature", HigherBound: maxTemperature, Actual: *cfg.temperature})
		}
	}
	// Return no error to indicate success.
	return nil
}
