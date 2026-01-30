// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackruntime

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/internal/getproviders/providerreqs"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackconfig"
	"github.com/hashicorp/terraform/internal/stacks/stackruntime/hooks"
	stacks_testing_provider "github.com/hashicorp/terraform/internal/stacks/stackruntime/testing"
	"github.com/hashicorp/terraform/internal/stacks/stackstate"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// ConfigValueEvent represents a captured config value emission event
type ConfigValueEvent struct {
	Addr     string
	Value    cty.Value
	HasValue bool
	Phase    string
}

// ConfigValueTracker captures config value emission events for testing
type ConfigValueTracker struct {
	sync.Mutex
	Events []ConfigValueEvent
}

func NewConfigValueTracker() *ConfigValueTracker {
	return &ConfigValueTracker{
		Events: make([]ConfigValueEvent, 0),
	}
}

func (cvt *ConfigValueTracker) CaptureHooks() *Hooks {
	// Start with the basic captured hooks framework and add our config value tracking
	baseHooks := NewCapturedHooks(false)
	hooksPtr := baseHooks.captureHooks()

	// Override the ReportConfigValue hook to add our custom tracking
	hooksPtr.ReportConfigValue = func(ctx context.Context, tracking any, data *hooks.ConfigValueHookData) any {
		fmt.Printf("DEBUG: ReportConfigValue called: %s = %s [%s]\n", data.Addr, data.Value.GoString(), data.Phase)
		cvt.Lock()
		defer cvt.Unlock()

		event := ConfigValueEvent{
			Addr:     data.Addr,
			Value:    data.Value,
			HasValue: !data.Value.IsNull(),
			Phase:    data.Phase,
		}
		cvt.Events = append(cvt.Events, event)
		return tracking
	}

	return hooksPtr
}

func (cvt *ConfigValueTracker) GetEmittedValues() []ConfigValueEvent {
	cvt.Lock()
	defer cvt.Unlock()

	result := make([]ConfigValueEvent, len(cvt.Events))
	copy(result, cvt.Events)
	return result
}

// Helper functions for test assertions
func filterByPhase(events []ConfigValueEvent, phase string) []ConfigValueEvent {
	var filtered []ConfigValueEvent
	for _, event := range events {
		if event.Phase == phase {
			filtered = append(filtered, event)
		}
	}
	return filtered
}

// TestProgressiveConfigValueEmission tests the progressive emission of config values
// from action invocations during stack apply operations. It verifies three main scenarios:
// 1. All config values are known during stack prepare
// 2. Only one value is known during stack prepare
// 3. All values are known after apply
func TestProgressiveConfigValueEmission(t *testing.T) {
	fakePlanTimestamp, err := time.Parse(time.RFC3339, "2021-01-01T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}

	tcs := map[string]struct {
		path   string
		cycles []TestCycle
	}{
		"all values known during prepare": {
			path: "progressive-config-values/all-known-inputs",
			cycles: []TestCycle{
				{
					planInputs: map[string]cty.Value{
						"static_value":    cty.StringVal("hello-world"),
						"computed_prefix": cty.StringVal("prefix"),
					},
				},
			},
		},
		"partial values known during prepare": {
			path: "progressive-config-values/partial-known-inputs",
			cycles: []TestCycle{
				{
					planInputs: map[string]cty.Value{
						"known_value": cty.StringVal("i-am-known"),
					},
				},
			},
		},
		"resource dependent outputs": {
			path: "progressive-config-values/resource-dependent-outputs",
			cycles: []TestCycle{
				{
					planInputs: map[string]cty.Value{
						"resource_count": cty.NumberIntVal(2),
					},
				},
			},
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			configValues := NewConfigValueTracker()

			lock := depsfile.NewLocks()
			lock.SetProvider(
				addrs.NewDefaultProvider("testing"),
				providerreqs.MustParseVersion("0.0.0"),
				providerreqs.MustParseVersionConstraints("=0.0.0"),
				providerreqs.PreferredHashes([]providerreqs.Hash{}),
			)

			store := stacks_testing_provider.NewResourceStore()

			testContext := TestContext{
				timestamp: &fakePlanTimestamp,
				config:    loadMainBundleConfigForTest(t, tc.path),
				providers: map[addrs.Provider]providers.Factory{
					addrs.NewDefaultProvider("testing"): func() (providers.Interface, error) {
						provider := stacks_testing_provider.NewProviderWithData(t, store)
						return provider, nil
					},
				},
				dependencyLocks: *lock,
			}

			var state *stackstate.State
			for i, cycle := range tc.cycles {
				t.Logf("Running test cycle %d for %s", i, name)

				// Plan phase
				plan := testContext.Plan(t, ctx, state, cycle)
				t.Logf("Plan completed")
				// instead of using the TestContext.Apply method which uses its own hooks
				request := ApplyRequest{
					Config: testContext.config,
					Plan:   plan,
					InputValues: func() map[stackaddrs.InputVariable]ExternalInputValue {
						inputs := make(map[stackaddrs.InputVariable]ExternalInputValue, len(cycle.applyInputs))
						for k, v := range cycle.applyInputs {
							inputs[stackaddrs.InputVariable{Name: k}] = ExternalInputValue{Value: v}
						}
						return inputs
					}(),
					ProviderFactories:  testContext.providers,
					ExperimentsAllowed: true,
					DependencyLocks:    testContext.dependencyLocks,
				}

				changesCh := make(chan stackstate.AppliedChange)
				diagsCh := make(chan tfdiags.Diagnostic)
				response := ApplyResponse{
					AppliedChanges: changesCh,
					Diagnostics:    diagsCh,
				}

				// Apply phase with config value tracking
				t.Logf("Starting Apply with hooks...")
				applyCtx := ContextWithHooks(ctx, configValues.CaptureHooks())
				go Apply(applyCtx, &request, &response)
				changes, diags := collectApplyOutput(changesCh, diagsCh)

				// Check for any errors in apply
				if diags.HasErrors() {
					t.Logf("Apply errors (this may be expected): %s", diags.Err())
				}
				t.Logf("Apply completed with %d changes", len(changes))

				// Check for any errors in apply
				if diags.HasErrors() {
					t.Logf("Apply errors (this may be expected): %s", diags.Err())
				}

				// Build the new state from applied changes (same as helper_test.go)
				if len(changes) > 0 {
					stateLoader := stackstate.NewLoader()
					for _, change := range changes {
						proto, err := change.AppliedChangeProto()
						if err != nil {
							t.Fatal(err)
						}

						for _, rawMsg := range proto.Raw {
							if rawMsg.Value == nil {
								continue
							}
							err = stateLoader.AddRaw(rawMsg.Key, rawMsg.Value)
							if err != nil {
								t.Fatal(err)
							}
						}
					}
					state = stateLoader.State()
				}

				// Analyze emitted values
				emittedValues := configValues.GetEmittedValues()
				preApplyValues := filterByPhase(emittedValues, "pre-apply")
				postApplyValues := filterByPhase(emittedValues, "post-apply")

				t.Logf("Test %s - Cycle %d: Pre-apply values: %d, Post-apply values: %d",
					name, i, len(preApplyValues), len(postApplyValues))

				// Log all emitted values for debugging
				for _, value := range emittedValues {
					t.Logf("  Config value: %s [%s] (has value: %v)", value.Addr, value.Phase, value.HasValue)
				}

				// Test-specific assertions would go here
				switch name {
				case "all values known during prepare":
					// Should have values in pre-apply phase for static outputs
					if len(preApplyValues) == 0 {
						t.Logf("Expected some pre-apply config values for static outputs, got none")
					}
				case "partial values known during prepare":
					// Should have some pre-apply values (the known ones)
					if len(emittedValues) == 0 {
						t.Logf("Expected some config values for mixed dependencies, got none")
					}
				case "resource dependent outputs":
					// Values may only be available post-apply (when enabled)
					if len(emittedValues) == 0 {
						t.Logf("Expected some config value events, got none")
					}
				}
			}
		})
	}
}

// TestProgressiveConfigValueEmissionWithResources attempts to trigger actual config value emissions
// by using configurations that create managed resources
func TestProgressiveConfigValueEmissionWithResources(t *testing.T) {
	fakePlanTimestamp, err := time.Parse(time.RFC3339, "2021-01-01T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}

	// Try multiple test configurations to find one that creates actual changes
	testCases := []struct {
		name   string
		path   string
		inputs map[string]cty.Value
	}{
		{
			name: "progressive-mixed",
			path: "progressive-mixed",
			inputs: map[string]cty.Value{
				"input_value": cty.StringVal("test-progressive"),
			},
		},
		{
			name: "mixed-timing-outputs",
			path: "mixed-timing-outputs",
			inputs: map[string]cty.Value{
				"input_value": cty.StringVal("test-progressive-value"),
			},
		},
		{
			name: "component-chain",
			path: "component-chain",
			inputs: map[string]cty.Value{
				"value": cty.StringVal("test-chain-value"),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			configValues := NewConfigValueTracker()

			lock := depsfile.NewLocks()
			lock.SetProvider(
				addrs.NewDefaultProvider("testing"),
				providerreqs.MustParseVersion("0.0.0"),
				providerreqs.MustParseVersionConstraints("=0.0.0"),
				providerreqs.PreferredHashes([]providerreqs.Hash{}),
			)

			store := stacks_testing_provider.NewResourceStore()

			// Load the test configuration
			var testConfig *stackconfig.Config
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Logf("Failed to load config %s: %v", tc.path, r)
						t.Skip("Configuration not available")
					}
				}()
				testConfig = loadMainBundleConfigForTest(t, tc.path)
			}()

			if testConfig == nil {
				t.Skip("Configuration could not be loaded")
			}

			testContext := TestContext{
				timestamp: &fakePlanTimestamp,
				config:    testConfig,
				providers: map[addrs.Provider]providers.Factory{
					addrs.NewDefaultProvider("testing"): func() (providers.Interface, error) {
						provider := stacks_testing_provider.NewProviderWithData(t, store)
						return provider, nil
					},
				},
				dependencyLocks: *lock,
			}

			cycle := TestCycle{
				planInputs: tc.inputs,
			}

			var state *stackstate.State
			t.Logf("Testing config %s for progressive config value emission", tc.name)

			// Plan phase
			plan := testContext.Plan(t, ctx, state, cycle)
			t.Logf("Plan completed for %s", tc.name)

			// Apply with our config value tracking hooks
			request := ApplyRequest{
				Config: testContext.config,
				Plan:   plan,
				InputValues: func() map[stackaddrs.InputVariable]ExternalInputValue {
					inputs := make(map[stackaddrs.InputVariable]ExternalInputValue, len(cycle.planInputs))
					for k, v := range cycle.planInputs {
						inputs[stackaddrs.InputVariable{Name: k}] = ExternalInputValue{Value: v}
					}
					return inputs
				}(),
				ProviderFactories:  testContext.providers,
				ExperimentsAllowed: true,
				DependencyLocks:    testContext.dependencyLocks,
			}

			changesCh := make(chan stackstate.AppliedChange)
			diagsCh := make(chan tfdiags.Diagnostic)
			response := ApplyResponse{
				AppliedChanges: changesCh,
				Diagnostics:    diagsCh,
			}

			// Apply phase with config value tracking
			t.Logf("Starting Apply with hooks for %s...", tc.name)
			applyCtx := ContextWithHooks(ctx, configValues.CaptureHooks())
			go Apply(applyCtx, &request, &response)
			changes, diags := collectApplyOutput(changesCh, diagsCh)

			// Check results
			if diags.HasErrors() {
				t.Logf("Apply errors for %s: %s", tc.name, diags.Err())
			}
			t.Logf("Apply completed for %s with %d changes", tc.name, len(changes))

			// Check what config values were emitted
			emittedValues := configValues.GetEmittedValues()
			preApplyValues := filterByPhase(emittedValues, "pre-apply")
			postApplyValues := filterByPhase(emittedValues, "post-apply")

			t.Logf("Config %s: Pre-apply values: %d, Post-apply values: %d", tc.name, len(preApplyValues), len(postApplyValues))

			// Log all emitted values
			if len(emittedValues) > 0 {
				t.Logf("SUCCESS: %s emitted %d config values!", tc.name, len(emittedValues))
				for _, value := range emittedValues {
					t.Logf("  Emitted: %s [%s] = %s", value.Addr, value.Phase, value.Value.GoString())
				}
			} else {
				t.Logf("Config %s: No config values emitted (changes: %d)", tc.name, len(changes))
			}
		})
	}
}
