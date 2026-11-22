// Copyright (c) 2026 thorsphere.
// All Rights Reserved. Use is governed by the Functional Source License v1.1
// (FSL-1.1-ALv2) that can be found in the LICENSE file.

package tsaiclient_test

// Import packages.
import (
	"math"    // Import the math package for floating-point constants.
	"strings" // Import the strings package for error detail checks.
	"testing" // Import the testing package for subtests.
	"time"    // Import the time package for timeouts.

	"github.com/thorsphere/lpstats"    // Import the lpstats package for floating-point pointers.
	"github.com/thorsphere/tsaiclient" // Import the tsaiclient package for testing.
	"github.com/thorsphere/tserr"      // Import the tserr package for error handling.
)

// TestNewConfig tests Config construction via NewConfig.
func TestNewConfig(t *testing.T) {
	// Run subtests in parallel.
	t.Parallel()
	// Define test cases.
	tests := []struct {
		name    string // The name of the test case.
		model   string // The model identifier under test.
		wantErr bool   // Whether an error is expected.
	}{
		{
			name:    "valid model",
			model:   "open/for-5",
			wantErr: false,
		},
		{
			name:    "valid model with surrounding whitespace",
			model:   "  open/for-5  ",
			wantErr: false,
		},
		{
			name:    "empty model",
			model:   "",
			wantErr: true,
		},
		{
			name:    "whitespace-only model",
			model:   "   \t\n  ",
			wantErr: true,
		},
	}
	// Iterate over test cases.
	for _, tt := range tests {
		// Run each test case as a subtest.
		t.Run(tt.name, func(t *testing.T) {
			// Run subtest in parallel.
			t.Parallel()
			// Create a new configuration and check whether the error matches expectations.
			cfg, err := tsaiclient.NewConfig(tt.model)
			// Check whether an expected error was missing.
			if tt.wantErr && err == nil {
				t.Fatal(tserr.NilFailed("NewConfig"))
			}
			// Check whether an unexpected error was returned.
			if !tt.wantErr && err != nil {
				t.Fatal(tserr.NilExpected(&tserr.NilExpectedArgs{Op: "NewConfig", Err: err}))
			}
			// Verify the returned configuration is non-nil when no error is expected.
			if !tt.wantErr && cfg == nil {
				// Fail the test if the configuration is nil.
				t.Fatal(tserr.NilFailed("NewConfig"))
			}
			// Verify the returned configuration is nil when an error is expected.
			if tt.wantErr && cfg != nil {
				// Fail the test if a non-nil configuration was returned on error.
				t.Fatal(tserr.NilExpected(&tserr.NilExpectedArgs{Op: "NewConfig"}))
			}
		})
	}
}

// TestConfigValidate tests the builder methods and validation logic of Config.Validate.
func TestConfigValidate(t *testing.T) {
	// Run subtests in parallel.
	t.Parallel()
	// Define test cases.
	tests := []struct {
		name                string        // The name of the test case.
		model               string        // The model identifier under test.
		maxCompletionTokens int           // The max completion tokens under test.
		temperature         *float64      // The optional sampling temperature under test; nil leaves it unset.
		clearTemperature    bool          // Whether to clear the temperature after setting it.
		reasoningEffort     string        // The reasoning effort under test.
		timeout             time.Duration // The per-call timeout under test.
		wantErr             bool          // Whether an error is expected.
		wantErrSub          string        // The expected error detail when validation fails.
	}{
		{
			name:                "defaults with temperature unset",
			model:               "open/for-5",
			maxCompletionTokens: 0,
			temperature:         nil,
			reasoningEffort:     "",
			wantErr:             false,
		},
		{
			name:        "explicit positive zero",
			model:       "open/for-5",
			temperature: lpstats.PtrFloat(0.0),
		},
		{
			name:        "explicit negative zero",
			model:       "open/for-5",
			temperature: lpstats.PtrFloat(math.Copysign(0, -1)),
		},
		{
			name:                "valid custom settings",
			model:               "open/for-5",
			maxCompletionTokens: 1024,
			temperature:         lpstats.PtrFloat(0.7),
			reasoningEffort:     "medium",
			wantErr:             false,
		},
		{
			name:                "reasoning effort trimmed of whitespace",
			model:               "open/for-5",
			maxCompletionTokens: 512,
			temperature:         lpstats.PtrFloat(2.0),
			reasoningEffort:     "  high  ",
			wantErr:             false,
		},
		{
			name:             "clear valid temperature",
			model:            "open/for-5",
			temperature:      lpstats.PtrFloat(0.7),
			clearTemperature: true,
		},
		{
			name:             "clear invalid temperature",
			model:            "open/for-5",
			temperature:      lpstats.PtrFloat(2.1),
			clearTemperature: true,
		},
		{
			name:                "empty model via WithModel",
			model:               "   ",
			maxCompletionTokens: 100,
			temperature:         lpstats.PtrFloat(1.0),
			reasoningEffort:     "low",
			wantErr:             true,
		},
		{
			name:                "negative max completion tokens",
			model:               "open/for-5",
			maxCompletionTokens: -1,
			temperature:         lpstats.PtrFloat(1.0),
			reasoningEffort:     "low",
			wantErr:             true,
		},
		{
			name:                "invalid reasoning effort",
			model:               "open/for-5",
			maxCompletionTokens: 100,
			temperature:         lpstats.PtrFloat(1.0),
			reasoningEffort:     "ultra",
			wantErr:             true,
		},
		{
			name:                "temperature is NaN",
			model:               "open/for-5",
			maxCompletionTokens: 100,
			temperature:         lpstats.PtrFloat(math.NaN()),
			reasoningEffort:     "none",
			wantErr:             true,
			wantErrSub:          "configuration temperature",
		},
		{
			name:        "temperature is positive infinity",
			model:       "open/for-5",
			temperature: lpstats.PtrFloat(math.Inf(1)),
			wantErr:     true,
			wantErrSub:  "temperature",
		},
		{
			name:        "temperature is negative infinity",
			model:       "open/for-5",
			temperature: lpstats.PtrFloat(math.Inf(-1)),
			wantErr:     true,
			wantErrSub:  "temperature",
		},
		{
			name:                "temperature below minimum",
			model:               "open/for-5",
			maxCompletionTokens: 100,
			temperature:         lpstats.PtrFloat(-0.1),
			reasoningEffort:     "minimal",
			wantErr:             true,
			wantErrSub:          "temperature",
		},
		{
			name:                "temperature above maximum",
			model:               "open/for-5",
			maxCompletionTokens: 100,
			temperature:         lpstats.PtrFloat(2.1),
			reasoningEffort:     "xhigh",
			wantErr:             true,
			wantErrSub:          "temperature",
		},
		{
			name:                "whitespace-only reasoning effort trims to omitted",
			model:               "open/for-5",
			maxCompletionTokens: 512,
			temperature:         lpstats.PtrFloat(1.0),
			reasoningEffort:     "   ",
			wantErr:             false,
		},
		{
			name:                "negative timeout",
			model:               "open/for-5",
			maxCompletionTokens: 100,
			temperature:         lpstats.PtrFloat(1.0),
			reasoningEffort:     "low",
			timeout:             -1 * time.Second,
			wantErr:             true,
		},
		{
			name:                "zero timeout falls back to default",
			model:               "open/for-5",
			maxCompletionTokens: 100,
			temperature:         lpstats.PtrFloat(1.0),
			reasoningEffort:     "low",
			timeout:             0,
			wantErr:             false,
		},
		{
			name:                "positive timeout",
			model:               "open/for-5",
			maxCompletionTokens: 100,
			temperature:         lpstats.PtrFloat(1.0),
			reasoningEffort:     "low",
			timeout:             5 * time.Minute,
			wantErr:             false,
		},
	}
	// Iterate over test cases.
	for _, tt := range tests {
		// Run each test case as a subtest.
		t.Run(tt.name, func(t *testing.T) {
			// Run subtest in parallel.
			t.Parallel()
			// Create a valid base configuration.
			cfg, err := tsaiclient.NewConfig("open/for-5")
			// Fail the test if creating the base configuration fails.
			if err != nil {
				// Report the unexpected configuration construction error.
				t.Fatal(tserr.NilExpected(&tserr.NilExpectedArgs{Op: "NewConfig", Err: err}))
			}
			// Apply the non-temperature builder methods under test.
			cfg.WithModel(tt.model).
				WithMaxCompletionTokens(tt.maxCompletionTokens).
				WithReasoningEffort(tt.reasoningEffort).
				WithTimeout(tt.timeout)
			// Set the sampling temperature only when the test case configures one.
			if tt.temperature != nil {
				// Apply the configured sampling temperature.
				cfg.WithTemperature(*tt.temperature)
			}
			// Clear the sampling temperature when requested by the test case.
			if tt.clearTemperature {
				// Verify clearing returns the same configuration for chaining.
				if got := cfg.WithoutTemperature(); got != cfg {
					// Report the unexpected builder result.
					t.Errorf("WithoutTemperature returned %p, want %p", got, cfg)
				}
			}
			// Validate the configuration.
			err = cfg.Validate()
			// Check whether an expected error was missing.
			if tt.wantErr && err == nil {
				t.Fatal(tserr.NilFailed("Validate"))
			}
			// Check whether an unexpected error was returned.
			if !tt.wantErr && err != nil {
				t.Fatal(tserr.NilExpected(&tserr.NilExpectedArgs{Op: "Validate", Err: err}))
			}
			// Verify temperature failures identify the invalid setting.
			if tt.wantErrSub != "" && err != nil && !strings.Contains(err.Error(), tt.wantErrSub) {
				// Report the unexpected error detail.
				t.Fatal(tserr.EqualStr(&tserr.EqualStrArgs{
					Var:    "Validate error detail",
					Actual: err.Error(),
					Want:   tt.wantErrSub,
				}))
			}
		})
	}
}

// TestConfigWithBuildersNil tests that WithModel, WithMaxCompletionTokens,
// WithTemperature, WithoutTemperature, WithReasoningEffort, and WithTimeout
// safely handle a nil Config receiver.
func TestConfigWithBuildersNil(t *testing.T) {
	// Run test in parallel.
	t.Parallel()
	// Declare a nil configuration.
	var cfg *tsaiclient.Config
	// Verify chained builder calls return nil without panicking.
	if got := cfg.WithModel("open/for-5").
		WithMaxCompletionTokens(100).
		WithTemperature(0.5).
		WithoutTemperature().
		WithReasoningEffort("low").
		WithTimeout(time.Minute); got != nil {
		// Fail the test if a non-nil configuration was returned.
		t.Fatal(tserr.NilExpected(&tserr.NilExpectedArgs{Op: "Config.With*"}))
	}
}

// TestConfigValidateNil tests the validation logic of Config.Validate when the configuration is nil.
func TestConfigValidateNil(t *testing.T) {
	// Run test in parallel.
	t.Parallel()
	// Declare a nil configuration.
	var cfg *tsaiclient.Config
	// Validate the configuration and check whether an error is returned.
	if err := cfg.Validate(); err == nil {
		// Fail the test if the error is nil.
		t.Fatal(tserr.NilFailed("Validate"))
	}
}
