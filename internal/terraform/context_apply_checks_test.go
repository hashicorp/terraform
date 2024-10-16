// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"testing"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/checks"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	testing_provider "github.com/hashicorp/terraform/internal/providers/testing"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// This file contains 'integration' tests for the Terraform check blocks.
//
// These tests could live in context_apply_test or context_apply2_test but given
// the size of those files, it makes sense to keep these check related tests
// grouped together.

type checksTestingStatus struct {
	status   checks.Status
	messages []string
}

func TestContextChecks(t *testing.T) {
	tests := map[string]struct {
		configs      map[string]string
		plan         map[string]checksTestingStatus
		planError    string
		planWarning  string
		apply        map[string]checksTestingStatus
		applyError   string
		applyWarning string
		state        *states.State
		provider     *testing_provider.MockProvider
		providerHook func(*testing_provider.MockProvider)
	}{
		"passing": {
			configs: map[string]string{
				"main.tf": `
provider "checks" {}

check "passing" {
  data "checks_object" "positive" {}

  assert {
    condition     = data.checks_object.positive.number >= 0
    error_message = "negative number"
  }
}
`,
			},
			plan: map[string]checksTestingStatus{
				"passing": {
					status: checks.StatusPass,
				},
			},
			apply: map[string]checksTestingStatus{
				"passing": {
					status: checks.StatusPass,
				},
			},
			provider: &testing_provider.MockProvider{
				Meta: "checks",
				GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
					DataSources: map[string]providers.Schema{
						"checks_object": {
							Block: &configschema.Block{
								Attributes: map[string]*configschema.Attribute{
									"number": {
										Type:     cty.Number,
										Computed: true,
									},
								},
							},
						},
					},
				},
				ReadDataSourceFn: func(request providers.ReadDataSourceRequest) providers.ReadDataSourceResponse {
					return providers.ReadDataSourceResponse{
						State: cty.ObjectVal(map[string]cty.Value{
							"number": cty.NumberIntVal(0),
						}),
					}
				},
			},
		},
		"failing": {
			configs: map[string]string{
				"main.tf": `
provider "checks" {}

check "failing" {
  data "checks_object" "positive" {}

  assert {
    condition     = data.checks_object.positive.number >= 0
    error_message = "negative number"
  }
}
`,
			},
			plan: map[string]checksTestingStatus{
				"failing": {
					status:   checks.StatusFail,
					messages: []string{"negative number"},
				},
			},
			planWarning: "Check block assertion failed: negative number",
			apply: map[string]checksTestingStatus{
				"failing": {
					status:   checks.StatusFail,
					messages: []string{"negative number"},
				},
			},
			applyWarning: "Check block assertion failed: negative number",
			provider: &testing_provider.MockProvider{
				Meta: "checks",
				GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
					DataSources: map[string]providers.Schema{
						"checks_object": {
							Block: &configschema.Block{
								Attributes: map[string]*configschema.Attribute{
									"number": {
										Type:     cty.Number,
										Computed: true,
									},
								},
							},
						},
					},
				},
				ReadDataSourceFn: func(request providers.ReadDataSourceRequest) providers.ReadDataSourceResponse {
					return providers.ReadDataSourceResponse{
						State: cty.ObjectVal(map[string]cty.Value{
							"number": cty.NumberIntVal(-1),
						}),
					}
				},
			},
		},
		"mixed": {
			configs: map[string]string{
				"main.tf": `
provider "checks" {}

check "failing" {
  data "checks_object" "neutral" {}

  assert {
    condition     = data.checks_object.neutral.number >= 0
    error_message = "negative number"
  }

  assert {
    condition = data.checks_object.neutral.number < 0
    error_message = "positive number"
  }
}
`,
			},
			plan: map[string]checksTestingStatus{
				"failing": {
					status:   checks.StatusFail,
					messages: []string{"positive number"},
				},
			},
			planWarning: "Check block assertion failed: positive number",
			apply: map[string]checksTestingStatus{
				"failing": {
					status:   checks.StatusFail,
					messages: []string{"positive number"},
				},
			},
			applyWarning: "Check block assertion failed: positive number",
			provider: &testing_provider.MockProvider{
				Meta: "checks",
				GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
					DataSources: map[string]providers.Schema{
						"checks_object": {
							Block: &configschema.Block{
								Attributes: map[string]*configschema.Attribute{
									"number": {
										Type:     cty.Number,
										Computed: true,
									},
								},
							},
						},
					},
				},
				ReadDataSourceFn: func(request providers.ReadDataSourceRequest) providers.ReadDataSourceResponse {
					return providers.ReadDataSourceResponse{
						State: cty.ObjectVal(map[string]cty.Value{
							"number": cty.NumberIntVal(0),
						}),
					}
				},
			},
		},
		"nested data blocks reload during apply": {
			configs: map[string]string{
				"main.tf": `
provider "checks" {}

data "checks_object" "data_block" {}

check "data_block" {
  assert {
    condition     = data.checks_object.data_block.number >= 0
    error_message = "negative number"
  }
}

check "nested_data_block" {
  data "checks_object" "nested_data_block" {}

  assert {
    condition     = data.checks_object.nested_data_block.number >= 0
    error_message = "negative number"
  }
}
`,
			},
			plan: map[string]checksTestingStatus{
				"nested_data_block": {
					status:   checks.StatusFail,
					messages: []string{"negative number"},
				},
				"data_block": {
					status:   checks.StatusFail,
					messages: []string{"negative number"},
				},
			},
			planWarning: "2 warnings:\n\n- Check block assertion failed: negative number\n- Check block assertion failed: negative number",
			apply: map[string]checksTestingStatus{
				"nested_data_block": {
					status: checks.StatusPass,
				},
				"data_block": {
					status:   checks.StatusFail,
					messages: []string{"negative number"},
				},
			},
			applyWarning: "Check block assertion failed: negative number",
			provider: &testing_provider.MockProvider{
				Meta: "checks",
				GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
					DataSources: map[string]providers.Schema{
						"checks_object": {
							Block: &configschema.Block{
								Attributes: map[string]*configschema.Attribute{
									"number": {
										Type:     cty.Number,
										Computed: true,
									},
								},
							},
						},
					},
				},
				ReadDataSourceFn: func(request providers.ReadDataSourceRequest) providers.ReadDataSourceResponse {
					return providers.ReadDataSourceResponse{
						State: cty.ObjectVal(map[string]cty.Value{
							"number": cty.NumberIntVal(-1),
						}),
					}
				},
			},
			providerHook: func(provider *testing_provider.MockProvider) {
				provider.ReadDataSourceFn = func(request providers.ReadDataSourceRequest) providers.ReadDataSourceResponse {
					// The data returned by the data sources are changing
					// between the plan and apply stage. The nested data block
					// will update to reflect this while the normal data block
					// will not detect the change.
					return providers.ReadDataSourceResponse{
						State: cty.ObjectVal(map[string]cty.Value{
							"number": cty.NumberIntVal(0),
						}),
					}
				}
			},
		},
		"returns unknown for unknown config": {
			configs: map[string]string{
				"main.tf": `
provider "checks" {}

resource "checks_object" "resource_block" {}

check "resource_block" {
  data "checks_object" "data_block" {
    id = checks_object.resource_block.id
  }

  assert {
    condition = data.checks_object.data_block.number >= 0
    error_message = "negative number"
  }
}
`,
			},
			plan: map[string]checksTestingStatus{
				"resource_block": {
					status: checks.StatusUnknown,
				},
			},
			planWarning: "Check block assertion known after apply: The condition could not be evaluated at this time, a result will be known when this plan is applied.",
			apply: map[string]checksTestingStatus{
				"resource_block": {
					status: checks.StatusPass,
				},
			},
			provider: &testing_provider.MockProvider{
				Meta: "checks",
				GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
					ResourceTypes: map[string]providers.Schema{
						"checks_object": {
							Block: &configschema.Block{
								Attributes: map[string]*configschema.Attribute{
									"id": {
										Type:     cty.String,
										Computed: true,
									},
								},
							},
						},
					},
					DataSources: map[string]providers.Schema{
						"checks_object": {
							Block: &configschema.Block{
								Attributes: map[string]*configschema.Attribute{
									"id": {
										Type:     cty.String,
										Required: true,
									},
									"number": {
										Type:     cty.Number,
										Computed: true,
									},
								},
							},
						},
					},
				},
				PlanResourceChangeFn: func(request providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
					return providers.PlanResourceChangeResponse{
						PlannedState: cty.ObjectVal(map[string]cty.Value{
							"id": cty.UnknownVal(cty.String),
						}),
					}
				},
				ApplyResourceChangeFn: func(request providers.ApplyResourceChangeRequest) providers.ApplyResourceChangeResponse {
					return providers.ApplyResourceChangeResponse{
						NewState: cty.ObjectVal(map[string]cty.Value{
							"id": cty.StringVal("7A9F887D-44C7-4281-80E5-578E41F99DFC"),
						}),
					}
				},
				ReadDataSourceFn: func(request providers.ReadDataSourceRequest) providers.ReadDataSourceResponse {
					values := request.Config.AsValueMap()
					if id, ok := values["id"]; ok {
						if id.IsKnown() && id.AsString() == "7A9F887D-44C7-4281-80E5-578E41F99DFC" {
							return providers.ReadDataSourceResponse{
								State: cty.ObjectVal(map[string]cty.Value{
									"id":     cty.StringVal("7A9F887D-44C7-4281-80E5-578E41F99DFC"),
									"number": cty.NumberIntVal(0),
								}),
							}
						}
					}

					return providers.ReadDataSourceResponse{
						Diagnostics: tfdiags.Diagnostics{tfdiags.Sourceless(tfdiags.Error, "shouldn't make it here", "really shouldn't make it here")},
					}
				},
			},
		},
		"failing nested data source doesn't block the plan": {
			configs: map[string]string{
				"main.tf": `
provider "checks" {}

check "error" {
  data "checks_object" "data_block" {}

  assert {
    condition = data.checks_object.data_block.number >= 0
    error_message = "negative number"
  }
}
`,
			},
			plan: map[string]checksTestingStatus{
				"error": {
					status: checks.StatusFail,
					messages: []string{
						"data source read failed: something bad happened and the provider couldn't read the data source",
					},
				},
			},
			planWarning: "data source read failed: something bad happened and the provider couldn't read the data source",
			apply: map[string]checksTestingStatus{
				"error": {
					status: checks.StatusFail,
					messages: []string{
						"data source read failed: something bad happened and the provider couldn't read the data source",
					},
				},
			},
			applyWarning: "data source read failed: something bad happened and the provider couldn't read the data source",
			provider: &testing_provider.MockProvider{
				Meta: "checks",
				GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
					DataSources: map[string]providers.Schema{
						"checks_object": {
							Block: &configschema.Block{
								Attributes: map[string]*configschema.Attribute{
									"number": {
										Type:     cty.Number,
										Computed: true,
									},
								},
							},
						},
					},
				},
				ReadDataSourceFn: func(request providers.ReadDataSourceRequest) providers.ReadDataSourceResponse {
					return providers.ReadDataSourceResponse{
						Diagnostics: tfdiags.Diagnostics{tfdiags.Sourceless(tfdiags.Error, "data source read failed", "something bad happened and the provider couldn't read the data source")},
					}
				},
			},
		}, "failing nested data source should prevent checks from executing": {
			configs: map[string]string{
				"main.tf": `
provider "checks" {}

resource "checks_object" "resource_block" {
  number = -1
}

check "error" {
  data "checks_object" "data_block" {}

  assert {
    condition = checks_object.resource_block.number >= 0
    error_message = "negative number"
  }
}
`,
			},
			state: states.BuildState(func(state *states.SyncState) {
				state.SetResourceInstanceCurrent(
					addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "checks_object",
						Name: "resource_block",
					}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
					&states.ResourceInstanceObjectSrc{
						Status:    states.ObjectReady,
						AttrsJSON: []byte(`{"number": -1}`),
					},
					addrs.AbsProviderConfig{
						Provider: addrs.NewDefaultProvider("checks"),
						Module:   addrs.RootModule,
					})
			}),
			plan: map[string]checksTestingStatus{
				"error": {
					status: checks.StatusFail,
					messages: []string{
						"data source read failed: something bad happened and the provider couldn't read the data source",
					},
				},
			},
			planWarning: "data source read failed: something bad happened and the provider couldn't read the data source",
			apply: map[string]checksTestingStatus{
				"error": {
					status: checks.StatusFail,
					messages: []string{
						"data source read failed: something bad happened and the provider couldn't read the data source",
					},
				},
			},
			applyWarning: "data source read failed: something bad happened and the provider couldn't read the data source",
			provider: &testing_provider.MockProvider{
				Meta: "checks",
				GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
					ResourceTypes: map[string]providers.Schema{
						"checks_object": {
							Block: &configschema.Block{
								Attributes: map[string]*configschema.Attribute{
									"number": {
										Type:     cty.Number,
										Required: true,
									},
								},
							},
						},
					},
					DataSources: map[string]providers.Schema{
						"checks_object": {
							Block: &configschema.Block{
								Attributes: map[string]*configschema.Attribute{
									"number": {
										Type:     cty.Number,
										Computed: true,
									},
								},
							},
						},
					},
				},
				PlanResourceChangeFn: func(request providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
					return providers.PlanResourceChangeResponse{
						PlannedState: cty.ObjectVal(map[string]cty.Value{
							"number": cty.NumberIntVal(-1),
						}),
					}
				},
				ApplyResourceChangeFn: func(request providers.ApplyResourceChangeRequest) providers.ApplyResourceChangeResponse {
					return providers.ApplyResourceChangeResponse{
						NewState: cty.ObjectVal(map[string]cty.Value{
							"number": cty.NumberIntVal(-1),
						}),
					}
				},
				ReadDataSourceFn: func(request providers.ReadDataSourceRequest) providers.ReadDataSourceResponse {
					return providers.ReadDataSourceResponse{
						Diagnostics: tfdiags.Diagnostics{tfdiags.Sourceless(tfdiags.Error, "data source read failed", "something bad happened and the provider couldn't read the data source")},
					}
				},
			},
		},
		"check failing in state and passing after plan and apply": {
			configs: map[string]string{
				"main.tf": `
provider "checks" {}

resource "checks_object" "resource" {
  number = 0
}

check "passing" {
  assert {
    condition     = checks_object.resource.number >= 0
    error_message = "negative number"
  }
}
`,
			},
			state: states.BuildState(func(state *states.SyncState) {
				state.SetResourceInstanceCurrent(
					addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "checks_object",
						Name: "resource",
					}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
					&states.ResourceInstanceObjectSrc{
						Status:    states.ObjectReady,
						AttrsJSON: []byte(`{"number": -1}`),
					},
					addrs.AbsProviderConfig{
						Provider: addrs.NewDefaultProvider("checks"),
						Module:   addrs.RootModule,
					})
			}),
			plan: map[string]checksTestingStatus{
				"passing": {
					status: checks.StatusPass,
				},
			},
			apply: map[string]checksTestingStatus{
				"passing": {
					status: checks.StatusPass,
				},
			},
			provider: &testing_provider.MockProvider{
				Meta: "checks",
				GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
					ResourceTypes: map[string]providers.Schema{
						"checks_object": {
							Block: &configschema.Block{
								Attributes: map[string]*configschema.Attribute{
									"number": {
										Type:     cty.Number,
										Required: true,
									},
								},
							},
						},
					},
				},
				PlanResourceChangeFn: func(request providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
					return providers.PlanResourceChangeResponse{
						PlannedState: cty.ObjectVal(map[string]cty.Value{
							"number": cty.NumberIntVal(0),
						}),
					}
				},
				ApplyResourceChangeFn: func(request providers.ApplyResourceChangeRequest) providers.ApplyResourceChangeResponse {
					return providers.ApplyResourceChangeResponse{
						NewState: cty.ObjectVal(map[string]cty.Value{
							"number": cty.NumberIntVal(0),
						}),
					}
				},
			},
		},
		"failing data source does block the plan": {
			configs: map[string]string{
				"main.tf": `
provider "checks" {}

data "checks_object" "data_block" {}

check "error" {
  assert {
    condition = data.checks_object.data_block.number >= 0
    error_message = "negative number"
  }
}
`,
			},
			planError: "data source read failed: something bad happened and the provider couldn't read the data source",
			provider: &testing_provider.MockProvider{
				Meta: "checks",
				GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
					DataSources: map[string]providers.Schema{
						"checks_object": {
							Block: &configschema.Block{
								Attributes: map[string]*configschema.Attribute{
									"number": {
										Type:     cty.Number,
										Computed: true,
									},
								},
							},
						},
					},
				},
				ReadDataSourceFn: func(request providers.ReadDataSourceRequest) providers.ReadDataSourceResponse {
					return providers.ReadDataSourceResponse{
						Diagnostics: tfdiags.Diagnostics{tfdiags.Sourceless(tfdiags.Error, "data source read failed", "something bad happened and the provider couldn't read the data source")},
					}
				},
			},
		},
		"invalid reference into check block": {
			configs: map[string]string{
				"main.tf": `
provider "checks" {}

data "checks_object" "data_block" {
  id = data.checks_object.nested_data_block.id
}

check "error" {
  data "checks_object" "nested_data_block" {}

  assert {
    condition = data.checks_object.data_block.number >= 0
    error_message = "negative number"
  }
}
`,
			},
			planError: "Reference to scoped resource: The referenced data resource \"checks_object\" \"nested_data_block\" is not available from this context.",
			provider: &testing_provider.MockProvider{
				Meta: "checks",
				GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
					DataSources: map[string]providers.Schema{
						"checks_object": {
							Block: &configschema.Block{
								Attributes: map[string]*configschema.Attribute{
									"id": {
										Type:     cty.String,
										Computed: true,
										Optional: true,
									},
								},
							},
						},
					},
				},
				ReadDataSourceFn: func(request providers.ReadDataSourceRequest) providers.ReadDataSourceResponse {
					input := request.Config.AsValueMap()
					if _, ok := input["id"]; ok {
						return providers.ReadDataSourceResponse{
							State: request.Config,
						}
					}

					return providers.ReadDataSourceResponse{
						State: cty.ObjectVal(map[string]cty.Value{
							"id": cty.UnknownVal(cty.String),
						}),
					}
				},
			},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			configs := testModuleInline(t, test.configs)
			ctx := testContext2(t, &ContextOpts{
				Providers: map[addrs.Provider]providers.Factory{
					addrs.NewDefaultProvider(test.provider.Meta.(string)): testProviderFuncFixed(test.provider),
				},
			})

			initialState := states.NewState()
			if test.state != nil {
				initialState = test.state
			}

			plan, diags := ctx.Plan(configs, initialState, &PlanOpts{
				Mode: plans.NormalMode,
			})
			if validateCheckDiagnostics(t, "planning", test.planWarning, test.planError, diags) {
				return
			}
			validateCheckResults(t, "planning", test.plan, plan.Checks)

			if test.providerHook != nil {
				// This gives an opportunity to change the behaviour of the
				// provider between the plan and apply stages.
				test.providerHook(test.provider)
			}

			state, diags := ctx.Apply(plan, configs, nil)
			if validateCheckDiagnostics(t, "apply", test.applyWarning, test.applyError, diags) {
				return
			}
			validateCheckResults(t, "apply", test.apply, state.CheckResults)
		})
	}
}

func TestContextChecks_DoesNotPanicOnModuleExpansion(t *testing.T) {
	// This is a bit of a special test, we're adding it to verify that
	// https://github.com/hashicorp/terraform/issues/34062 is fixed.
	//
	// Essentially we make a check block in a child module that depends on a
	// resource that has no changes. We don't care about the actual behaviour
	// of the check block. We just don't want the apply operation to crash.

	m := testModuleInline(t, map[string]string{
		"main.tf": `
module "panic_at_the_disco" {
   source = "./panic"
}
`,
		"panic/main.tf": `
resource "test_object" "object" {
    test_string = "Hello, world!"
}

check "check_should_not_panic" {
    assert {
         condition     = test_object.object.test_string == "Hello, world!"
         error_message = "condition violated"
    }
}
`,
	})

	p := simpleMockProvider()

	ctx := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	plan, diags := ctx.Plan(m, states.BuildState(func(state *states.SyncState) {
		state.SetResourceInstanceCurrent(
			mustResourceInstanceAddr("module.panic_at_the_disco.test_object.object"),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"test_string":"Hello, world!"}`),
				Status:    states.ObjectReady,
			},
			mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
		)
	}), DefaultPlanOpts)
	assertNoErrors(t, diags)

	_, diags = ctx.Apply(plan, m, nil)
	assertNoErrors(t, diags)
}

func validateCheckDiagnostics(t *testing.T, stage string, expectedWarning, expectedError string, actual tfdiags.Diagnostics) bool {
	if expectedError != "" {
		if !actual.HasErrors() {
			t.Errorf("expected %s to error with \"%s\", but no errors were returned", stage, expectedError)
		} else if expectedError != actual.Err().Error() {
			t.Errorf("expected %s to error with \"%s\" but found \"%s\"", stage, expectedError, actual.Err())
		}

		// If we expected an error then we won't finish the rest of the test.
		return true
	}

	if expectedWarning != "" {
		warnings := actual.ErrWithWarnings()
		if actual.ErrWithWarnings() == nil {
			t.Errorf("expected %s to warn with \"%s\", but no errors were returned", stage, expectedWarning)
		} else if expectedWarning != warnings.Error() {
			t.Errorf("expected %s to warn with \"%s\" but found \"%s\"", stage, expectedWarning, warnings)
		}
	} else {
		if actual.ErrWithWarnings() != nil {
			t.Errorf("expected %s to produce no diagnostics but found \"%s\"", stage, actual.ErrWithWarnings())
		}
	}

	assertNoErrors(t, actual)
	return false
}

func validateCheckResults(t *testing.T, stage string, expected map[string]checksTestingStatus, actual *states.CheckResults) {

	// Just a quick sanity check that the plan or apply process didn't create
	// some non-existent checks.
	if len(expected) != len(actual.ConfigResults.Keys()) {
		t.Errorf("expected %d check results but found %d after %s", len(expected), len(actual.ConfigResults.Keys()), stage)
	}

	// Now, lets make sure the checks all match what we expect.
	for check, want := range expected {
		results := actual.GetObjectResult(addrs.Check{
			Name: check,
		}.Absolute(addrs.RootModuleInstance))

		if results.Status != want.status {
			t.Errorf("%s: wanted %s but got %s after %s", check, want.status, results.Status, stage)
		}

		if len(want.messages) != len(results.FailureMessages) {
			t.Errorf("%s: expected %d failure messages but had %d after %s", check, len(want.messages), len(results.FailureMessages), stage)
		}

		max := len(want.messages)
		if len(results.FailureMessages) > max {
			max = len(results.FailureMessages)
		}

		for ix := 0; ix < max; ix++ {
			var expected, actual string
			if ix < len(want.messages) {
				expected = want.messages[ix]
			}
			if ix < len(results.FailureMessages) {
				actual = results.FailureMessages[ix]
			}

			// Order matters!
			if actual != expected {
				t.Errorf("%s: expected failure message at %d to be \"%s\" but was \"%s\" after %s", check, ix, expected, actual, stage)
			}
		}

	}
}
