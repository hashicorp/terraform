// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package moduletest

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestRun_ValidateExpectedFailures(t *testing.T) {

	type output struct {
		Description tfdiags.Description
		Severity    tfdiags.Severity
	}

	tcs := map[string]struct {
		ExpectedFailures []string
		Input            tfdiags.Diagnostics
		Output           []output
	}{
		"empty": {
			ExpectedFailures: nil,
			Input:            nil,
			Output:           nil,
		},
		"carries through simple diags": {
			Input: createDiagnostics(func(diags tfdiags.Diagnostics) tfdiags.Diagnostics {

				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "simple error",
					Detail:   "want to see this in the returned set",
				})

				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagWarning,
					Summary:  "simple warning",
					Detail:   "want to see this in the returned set",
				})

				return diags
			}),
			Output: []output{
				{
					Description: tfdiags.Description{
						Summary: "simple error",
						Detail:  "want to see this in the returned set",
					},
					Severity: tfdiags.Error,
				},
				{
					Description: tfdiags.Description{
						Summary: "simple warning",
						Detail:  "want to see this in the returned set",
					},
					Severity: tfdiags.Warning,
				},
			},
		},
		"expected failures did not fail": {
			ExpectedFailures: []string{
				"check.example",
			},
			Input: nil,
			Output: []output{
				{
					Description: tfdiags.Description{
						Summary: "Missing expected failure",
						Detail:  "The checkable object, check.example, was expected to report an error but did not.",
					},
					Severity: tfdiags.Error,
				},
			},
		},
		"outputs": {
			ExpectedFailures: []string{
				"output.expected_one",
				"output.expected_two",
			},
			Input: createDiagnostics(func(diags tfdiags.Diagnostics) tfdiags.Diagnostics {

				// First, let's create an output that failed that isn't
				// expected. This should be unaffected by our function.
				diags = diags.Append(
					&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "unexpected failure",
						Detail:   "this should not be removed",
						Extra: &addrs.CheckRuleDiagnosticExtra{
							CheckRule: addrs.NewCheckRule(addrs.AbsOutputValue{
								Module:      addrs.RootModuleInstance,
								OutputValue: addrs.OutputValue{Name: "unexpected"},
							}, addrs.OutputPrecondition, 0),
						},
					})

				// Second, let's create an output that failed but is expected.
				// Our function should remove this from the set of diags.
				diags = diags.Append(
					&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "expected failure",
						Detail:   "this should be removed",
						Extra: &addrs.CheckRuleDiagnosticExtra{
							CheckRule: addrs.NewCheckRule(addrs.AbsOutputValue{
								Module:      addrs.RootModuleInstance,
								OutputValue: addrs.OutputValue{Name: "expected_one"},
							}, addrs.OutputPrecondition, 0),
						},
					})

				diags = diags.Append(
					&hcl.Diagnostic{
						Severity: hcl.DiagWarning,
						Summary:  "expected warning",
						Detail:   "this should not be removed",
						Extra: &addrs.CheckRuleDiagnosticExtra{
							CheckRule: addrs.NewCheckRule(addrs.AbsOutputValue{
								Module:      addrs.RootModuleInstance,
								OutputValue: addrs.OutputValue{Name: "expected_one"},
							}, addrs.OutputPrecondition, 0),
						},
					})

				// The error we are adding here is for expected_two but in a
				// child module. We expect that this diagnostic shouldn't
				// trigger our expected failure, and that an extra diagnostic
				// should be created complaining that the output wasn't actually
				// triggered.

				diags = diags.Append(
					&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "error in child module",
						Detail:   "this should not be removed",
						Extra: &addrs.CheckRuleDiagnosticExtra{
							CheckRule: addrs.NewCheckRule(addrs.AbsOutputValue{
								Module: []addrs.ModuleInstanceStep{
									{
										Name: "child_module",
									},
								},
								OutputValue: addrs.OutputValue{Name: "expected_two"},
							}, addrs.OutputPrecondition, 0),
						},
					})

				return diags
			}),
			Output: []output{
				{
					Description: tfdiags.Description{
						Summary: "unexpected failure",
						Detail:  "this should not be removed",
					},
					Severity: tfdiags.Error,
				},
				{
					Description: tfdiags.Description{
						Summary: "expected warning",
						Detail:  "this should not be removed",
					},
					Severity: tfdiags.Warning,
				},
				{
					Description: tfdiags.Description{
						Summary: "error in child module",
						Detail:  "this should not be removed",
					},
					Severity: tfdiags.Error,
				},
				{
					Description: tfdiags.Description{
						Summary: "Missing expected failure",
						Detail:  "The checkable object, output.expected_two, was expected to report an error but did not.",
					},
					Severity: tfdiags.Error,
				},
			},
		},
		"variables": {
			ExpectedFailures: []string{
				"var.expected_one",
				"var.expected_two",
			},
			Input: createDiagnostics(func(diags tfdiags.Diagnostics) tfdiags.Diagnostics {

				// First, let's create an input that failed that isn't
				// expected. This should be unaffected by our function.
				diags = diags.Append(
					&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "unexpected failure",
						Detail:   "this should not be removed",
						Extra: &addrs.CheckRuleDiagnosticExtra{
							CheckRule: addrs.NewCheckRule(addrs.AbsInputVariableInstance{
								Module:   addrs.RootModuleInstance,
								Variable: addrs.InputVariable{Name: "unexpected"},
							}, addrs.InputValidation, 0),
						},
					})

				// Second, let's create an input that failed but is expected.
				// Our function should remove this from the set of diags.
				diags = diags.Append(
					&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "expected failure",
						Detail:   "this should be removed",
						Extra: &addrs.CheckRuleDiagnosticExtra{
							CheckRule: addrs.NewCheckRule(addrs.AbsInputVariableInstance{
								Module:   addrs.RootModuleInstance,
								Variable: addrs.InputVariable{Name: "expected_one"},
							}, addrs.InputValidation, 0),
						},
					})

				diags = diags.Append(
					&hcl.Diagnostic{
						Severity: hcl.DiagWarning,
						Summary:  "expected warning",
						Detail:   "this should not be removed",
						Extra: &addrs.CheckRuleDiagnosticExtra{
							CheckRule: addrs.NewCheckRule(addrs.AbsInputVariableInstance{
								Module:   addrs.RootModuleInstance,
								Variable: addrs.InputVariable{Name: "expected_one"},
							}, addrs.InputValidation, 0),
						},
					})

				// The error we are adding here is for expected_two but in a
				// child module. We expect that this diagnostic shouldn't
				// trigger our expected failure, and that an extra diagnostic
				// should be created complaining that the output wasn't actually
				// triggered.

				diags = diags.Append(
					&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "error in child module",
						Detail:   "this should not be removed",
						Extra: &addrs.CheckRuleDiagnosticExtra{
							CheckRule: addrs.NewCheckRule(addrs.AbsInputVariableInstance{
								Module: []addrs.ModuleInstanceStep{
									{
										Name: "child_module",
									},
								},
								Variable: addrs.InputVariable{Name: "expected_two"},
							}, addrs.InputValidation, 0),
						},
					})

				return diags
			}),
			Output: []output{
				{
					Description: tfdiags.Description{
						Summary: "unexpected failure",
						Detail:  "this should not be removed",
					},
					Severity: tfdiags.Error,
				},
				{
					Description: tfdiags.Description{
						Summary: "expected warning",
						Detail:  "this should not be removed",
					},
					Severity: tfdiags.Warning,
				},
				{
					Description: tfdiags.Description{
						Summary: "error in child module",
						Detail:  "this should not be removed",
					},
					Severity: tfdiags.Error,
				},
				{
					Description: tfdiags.Description{
						Summary: "Missing expected failure",
						Detail:  "The checkable object, var.expected_two, was expected to report an error but did not.",
					},
					Severity: tfdiags.Error,
				},
			},
		},
		"resources": {
			ExpectedFailures: []string{
				"test_instance.single",
				"test_instance.all_instances",
				"test_instance.instance[0]",
				"test_instance.instance[2]",
				"test_instance.missing",
			},
			Input: createDiagnostics(func(diags tfdiags.Diagnostics) tfdiags.Diagnostics {
				// First, we'll create an unexpected failure that should be
				// carried through untouched.
				diags = diags.Append(
					&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "unexpected failure",
						Detail:   "this should not be removed",
						Extra: &addrs.CheckRuleDiagnosticExtra{
							CheckRule: addrs.NewCheckRule(addrs.AbsResourceInstance{
								Module: addrs.RootModuleInstance,
								Resource: addrs.ResourceInstance{
									Resource: addrs.Resource{
										Mode: addrs.ManagedResourceMode,
										Type: "test_instance",
										Name: "unexpected",
									},
								},
							}, addrs.ResourcePrecondition, 0),
						},
					})

				// Second, we'll create a failure from our test_instance.single
				// resource that should be removed.
				diags = diags.Append(
					&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "expected failure in test_instance.single",
						Detail:   "this should be removed",
						Extra: &addrs.CheckRuleDiagnosticExtra{
							CheckRule: addrs.NewCheckRule(addrs.AbsResourceInstance{
								Module: addrs.RootModuleInstance,
								Resource: addrs.ResourceInstance{
									Resource: addrs.Resource{
										Mode: addrs.ManagedResourceMode,
										Type: "test_instance",
										Name: "single",
									},
								},
							}, addrs.ResourcePrecondition, 0),
						},
					})

				// Third, we'll create a warning from our test_instance.single
				// resource that should be propagated as it is only a warning.
				diags = diags.Append(
					&hcl.Diagnostic{
						Severity: hcl.DiagWarning,
						Summary:  "expected warning in test_instance.single",
						Detail:   "this should not be removed",
						Extra: &addrs.CheckRuleDiagnosticExtra{
							CheckRule: addrs.NewCheckRule(addrs.AbsResourceInstance{
								Module: addrs.RootModuleInstance,
								Resource: addrs.ResourceInstance{
									Resource: addrs.Resource{
										Mode: addrs.ManagedResourceMode,
										Type: "test_instance",
										Name: "single",
									},
								},
							}, addrs.ResourcePrecondition, 0),
						},
					})

				// Fourth, we'll create diagnostics from several instances of
				// the test_instance.all_instances which should all be removed.
				diags = diags.Append(
					&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "expected failure in test_instance.all_instances[0]",
						Detail:   "this should be removed",
						Extra: &addrs.CheckRuleDiagnosticExtra{
							CheckRule: addrs.NewCheckRule(addrs.AbsResourceInstance{
								Module: addrs.RootModuleInstance,
								Resource: addrs.ResourceInstance{
									Resource: addrs.Resource{
										Mode: addrs.ManagedResourceMode,
										Type: "test_instance",
										Name: "all_instances",
									},
									Key: addrs.IntKey(0),
								},
							}, addrs.ResourcePrecondition, 0),
						},
					})
				diags = diags.Append(
					&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "expected failure in test_instance.all_instances[1]",
						Detail:   "this should be removed",
						Extra: &addrs.CheckRuleDiagnosticExtra{
							CheckRule: addrs.NewCheckRule(addrs.AbsResourceInstance{
								Module: addrs.RootModuleInstance,
								Resource: addrs.ResourceInstance{
									Resource: addrs.Resource{
										Mode: addrs.ManagedResourceMode,
										Type: "test_instance",
										Name: "all_instances",
									},
									Key: addrs.IntKey(1),
								},
							}, addrs.ResourcePrecondition, 0),
						},
					})
				diags = diags.Append(
					&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "expected failure in test_instance.all_instances[2]",
						Detail:   "this should be removed",
						Extra: &addrs.CheckRuleDiagnosticExtra{
							CheckRule: addrs.NewCheckRule(addrs.AbsResourceInstance{
								Module: addrs.RootModuleInstance,
								Resource: addrs.ResourceInstance{
									Resource: addrs.Resource{
										Mode: addrs.ManagedResourceMode,
										Type: "test_instance",
										Name: "all_instances",
									},
									Key: addrs.IntKey(2),
								},
							}, addrs.ResourcePrecondition, 0),
						},
					})

				// Fifth, we'll create diagnostics for several instances of
				// the test_instance.instance resource, only some of which
				// should be removed.
				diags = diags.Append(
					&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "expected failure in test_instance.instance[0]",
						Detail:   "this should be removed",
						Extra: &addrs.CheckRuleDiagnosticExtra{
							CheckRule: addrs.NewCheckRule(addrs.AbsResourceInstance{
								Module: addrs.RootModuleInstance,
								Resource: addrs.ResourceInstance{
									Resource: addrs.Resource{
										Mode: addrs.ManagedResourceMode,
										Type: "test_instance",
										Name: "instance",
									},
									Key: addrs.IntKey(0),
								},
							}, addrs.ResourcePrecondition, 0),
						},
					})
				diags = diags.Append(
					&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "expected failure in test_instance.instance[1]",
						Detail:   "this should not be removed",
						Extra: &addrs.CheckRuleDiagnosticExtra{
							CheckRule: addrs.NewCheckRule(addrs.AbsResourceInstance{
								Module: addrs.RootModuleInstance,
								Resource: addrs.ResourceInstance{
									Resource: addrs.Resource{
										Mode: addrs.ManagedResourceMode,
										Type: "test_instance",
										Name: "instance",
									},
									Key: addrs.IntKey(1),
								},
							}, addrs.ResourcePrecondition, 0),
						},
					})
				diags = diags.Append(
					&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "expected failure in test_instance.instance[2]",
						Detail:   "this should be removed",
						Extra: &addrs.CheckRuleDiagnosticExtra{
							CheckRule: addrs.NewCheckRule(addrs.AbsResourceInstance{
								Module: addrs.RootModuleInstance,
								Resource: addrs.ResourceInstance{
									Resource: addrs.Resource{
										Mode: addrs.ManagedResourceMode,
										Type: "test_instance",
										Name: "instance",
									},
									Key: addrs.IntKey(2),
								},
							}, addrs.ResourcePrecondition, 0),
						},
					})

				// Finally, we'll create an error that originated from
				// test_instance.missing but in a child module which shouldn't
				// be removed.
				diags = diags.Append(
					&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "failure in child module",
						Detail:   "this should not be removed",
						Extra: &addrs.CheckRuleDiagnosticExtra{
							CheckRule: addrs.NewCheckRule(addrs.AbsResourceInstance{
								Module: []addrs.ModuleInstanceStep{
									{
										Name: "child_module",
									},
								},
								Resource: addrs.ResourceInstance{
									Resource: addrs.Resource{
										Mode: addrs.ManagedResourceMode,
										Type: "test_instance",
										Name: "missing",
									},
								},
							}, addrs.ResourcePrecondition, 0),
						},
					})

				return diags
			}),
			Output: []output{
				{
					Description: tfdiags.Description{
						Summary: "unexpected failure",
						Detail:  "this should not be removed",
					},
					Severity: tfdiags.Error,
				},
				{
					Description: tfdiags.Description{
						Summary: "expected warning in test_instance.single",
						Detail:  "this should not be removed",
					},
					Severity: tfdiags.Warning,
				},
				{
					Description: tfdiags.Description{
						Summary: "expected failure in test_instance.instance[1]",
						Detail:  "this should not be removed",
					},
					Severity: tfdiags.Error,
				},
				{
					Description: tfdiags.Description{
						Summary: "failure in child module",
						Detail:  "this should not be removed",
					},
					Severity: tfdiags.Error,
				},
				{
					Description: tfdiags.Description{
						Summary: "Missing expected failure",
						Detail:  "The checkable object, test_instance.missing, was expected to report an error but did not.",
					},
					Severity: tfdiags.Error,
				},
			},
		},
		"check_assertions": {
			ExpectedFailures: []string{
				"check.expected",
				"check.missing",
			},
			Input: createDiagnostics(func(diags tfdiags.Diagnostics) tfdiags.Diagnostics {
				// First, we'll add an unexpected warning from a check block
				// assertion that should get upgraded to an error.
				diags = diags.Append(
					&hcl.Diagnostic{
						Severity: hcl.DiagWarning,
						Summary:  "unexpected failure",
						Detail:   "this should upgrade and not be removed",
						Extra: &addrs.CheckRuleDiagnosticExtra{
							CheckRule: addrs.NewCheckRule(addrs.AbsCheck{
								Module: addrs.RootModuleInstance,
								Check: addrs.Check{
									Name: "unexpected",
								},
							}, addrs.CheckAssertion, 0),
						},
					})

				// Second, we'll add an unexpected warning from a check block
				// in a child module that should get upgrade to error.
				diags = diags.Append(
					&hcl.Diagnostic{
						Severity: hcl.DiagWarning,
						Summary:  "expected failure in child module",
						Detail:   "this should upgrade and not be removed",
						Extra: &addrs.CheckRuleDiagnosticExtra{
							CheckRule: addrs.NewCheckRule(addrs.AbsCheck{
								Module: []addrs.ModuleInstanceStep{
									{
										Name: "child_module",
									},
								},
								Check: addrs.Check{
									Name: "expected",
								},
							}, addrs.CheckAssertion, 0),
						},
					})

				// Third, we'll add an expected warning from a check block
				// assertion that should be removed.
				diags = diags.Append(
					&hcl.Diagnostic{
						Severity: hcl.DiagWarning,
						Summary:  "expected failure",
						Detail:   "this should be removed",
						Extra: &addrs.CheckRuleDiagnosticExtra{
							CheckRule: addrs.NewCheckRule(addrs.AbsCheck{
								Module: addrs.RootModuleInstance,
								Check: addrs.Check{
									Name: "expected",
								},
							}, addrs.CheckAssertion, 0),
						},
					})

				// The second expected failure has no diagnostics, we just want
				// to make sure that a new diagnostic is added for this case.

				return diags
			}),
			Output: []output{
				{
					Description: tfdiags.Description{
						Summary: "unexpected failure",
						Detail:  "this should upgrade and not be removed",
					},
					Severity: tfdiags.Error,
				},
				{
					Description: tfdiags.Description{
						Summary: "expected failure in child module",
						Detail:  "this should upgrade and not be removed",
					},
					Severity: tfdiags.Error,
				},
				{
					Description: tfdiags.Description{
						Summary: "Missing expected failure",
						Detail:  "The checkable object, check.missing, was expected to report an error but did not.",
					},
					Severity: tfdiags.Error,
				},
			},
		},
		"check_data_sources": {
			ExpectedFailures: []string{
				"check.expected",
			},
			Input: createDiagnostics(func(diags tfdiags.Diagnostics) tfdiags.Diagnostics {
				// First, we'll add an unexpected warning from a check block
				// assertion that should be propagated as an error.
				diags = diags.Append(
					tfdiags.Override(
						tfdiags.Sourceless(tfdiags.Error, "unexpected failure", "this should be an error and not removed"),
						tfdiags.Warning,
						func() tfdiags.DiagnosticExtraWrapper {
							return &addrs.CheckRuleDiagnosticExtra{
								CheckRule: addrs.NewCheckRule(addrs.AbsCheck{
									Module: addrs.RootModuleInstance,
									Check: addrs.Check{
										Name: "unexpected",
									},
								}, addrs.CheckDataResource, 0),
							}
						}))

				// Second, we'll add an unexpected warning from a check block
				// assertion that should remain as a warning.
				diags = diags.Append(
					tfdiags.Override(
						tfdiags.Sourceless(tfdiags.Warning, "unexpected warning", "this should be a warning and not removed"),
						tfdiags.Warning,
						func() tfdiags.DiagnosticExtraWrapper {
							return &addrs.CheckRuleDiagnosticExtra{
								CheckRule: addrs.NewCheckRule(addrs.AbsCheck{
									Module: addrs.RootModuleInstance,
									Check: addrs.Check{
										Name: "unexpected",
									},
								}, addrs.CheckDataResource, 0),
							}
						}))

				// Third, we'll add an unexpected warning from a check block
				// in a child module that should be propagated as an error.
				diags = diags.Append(
					tfdiags.Override(
						tfdiags.Sourceless(tfdiags.Error, "expected failure from child module", "this should be an error and not removed"),
						tfdiags.Warning,
						func() tfdiags.DiagnosticExtraWrapper {
							return &addrs.CheckRuleDiagnosticExtra{
								CheckRule: addrs.NewCheckRule(addrs.AbsCheck{
									Module: []addrs.ModuleInstanceStep{
										{
											Name: "child_module",
										},
									},
									Check: addrs.Check{
										Name: "expected",
									},
								}, addrs.CheckDataResource, 0),
							}
						}))

				// Fourth, we'll add an expected warning that should be removed.
				diags = diags.Append(
					tfdiags.Override(
						tfdiags.Sourceless(tfdiags.Error, "expected failure", "this should be removed"),
						tfdiags.Warning,
						func() tfdiags.DiagnosticExtraWrapper {
							return &addrs.CheckRuleDiagnosticExtra{
								CheckRule: addrs.NewCheckRule(addrs.AbsCheck{
									Module: addrs.RootModuleInstance,
									Check: addrs.Check{
										Name: "expected",
									},
								}, addrs.CheckDataResource, 0),
							}
						}))

				return diags
			}),
			Output: []output{
				{
					Description: tfdiags.Description{
						Summary: "unexpected failure",
						Detail:  "this should be an error and not removed",
					},
					Severity: tfdiags.Error,
				},
				{
					Description: tfdiags.Description{
						Summary: "unexpected warning",
						Detail:  "this should be a warning and not removed",
					},
					Severity: tfdiags.Warning,
				},
				{
					Description: tfdiags.Description{
						Summary: "expected failure from child module",
						Detail:  "this should be an error and not removed",
					},
					Severity: tfdiags.Error,
				},
			},
		},
	}
	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			var traversals []hcl.Traversal
			for _, ef := range tc.ExpectedFailures {
				traversal, diags := hclsyntax.ParseTraversalAbs([]byte(ef), "foo.tf", hcl.Pos{Line: 1, Column: 1})
				if diags.HasErrors() {
					t.Errorf("invalid expected failure %s: %v", ef, diags.Error())
				}
				traversals = append(traversals, traversal)
			}

			if t.Failed() {
				return
			}

			run := Run{
				Config: &configs.TestRun{
					ExpectFailures: traversals,
				},
			}

			out := run.ValidateExpectedFailures(tc.Input)
			ix := 0
			for ; ix < len(tc.Output); ix++ {
				expected := tc.Output[ix]

				if ix >= len(out) {
					t.Errorf("missing diagnostic at %d, expected: [%s] %s, %s", ix, expected.Severity, expected.Description.Summary, expected.Description.Detail)
					continue
				}

				actual := output{
					Description: out[ix].Description(),
					Severity:    out[ix].Severity(),
				}

				if diff := cmp.Diff(expected, actual); len(diff) > 0 {
					t.Errorf("mismatched diagnostic at %d:\n%s", ix, diff)
				}
			}

			for ; ix < len(out); ix++ {
				actual := out[ix]
				t.Errorf("additional diagnostic at %d: [%s] %s, %s", ix, actual.Severity(), actual.Description().Summary, actual.Description().Detail)
			}
		})
	}
}

func createDiagnostics(populate func(diags tfdiags.Diagnostics) tfdiags.Diagnostics) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	diags = populate(diags)
	return diags
}
