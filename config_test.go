// Copyright (c) 2026 thorsphere.
// All Rights Reserved. Use is governed by the Functional Source License v1.1
// (FSL-1.1-ALv2) that can be found in the LICENSE file.

package tsaiclient_test

// Import packages.
import (
	"math"    // Import the math package for floating-point constants.
	"testing" // Import the testing package for subtests.
	"time"    // Import the time package for timeouts.

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
		temperature         float64       // The sampling temperature under test.
		reasoningEffort     string        // The reasoning effort under test.
		timeout             time.Duration // The per-call timeout under test.
		wantErr             bool          // Whether an error is expected.
	}{
		{
			name:                "defaults only",
			model:               "open/for-5",
			maxCompletionTokens: 0,
			temperature:         0,
			reasoningEffort:     "",
			wantErr:             false,
		},
		{
			name:                "valid custom settings",
			model:               "open/for-5",
			maxCompletionTokens: 1024,
			temperature:         0.7,
			reasoningEffort:     "medium",
			wantErr:             false,
		},
		{
			name:                "reasoning effort trimmed of whitespace",
			model:               "open/for-5",
			maxCompletionTokens: 512,
			temperature:         2.0,
			reasoningEffort:     "  high  ",
			wantErr:             false,
		},
		{
			name:                "empty model via WithModel",
			model:               "   ",
			maxCompletionTokens: 100,
			temperature:         1.0,
			reasoningEffort:     "low",
			wantErr:             true,
		},
		{
			name:                "negative max completion tokens",
			model:               "open/for-5",
			maxCompletionTokens: -1,
			temperature:         1.0,
			reasoningEffort:     "low",
			wantErr:             true,
		},
		{
			name:                "invalid reasoning effort",
			model:               "open/for-5",
			maxCompletionTokens: 100,
			temperature:         1.0,
			reasoningEffort:     "ultra",
			wantErr:             true,
		},
		{
			name:                "temperature is NaN",
			model:               "open/for-5",
			maxCompletionTokens: 100,
			temperature:         math.NaN(),
			reasoningEffort:     "none",
			wantErr:             true,
		},
		{
			name:                "temperature below minimum",
			model:               "open/for-5",
			maxCompletionTokens: 100,
			temperature:         -0.1,
			reasoningEffort:     "minimal",
			wantErr:             true,
		},
		{
			name:                "temperature above maximum",
			model:               "open/for-5",
			maxCompletionTokens: 100,
			temperature:         2.1,
			reasoningEffort:     "xhigh",
			wantErr:             true,
		},
		{
			name:                "whitespace-only reasoning effort trims to omitted",
			model:               "open/for-5",
			maxCompletionTokens: 512,
			temperature:         1.0,
			reasoningEffort:     "   ",
			wantErr:             false,
		},
		{
			name:                "negative timeout",
			model:               "open/for-5",
			maxCompletionTokens: 100,
			temperature:         1.0,
			reasoningEffort:     "low",
			timeout:             -1 * time.Second,
			wantErr:             true,
		},
		{
			name:                "zero timeout falls back to default",
			model:               "open/for-5",
			maxCompletionTokens: 100,
			temperature:         1.0,
			reasoningEffort:     "low",
			timeout:             0,
			wantErr:             false,
		},
		{
			name:                "positive timeout",
			model:               "open/for-5",
			maxCompletionTokens: 100,
			temperature:         1.0,
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
				t.Fatal(err)
			}
			// Apply all builder methods under test.
			cfg.WithModel(tt.model).
				WithMaxCompletionTokens(tt.maxCompletionTokens).
				WithTemperature(tt.temperature).
				WithReasoningEffort(tt.reasoningEffort).
				WithTimeout(tt.timeout)
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
		})
	}
}

// TestConfigWithBuildersNil tests that WithModel, WithMaxCompletionTokens,
// WithTemperature, WithReasoningEffort, and WithTimeout safely handle a nil
// Config receiver.
func TestConfigWithBuildersNil(t *testing.T) {
    // Run test in parallel.
    t.Parallel()
    // Declare a nil configuration.
    var cfg *tsaiclient.Config
    // Verify chained builder calls return nil without panicking.
    if got := cfg.WithModel("open/for-5").
        WithMaxCompletionTokens(100).
        WithTemperature(0.5).
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
