// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/terminal"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestTestHuman_Conclusion(t *testing.T) {
	tcs := map[string]struct {
		Suite    *moduletest.Suite
		Expected string
	}{
		"no tests": {
			Suite:    &moduletest.Suite{},
			Expected: "\nExecuted 0 tests.\n",
		},

		"only skipped tests": {
			Suite: &moduletest.Suite{
				Status: moduletest.Skip,
				Files: map[string]*moduletest.File{
					"descriptive_test_name.tftest.hcl": {
						Name:   "descriptive_test_name.tftest.hcl",
						Status: moduletest.Skip,
						Runs: []*moduletest.Run{
							{
								Name:   "test_one",
								Status: moduletest.Skip,
							},
							{
								Name:   "test_two",
								Status: moduletest.Skip,
							},
							{
								Name:   "test_three",
								Status: moduletest.Skip,
							},
						},
					},
					"other_descriptive_test_name.tftest.hcl": {
						Name:   "other_descriptive_test_name.tftest.hcl",
						Status: moduletest.Skip,
						Runs: []*moduletest.Run{
							{
								Name:   "test_one",
								Status: moduletest.Skip,
							},
							{
								Name:   "test_two",
								Status: moduletest.Skip,
							},
							{
								Name:   "test_three",
								Status: moduletest.Skip,
							},
						},
					},
				},
			},
			Expected: "\nExecuted 0 tests, 6 skipped.\n",
		},

		"only passed tests": {
			Suite: &moduletest.Suite{
				Status: moduletest.Pass,
				Files: map[string]*moduletest.File{
					"descriptive_test_name.tftest.hcl": {
						Name:   "descriptive_test_name.tftest.hcl",
						Status: moduletest.Pass,
						Runs: []*moduletest.Run{
							{
								Name:   "test_one",
								Status: moduletest.Pass,
							},
							{
								Name:   "test_two",
								Status: moduletest.Pass,
							},
							{
								Name:   "test_three",
								Status: moduletest.Pass,
							},
						},
					},
					"other_descriptive_test_name.tftest.hcl": {
						Name:   "other_descriptive_test_name.tftest.hcl",
						Status: moduletest.Pass,
						Runs: []*moduletest.Run{
							{
								Name:   "test_one",
								Status: moduletest.Pass,
							},
							{
								Name:   "test_two",
								Status: moduletest.Pass,
							},
							{
								Name:   "test_three",
								Status: moduletest.Pass,
							},
						},
					},
				},
			},
			Expected: "\nSuccess! 6 passed, 0 failed.\n",
		},

		"passed and skipped tests": {
			Suite: &moduletest.Suite{
				Status: moduletest.Pass,
				Files: map[string]*moduletest.File{
					"descriptive_test_name.tftest.hcl": {
						Name:   "descriptive_test_name.tftest.hcl",
						Status: moduletest.Pass,
						Runs: []*moduletest.Run{
							{
								Name:   "test_one",
								Status: moduletest.Pass,
							},
							{
								Name:   "test_two",
								Status: moduletest.Skip,
							},
							{
								Name:   "test_three",
								Status: moduletest.Pass,
							},
						},
					},
					"other_descriptive_test_name.tftest.hcl": {
						Name:   "other_descriptive_test_name.tftest.hcl",
						Status: moduletest.Pass,
						Runs: []*moduletest.Run{
							{
								Name:   "test_one",
								Status: moduletest.Skip,
							},
							{
								Name:   "test_two",
								Status: moduletest.Pass,
							},
							{
								Name:   "test_three",
								Status: moduletest.Pass,
							},
						},
					},
				},
			},
			Expected: "\nSuccess! 4 passed, 0 failed, 2 skipped.\n",
		},

		"only failed tests": {
			Suite: &moduletest.Suite{
				Status: moduletest.Fail,
				Files: map[string]*moduletest.File{
					"descriptive_test_name.tftest.hcl": {
						Name:   "descriptive_test_name.tftest.hcl",
						Status: moduletest.Fail,
						Runs: []*moduletest.Run{
							{
								Name:   "test_one",
								Status: moduletest.Fail,
							},
							{
								Name:   "test_two",
								Status: moduletest.Fail,
							},
							{
								Name:   "test_three",
								Status: moduletest.Fail,
							},
						},
					},
					"other_descriptive_test_name.tftest.hcl": {
						Name:   "other_descriptive_test_name.tftest.hcl",
						Status: moduletest.Fail,
						Runs: []*moduletest.Run{
							{
								Name:   "test_one",
								Status: moduletest.Fail,
							},
							{
								Name:   "test_two",
								Status: moduletest.Fail,
							},
							{
								Name:   "test_three",
								Status: moduletest.Fail,
							},
						},
					},
				},
			},
			Expected: "\nFailure! 0 passed, 6 failed.\n",
		},

		"failed and skipped tests": {
			Suite: &moduletest.Suite{
				Status: moduletest.Fail,
				Files: map[string]*moduletest.File{
					"descriptive_test_name.tftest.hcl": {
						Name:   "descriptive_test_name.tftest.hcl",
						Status: moduletest.Fail,
						Runs: []*moduletest.Run{
							{
								Name:   "test_one",
								Status: moduletest.Fail,
							},
							{
								Name:   "test_two",
								Status: moduletest.Skip,
							},
							{
								Name:   "test_three",
								Status: moduletest.Fail,
							},
						},
					},
					"other_descriptive_test_name.tftest.hcl": {
						Name:   "other_descriptive_test_name.tftest.hcl",
						Status: moduletest.Fail,
						Runs: []*moduletest.Run{
							{
								Name:   "test_one",
								Status: moduletest.Fail,
							},
							{
								Name:   "test_two",
								Status: moduletest.Fail,
							},
							{
								Name:   "test_three",
								Status: moduletest.Skip,
							},
						},
					},
				},
			},
			Expected: "\nFailure! 0 passed, 4 failed, 2 skipped.\n",
		},

		"failed, passed and skipped tests": {
			Suite: &moduletest.Suite{
				Status: moduletest.Fail,
				Files: map[string]*moduletest.File{
					"descriptive_test_name.tftest.hcl": {
						Name:   "descriptive_test_name.tftest.hcl",
						Status: moduletest.Fail,
						Runs: []*moduletest.Run{
							{
								Name:   "test_one",
								Status: moduletest.Fail,
							},
							{
								Name:   "test_two",
								Status: moduletest.Pass,
							},
							{
								Name:   "test_three",
								Status: moduletest.Skip,
							},
						},
					},
					"other_descriptive_test_name.tftest.hcl": {
						Name:   "other_descriptive_test_name.tftest.hcl",
						Status: moduletest.Fail,
						Runs: []*moduletest.Run{
							{
								Name:   "test_one",
								Status: moduletest.Skip,
							},
							{
								Name:   "test_two",
								Status: moduletest.Fail,
							},
							{
								Name:   "test_three",
								Status: moduletest.Pass,
							},
						},
					},
				},
			},
			Expected: "\nFailure! 2 passed, 2 failed, 2 skipped.\n",
		},

		"failed and errored tests": {
			Suite: &moduletest.Suite{
				Status: moduletest.Error,
				Files: map[string]*moduletest.File{
					"descriptive_test_name.tftest.hcl": {
						Name:   "descriptive_test_name.tftest.hcl",
						Status: moduletest.Error,
						Runs: []*moduletest.Run{
							{
								Name:   "test_one",
								Status: moduletest.Fail,
							},
							{
								Name:   "test_two",
								Status: moduletest.Error,
							},
							{
								Name:   "test_three",
								Status: moduletest.Fail,
							},
						},
					},
					"other_descriptive_test_name.tftest.hcl": {
						Name:   "other_descriptive_test_name.tftest.hcl",
						Status: moduletest.Error,
						Runs: []*moduletest.Run{
							{
								Name:   "test_one",
								Status: moduletest.Fail,
							},
							{
								Name:   "test_two",
								Status: moduletest.Error,
							},
							{
								Name:   "test_three",
								Status: moduletest.Error,
							},
						},
					},
				},
			},
			Expected: "\nFailure! 0 passed, 6 failed.\n",
		},

		"failed, errored, passed, and skipped tests": {
			Suite: &moduletest.Suite{
				Status: moduletest.Error,
				Files: map[string]*moduletest.File{
					"descriptive_test_name.tftest.hcl": {
						Name:   "descriptive_test_name.tftest.hcl",
						Status: moduletest.Fail,
						Runs: []*moduletest.Run{
							{
								Name:   "test_one",
								Status: moduletest.Pass,
							},
							{
								Name:   "test_two",
								Status: moduletest.Pass,
							},
							{
								Name:   "test_three",
								Status: moduletest.Fail,
							},
						},
					},
					"other_descriptive_test_name.tftest.hcl": {
						Name:   "other_descriptive_test_name.tftest.hcl",
						Status: moduletest.Error,
						Runs: []*moduletest.Run{
							{
								Name:   "test_one",
								Status: moduletest.Error,
							},
							{
								Name:   "test_two",
								Status: moduletest.Skip,
							},
							{
								Name:   "test_three",
								Status: moduletest.Skip,
							},
						},
					},
				},
			},
			Expected: "\nFailure! 2 passed, 2 failed, 2 skipped.\n",
		},
	}
	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {

			streams, done := terminal.StreamsForTesting(t)
			view := NewTest(arguments.ViewHuman, NewView(streams))

			view.Conclusion(tc.Suite)

			actual := done(t).Stdout()
			expected := tc.Expected
			if diff := cmp.Diff(expected, actual); len(diff) > 0 {
				t.Fatalf("expected:\n%s\nactual:\n%s\ndiff:\n%s", expected, actual, diff)
			}
		})
	}
}

func TestTestHuman_File(t *testing.T) {
	tcs := map[string]struct {
		File     *moduletest.File
		Progress moduletest.Progress
		Expected string
	}{
		"pass": {
			File:     &moduletest.File{Name: "main.tf", Status: moduletest.Pass},
			Progress: moduletest.Complete,
			Expected: "main.tf... pass\n",
		},

		"pending": {
			File:     &moduletest.File{Name: "main.tf", Status: moduletest.Pending},
			Progress: moduletest.Complete,
			Expected: "main.tf... pending\n",
		},

		"skip": {
			File:     &moduletest.File{Name: "main.tf", Status: moduletest.Skip},
			Progress: moduletest.Complete,
			Expected: "main.tf... skip\n",
		},

		"fail": {
			File:     &moduletest.File{Name: "main.tf", Status: moduletest.Fail},
			Progress: moduletest.Complete,
			Expected: "main.tf... fail\n",
		},

		"error": {
			File:     &moduletest.File{Name: "main.tf", Status: moduletest.Error},
			Progress: moduletest.Complete,
			Expected: "main.tf... fail\n",
		},
		"starting": {
			File:     &moduletest.File{Name: "main.tftest.hcl", Status: moduletest.Pending},
			Progress: moduletest.Starting,
			Expected: "main.tftest.hcl... in progress\n",
		},
		"tear_down": {
			File:     &moduletest.File{Name: "main.tftest.hcl", Status: moduletest.Pending},
			Progress: moduletest.TearDown,
			Expected: "main.tftest.hcl... tearing down\n",
		},
	}
	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {

			streams, done := terminal.StreamsForTesting(t)
			view := NewTest(arguments.ViewHuman, NewView(streams))

			view.File(tc.File, tc.Progress)

			actual := done(t).Stdout()
			expected := tc.Expected
			if diff := cmp.Diff(expected, actual); len(diff) > 0 {
				t.Fatalf("expected:\n%s\nactual:\n%s\ndiff:\n%s", expected, actual, diff)
			}
		})
	}
}

func TestTestHuman_Run(t *testing.T) {
	tcs := map[string]struct {
		Run      *moduletest.Run
		Progress moduletest.Progress
		StdOut   string
		StdErr   string
	}{
		"pass": {
			Run:      &moduletest.Run{Name: "run_block", Status: moduletest.Pass},
			Progress: moduletest.Complete,
			StdOut:   "  run \"run_block\"... pass\n",
		},

		"pass_with_diags": {
			Run: &moduletest.Run{
				Name:        "run_block",
				Status:      moduletest.Pass,
				Diagnostics: tfdiags.Diagnostics{tfdiags.Sourceless(tfdiags.Warning, "a warning occurred", "some warning happened during this test")},
			},
			Progress: moduletest.Complete,
			StdOut: `  run "run_block"... pass

Warning: a warning occurred

some warning happened during this test

`,
		},

		"pending": {
			Run:      &moduletest.Run{Name: "run_block", Status: moduletest.Pending},
			Progress: moduletest.Complete,
			StdOut:   "  run \"run_block\"... pending\n",
		},

		"skip": {
			Run:      &moduletest.Run{Name: "run_block", Status: moduletest.Skip},
			Progress: moduletest.Complete,
			StdOut:   "  run \"run_block\"... skip\n",
		},

		"fail": {
			Run:      &moduletest.Run{Name: "run_block", Status: moduletest.Fail},
			Progress: moduletest.Complete,
			StdOut:   "  run \"run_block\"... fail\n",
		},

		"fail_with_diags": {
			Run: &moduletest.Run{
				Name:   "run_block",
				Status: moduletest.Fail,
				Diagnostics: tfdiags.Diagnostics{
					tfdiags.Sourceless(tfdiags.Error, "a comparison failed", "details details details"),
					tfdiags.Sourceless(tfdiags.Error, "a second comparison failed", "other details"),
				},
			},
			Progress: moduletest.Complete,
			StdOut:   "  run \"run_block\"... fail\n",
			StdErr: `
Error: a comparison failed

details details details

Error: a second comparison failed

other details
`,
		},

		"error": {
			Run:      &moduletest.Run{Name: "run_block", Status: moduletest.Error},
			Progress: moduletest.Complete,
			StdOut:   "  run \"run_block\"... fail\n",
		},

		"error_with_diags": {
			Run: &moduletest.Run{
				Name:        "run_block",
				Status:      moduletest.Error,
				Diagnostics: tfdiags.Diagnostics{tfdiags.Sourceless(tfdiags.Error, "an error occurred", "something bad happened during this test")},
			},
			Progress: moduletest.Complete,
			StdOut:   "  run \"run_block\"... fail\n",
			StdErr: `
Error: an error occurred

something bad happened during this test
`,
		},
		"verbose_plan": {
			Run: &moduletest.Run{
				Name:   "run_block",
				Status: moduletest.Pass,
				Config: &configs.TestRun{
					Command: configs.PlanTestCommand,
				},
				Verbose: &moduletest.Verbose{
					Plan: &plans.Plan{
						Changes: &plans.ChangesSrc{
							Resources: []*plans.ResourceInstanceChangeSrc{
								{
									Addr: addrs.AbsResourceInstance{
										Module: addrs.RootModuleInstance,
										Resource: addrs.ResourceInstance{
											Resource: addrs.Resource{
												Mode: addrs.ManagedResourceMode,
												Type: "test_resource",
												Name: "creating",
											},
										},
									},
									PrevRunAddr: addrs.AbsResourceInstance{
										Module: addrs.RootModuleInstance,
										Resource: addrs.ResourceInstance{
											Resource: addrs.Resource{
												Mode: addrs.ManagedResourceMode,
												Type: "test_resource",
												Name: "creating",
											},
										},
									},
									ProviderAddr: addrs.AbsProviderConfig{
										Module: addrs.RootModule,
										Provider: addrs.Provider{
											Hostname:  addrs.DefaultProviderRegistryHost,
											Namespace: "hashicorp",
											Type:      "test",
										},
									},
									ChangeSrc: plans.ChangeSrc{
										Action: plans.Create,
										After: dynamicValue(
											t,
											cty.ObjectVal(map[string]cty.Value{
												"value": cty.StringVal("Hello, world!"),
											}),
											cty.Object(map[string]cty.Type{
												"value": cty.String,
											})),
									},
								},
							},
						},
					},
					State:  states.NewState(), // empty state
					Config: &configs.Config{},
					Providers: map[addrs.Provider]providers.ProviderSchema{
						addrs.Provider{
							Hostname:  addrs.DefaultProviderRegistryHost,
							Namespace: "hashicorp",
							Type:      "test",
						}: {
							ResourceTypes: map[string]providers.Schema{
								"test_resource": {
									Block: &configschema.Block{
										Attributes: map[string]*configschema.Attribute{
											"value": {
												Type: cty.String,
											},
										},
									},
								},
							},
						},
					},
				},
			},
			Progress: moduletest.Complete,
			StdOut: `  run "run_block"... pass

Terraform used the selected providers to generate the following execution
plan. Resource actions are indicated with the following symbols:
  + create

Terraform will perform the following actions:

  # test_resource.creating will be created
  + resource "test_resource" "creating" {
      + value = "Hello, world!"
    }

Plan: 1 to add, 0 to change, 0 to destroy.

`,
		},
		"verbose_apply": {
			Run: &moduletest.Run{
				Name:   "run_block",
				Status: moduletest.Pass,
				Config: &configs.TestRun{
					Command: configs.ApplyTestCommand,
				},
				Verbose: &moduletest.Verbose{
					Plan: &plans.Plan{}, // empty plan
					State: states.BuildState(func(state *states.SyncState) {
						state.SetResourceInstanceCurrent(
							addrs.AbsResourceInstance{
								Module: addrs.RootModuleInstance,
								Resource: addrs.ResourceInstance{
									Resource: addrs.Resource{
										Mode: addrs.ManagedResourceMode,
										Type: "test_resource",
										Name: "creating",
									},
								},
							},
							&states.ResourceInstanceObjectSrc{
								AttrsJSON: []byte(`{"value":"foobar"}`),
							},
							addrs.AbsProviderConfig{
								Module: addrs.RootModule,
								Provider: addrs.Provider{
									Hostname:  addrs.DefaultProviderRegistryHost,
									Namespace: "hashicorp",
									Type:      "test",
								},
							})
					}),
					Config: &configs.Config{},
					Providers: map[addrs.Provider]providers.ProviderSchema{
						addrs.Provider{
							Hostname:  addrs.DefaultProviderRegistryHost,
							Namespace: "hashicorp",
							Type:      "test",
						}: {
							ResourceTypes: map[string]providers.Schema{
								"test_resource": {
									Block: &configschema.Block{
										Attributes: map[string]*configschema.Attribute{
											"value": {
												Type: cty.String,
											},
										},
									},
								},
							},
						},
					},
				},
			},
			Progress: moduletest.Complete,
			StdOut: `  run "run_block"... pass

# test_resource.creating:
resource "test_resource" "creating" {
    value = "foobar"
}

`,
		},
		// These next three tests should print nothing, as we only report on
		// progress complete.
		"progress_starting": {
			Run:      &moduletest.Run{Name: "run_block", Status: moduletest.Pass},
			Progress: moduletest.Starting,
		},
		"progress_running": {
			Run:      &moduletest.Run{Name: "run_block", Status: moduletest.Pass},
			Progress: moduletest.Running,
		},
		"progress_teardown": {
			Run:      &moduletest.Run{Name: "run_block", Status: moduletest.Pass},
			Progress: moduletest.TearDown,
		},
	}
	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			file := &moduletest.File{
				Name: "main.tftest.hcl",
			}

			streams, done := terminal.StreamsForTesting(t)
			view := NewTest(arguments.ViewHuman, NewView(streams))

			view.Run(tc.Run, file, tc.Progress, 0)

			output := done(t)
			actual, expected := output.Stdout(), tc.StdOut
			if diff := cmp.Diff(expected, actual); len(diff) > 0 {
				t.Errorf("expected:\n%s\nactual:\n%s\ndiff:\n%s", expected, actual, diff)
			}

			actual, expected = output.Stderr(), tc.StdErr
			if diff := cmp.Diff(expected, actual); len(diff) > 0 {
				t.Errorf("expected:\n%s\nactual:\n%s\ndiff:\n%s", expected, actual, diff)
			}
		})
	}
}

func TestTestHuman_DestroySummary(t *testing.T) {
	tcs := map[string]struct {
		diags  tfdiags.Diagnostics
		run    *moduletest.Run
		file   *moduletest.File
		state  *states.State
		stdout string
		stderr string
	}{
		"empty": {
			diags: nil,
			file:  &moduletest.File{Name: "main.tftest.hcl"},
			state: states.NewState(),
		},
		"empty_state_only_warnings": {
			diags: tfdiags.Diagnostics{
				tfdiags.Sourceless(tfdiags.Warning, "first warning", "some thing not very bad happened"),
				tfdiags.Sourceless(tfdiags.Warning, "second warning", "some thing not very bad happened again"),
			},
			file:  &moduletest.File{Name: "main.tftest.hcl"},
			state: states.NewState(),
			stdout: `
Warning: first warning

some thing not very bad happened

Warning: second warning

some thing not very bad happened again
`,
		},
		"empty_state_with_errors": {
			diags: tfdiags.Diagnostics{
				tfdiags.Sourceless(tfdiags.Warning, "first warning", "some thing not very bad happened"),
				tfdiags.Sourceless(tfdiags.Warning, "second warning", "some thing not very bad happened again"),
				tfdiags.Sourceless(tfdiags.Error, "first error", "this time it is very bad"),
			},
			file:  &moduletest.File{Name: "main.tftest.hcl"},
			state: states.NewState(),
			stdout: `
Warning: first warning

some thing not very bad happened

Warning: second warning

some thing not very bad happened again
`,
			stderr: `Terraform encountered an error destroying resources created while executing
main.tftest.hcl.

Error: first error

this time it is very bad
`,
		},
		"error_from_run": {
			diags: tfdiags.Diagnostics{
				tfdiags.Sourceless(tfdiags.Error, "first error", "this time it is very bad"),
			},
			run:   &moduletest.Run{Name: "run_block"},
			file:  &moduletest.File{Name: "main.tftest.hcl"},
			state: states.NewState(),
			stderr: `Terraform encountered an error destroying resources created while executing
main.tftest.hcl/run_block.

Error: first error

this time it is very bad
`,
		},
		"state_only_warnings": {
			diags: tfdiags.Diagnostics{
				tfdiags.Sourceless(tfdiags.Warning, "first warning", "some thing not very bad happened"),
				tfdiags.Sourceless(tfdiags.Warning, "second warning", "some thing not very bad happened again"),
			},
			file: &moduletest.File{Name: "main.tftest.hcl"},
			state: states.BuildState(func(state *states.SyncState) {
				state.SetResourceInstanceCurrent(
					addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "test",
						Name: "foo",
					}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
					&states.ResourceInstanceObjectSrc{
						Status: states.ObjectReady,
					},
					addrs.AbsProviderConfig{
						Module:   addrs.RootModule,
						Provider: addrs.NewDefaultProvider("test"),
					})
				state.SetResourceInstanceCurrent(
					addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "test",
						Name: "bar",
					}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
					&states.ResourceInstanceObjectSrc{
						Status: states.ObjectReady,
					},
					addrs.AbsProviderConfig{
						Module:   addrs.RootModule,
						Provider: addrs.NewDefaultProvider("test"),
					})
				state.SetResourceInstanceDeposed(
					addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "test",
						Name: "bar",
					}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
					"0fcb640a",
					&states.ResourceInstanceObjectSrc{
						Status: states.ObjectReady,
					},
					addrs.AbsProviderConfig{
						Module:   addrs.RootModule,
						Provider: addrs.NewDefaultProvider("test"),
					})
			}),
			stdout: `
Warning: first warning

some thing not very bad happened

Warning: second warning

some thing not very bad happened again
`,
			stderr: `
Terraform left the following resources in state after executing
main.tftest.hcl, and they need to be cleaned up manually:
  - test.bar
  - test.bar (0fcb640a)
  - test.foo
`,
		},
		"state_with_errors": {
			diags: tfdiags.Diagnostics{
				tfdiags.Sourceless(tfdiags.Warning, "first warning", "some thing not very bad happened"),
				tfdiags.Sourceless(tfdiags.Warning, "second warning", "some thing not very bad happened again"),
				tfdiags.Sourceless(tfdiags.Error, "first error", "this time it is very bad"),
			},
			file: &moduletest.File{Name: "main.tftest.hcl"},
			state: states.BuildState(func(state *states.SyncState) {
				state.SetResourceInstanceCurrent(
					addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "test",
						Name: "foo",
					}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
					&states.ResourceInstanceObjectSrc{
						Status: states.ObjectReady,
					},
					addrs.AbsProviderConfig{
						Module:   addrs.RootModule,
						Provider: addrs.NewDefaultProvider("test"),
					})
				state.SetResourceInstanceCurrent(
					addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "test",
						Name: "bar",
					}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
					&states.ResourceInstanceObjectSrc{
						Status: states.ObjectReady,
					},
					addrs.AbsProviderConfig{
						Module:   addrs.RootModule,
						Provider: addrs.NewDefaultProvider("test"),
					})
				state.SetResourceInstanceDeposed(
					addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "test",
						Name: "bar",
					}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
					"0fcb640a",
					&states.ResourceInstanceObjectSrc{
						Status: states.ObjectReady,
					},
					addrs.AbsProviderConfig{
						Module:   addrs.RootModule,
						Provider: addrs.NewDefaultProvider("test"),
					})
			}),
			stdout: `
Warning: first warning

some thing not very bad happened

Warning: second warning

some thing not very bad happened again
`,
			stderr: `Terraform encountered an error destroying resources created while executing
main.tftest.hcl.

Error: first error

this time it is very bad

Terraform left the following resources in state after executing
main.tftest.hcl, and they need to be cleaned up manually:
  - test.bar
  - test.bar (0fcb640a)
  - test.foo
`,
		},
	}
	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			streams, done := terminal.StreamsForTesting(t)
			view := NewTest(arguments.ViewHuman, NewView(streams))

			view.DestroySummary(tc.diags, tc.run, tc.file, tc.state)

			output := done(t)
			actual, expected := output.Stdout(), tc.stdout
			if diff := cmp.Diff(expected, actual); len(diff) > 0 {
				t.Errorf("expected:\n%s\nactual:\n%s\ndiff:\n%s", expected, actual, diff)
			}

			actual, expected = output.Stderr(), tc.stderr
			if diff := cmp.Diff(expected, actual); len(diff) > 0 {
				t.Errorf("expected:\n%s\nactual:\n%s\ndiff:\n%s", expected, actual, diff)
			}
		})
	}
}

func TestTestHuman_FatalInterruptSummary(t *testing.T) {
	tcs := map[string]struct {
		states  map[*moduletest.Run]*states.State
		run     *moduletest.Run
		created []*plans.ResourceInstanceChangeSrc
		want    string
	}{
		"no_state_only_plan": {
			states: make(map[*moduletest.Run]*states.State),
			run: &moduletest.Run{
				Config: &configs.TestRun{},
				Name:   "run_block",
			},
			created: []*plans.ResourceInstanceChangeSrc{
				{
					Addr: addrs.AbsResourceInstance{
						Module: addrs.RootModuleInstance,
						Resource: addrs.ResourceInstance{
							Resource: addrs.Resource{
								Mode: addrs.ManagedResourceMode,
								Type: "test_instance",
								Name: "one",
							},
						},
					},
					ChangeSrc: plans.ChangeSrc{
						Action: plans.Create,
					},
				},
				{
					Addr: addrs.AbsResourceInstance{
						Module: addrs.RootModuleInstance,
						Resource: addrs.ResourceInstance{
							Resource: addrs.Resource{
								Mode: addrs.ManagedResourceMode,
								Type: "test_instance",
								Name: "two",
							},
						},
					},
					ChangeSrc: plans.ChangeSrc{
						Action: plans.Create,
					},
				},
			},
			want: `
Terraform was interrupted while executing main.tftest.hcl, and may not have
performed the expected cleanup operations.

Terraform was in the process of creating the following resources for
"run_block" from the module under test, and they may not have been destroyed:
  - test_instance.one
  - test_instance.two
`,
		},
		"file_state_no_plan": {
			states: map[*moduletest.Run]*states.State{
				nil: states.BuildState(func(state *states.SyncState) {
					state.SetResourceInstanceCurrent(
						addrs.AbsResourceInstance{
							Module: addrs.RootModuleInstance,
							Resource: addrs.ResourceInstance{
								Resource: addrs.Resource{
									Mode: addrs.ManagedResourceMode,
									Type: "test_instance",
									Name: "one",
								},
							},
						},
						&states.ResourceInstanceObjectSrc{},
						addrs.AbsProviderConfig{})

					state.SetResourceInstanceCurrent(
						addrs.AbsResourceInstance{
							Module: addrs.RootModuleInstance,
							Resource: addrs.ResourceInstance{
								Resource: addrs.Resource{
									Mode: addrs.ManagedResourceMode,
									Type: "test_instance",
									Name: "two",
								},
							},
						},
						&states.ResourceInstanceObjectSrc{},
						addrs.AbsProviderConfig{})
				}),
			},
			created: nil,
			want: `
Terraform was interrupted while executing main.tftest.hcl, and may not have
performed the expected cleanup operations.

Terraform has already created the following resources from the module under
test:
  - test_instance.one
  - test_instance.two
`,
		},
		"run_states_no_plan": {
			states: map[*moduletest.Run]*states.State{
				&moduletest.Run{
					Name: "setup_block",
					Config: &configs.TestRun{
						Module: &configs.TestRunModuleCall{
							Source: addrs.ModuleSourceLocal("../setup"),
						},
					},
				}: states.BuildState(func(state *states.SyncState) {
					state.SetResourceInstanceCurrent(
						addrs.AbsResourceInstance{
							Module: addrs.RootModuleInstance,
							Resource: addrs.ResourceInstance{
								Resource: addrs.Resource{
									Mode: addrs.ManagedResourceMode,
									Type: "test_instance",
									Name: "one",
								},
							},
						},
						&states.ResourceInstanceObjectSrc{},
						addrs.AbsProviderConfig{})

					state.SetResourceInstanceCurrent(
						addrs.AbsResourceInstance{
							Module: addrs.RootModuleInstance,
							Resource: addrs.ResourceInstance{
								Resource: addrs.Resource{
									Mode: addrs.ManagedResourceMode,
									Type: "test_instance",
									Name: "two",
								},
							},
						},
						&states.ResourceInstanceObjectSrc{},
						addrs.AbsProviderConfig{})
				}),
			},
			created: nil,
			want: `
Terraform was interrupted while executing main.tftest.hcl, and may not have
performed the expected cleanup operations.

Terraform has already created the following resources for "setup_block" from
"../setup":
  - test_instance.one
  - test_instance.two
`,
		},
		"all_states_with_plan": {
			states: map[*moduletest.Run]*states.State{
				&moduletest.Run{
					Name: "setup_block",
					Config: &configs.TestRun{
						Module: &configs.TestRunModuleCall{
							Source: addrs.ModuleSourceLocal("../setup"),
						},
					},
				}: states.BuildState(func(state *states.SyncState) {
					state.SetResourceInstanceCurrent(
						addrs.AbsResourceInstance{
							Module: addrs.RootModuleInstance,
							Resource: addrs.ResourceInstance{
								Resource: addrs.Resource{
									Mode: addrs.ManagedResourceMode,
									Type: "test_instance",
									Name: "setup_one",
								},
							},
						},
						&states.ResourceInstanceObjectSrc{},
						addrs.AbsProviderConfig{})

					state.SetResourceInstanceCurrent(
						addrs.AbsResourceInstance{
							Module: addrs.RootModuleInstance,
							Resource: addrs.ResourceInstance{
								Resource: addrs.Resource{
									Mode: addrs.ManagedResourceMode,
									Type: "test_instance",
									Name: "setup_two",
								},
							},
						},
						&states.ResourceInstanceObjectSrc{},
						addrs.AbsProviderConfig{})
				}),
				nil: states.BuildState(func(state *states.SyncState) {
					state.SetResourceInstanceCurrent(
						addrs.AbsResourceInstance{
							Module: addrs.RootModuleInstance,
							Resource: addrs.ResourceInstance{
								Resource: addrs.Resource{
									Mode: addrs.ManagedResourceMode,
									Type: "test_instance",
									Name: "one",
								},
							},
						},
						&states.ResourceInstanceObjectSrc{},
						addrs.AbsProviderConfig{})

					state.SetResourceInstanceCurrent(
						addrs.AbsResourceInstance{
							Module: addrs.RootModuleInstance,
							Resource: addrs.ResourceInstance{
								Resource: addrs.Resource{
									Mode: addrs.ManagedResourceMode,
									Type: "test_instance",
									Name: "two",
								},
							},
						},
						&states.ResourceInstanceObjectSrc{},
						addrs.AbsProviderConfig{})
				}),
			},
			created: []*plans.ResourceInstanceChangeSrc{
				{
					Addr: addrs.AbsResourceInstance{
						Module: addrs.RootModuleInstance,
						Resource: addrs.ResourceInstance{
							Resource: addrs.Resource{
								Mode: addrs.ManagedResourceMode,
								Type: "test_instance",
								Name: "new_one",
							},
						},
					},
					ChangeSrc: plans.ChangeSrc{
						Action: plans.Create,
					},
				},
				{
					Addr: addrs.AbsResourceInstance{
						Module: addrs.RootModuleInstance,
						Resource: addrs.ResourceInstance{
							Resource: addrs.Resource{
								Mode: addrs.ManagedResourceMode,
								Type: "test_instance",
								Name: "new_two",
							},
						},
					},
					ChangeSrc: plans.ChangeSrc{
						Action: plans.Create,
					},
				},
			},
			run: &moduletest.Run{
				Config: &configs.TestRun{},
				Name:   "run_block",
			},
			want: `
Terraform was interrupted while executing main.tftest.hcl, and may not have
performed the expected cleanup operations.

Terraform has already created the following resources from the module under
test:
  - test_instance.one
  - test_instance.two

Terraform has already created the following resources for "setup_block" from
"../setup":
  - test_instance.setup_one
  - test_instance.setup_two

Terraform was in the process of creating the following resources for
"run_block" from the module under test, and they may not have been destroyed:
  - test_instance.new_one
  - test_instance.new_two
`,
		},
	}
	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			streams, done := terminal.StreamsForTesting(t)
			view := NewTest(arguments.ViewHuman, NewView(streams))

			file := &moduletest.File{
				Name: "main.tftest.hcl",
				Runs: func() []*moduletest.Run {
					var runs []*moduletest.Run
					for run := range tc.states {
						if run != nil {
							runs = append(runs, run)
						}
					}
					return runs
				}(),
			}

			view.FatalInterruptSummary(tc.run, file, tc.states, tc.created)
			actual, expected := done(t).Stderr(), tc.want
			if diff := cmp.Diff(expected, actual); len(diff) > 0 {
				t.Errorf("expected:\n%s\nactual:\n%s\ndiff:\n%s", expected, actual, diff)
			}
		})
	}
}

func TestTestJSON_Abstract(t *testing.T) {
	tcs := map[string]struct {
		suite *moduletest.Suite
		want  []map[string]interface{}
	}{
		"single": {
			suite: &moduletest.Suite{
				Files: map[string]*moduletest.File{
					"main.tftest.hcl": {
						Runs: []*moduletest.Run{
							{
								Name: "setup",
							},
						},
					},
				},
			},
			want: []map[string]interface{}{
				{
					"@level":   "info",
					"@message": "Found 1 file and 1 run block",
					"@module":  "terraform.ui",
					"test_abstract": map[string]interface{}{
						"main.tftest.hcl": []interface{}{
							"setup",
						},
					},
					"type": "test_abstract",
				},
			},
		},
		"plural": {
			suite: &moduletest.Suite{
				Files: map[string]*moduletest.File{
					"main.tftest.hcl": {
						Runs: []*moduletest.Run{
							{
								Name: "setup",
							},
							{
								Name: "test",
							},
						},
					},
					"other.tftest.hcl": {
						Runs: []*moduletest.Run{
							{
								Name: "test",
							},
						},
					},
				},
			},
			want: []map[string]interface{}{
				{
					"@level":   "info",
					"@message": "Found 2 files and 3 run blocks",
					"@module":  "terraform.ui",
					"test_abstract": map[string]interface{}{
						"main.tftest.hcl": []interface{}{
							"setup",
							"test",
						},
						"other.tftest.hcl": []interface{}{
							"test",
						},
					},
					"type": "test_abstract",
				},
			},
		},
	}
	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			streams, done := terminal.StreamsForTesting(t)
			view := NewTest(arguments.ViewJSON, NewView(streams))

			view.Abstract(tc.suite)
			testJSONViewOutputEquals(t, done(t).All(), tc.want)
		})
	}
}

func TestTestJSON_Conclusion(t *testing.T) {
	tcs := map[string]struct {
		suite *moduletest.Suite
		want  []map[string]interface{}
	}{
		"no tests": {
			suite: &moduletest.Suite{},
			want: []map[string]interface{}{
				{
					"@level":   "info",
					"@message": "Executed 0 tests.",
					"@module":  "terraform.ui",
					"test_summary": map[string]interface{}{
						"status":  "pending",
						"errored": 0.0,
						"failed":  0.0,
						"passed":  0.0,
						"skipped": 0.0,
					},
					"type": "test_summary",
				},
			},
		},

		"only skipped tests": {
			suite: &moduletest.Suite{
				Status: moduletest.Skip,
				Files: map[string]*moduletest.File{
					"descriptive_test_name.tftest.hcl": {
						Name:   "descriptive_test_name.tftest.hcl",
						Status: moduletest.Skip,
						Runs: []*moduletest.Run{
							{
								Name:   "test_one",
								Status: moduletest.Skip,
							},
							{
								Name:   "test_two",
								Status: moduletest.Skip,
							},
							{
								Name:   "test_three",
								Status: moduletest.Skip,
							},
						},
					},
					"other_descriptive_test_name.tftest.hcl": {
						Name:   "other_descriptive_test_name.tftest.hcl",
						Status: moduletest.Skip,
						Runs: []*moduletest.Run{
							{
								Name:   "test_one",
								Status: moduletest.Skip,
							},
							{
								Name:   "test_two",
								Status: moduletest.Skip,
							},
							{
								Name:   "test_three",
								Status: moduletest.Skip,
							},
						},
					},
				},
			},
			want: []map[string]interface{}{
				{
					"@level":   "info",
					"@message": "Executed 0 tests, 6 skipped.",
					"@module":  "terraform.ui",
					"test_summary": map[string]interface{}{
						"status":  "skip",
						"errored": 0.0,
						"failed":  0.0,
						"passed":  0.0,
						"skipped": 6.0,
					},
					"type": "test_summary",
				},
			},
		},

		"only passed tests": {
			suite: &moduletest.Suite{
				Status: moduletest.Pass,
				Files: map[string]*moduletest.File{
					"descriptive_test_name.tftest.hcl": {
						Name:   "descriptive_test_name.tftest.hcl",
						Status: moduletest.Pass,
						Runs: []*moduletest.Run{
							{
								Name:   "test_one",
								Status: moduletest.Pass,
							},
							{
								Name:   "test_two",
								Status: moduletest.Pass,
							},
							{
								Name:   "test_three",
								Status: moduletest.Pass,
							},
						},
					},
					"other_descriptive_test_name.tftest.hcl": {
						Name:   "other_descriptive_test_name.tftest.hcl",
						Status: moduletest.Pass,
						Runs: []*moduletest.Run{
							{
								Name:   "test_one",
								Status: moduletest.Pass,
							},
							{
								Name:   "test_two",
								Status: moduletest.Pass,
							},
							{
								Name:   "test_three",
								Status: moduletest.Pass,
							},
						},
					},
				},
			},
			want: []map[string]interface{}{
				{
					"@level":   "info",
					"@message": "Success! 6 passed, 0 failed.",
					"@module":  "terraform.ui",
					"test_summary": map[string]interface{}{
						"status":  "pass",
						"errored": 0.0,
						"failed":  0.0,
						"passed":  6.0,
						"skipped": 0.0,
					},
					"type": "test_summary",
				},
			},
		},

		"passed and skipped tests": {
			suite: &moduletest.Suite{
				Status: moduletest.Pass,
				Files: map[string]*moduletest.File{
					"descriptive_test_name.tftest.hcl": {
						Name:   "descriptive_test_name.tftest.hcl",
						Status: moduletest.Pass,
						Runs: []*moduletest.Run{
							{
								Name:   "test_one",
								Status: moduletest.Pass,
							},
							{
								Name:   "test_two",
								Status: moduletest.Skip,
							},
							{
								Name:   "test_three",
								Status: moduletest.Pass,
							},
						},
					},
					"other_descriptive_test_name.tftest.hcl": {
						Name:   "other_descriptive_test_name.tftest.hcl",
						Status: moduletest.Pass,
						Runs: []*moduletest.Run{
							{
								Name:   "test_one",
								Status: moduletest.Skip,
							},
							{
								Name:   "test_two",
								Status: moduletest.Pass,
							},
							{
								Name:   "test_three",
								Status: moduletest.Pass,
							},
						},
					},
				},
			},
			want: []map[string]interface{}{
				{
					"@level":   "info",
					"@message": "Success! 4 passed, 0 failed, 2 skipped.",
					"@module":  "terraform.ui",
					"test_summary": map[string]interface{}{
						"status":  "pass",
						"errored": 0.0,
						"failed":  0.0,
						"passed":  4.0,
						"skipped": 2.0,
					},
					"type": "test_summary",
				},
			},
		},

		"only failed tests": {
			suite: &moduletest.Suite{
				Status: moduletest.Fail,
				Files: map[string]*moduletest.File{
					"descriptive_test_name.tftest.hcl": {
						Name:   "descriptive_test_name.tftest.hcl",
						Status: moduletest.Fail,
						Runs: []*moduletest.Run{
							{
								Name:   "test_one",
								Status: moduletest.Fail,
							},
							{
								Name:   "test_two",
								Status: moduletest.Fail,
							},
							{
								Name:   "test_three",
								Status: moduletest.Fail,
							},
						},
					},
					"other_descriptive_test_name.tftest.hcl": {
						Name:   "other_descriptive_test_name.tftest.hcl",
						Status: moduletest.Fail,
						Runs: []*moduletest.Run{
							{
								Name:   "test_one",
								Status: moduletest.Fail,
							},
							{
								Name:   "test_two",
								Status: moduletest.Fail,
							},
							{
								Name:   "test_three",
								Status: moduletest.Fail,
							},
						},
					},
				},
			},
			want: []map[string]interface{}{
				{
					"@level":   "info",
					"@message": "Failure! 0 passed, 6 failed.",
					"@module":  "terraform.ui",
					"test_summary": map[string]interface{}{
						"status":  "fail",
						"errored": 0.0,
						"failed":  6.0,
						"passed":  0.0,
						"skipped": 0.0,
					},
					"type": "test_summary",
				},
			},
		},

		"failed and skipped tests": {
			suite: &moduletest.Suite{
				Status: moduletest.Fail,
				Files: map[string]*moduletest.File{
					"descriptive_test_name.tftest.hcl": {
						Name:   "descriptive_test_name.tftest.hcl",
						Status: moduletest.Fail,
						Runs: []*moduletest.Run{
							{
								Name:   "test_one",
								Status: moduletest.Fail,
							},
							{
								Name:   "test_two",
								Status: moduletest.Skip,
							},
							{
								Name:   "test_three",
								Status: moduletest.Fail,
							},
						},
					},
					"other_descriptive_test_name.tftest.hcl": {
						Name:   "other_descriptive_test_name.tftest.hcl",
						Status: moduletest.Fail,
						Runs: []*moduletest.Run{
							{
								Name:   "test_one",
								Status: moduletest.Fail,
							},
							{
								Name:   "test_two",
								Status: moduletest.Fail,
							},
							{
								Name:   "test_three",
								Status: moduletest.Skip,
							},
						},
					},
				},
			},
			want: []map[string]interface{}{
				{
					"@level":   "info",
					"@message": "Failure! 0 passed, 4 failed, 2 skipped.",
					"@module":  "terraform.ui",
					"test_summary": map[string]interface{}{
						"status":  "fail",
						"errored": 0.0,
						"failed":  4.0,
						"passed":  0.0,
						"skipped": 2.0,
					},
					"type": "test_summary",
				},
			},
		},

		"failed, passed and skipped tests": {
			suite: &moduletest.Suite{
				Status: moduletest.Fail,
				Files: map[string]*moduletest.File{
					"descriptive_test_name.tftest.hcl": {
						Name:   "descriptive_test_name.tftest.hcl",
						Status: moduletest.Fail,
						Runs: []*moduletest.Run{
							{
								Name:   "test_one",
								Status: moduletest.Fail,
							},
							{
								Name:   "test_two",
								Status: moduletest.Pass,
							},
							{
								Name:   "test_three",
								Status: moduletest.Skip,
							},
						},
					},
					"other_descriptive_test_name.tftest.hcl": {
						Name:   "other_descriptive_test_name.tftest.hcl",
						Status: moduletest.Fail,
						Runs: []*moduletest.Run{
							{
								Name:   "test_one",
								Status: moduletest.Skip,
							},
							{
								Name:   "test_two",
								Status: moduletest.Fail,
							},
							{
								Name:   "test_three",
								Status: moduletest.Pass,
							},
						},
					},
				},
			},
			want: []map[string]interface{}{
				{
					"@level":   "info",
					"@message": "Failure! 2 passed, 2 failed, 2 skipped.",
					"@module":  "terraform.ui",
					"test_summary": map[string]interface{}{
						"status":  "fail",
						"errored": 0.0,
						"failed":  2.0,
						"passed":  2.0,
						"skipped": 2.0,
					},
					"type": "test_summary",
				},
			},
		},

		"failed and errored tests": {
			suite: &moduletest.Suite{
				Status: moduletest.Error,
				Files: map[string]*moduletest.File{
					"descriptive_test_name.tftest.hcl": {
						Name:   "descriptive_test_name.tftest.hcl",
						Status: moduletest.Error,
						Runs: []*moduletest.Run{
							{
								Name:   "test_one",
								Status: moduletest.Fail,
							},
							{
								Name:   "test_two",
								Status: moduletest.Error,
							},
							{
								Name:   "test_three",
								Status: moduletest.Fail,
							},
						},
					},
					"other_descriptive_test_name.tftest.hcl": {
						Name:   "other_descriptive_test_name.tftest.hcl",
						Status: moduletest.Error,
						Runs: []*moduletest.Run{
							{
								Name:   "test_one",
								Status: moduletest.Fail,
							},
							{
								Name:   "test_two",
								Status: moduletest.Error,
							},
							{
								Name:   "test_three",
								Status: moduletest.Error,
							},
						},
					},
				},
			},
			want: []map[string]interface{}{
				{
					"@level":   "info",
					"@message": "Failure! 0 passed, 6 failed.",
					"@module":  "terraform.ui",
					"test_summary": map[string]interface{}{
						"status":  "error",
						"errored": 3.0,
						"failed":  3.0,
						"passed":  0.0,
						"skipped": 0.0,
					},
					"type": "test_summary",
				},
			},
		},

		"failed, errored, passed, and skipped tests": {
			suite: &moduletest.Suite{
				Status: moduletest.Error,
				Files: map[string]*moduletest.File{
					"descriptive_test_name.tftest.hcl": {
						Name:   "descriptive_test_name.tftest.hcl",
						Status: moduletest.Fail,
						Runs: []*moduletest.Run{
							{
								Name:   "test_one",
								Status: moduletest.Pass,
							},
							{
								Name:   "test_two",
								Status: moduletest.Pass,
							},
							{
								Name:   "test_three",
								Status: moduletest.Fail,
							},
						},
					},
					"other_descriptive_test_name.tftest.hcl": {
						Name:   "other_descriptive_test_name.tftest.hcl",
						Status: moduletest.Error,
						Runs: []*moduletest.Run{
							{
								Name:   "test_one",
								Status: moduletest.Error,
							},
							{
								Name:   "test_two",
								Status: moduletest.Skip,
							},
							{
								Name:   "test_three",
								Status: moduletest.Skip,
							},
						},
					},
				},
			},
			want: []map[string]interface{}{
				{
					"@level":   "info",
					"@message": "Failure! 2 passed, 2 failed, 2 skipped.",
					"@module":  "terraform.ui",
					"test_summary": map[string]interface{}{
						"status":  "error",
						"errored": 1.0,
						"failed":  1.0,
						"passed":  2.0,
						"skipped": 2.0,
					},
					"type": "test_summary",
				},
			},
		},
	}
	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			streams, done := terminal.StreamsForTesting(t)
			view := NewTest(arguments.ViewJSON, NewView(streams))

			view.Conclusion(tc.suite)
			testJSONViewOutputEquals(t, done(t).All(), tc.want)
		})
	}
}

func TestTestJSON_DestroySummary(t *testing.T) {
	tcs := map[string]struct {
		file  *moduletest.File
		run   *moduletest.Run
		state *states.State
		diags tfdiags.Diagnostics
		want  []map[string]interface{}
	}{
		"empty_state_only_warnings": {
			diags: tfdiags.Diagnostics{
				tfdiags.Sourceless(tfdiags.Warning, "first warning", "something not very bad happened"),
				tfdiags.Sourceless(tfdiags.Warning, "second warning", "something not very bad happened again"),
			},
			file:  &moduletest.File{Name: "main.tftest.hcl"},
			state: states.NewState(),
			want: []map[string]interface{}{
				{
					"@level":    "warn",
					"@message":  "Warning: first warning",
					"@module":   "terraform.ui",
					"@testfile": "main.tftest.hcl",
					"diagnostic": map[string]interface{}{
						"detail":   "something not very bad happened",
						"severity": "warning",
						"summary":  "first warning",
					},
					"type": "diagnostic",
				},
				{
					"@level":    "warn",
					"@message":  "Warning: second warning",
					"@module":   "terraform.ui",
					"@testfile": "main.tftest.hcl",
					"diagnostic": map[string]interface{}{
						"detail":   "something not very bad happened again",
						"severity": "warning",
						"summary":  "second warning",
					},
					"type": "diagnostic",
				},
			},
		},
		"empty_state_with_errors": {
			diags: tfdiags.Diagnostics{
				tfdiags.Sourceless(tfdiags.Warning, "first warning", "something not very bad happened"),
				tfdiags.Sourceless(tfdiags.Warning, "second warning", "something not very bad happened again"),
				tfdiags.Sourceless(tfdiags.Error, "first error", "this time it is very bad"),
			},
			file:  &moduletest.File{Name: "main.tftest.hcl"},
			state: states.NewState(),
			want: []map[string]interface{}{
				{
					"@level":    "warn",
					"@message":  "Warning: first warning",
					"@module":   "terraform.ui",
					"@testfile": "main.tftest.hcl",
					"diagnostic": map[string]interface{}{
						"detail":   "something not very bad happened",
						"severity": "warning",
						"summary":  "first warning",
					},
					"type": "diagnostic",
				},
				{
					"@level":    "warn",
					"@message":  "Warning: second warning",
					"@module":   "terraform.ui",
					"@testfile": "main.tftest.hcl",
					"diagnostic": map[string]interface{}{
						"detail":   "something not very bad happened again",
						"severity": "warning",
						"summary":  "second warning",
					},
					"type": "diagnostic",
				},
				{
					"@level":    "error",
					"@message":  "Error: first error",
					"@module":   "terraform.ui",
					"@testfile": "main.tftest.hcl",
					"diagnostic": map[string]interface{}{
						"detail":   "this time it is very bad",
						"severity": "error",
						"summary":  "first error",
					},
					"type": "diagnostic",
				},
			},
		},
		"state_from_run": {
			file: &moduletest.File{Name: "main.tftest.hcl"},
			run:  &moduletest.Run{Name: "run_block"},
			state: states.BuildState(func(state *states.SyncState) {
				state.SetResourceInstanceCurrent(
					addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "test",
						Name: "foo",
					}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
					&states.ResourceInstanceObjectSrc{
						Status: states.ObjectReady,
					},
					addrs.AbsProviderConfig{
						Module:   addrs.RootModule,
						Provider: addrs.NewDefaultProvider("test"),
					})
			}),
			want: []map[string]interface{}{
				{
					"@level":    "error",
					"@message":  "Terraform left some resources in state after executing main.tftest.hcl/run_block, they need to be cleaned up manually.",
					"@module":   "terraform.ui",
					"@testfile": "main.tftest.hcl",
					"@testrun":  "run_block",
					"test_cleanup": map[string]interface{}{
						"failed_resources": []interface{}{
							map[string]interface{}{
								"instance": "test.foo",
							},
						},
					},
					"type": "test_cleanup",
				},
			},
		},
		"state_only_warnings": {
			diags: tfdiags.Diagnostics{
				tfdiags.Sourceless(tfdiags.Warning, "first warning", "something not very bad happened"),
				tfdiags.Sourceless(tfdiags.Warning, "second warning", "something not very bad happened again"),
			},
			file: &moduletest.File{Name: "main.tftest.hcl"},
			state: states.BuildState(func(state *states.SyncState) {
				state.SetResourceInstanceCurrent(
					addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "test",
						Name: "foo",
					}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
					&states.ResourceInstanceObjectSrc{
						Status: states.ObjectReady,
					},
					addrs.AbsProviderConfig{
						Module:   addrs.RootModule,
						Provider: addrs.NewDefaultProvider("test"),
					})
				state.SetResourceInstanceCurrent(
					addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "test",
						Name: "bar",
					}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
					&states.ResourceInstanceObjectSrc{
						Status: states.ObjectReady,
					},
					addrs.AbsProviderConfig{
						Module:   addrs.RootModule,
						Provider: addrs.NewDefaultProvider("test"),
					})
				state.SetResourceInstanceDeposed(
					addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "test",
						Name: "bar",
					}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
					"0fcb640a",
					&states.ResourceInstanceObjectSrc{
						Status: states.ObjectReady,
					},
					addrs.AbsProviderConfig{
						Module:   addrs.RootModule,
						Provider: addrs.NewDefaultProvider("test"),
					})
			}),
			want: []map[string]interface{}{
				{
					"@level":    "error",
					"@message":  "Terraform left some resources in state after executing main.tftest.hcl, they need to be cleaned up manually.",
					"@module":   "terraform.ui",
					"@testfile": "main.tftest.hcl",
					"test_cleanup": map[string]interface{}{
						"failed_resources": []interface{}{
							map[string]interface{}{
								"instance": "test.bar",
							},
							map[string]interface{}{
								"instance":    "test.bar",
								"deposed_key": "0fcb640a",
							},
							map[string]interface{}{
								"instance": "test.foo",
							},
						},
					},
					"type": "test_cleanup",
				},
				{
					"@level":    "warn",
					"@message":  "Warning: first warning",
					"@module":   "terraform.ui",
					"@testfile": "main.tftest.hcl",
					"diagnostic": map[string]interface{}{
						"detail":   "something not very bad happened",
						"severity": "warning",
						"summary":  "first warning",
					},
					"type": "diagnostic",
				},
				{
					"@level":    "warn",
					"@message":  "Warning: second warning",
					"@module":   "terraform.ui",
					"@testfile": "main.tftest.hcl",
					"diagnostic": map[string]interface{}{
						"detail":   "something not very bad happened again",
						"severity": "warning",
						"summary":  "second warning",
					},
					"type": "diagnostic",
				},
			},
		},
		"state_with_errors": {
			diags: tfdiags.Diagnostics{
				tfdiags.Sourceless(tfdiags.Warning, "first warning", "something not very bad happened"),
				tfdiags.Sourceless(tfdiags.Warning, "second warning", "something not very bad happened again"),
				tfdiags.Sourceless(tfdiags.Error, "first error", "this time it is very bad"),
			},
			file: &moduletest.File{Name: "main.tftest.hcl"},
			state: states.BuildState(func(state *states.SyncState) {
				state.SetResourceInstanceCurrent(
					addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "test",
						Name: "foo",
					}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
					&states.ResourceInstanceObjectSrc{
						Status: states.ObjectReady,
					},
					addrs.AbsProviderConfig{
						Module:   addrs.RootModule,
						Provider: addrs.NewDefaultProvider("test"),
					})
				state.SetResourceInstanceCurrent(
					addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "test",
						Name: "bar",
					}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
					&states.ResourceInstanceObjectSrc{
						Status: states.ObjectReady,
					},
					addrs.AbsProviderConfig{
						Module:   addrs.RootModule,
						Provider: addrs.NewDefaultProvider("test"),
					})
				state.SetResourceInstanceDeposed(
					addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "test",
						Name: "bar",
					}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
					"0fcb640a",
					&states.ResourceInstanceObjectSrc{
						Status: states.ObjectReady,
					},
					addrs.AbsProviderConfig{
						Module:   addrs.RootModule,
						Provider: addrs.NewDefaultProvider("test"),
					})
			}),
			want: []map[string]interface{}{
				{
					"@level":    "error",
					"@message":  "Terraform left some resources in state after executing main.tftest.hcl, they need to be cleaned up manually.",
					"@module":   "terraform.ui",
					"@testfile": "main.tftest.hcl",
					"test_cleanup": map[string]interface{}{
						"failed_resources": []interface{}{
							map[string]interface{}{
								"instance": "test.bar",
							},
							map[string]interface{}{
								"instance":    "test.bar",
								"deposed_key": "0fcb640a",
							},
							map[string]interface{}{
								"instance": "test.foo",
							},
						},
					},
					"type": "test_cleanup",
				},
				{
					"@level":    "warn",
					"@message":  "Warning: first warning",
					"@module":   "terraform.ui",
					"@testfile": "main.tftest.hcl",
					"diagnostic": map[string]interface{}{
						"detail":   "something not very bad happened",
						"severity": "warning",
						"summary":  "first warning",
					},
					"type": "diagnostic",
				},
				{
					"@level":    "warn",
					"@message":  "Warning: second warning",
					"@module":   "terraform.ui",
					"@testfile": "main.tftest.hcl",
					"diagnostic": map[string]interface{}{
						"detail":   "something not very bad happened again",
						"severity": "warning",
						"summary":  "second warning",
					},
					"type": "diagnostic",
				},
				{
					"@level":    "error",
					"@message":  "Error: first error",
					"@module":   "terraform.ui",
					"@testfile": "main.tftest.hcl",
					"diagnostic": map[string]interface{}{
						"detail":   "this time it is very bad",
						"severity": "error",
						"summary":  "first error",
					},
					"type": "diagnostic",
				},
			},
		},
	}
	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			streams, done := terminal.StreamsForTesting(t)
			view := NewTest(arguments.ViewJSON, NewView(streams))

			view.DestroySummary(tc.diags, tc.run, tc.file, tc.state)
			testJSONViewOutputEquals(t, done(t).All(), tc.want)
		})
	}
}

func TestTestJSON_File(t *testing.T) {
	tcs := map[string]struct {
		file     *moduletest.File
		progress moduletest.Progress
		want     []map[string]interface{}
	}{
		"pass": {
			file:     &moduletest.File{Name: "main.tf", Status: moduletest.Pass},
			progress: moduletest.Complete,
			want: []map[string]interface{}{
				{
					"@level":    "info",
					"@message":  "main.tf... pass",
					"@module":   "terraform.ui",
					"@testfile": "main.tf",
					"test_file": map[string]interface{}{
						"path":     "main.tf",
						"progress": "complete",
						"status":   "pass",
					},
					"type": "test_file",
				},
			},
		},

		"pending": {
			file:     &moduletest.File{Name: "main.tf", Status: moduletest.Pending},
			progress: moduletest.Complete,
			want: []map[string]interface{}{
				{
					"@level":    "info",
					"@message":  "main.tf... pending",
					"@module":   "terraform.ui",
					"@testfile": "main.tf",
					"test_file": map[string]interface{}{
						"path":     "main.tf",
						"progress": "complete",
						"status":   "pending",
					},
					"type": "test_file",
				},
			},
		},

		"skip": {
			file:     &moduletest.File{Name: "main.tf", Status: moduletest.Skip},
			progress: moduletest.Complete,
			want: []map[string]interface{}{
				{
					"@level":    "info",
					"@message":  "main.tf... skip",
					"@module":   "terraform.ui",
					"@testfile": "main.tf",
					"test_file": map[string]interface{}{
						"path":     "main.tf",
						"progress": "complete",
						"status":   "skip",
					},
					"type": "test_file",
				},
			},
		},

		"fail": {
			file:     &moduletest.File{Name: "main.tf", Status: moduletest.Fail},
			progress: moduletest.Complete,
			want: []map[string]interface{}{
				{
					"@level":    "info",
					"@message":  "main.tf... fail",
					"@module":   "terraform.ui",
					"@testfile": "main.tf",
					"test_file": map[string]interface{}{
						"path":     "main.tf",
						"progress": "complete",
						"status":   "fail",
					},
					"type": "test_file",
				},
			},
		},

		"error": {
			file:     &moduletest.File{Name: "main.tf", Status: moduletest.Error},
			progress: moduletest.Complete,
			want: []map[string]interface{}{
				{
					"@level":    "info",
					"@message":  "main.tf... fail",
					"@module":   "terraform.ui",
					"@testfile": "main.tf",
					"test_file": map[string]interface{}{
						"path":     "main.tf",
						"progress": "complete",
						"status":   "error",
					},
					"type": "test_file",
				},
			},
		},

		"starting": {
			file:     &moduletest.File{Name: "main.tftest.hcl", Status: moduletest.Pending},
			progress: moduletest.Starting,
			want: []map[string]interface{}{
				{
					"@level":    "info",
					"@message":  "main.tftest.hcl... in progress",
					"@module":   "terraform.ui",
					"@testfile": "main.tftest.hcl",
					"test_file": map[string]interface{}{
						"path":     "main.tftest.hcl",
						"progress": "starting",
					},
					"type": "test_file",
				},
			},
		},

		"tear_down": {
			file:     &moduletest.File{Name: "main.tftest.hcl", Status: moduletest.Pending},
			progress: moduletest.TearDown,
			want: []map[string]interface{}{
				{
					"@level":    "info",
					"@message":  "main.tftest.hcl... tearing down",
					"@module":   "terraform.ui",
					"@testfile": "main.tftest.hcl",
					"test_file": map[string]interface{}{
						"path":     "main.tftest.hcl",
						"progress": "teardown",
					},
					"type": "test_file",
				},
			},
		},
	}
	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			streams, done := terminal.StreamsForTesting(t)
			view := NewTest(arguments.ViewJSON, NewView(streams))

			view.File(tc.file, tc.progress)
			testJSONViewOutputEquals(t, done(t).All(), tc.want)
		})
	}
}

func TestTestJSON_Run(t *testing.T) {
	tcs := map[string]struct {
		run      *moduletest.Run
		progress moduletest.Progress
		elapsed  int64
		want     []map[string]interface{}
	}{
		"starting": {
			run:      &moduletest.Run{Name: "run_block", Status: moduletest.Pass},
			progress: moduletest.Starting,
			want: []map[string]interface{}{
				{
					"@level":    "info",
					"@message":  "  \"run_block\"... in progress",
					"@module":   "terraform.ui",
					"@testfile": "main.tftest.hcl",
					"@testrun":  "run_block",
					"test_run": map[string]interface{}{
						"path":     "main.tftest.hcl",
						"run":      "run_block",
						"progress": "starting",
						"elapsed":  float64(0),
					},
					"type": "test_run",
				},
			},
		},

		"running": {
			run:      &moduletest.Run{Name: "run_block", Status: moduletest.Pass},
			progress: moduletest.Running,
			elapsed:  2024,
			want: []map[string]interface{}{
				{
					"@level":    "info",
					"@message":  "  \"run_block\"... in progress",
					"@module":   "terraform.ui",
					"@testfile": "main.tftest.hcl",
					"@testrun":  "run_block",
					"test_run": map[string]interface{}{
						"path":     "main.tftest.hcl",
						"run":      "run_block",
						"progress": "running",
						"elapsed":  float64(2024),
					},
					"type": "test_run",
				},
			},
		},

		"teardown": {
			run:      &moduletest.Run{Name: "run_block", Status: moduletest.Pass},
			progress: moduletest.TearDown,
			want: []map[string]interface{}{
				{
					"@level":    "info",
					"@message":  "  \"run_block\"... tearing down",
					"@module":   "terraform.ui",
					"@testfile": "main.tftest.hcl",
					"@testrun":  "run_block",
					"test_run": map[string]interface{}{
						"path":     "main.tftest.hcl",
						"run":      "run_block",
						"progress": "teardown",
						"elapsed":  float64(0),
					},
					"type": "test_run",
				},
			},
		},

		"pass": {
			run:      &moduletest.Run{Name: "run_block", Status: moduletest.Pass},
			progress: moduletest.Complete,
			want: []map[string]interface{}{
				{
					"@level":    "info",
					"@message":  "  \"run_block\"... pass",
					"@module":   "terraform.ui",
					"@testfile": "main.tftest.hcl",
					"@testrun":  "run_block",
					"test_run": map[string]interface{}{
						"path":     "main.tftest.hcl",
						"run":      "run_block",
						"progress": "complete",
						"status":   "pass",
					},
					"type": "test_run",
				},
			},
		},

		"pass_with_diags": {
			run: &moduletest.Run{
				Name:        "run_block",
				Status:      moduletest.Pass,
				Diagnostics: tfdiags.Diagnostics{tfdiags.Sourceless(tfdiags.Warning, "a warning occurred", "some warning happened during this test")},
			},
			progress: moduletest.Complete,
			want: []map[string]interface{}{
				{
					"@level":    "info",
					"@message":  "  \"run_block\"... pass",
					"@module":   "terraform.ui",
					"@testfile": "main.tftest.hcl",
					"@testrun":  "run_block",
					"test_run": map[string]interface{}{
						"path":     "main.tftest.hcl",
						"run":      "run_block",
						"progress": "complete",
						"status":   "pass",
					},
					"type": "test_run",
				},
				{
					"@level":    "warn",
					"@message":  "Warning: a warning occurred",
					"@module":   "terraform.ui",
					"@testfile": "main.tftest.hcl",
					"@testrun":  "run_block",
					"diagnostic": map[string]interface{}{
						"detail":   "some warning happened during this test",
						"severity": "warning",
						"summary":  "a warning occurred",
					},
					"type": "diagnostic",
				},
			},
		},

		"pending": {
			run:      &moduletest.Run{Name: "run_block", Status: moduletest.Pending},
			progress: moduletest.Complete,
			want: []map[string]interface{}{
				{
					"@level":    "info",
					"@message":  "  \"run_block\"... pending",
					"@module":   "terraform.ui",
					"@testfile": "main.tftest.hcl",
					"@testrun":  "run_block",
					"test_run": map[string]interface{}{
						"path":     "main.tftest.hcl",
						"run":      "run_block",
						"progress": "complete",
						"status":   "pending",
					},
					"type": "test_run",
				},
			},
		},

		"skip": {
			run:      &moduletest.Run{Name: "run_block", Status: moduletest.Skip},
			progress: moduletest.Complete,
			want: []map[string]interface{}{
				{
					"@level":    "info",
					"@message":  "  \"run_block\"... skip",
					"@module":   "terraform.ui",
					"@testfile": "main.tftest.hcl",
					"@testrun":  "run_block",
					"test_run": map[string]interface{}{
						"path":     "main.tftest.hcl",
						"run":      "run_block",
						"progress": "complete",
						"status":   "skip",
					},
					"type": "test_run",
				},
			},
		},

		"fail": {
			run:      &moduletest.Run{Name: "run_block", Status: moduletest.Fail},
			progress: moduletest.Complete,
			want: []map[string]interface{}{
				{
					"@level":    "info",
					"@message":  "  \"run_block\"... fail",
					"@module":   "terraform.ui",
					"@testfile": "main.tftest.hcl",
					"@testrun":  "run_block",
					"test_run": map[string]interface{}{
						"path":     "main.tftest.hcl",
						"run":      "run_block",
						"progress": "complete",
						"status":   "fail",
					},
					"type": "test_run",
				},
			},
		},

		"fail_with_diags": {
			run: &moduletest.Run{
				Name:   "run_block",
				Status: moduletest.Fail,
				Diagnostics: tfdiags.Diagnostics{
					tfdiags.Sourceless(tfdiags.Error, "a comparison failed", "details details details"),
					tfdiags.Sourceless(tfdiags.Error, "a second comparison failed", "other details"),
				},
			},
			progress: moduletest.Complete,
			want: []map[string]interface{}{
				{
					"@level":    "info",
					"@message":  "  \"run_block\"... fail",
					"@module":   "terraform.ui",
					"@testfile": "main.tftest.hcl",
					"@testrun":  "run_block",
					"test_run": map[string]interface{}{
						"path":     "main.tftest.hcl",
						"run":      "run_block",
						"progress": "complete",
						"status":   "fail",
					},
					"type": "test_run",
				},
				{
					"@level":    "error",
					"@message":  "Error: a comparison failed",
					"@module":   "terraform.ui",
					"@testfile": "main.tftest.hcl",
					"@testrun":  "run_block",
					"diagnostic": map[string]interface{}{
						"detail":   "details details details",
						"severity": "error",
						"summary":  "a comparison failed",
					},
					"type": "diagnostic",
				},
				{
					"@level":    "error",
					"@message":  "Error: a second comparison failed",
					"@module":   "terraform.ui",
					"@testfile": "main.tftest.hcl",
					"@testrun":  "run_block",
					"diagnostic": map[string]interface{}{
						"detail":   "other details",
						"severity": "error",
						"summary":  "a second comparison failed",
					},
					"type": "diagnostic",
				},
			},
		},

		"error": {
			run:      &moduletest.Run{Name: "run_block", Status: moduletest.Error},
			progress: moduletest.Complete,
			want: []map[string]interface{}{
				{
					"@level":    "info",
					"@message":  "  \"run_block\"... fail",
					"@module":   "terraform.ui",
					"@testfile": "main.tftest.hcl",
					"@testrun":  "run_block",
					"test_run": map[string]interface{}{
						"path":     "main.tftest.hcl",
						"run":      "run_block",
						"progress": "complete",
						"status":   "error",
					},
					"type": "test_run",
				},
			},
		},

		"error_with_diags": {
			run: &moduletest.Run{
				Name:        "run_block",
				Status:      moduletest.Error,
				Diagnostics: tfdiags.Diagnostics{tfdiags.Sourceless(tfdiags.Error, "an error occurred", "something bad happened during this test")},
			},
			progress: moduletest.Complete,
			want: []map[string]interface{}{
				{
					"@level":    "info",
					"@message":  "  \"run_block\"... fail",
					"@module":   "terraform.ui",
					"@testfile": "main.tftest.hcl",
					"@testrun":  "run_block",
					"test_run": map[string]interface{}{
						"path":     "main.tftest.hcl",
						"run":      "run_block",
						"progress": "complete",
						"status":   "error",
					},
					"type": "test_run",
				},
				{
					"@level":    "error",
					"@message":  "Error: an error occurred",
					"@module":   "terraform.ui",
					"@testfile": "main.tftest.hcl",
					"@testrun":  "run_block",
					"diagnostic": map[string]interface{}{
						"detail":   "something bad happened during this test",
						"severity": "error",
						"summary":  "an error occurred",
					},
					"type": "diagnostic",
				},
			},
		},

		"verbose_plan": {
			run: &moduletest.Run{
				Name:   "run_block",
				Status: moduletest.Pass,
				Config: &configs.TestRun{
					Command: configs.PlanTestCommand,
				},
				Verbose: &moduletest.Verbose{
					Plan: &plans.Plan{
						Changes: &plans.ChangesSrc{
							Resources: []*plans.ResourceInstanceChangeSrc{
								{
									Addr: addrs.AbsResourceInstance{
										Module: addrs.RootModuleInstance,
										Resource: addrs.ResourceInstance{
											Resource: addrs.Resource{
												Mode: addrs.ManagedResourceMode,
												Type: "test_resource",
												Name: "creating",
											},
										},
									},
									PrevRunAddr: addrs.AbsResourceInstance{
										Module: addrs.RootModuleInstance,
										Resource: addrs.ResourceInstance{
											Resource: addrs.Resource{
												Mode: addrs.ManagedResourceMode,
												Type: "test_resource",
												Name: "creating",
											},
										},
									},
									ProviderAddr: addrs.AbsProviderConfig{
										Module: addrs.RootModule,
										Provider: addrs.Provider{
											Hostname:  addrs.DefaultProviderRegistryHost,
											Namespace: "hashicorp",
											Type:      "test",
										},
									},
									ChangeSrc: plans.ChangeSrc{
										Action: plans.Create,
										After: dynamicValue(
											t,
											cty.ObjectVal(map[string]cty.Value{
												"value": cty.StringVal("foobar"),
											}),
											cty.Object(map[string]cty.Type{
												"value": cty.String,
											})),
									},
								},
							},
						},
					},
					State: states.NewState(), // empty state
					Config: &configs.Config{
						Module: &configs.Module{
							ProviderRequirements: &configs.RequiredProviders{},
						},
					},
					Providers: map[addrs.Provider]providers.ProviderSchema{
						addrs.Provider{
							Hostname:  addrs.DefaultProviderRegistryHost,
							Namespace: "hashicorp",
							Type:      "test",
						}: {
							ResourceTypes: map[string]providers.Schema{
								"test_resource": {
									Block: &configschema.Block{
										Attributes: map[string]*configschema.Attribute{
											"value": {
												Type: cty.String,
											},
										},
									},
								},
							},
						},
					},
				},
			},
			progress: moduletest.Complete,
			want: []map[string]interface{}{
				{
					"@level":    "info",
					"@message":  "  \"run_block\"... pass",
					"@module":   "terraform.ui",
					"@testfile": "main.tftest.hcl",
					"@testrun":  "run_block",
					"test_run": map[string]interface{}{
						"path":     "main.tftest.hcl",
						"run":      "run_block",
						"progress": "complete",
						"status":   "pass",
					},
					"type": "test_run",
				},
				{
					"@level":    "info",
					"@message":  "-verbose flag enabled, printing plan",
					"@module":   "terraform.ui",
					"@testfile": "main.tftest.hcl",
					"@testrun":  "run_block",
					"test_plan": map[string]interface{}{
						"plan_format_version":     "1.2",
						"provider_format_version": "1.0",
						"resource_changes": []interface{}{
							map[string]interface{}{
								"address": "test_resource.creating",
								"change": map[string]interface{}{
									"actions": []interface{}{"create"},
									"after": map[string]interface{}{
										"value": "foobar",
									},
									"after_sensitive":  map[string]interface{}{},
									"after_unknown":    map[string]interface{}{},
									"before":           nil,
									"before_sensitive": false,
								},
								"mode":          "managed",
								"name":          "creating",
								"provider_name": "registry.terraform.io/hashicorp/test",
								"type":          "test_resource",
							},
						},
						"provider_schemas": map[string]interface{}{
							"registry.terraform.io/hashicorp/test": map[string]interface{}{
								"provider": map[string]interface{}{
									"version": 0.0,
								},
								"resource_schemas": map[string]interface{}{
									"test_resource": map[string]interface{}{
										"block": map[string]interface{}{
											"attributes": map[string]interface{}{
												"value": map[string]interface{}{
													"description_kind": "plain",
													"type":             "string",
												},
											},
											"description_kind": "plain",
										},
										"version": 0.0,
									},
								},
							},
						},
					},
					"type": "test_plan",
				},
			},
		},

		"verbose_apply": {
			run: &moduletest.Run{
				Name:   "run_block",
				Status: moduletest.Pass,
				Config: &configs.TestRun{
					Command: configs.ApplyTestCommand,
				},
				Verbose: &moduletest.Verbose{
					Plan: &plans.Plan{}, // empty plan
					State: states.BuildState(func(state *states.SyncState) {
						state.SetResourceInstanceCurrent(
							addrs.AbsResourceInstance{
								Module: addrs.RootModuleInstance,
								Resource: addrs.ResourceInstance{
									Resource: addrs.Resource{
										Mode: addrs.ManagedResourceMode,
										Type: "test_resource",
										Name: "creating",
									},
								},
							},
							&states.ResourceInstanceObjectSrc{
								AttrsJSON: []byte(`{"value":"foobar"}`),
							},
							addrs.AbsProviderConfig{
								Module: addrs.RootModule,
								Provider: addrs.Provider{
									Hostname:  addrs.DefaultProviderRegistryHost,
									Namespace: "hashicorp",
									Type:      "test",
								},
							})
					}),
					Config: &configs.Config{
						Module: &configs.Module{},
					},
					Providers: map[addrs.Provider]providers.ProviderSchema{
						addrs.Provider{
							Hostname:  addrs.DefaultProviderRegistryHost,
							Namespace: "hashicorp",
							Type:      "test",
						}: {
							ResourceTypes: map[string]providers.Schema{
								"test_resource": {
									Block: &configschema.Block{
										Attributes: map[string]*configschema.Attribute{
											"value": {
												Type: cty.String,
											},
										},
									},
								},
							},
						},
					},
				},
			},
			progress: moduletest.Complete,
			want: []map[string]interface{}{
				{
					"@level":    "info",
					"@message":  "  \"run_block\"... pass",
					"@module":   "terraform.ui",
					"@testfile": "main.tftest.hcl",
					"@testrun":  "run_block",
					"test_run": map[string]interface{}{
						"path":     "main.tftest.hcl",
						"run":      "run_block",
						"progress": "complete",
						"status":   "pass",
					},
					"type": "test_run",
				},
				{
					"@level":    "info",
					"@message":  "-verbose flag enabled, printing state",
					"@module":   "terraform.ui",
					"@testfile": "main.tftest.hcl",
					"@testrun":  "run_block",
					"test_state": map[string]interface{}{
						"state_format_version":    "1.0",
						"provider_format_version": "1.0",
						"root_module": map[string]interface{}{
							"resources": []interface{}{
								map[string]interface{}{
									"address":          "test_resource.creating",
									"mode":             "managed",
									"name":             "creating",
									"provider_name":    "registry.terraform.io/hashicorp/test",
									"schema_version":   0.0,
									"sensitive_values": map[string]interface{}{},
									"type":             "test_resource",
									"values": map[string]interface{}{
										"value": "foobar",
									},
								},
							},
						},
						"provider_schemas": map[string]interface{}{
							"registry.terraform.io/hashicorp/test": map[string]interface{}{
								"provider": map[string]interface{}{
									"version": 0.0,
								},
								"resource_schemas": map[string]interface{}{
									"test_resource": map[string]interface{}{
										"block": map[string]interface{}{
											"attributes": map[string]interface{}{
												"value": map[string]interface{}{
													"description_kind": "plain",
													"type":             "string",
												},
											},
											"description_kind": "plain",
										},
										"version": 0.0,
									},
								},
							},
						},
					},
					"type": "test_state",
				},
			},
		},
	}
	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			streams, done := terminal.StreamsForTesting(t)
			view := NewTest(arguments.ViewJSON, NewView(streams))

			file := &moduletest.File{Name: "main.tftest.hcl"}

			view.Run(tc.run, file, tc.progress, tc.elapsed)
			testJSONViewOutputEquals(t, done(t).All(), tc.want, cmp.FilterPath(func(path cmp.Path) bool {
				return strings.Contains(path.Last().String(), "version") || strings.Contains(path.Last().String(), "timestamp")
			}, cmp.Ignore()))
		})
	}
}

func TestTestJSON_FatalInterruptSummary(t *testing.T) {
	tcs := map[string]struct {
		states  map[*moduletest.Run]*states.State
		changes []*plans.ResourceInstanceChangeSrc
		want    []map[string]interface{}
	}{
		"no_state_only_plan": {
			states: make(map[*moduletest.Run]*states.State),
			changes: []*plans.ResourceInstanceChangeSrc{
				{
					Addr: addrs.AbsResourceInstance{
						Module: addrs.RootModuleInstance,
						Resource: addrs.ResourceInstance{
							Resource: addrs.Resource{
								Mode: addrs.ManagedResourceMode,
								Type: "test_instance",
								Name: "one",
							},
						},
					},
					ChangeSrc: plans.ChangeSrc{
						Action: plans.Create,
					},
				},
				{
					Addr: addrs.AbsResourceInstance{
						Module: addrs.RootModuleInstance,
						Resource: addrs.ResourceInstance{
							Resource: addrs.Resource{
								Mode: addrs.ManagedResourceMode,
								Type: "test_instance",
								Name: "two",
							},
						},
					},
					ChangeSrc: plans.ChangeSrc{
						Action: plans.Create,
					},
				},
			},
			want: []map[string]interface{}{
				{
					"@level":    "error",
					"@message":  "Terraform was interrupted during test execution, and may not have performed the expected cleanup operations.",
					"@module":   "terraform.ui",
					"@testfile": "main.tftest.hcl",
					"@testrun":  "run_block",
					"test_interrupt": map[string]interface{}{
						"planned": []interface{}{
							"test_instance.one",
							"test_instance.two",
						},
					},
					"type": "test_interrupt",
				},
			},
		},
		"file_state_no_plan": {
			states: map[*moduletest.Run]*states.State{
				nil: states.BuildState(func(state *states.SyncState) {
					state.SetResourceInstanceCurrent(
						addrs.AbsResourceInstance{
							Module: addrs.RootModuleInstance,
							Resource: addrs.ResourceInstance{
								Resource: addrs.Resource{
									Mode: addrs.ManagedResourceMode,
									Type: "test_instance",
									Name: "one",
								},
							},
						},
						&states.ResourceInstanceObjectSrc{},
						addrs.AbsProviderConfig{})

					state.SetResourceInstanceCurrent(
						addrs.AbsResourceInstance{
							Module: addrs.RootModuleInstance,
							Resource: addrs.ResourceInstance{
								Resource: addrs.Resource{
									Mode: addrs.ManagedResourceMode,
									Type: "test_instance",
									Name: "two",
								},
							},
						},
						&states.ResourceInstanceObjectSrc{},
						addrs.AbsProviderConfig{})
				}),
			},
			changes: nil,
			want: []map[string]interface{}{
				{
					"@level":    "error",
					"@message":  "Terraform was interrupted during test execution, and may not have performed the expected cleanup operations.",
					"@module":   "terraform.ui",
					"@testfile": "main.tftest.hcl",
					"@testrun":  "run_block",
					"test_interrupt": map[string]interface{}{
						"state": []interface{}{
							map[string]interface{}{
								"instance": "test_instance.one",
							},
							map[string]interface{}{
								"instance": "test_instance.two",
							},
						},
					},
					"type": "test_interrupt",
				},
			},
		},
		"run_states_no_plan": {
			states: map[*moduletest.Run]*states.State{
				&moduletest.Run{Name: "setup_block"}: states.BuildState(func(state *states.SyncState) {
					state.SetResourceInstanceCurrent(
						addrs.AbsResourceInstance{
							Module: addrs.RootModuleInstance,
							Resource: addrs.ResourceInstance{
								Resource: addrs.Resource{
									Mode: addrs.ManagedResourceMode,
									Type: "test_instance",
									Name: "one",
								},
							},
						},
						&states.ResourceInstanceObjectSrc{},
						addrs.AbsProviderConfig{})

					state.SetResourceInstanceCurrent(
						addrs.AbsResourceInstance{
							Module: addrs.RootModuleInstance,
							Resource: addrs.ResourceInstance{
								Resource: addrs.Resource{
									Mode: addrs.ManagedResourceMode,
									Type: "test_instance",
									Name: "two",
								},
							},
						},
						&states.ResourceInstanceObjectSrc{},
						addrs.AbsProviderConfig{})
				}),
			},
			changes: nil,
			want: []map[string]interface{}{
				{
					"@level":    "error",
					"@message":  "Terraform was interrupted during test execution, and may not have performed the expected cleanup operations.",
					"@module":   "terraform.ui",
					"@testfile": "main.tftest.hcl",
					"@testrun":  "run_block",
					"test_interrupt": map[string]interface{}{
						"states": map[string]interface{}{
							"setup_block": []interface{}{
								map[string]interface{}{
									"instance": "test_instance.one",
								},
								map[string]interface{}{
									"instance": "test_instance.two",
								},
							},
						},
					},
					"type": "test_interrupt",
				},
			},
		},
		"all_states_with_plan": {
			states: map[*moduletest.Run]*states.State{
				&moduletest.Run{Name: "setup_block"}: states.BuildState(func(state *states.SyncState) {
					state.SetResourceInstanceCurrent(
						addrs.AbsResourceInstance{
							Module: addrs.RootModuleInstance,
							Resource: addrs.ResourceInstance{
								Resource: addrs.Resource{
									Mode: addrs.ManagedResourceMode,
									Type: "test_instance",
									Name: "setup_one",
								},
							},
						},
						&states.ResourceInstanceObjectSrc{},
						addrs.AbsProviderConfig{})

					state.SetResourceInstanceCurrent(
						addrs.AbsResourceInstance{
							Module: addrs.RootModuleInstance,
							Resource: addrs.ResourceInstance{
								Resource: addrs.Resource{
									Mode: addrs.ManagedResourceMode,
									Type: "test_instance",
									Name: "setup_two",
								},
							},
						},
						&states.ResourceInstanceObjectSrc{},
						addrs.AbsProviderConfig{})
				}),
				nil: states.BuildState(func(state *states.SyncState) {
					state.SetResourceInstanceCurrent(
						addrs.AbsResourceInstance{
							Module: addrs.RootModuleInstance,
							Resource: addrs.ResourceInstance{
								Resource: addrs.Resource{
									Mode: addrs.ManagedResourceMode,
									Type: "test_instance",
									Name: "one",
								},
							},
						},
						&states.ResourceInstanceObjectSrc{},
						addrs.AbsProviderConfig{})

					state.SetResourceInstanceCurrent(
						addrs.AbsResourceInstance{
							Module: addrs.RootModuleInstance,
							Resource: addrs.ResourceInstance{
								Resource: addrs.Resource{
									Mode: addrs.ManagedResourceMode,
									Type: "test_instance",
									Name: "two",
								},
							},
						},
						&states.ResourceInstanceObjectSrc{},
						addrs.AbsProviderConfig{})
				}),
			},
			changes: []*plans.ResourceInstanceChangeSrc{
				{
					Addr: addrs.AbsResourceInstance{
						Module: addrs.RootModuleInstance,
						Resource: addrs.ResourceInstance{
							Resource: addrs.Resource{
								Mode: addrs.ManagedResourceMode,
								Type: "test_instance",
								Name: "new_one",
							},
						},
					},
					ChangeSrc: plans.ChangeSrc{
						Action: plans.Create,
					},
				},
				{
					Addr: addrs.AbsResourceInstance{
						Module: addrs.RootModuleInstance,
						Resource: addrs.ResourceInstance{
							Resource: addrs.Resource{
								Mode: addrs.ManagedResourceMode,
								Type: "test_instance",
								Name: "new_two",
							},
						},
					},
					ChangeSrc: plans.ChangeSrc{
						Action: plans.Create,
					},
				},
			},
			want: []map[string]interface{}{
				{
					"@level":    "error",
					"@message":  "Terraform was interrupted during test execution, and may not have performed the expected cleanup operations.",
					"@module":   "terraform.ui",
					"@testfile": "main.tftest.hcl",
					"@testrun":  "run_block",
					"test_interrupt": map[string]interface{}{
						"state": []interface{}{
							map[string]interface{}{
								"instance": "test_instance.one",
							},
							map[string]interface{}{
								"instance": "test_instance.two",
							},
						},
						"states": map[string]interface{}{
							"setup_block": []interface{}{
								map[string]interface{}{
									"instance": "test_instance.setup_one",
								},
								map[string]interface{}{
									"instance": "test_instance.setup_two",
								},
							},
						},
						"planned": []interface{}{
							"test_instance.new_one",
							"test_instance.new_two",
						},
					},
					"type": "test_interrupt",
				},
			},
		},
	}
	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			streams, done := terminal.StreamsForTesting(t)
			view := NewTest(arguments.ViewJSON, NewView(streams))

			file := &moduletest.File{Name: "main.tftest.hcl"}
			run := &moduletest.Run{Name: "run_block"}

			view.FatalInterruptSummary(run, file, tc.states, tc.changes)
			testJSONViewOutputEquals(t, done(t).All(), tc.want)
		})
	}
}

func dynamicValue(t *testing.T, value cty.Value, typ cty.Type) plans.DynamicValue {
	d, err := plans.NewDynamicValue(value, typ)
	if err != nil {
		t.Fatalf("failed to create dynamic value: %s", err)
	}
	return d
}
