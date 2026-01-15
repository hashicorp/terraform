// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hashicorp/cli"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	testing_command "github.com/hashicorp/terraform/internal/command/testing"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configload"
	"github.com/hashicorp/terraform/internal/getproviders"
	"github.com/hashicorp/terraform/internal/initwd"
	teststates "github.com/hashicorp/terraform/internal/moduletest/states"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/registry"
	"github.com/hashicorp/terraform/internal/states/statemgr"
	"github.com/hashicorp/terraform/internal/terminal"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestTest_Runs(t *testing.T) {
	tcs := map[string]struct {
		override              string
		args                  []string
		envVars               map[string]string
		expectedOut           []string
		expectedErr           []string
		expectedResourceCount int
		code                  int
		initCode              int
		skip                  bool
		description           string
	}{
		"simple_pass": {
			expectedOut: []string{"1 passed, 0 failed."},
			code:        0,
		},
		"top-dir-only-test-files": {
			expectedOut: []string{"1 passed, 0 failed."},
			code:        0,
		},
		"top-dir-only-nested-test-files": {
			expectedOut: []string{"1 passed, 0 failed."},
			code:        0,
		},
		"simple_pass_nested": {
			expectedOut: []string{"1 passed, 0 failed."},
			code:        0,
		},
		"simple_pass_count": {
			expectedOut: []string{"1 passed, 0 failed."},
			args:        []string{"-run-parallelism", "1"},
			code:        0,
		},
		"simple_pass_nested_alternate": {
			args:        []string{"-test-directory", "other"},
			expectedOut: []string{"1 passed, 0 failed."},
			code:        0,
		},
		"simple_pass_very_nested": {
			args:        []string{"-test-directory", "tests/subdir"},
			expectedOut: []string{"1 passed, 0 failed."},
			code:        0,
		},
		"simple_pass_cmd_parallel": {
			override:    "simple_pass",
			args:        []string{"-parallelism", "1"},
			expectedOut: []string{"1 passed, 0 failed."},
			code:        0,
			description: "simple_pass with parallelism set to 1",
		},
		"simple_pass_very_nested_alternate": {
			override:    "simple_pass_very_nested",
			args:        []string{"-test-directory", "./tests/subdir"},
			expectedOut: []string{"1 passed, 0 failed."},
			code:        0,
		},
		"simple_pass_bad_test_directory": {
			override:    "simple_pass",
			args:        []string{"-test-directory", "../tests"},
			expectedErr: []string{"Invalid testing directory"},
			code:        1,
		},
		"simple_pass_bad_test_directory_abs": {
			override:    "simple_pass",
			args:        []string{"-test-directory", "/home/username/config/tests"},
			expectedErr: []string{"Invalid testing directory"},
			code:        1,
		},
		"pass_with_locals": {
			expectedOut: []string{"1 passed, 0 failed."},
			code:        0,
		},
		"pass_with_outputs": {
			expectedOut: []string{"1 passed, 0 failed."},
			code:        0,
		},
		"pass_with_variables": {
			expectedOut: []string{"2 passed, 0 failed."},
			code:        0,
		},
		"plan_then_apply": {
			expectedOut: []string{"2 passed, 0 failed."},
			code:        0,
		},
		"expect_failures_checks": {
			expectedOut: []string{"1 passed, 0 failed."},
			code:        0,
		},
		"expect_failures_inputs": {
			expectedOut: []string{"1 passed, 0 failed."},
			code:        0,
		},
		"expect_failures_resources": {
			expectedOut: []string{"1 passed, 0 failed."},
			code:        0,
		},
		"expect_failures_outputs": {
			expectedOut: []string{"1 passed, 0 failed."},
			code:        0,
		},
		"expect_failures_checks_verbose": {
			override:    "expect_failures_checks",
			args:        []string{"-verbose"},
			expectedOut: []string{"1 passed, 0 failed.", "Warning: Check block assertion failed"},
			code:        0,
		},
		"expect_failures_inputs_verbose": {
			override:    "expect_failures_inputs",
			args:        []string{"-verbose"},
			expectedOut: []string{"1 passed, 0 failed."},
			expectedErr: []string{"Error: Invalid value for variable"},
			code:        0,
		},
		"expect_failures_resources_verbose": {
			override:    "expect_failures_resources",
			args:        []string{"-verbose"},
			expectedOut: []string{"1 passed, 0 failed."},
			expectedErr: []string{"Error: Resource postcondition failed"},
			code:        0,
		},
		"expect_failures_outputs_verbose": {
			override:    "expect_failures_outputs",
			args:        []string{"-verbose"},
			expectedOut: []string{"1 passed, 0 failed."},
			expectedErr: []string{"Error: Module output value precondition failed"},
			code:        0,
		},
		"multiple_files": {
			expectedOut: []string{"2 passed, 0 failed"},
			code:        0,
		},
		"multiple_files_with_filter": {
			override:    "multiple_files",
			args:        []string{"-filter=one.tftest.hcl"},
			expectedOut: []string{"1 passed, 0 failed"},
			code:        0,
		},
		"no_state": {
			expectedOut: []string{"0 passed, 1 failed"},
			expectedErr: []string{"No value for required variable"},
			description: "the run apply fails, causing it to produce a nil state.",
			code:        1,
		},
		"variables": {
			expectedOut: []string{"2 passed, 0 failed"},
			code:        0,
		},
		"variables_overridden": {
			override:    "variables",
			args:        []string{"-var=input=foo"},
			expectedOut: []string{"1 passed, 1 failed"},
			expectedErr: []string{`invalid value`},
			code:        1,
		},
		"simple_fail": {
			expectedOut: []string{"0 passed, 1 failed."},
			expectedErr: []string{"invalid value", `│ - "bar"
    │ + "zap"`},
			code: 1,
		},
		"custom_condition_checks": {
			expectedOut: []string{"0 passed, 1 failed."},
			expectedErr: []string{"this really should fail"},
			code:        1,
		},
		"custom_condition_inputs": {
			expectedOut: []string{"0 passed, 1 failed."},
			expectedErr: []string{"this should definitely fail"},
			code:        1,
		},
		"custom_condition_outputs": {
			expectedOut: []string{"0 passed, 1 failed."},
			expectedErr: []string{"this should fail"},
			code:        1,
		},
		"custom_condition_resources": {
			expectedOut: []string{"0 passed, 1 failed."},
			expectedErr: []string{"this really should fail"},
			code:        1,
		},
		"no_providers_in_main": {
			expectedOut: []string{"1 passed, 0 failed"},
			code:        0,
		},
		"default_variables": {
			expectedOut: []string{"1 passed, 0 failed."},
			code:        0,
		},
		"undefined_variables": {
			expectedOut: []string{"1 passed, 0 failed."},
			code:        0,
		},
		"shared_state": {
			expectedOut: []string{"8 passed, 0 failed."},
			code:        0,
		},
		"shared_state_object": {
			expectedOut: []string{"2 passed, 0 failed."},
			code:        0,
		},
		"variable_references": {
			expectedOut: []string{"3 passed, 0 failed."},
			args:        []string{"-var=global=\"triple\""},
			code:        0,
		},
		"unreferenced_global_variable": {
			override:    "variable_references",
			expectedOut: []string{"3 passed, 0 failed."},
			// The other variable shouldn't pass validation, but it won't be
			// referenced anywhere so should just be ignored.
			args: []string{"-var=global=\"triple\"", "-var=other=bad"},
			code: 0,
		},
		"variables_types": {
			expectedOut: []string{"1 passed, 0 failed."},
			args:        []string{"-var=number_input=0", "-var=string_input=Hello, world!", "-var=list_input=[\"Hello\",\"world\"]"},
			code:        0,
		},
		"null-outputs": {
			expectedOut: []string{"2 passed, 0 failed."},
			code:        0,
		},
		"destroy_fail": {
			expectedOut:           []string{"3 passed, 0 failed."},
			expectedErr:           []string{`Terraform left the following resources in state`},
			code:                  1,
			expectedResourceCount: 4,
		},
		"default_optional_values": {
			expectedOut: []string{"4 passed, 0 failed."},
			code:        0,
		},
		"tfvars_in_test_dir": {
			expectedOut: []string{"2 passed, 0 failed."},
			code:        0,
		},
		"auto_tfvars_in_test_dir": {
			override:    "tfvars_in_test_dir",
			args:        []string{"-test-directory=alternate"},
			expectedOut: []string{"2 passed, 0 failed."},
			code:        0,
		},
		"functions_available": {
			expectedOut: []string{"2 passed, 0 failed."},
			code:        0,
		},
		"provider-functions-available": {
			expectedOut: []string{"1 passed, 0 failed."},
			code:        0,
		},
		"mocking": {
			expectedOut: []string{"10 passed, 0 failed."},
			code:        0,
		},
		"mocking-invalid": {
			expectedErr: []string{
				"The override_during attribute must be a value of plan or apply.",
			},
			initCode: 1,
		},
		"mocking-error": {
			expectedErr: []string{
				"Unknown condition value",
				"plan_mocked_overridden.tftest.hcl",
				"test_resource.primary[0].id",
				"plan_mocked_provider.tftest.hcl",
				"test_resource.secondary[0].id",
			},
			code: 1,
		},
		"dangling_data_block": {
			expectedOut: []string{"2 passed, 0 failed."},
			code:        0,
		},
		"skip_destroy_on_empty": {
			expectedOut: []string{"3 passed, 0 failed."},
			code:        0,
		},
		"empty_module_with_output": {
			expectedOut: []string{"1 passed, 0 failed."},
			code:        0,
		},
		"global_var_refs": {
			expectedOut: []string{"1 passed, 0 failed, 2 skipped."},
			expectedErr: []string{"The input variable \"env_var_input\" does not exist within this test file", "The input variable \"setup\" does not exist within this test file"},
			code:        1,
		},
		"global_var_ref_in_suite_var": {
			expectedOut: []string{"1 passed, 0 failed."},
			code:        0,
		},
		"env-vars": {
			expectedOut: []string{"1 passed, 0 failed."},
			envVars: map[string]string{
				"TF_VAR_input": "foo",
			},
			code: 0,
		},
		"env-vars-in-module": {
			expectedOut: []string{"2 passed, 0 failed."},
			envVars: map[string]string{
				"TF_VAR_input": "foo",
			},
			code: 0,
		},
		"ephemeral_input": {
			expectedOut: []string{"2 passed, 0 failed."},
			code:        0,
		},
		"ephemeral_input_with_error": {
			expectedOut: []string{"Error message refers to ephemeral values", "1 passed, 1 failed."},
			expectedErr: []string{"Test assertion failed",
				`│ - "(ephemeral value)"
    │ + "bar"`},
			code: 1,
		},
		"ephemeral_resource": {
			expectedOut: []string{"0 passed, 1 failed."},
			expectedErr: []string{"Ephemeral resource instance has expired", "Ephemeral resources cannot be asserted"},
			code:        1,
		},
		"with_state_key": {
			expectedOut: []string{"3 passed, 1 failed."},
			expectedErr: []string{"Test assertion failed", "resource renamed without moved block"},
			code:        1,
		},
		"unapplyable-plan": {
			expectedOut: []string{"0 passed, 1 failed."},
			expectedErr: []string{"Cannot apply non-applyable plan"},
			code:        1,
		},
		"write-only-attributes": {
			expectedOut: []string{"1 passed, 0 failed."},
			code:        0,
		},
		"write-only-attributes-mocked": {
			expectedOut: []string{"1 passed, 0 failed."},
			code:        0,
		},
		"write-only-attributes-overridden": {
			expectedOut: []string{"1 passed, 0 failed."},
			code:        0,
		},
		"with-default-variables": {
			args:        []string{"-var=input_two=universe"},
			expectedOut: []string{"2 passed, 0 failed."},
			code:        0,
		},
		"parallel-errors": {
			expectedOut: []string{"1 passed, 1 failed, 1 skipped."},
			expectedErr: []string{"Invalid condition run"},
			code:        1,
		},
		"write-into-default-state": {
			args:        []string{"-verbose"},
			expectedOut: []string{"test_resource.two will be destroyed"},
			code:        0,
		},
		"prevent-destroy": {
			expectedOut: []string{"1 passed, 0 failed."},
			code:        0,
		},
		"deferred_changes": {
			args:        []string{"-allow-deferral"},
			expectedOut: []string{"3 passed, 0 failed."},
			code:        0,
		},
		"ephemeral_output": {
			code: 0,
		},
		"ephemeral_output_referenced": {
			code: 0,
		},
		"no-tests": {
			code: 0,
		},
		"simple_pass_function": {
			expectedOut:           []string{"2 passed, 0 failed."},
			code:                  0,
			expectedResourceCount: 0,
		},
		"mocking-invalid-outputs": {
			expectedErr: []string{
				"Invalid outputs attribute",
			},
			code: 1,
		},
		"mock-sources-inline": {
			expectedOut: []string{"2 passed, 0 failed."},
			code:        0,
		},
	}
	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			if tc.skip {
				t.Skip()
			}

			for k, v := range tc.envVars {
				os.Setenv(k, v)
			}
			defer func() {
				for k := range tc.envVars {
					os.Unsetenv(k)
				}
			}()

			file := name
			if len(tc.override) > 0 {
				file = tc.override
			}

			td := t.TempDir()
			testCopyDir(t, testFixturePath(path.Join("test", file)), td)
			t.Chdir(td)

			store := &testing_command.ResourceStore{
				Data: make(map[string]cty.Value),
			}
			providerSource, close := newMockProviderSource(t, map[string][]string{
				"test": {"1.0.0"},
			})
			defer close()

			streams, done := terminal.StreamsForTesting(t)
			view := views.NewView(streams)
			ui := new(cli.MockUi)

			meta := Meta{
				testingOverrides: &testingOverrides{
					Providers: map[addrs.Provider]providers.Factory{
						addrs.NewDefaultProvider("test"): func() (providers.Interface, error) {
							return testing_command.NewProvider(store).Provider, nil
						},
					},
				},
				Ui:                        ui,
				View:                      view,
				Streams:                   streams,
				ProviderSource:            providerSource,
				AllowExperimentalFeatures: true,
			}

			init := &InitCommand{
				Meta: meta,
			}

			if code := init.Run(nil); code != tc.initCode {
				output := done(t)
				t.Fatalf("expected status code %d but got %d: %s", tc.initCode, code, output.All())
			}

			if tc.initCode > 0 {
				// Then we don't expect the init step to succeed. So we'll check
				// the init output for our expected error messages and outputs.
				output := done(t).All()
				stdout, stderr := output, output

				if len(tc.expectedOut) > 0 {
					for _, expectedOut := range tc.expectedOut {
						if !strings.Contains(stdout, expectedOut) {
							t.Errorf("output didn't contain expected string:\n\n%s", stdout)
						}
					}
				}

				if len(tc.expectedErr) > 0 {
					for _, expectedErr := range tc.expectedErr {
						if !strings.Contains(stderr, expectedErr) {
							t.Errorf("error didn't contain expected string:\n\n%s", stderr)
						}
					}
				} else if stderr != "" {
					t.Errorf("unexpected stderr output\n%s", stderr)
				}

				// If `terraform init` failed, then we don't expect that
				// `terraform test` will have run at all, so we can just return
				// here.
				return
			}

			// discard the output from the init command
			done(t)

			// Reset the streams for the next command.
			streams, done = terminal.StreamsForTesting(t)
			meta.Streams = streams
			meta.View = views.NewView(streams)

			c := &TestCommand{
				Meta: meta,
			}

			code := c.Run(append(tc.args, "-no-color"))
			output := done(t)

			if code != tc.code {
				t.Errorf("expected status code %d but got %d:\n\n%s", tc.code, code, output.All())
			}

			if len(tc.expectedOut) > 0 {
				for _, expectedOut := range tc.expectedOut {
					if !strings.Contains(output.Stdout(), expectedOut) {
						t.Errorf("output didn't contain expected string (%q):\n\n%s", expectedOut, output.Stdout())
					}
				}
			}

			if len(tc.expectedErr) > 0 {
				for _, expectedErr := range tc.expectedErr {
					if !strings.Contains(output.Stderr(), expectedErr) {
						t.Errorf("error didn't contain expected string (%q):\n\n%s", expectedErr, output.Stderr())
					}
				}
			} else if output.Stderr() != "" {
				t.Errorf("unexpected stderr output\n%s", output.Stderr())
			}

			if len(store.Data) != tc.expectedResourceCount {
				t.Errorf("should have left %d resources on completion but left %v", tc.expectedResourceCount, len(store.Data))
			}
		})
	}
}

func TestTest_Interrupt(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath(path.Join("test", "with_interrupt")), td)
	t.Chdir(td)

	provider := testing_command.NewProvider(nil)
	view, done := testView(t)

	interrupt := make(chan struct{})
	provider.Interrupt = interrupt

	c := &TestCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(provider.Provider),
			View:             view,
			ShutdownCh:       interrupt,
		},
	}

	c.Run(nil)
	output := done(t).All()

	if !strings.Contains(output, "Interrupt received") {
		t.Errorf("output didn't produce the right output:\n\n%s", output)
	}

	if provider.ResourceCount() > 0 {
		// we asked for a nice stop in this one, so it should still have tidied everything up.
		t.Errorf("should have deleted all resources on completion but left %v", provider.ResourceString())
	}
}

func TestTest_DestroyFail(t *testing.T) {
	// Testing that when a cleanup fails, we leave behind state files of the failed
	// resources, and that the test command fails with a non-zero exit code.
	td := t.TempDir()
	testCopyDir(t, testFixturePath(path.Join("test", "destroy_fail")), td)
	t.Chdir(td)

	provider := testing_command.NewProvider(nil)
	providerSource, close := newMockProviderSource(t, map[string][]string{
		"test": {"1.0.0"},
	})
	defer close()

	view, done := testView(t)

	meta := Meta{
		testingOverrides: metaOverridesForProvider(provider.Provider),
		View:             view,
		ProviderSource:   providerSource,
	}

	init := &InitCommand{Meta: meta}
	if code := init.Run(nil); code != 0 {
		output := done(t)
		t.Fatalf("expected status code %d but got %d: %s", 0, code, output.All())
	}

	interrupt := make(chan struct{})
	provider.Interrupt = interrupt
	view, done = testView(t)

	c := &TestCommand{
		Meta: Meta{
			testingOverrides:          metaOverridesForProvider(provider.Provider),
			View:                      view,
			ShutdownCh:                interrupt,
			AllowExperimentalFeatures: true,
		},
	}

	c.Run([]string{"-no-color"})
	output := done(t)
	err := output.Stderr()

	cleanupMessage := `main.tftest.hcl... in progress
  run "setup"... pass
  run "single"... pass
  run "double"... pass
main.tftest.hcl... tearing down
main.tftest.hcl... fail

Failure! 3 passed, 0 failed.
`

	cleanupErr := `Terraform encountered an error destroying resources created while executing
main.tftest.hcl/double.

Error: Failed to destroy resource

destroy_fail is set to true

Error: Failed to destroy resource

destroy_fail is set to true

Terraform left the following resources in state after executing
main.tftest.hcl/double, and they need to be cleaned up manually:
  - test_resource.another
  - test_resource.resource
Terraform encountered an error destroying resources created while executing
main.tftest.hcl/single.

Error: Failed to destroy resource

destroy_fail is set to true

Error: Failed to destroy resource

destroy_fail is set to true

Terraform left the following resources in state after executing
main.tftest.hcl/single, and they need to be cleaned up manually:
  - test_resource.another
  - test_resource.resource
`

	// It's really important that the above message is printed, so we're testing
	// for it specifically and making sure it contains all the resources.
	if diff := cmp.Diff(cleanupErr, err); diff != "" {
		t.Errorf("expected err to be\n%s\n\nbut got\n%s\n\n diff:\n%s\n", cleanupErr, err, diff)
	}
	if diff := cmp.Diff(cleanupMessage, output.Stdout()); diff != "" {
		t.Errorf("expected output to be \n%s\n\nbut got \n%s\n\n diff:\n%s\n", cleanupMessage, output.Stdout(), diff)
	}

	if provider.ResourceCount() != 4 {
		t.Errorf("should not have deleted all resources on completion but only has %v", provider.ResourceString())
	}

	expectedStates := map[string][]string{
		"main.":       {"test_resource.another", "test_resource.resource"},
		"main.double": {"test_resource.another", "test_resource.resource"},
	}
	if diff := cmp.Diff(expectedStates, statesFromManifest(t, td)); diff != "" {
		t.Fatalf("unexpected states: %s", diff)
	}

	t.Run("cleanup failed state", func(t *testing.T) {
		interrupt := make(chan struct{})
		provider.Interrupt = interrupt
		provider.Provider.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
			return providers.PlanResourceChangeResponse{
				PlannedState: req.ProposedNewState,
			}
		}
		provider.Provider.ApplyResourceChangeFn = func(req providers.ApplyResourceChangeRequest) providers.ApplyResourceChangeResponse {
			return providers.ApplyResourceChangeResponse{
				NewState: req.PlannedState,
			}
		}
		view, done = testView(t)

		c := &TestCleanupCommand{
			Meta: Meta{
				testingOverrides:          metaOverridesForProvider(provider.Provider),
				View:                      view,
				ShutdownCh:                interrupt,
				AllowExperimentalFeatures: true,
			},
		}

		c.Run([]string{"-no-color"})
		output := done(t)

		actualCleanup := output.Stdout()
		expectedCleanup := `main.tftest.hcl... in progress
main.tftest.hcl... tearing down
main.tftest.hcl... pass

Success!
`
		if diff := cmp.Diff(expectedCleanup, actualCleanup); diff != "" {
			t.Fatalf("unexpected cleanup output: expected %s\n, got %s\n, diff: %s", expectedCleanup, actualCleanup, diff)
		}

		expectedStates := map[string][]string{}

		if diff := cmp.Diff(expectedStates, statesFromManifest(t, td)); diff != "" {
			t.Fatalf("unexpected states after cleanup: %s", diff)
		}
	})
}

func TestTest_Cleanup(t *testing.T) {
	// function to consolidate the test command that should generate some state files and manifest
	// It also does assertions.
	executeTestCmd := func(provider *testing_command.TestProvider, providerSource *getproviders.MockSource) (td string) {
		td = t.TempDir()
		testCopyDir(t, testFixturePath(path.Join("test", "cleanup")), td)
		t.Chdir(td)

		view, done := testView(t)
		meta := Meta{
			testingOverrides:          metaOverridesForProvider(provider.Provider),
			View:                      view,
			ProviderSource:            providerSource,
			AllowExperimentalFeatures: true,
		}

		init := &InitCommand{Meta: meta}
		if code := init.Run(nil); code != 0 {
			output := done(t)
			t.Fatalf("expected status code %d but got %d: %s", 0, code, output.All())
		}
		interrupt := make(chan struct{})
		provider.Interrupt = interrupt
		view, done = testView(t)

		c := &TestCommand{
			Meta: Meta{
				testingOverrides:          metaOverridesForProvider(provider.Provider),
				View:                      view,
				ShutdownCh:                interrupt,
				AllowExperimentalFeatures: true,
			},
		}

		c.Run([]string{"-no-color"})
		output := done(t)

		message := `main.tftest.hcl... in progress
  run "test"... pass
  run "test_two"... pass
  run "test_three"... pass
  run "test_four"... pass
main.tftest.hcl... tearing down
main.tftest.hcl... fail

Failure! 4 passed, 0 failed.
`

		outputErr := `Terraform encountered an error destroying resources created while executing
main.tftest.hcl/test_three.

Error: Failed to destroy resource

destroy_fail is set to true

Terraform left the following resources in state after executing
main.tftest.hcl/test_three, and they need to be cleaned up manually:
  - test_resource.resource
`
		if diff := cmp.Diff(outputErr, output.Stderr()); diff != "" {
			t.Errorf("expected err to be %s\n\nbut got %s\n\n diff:%s\n", outputErr, output.Stderr(), diff)
		}
		if diff := cmp.Diff(message, output.Stdout()); diff != "" {
			t.Errorf("expected output to be %s\n\nbut got %s\n\n diff:%s\n", message, output.Stdout(), diff)
		}

		if provider.ResourceCount() != 2 {
			t.Errorf("should have 2 resources on completion but only has %v", provider.ResourceString())
		}

		expectedStates := map[string][]string{
			"main.":            {"test_resource.resource"},
			"main.state_three": {"test_resource.resource"},
		}
		actual := removeOutputs(statesFromManifest(t, td))

		if diff := cmp.Diff(expectedStates, actual); diff != "" {
			t.Fatalf("unexpected states: %s", diff)
		}

		return
	}

	t.Run("cleanup all left-over state", func(t *testing.T) {
		provider := testing_command.NewProvider(nil)
		providerSource, close := newMockProviderSource(t, map[string][]string{
			"test": {"1.0.0"},
		})
		defer close()

		// Run the test command to create the state
		td := executeTestCmd(provider, providerSource)
		interrupt := make(chan struct{})
		provider.Interrupt = interrupt
		provider.Provider.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
			return providers.PlanResourceChangeResponse{
				PlannedState: req.ProposedNewState,
			}
		}
		provider.Provider.ApplyResourceChangeFn = func(req providers.ApplyResourceChangeRequest) providers.ApplyResourceChangeResponse {
			var diags tfdiags.Diagnostics
			// Simulate an error during apply, unless it is a destroy operation
			if !req.PlannedState.IsNull() {
				diags = diags.Append(fmt.Errorf("apply error"))
			}
			return providers.ApplyResourceChangeResponse{
				NewState:    req.PlannedState,
				Diagnostics: diags,
			}
		}
		view, done := testView(t)

		c := &TestCleanupCommand{
			Meta: Meta{
				testingOverrides:          metaOverridesForProvider(provider.Provider),
				View:                      view,
				ShutdownCh:                interrupt,
				AllowExperimentalFeatures: true,
			},
		}

		c.Run([]string{"-no-color"})
		output := done(t)

		expectedCleanup := `main.tftest.hcl... in progress
main.tftest.hcl... tearing down
main.tftest.hcl... pass

Success!
`
		if diff := cmp.Diff(expectedCleanup, output.Stdout()); diff != "" {
			t.Errorf("unexpected cleanup output: expected %s\n, got %s\n, diff: %s", expectedCleanup, output.Stdout(), diff)
		}

		expectedStates := map[string][]string{}
		actualStates := removeOutputs(statesFromManifest(t, td))

		if diff := cmp.Diff(expectedStates, actualStates); diff != "" {
			t.Fatalf("unexpected states after cleanup: %s", diff)
		}
	})

	t.Run("cleanup failed state only (-repair)", func(t *testing.T) {
		provider := testing_command.NewProvider(nil)
		providerSource, close := newMockProviderSource(t, map[string][]string{
			"test": {"1.0.0"},
		})
		defer close()

		// Run the test command to create the state
		td := executeTestCmd(provider, providerSource)

		interrupt := make(chan struct{})
		provider.Interrupt = interrupt
		provider.Provider.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
			return providers.PlanResourceChangeResponse{
				PlannedState: req.ProposedNewState,
			}
		}
		provider.Provider.ApplyResourceChangeFn = func(req providers.ApplyResourceChangeRequest) providers.ApplyResourceChangeResponse {
			return providers.ApplyResourceChangeResponse{
				NewState: req.PlannedState,
			}
		}
		view, done := testView(t)

		c := &TestCleanupCommand{
			Meta: Meta{
				testingOverrides:          metaOverridesForProvider(provider.Provider),
				View:                      view,
				ShutdownCh:                interrupt,
				AllowExperimentalFeatures: true,
			},
		}

		c.Run([]string{"-no-color", "-repair"})
		output := done(t)

		expectedCleanup := `main.tftest.hcl... in progress
main.tftest.hcl... tearing down
main.tftest.hcl... pass

Success!
`
		if diff := cmp.Diff(expectedCleanup, output.Stdout()); diff != "" {
			t.Fatalf("unexpected cleanup output: expected %s\n, got %s\n, diff: %s", expectedCleanup, output.Stdout(), diff)
		}

		expectedStates := map[string][]string{
			"main.": {"test_resource.resource"},
		}
		actual := removeOutputs(statesFromManifest(t, td))

		if diff := cmp.Diff(expectedStates, actual); diff != "" {
			t.Fatalf("unexpected states after cleanup: %s", diff)
		}
	})
}

func TestTest_CleanupActuallyCleansUp(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath(path.Join("test", "skip_cleanup_simple")), td)
	t.Chdir(td)

	provider := testing_command.NewProvider(nil)
	providerSource, close := newMockProviderSource(t, map[string][]string{
		"test": {"1.0.0"},
	})
	defer close()

	streams, done := terminal.StreamsForTesting(t)
	view := views.NewView(streams)
	ui := new(cli.MockUi)

	meta := Meta{
		testingOverrides:          metaOverridesForProvider(provider.Provider),
		Ui:                        ui,
		View:                      view,
		Streams:                   streams,
		ProviderSource:            providerSource,
		AllowExperimentalFeatures: true,
	}

	init := &InitCommand{
		Meta: meta,
	}

	output := done(t)

	if code := init.Run(nil); code != 0 {
		t.Fatalf("expected status code 0 but got %d: %s", code, output.All())
	}

	// Reset the streams for the next command.
	streams, done = terminal.StreamsForTesting(t)
	meta.Streams = streams
	meta.View = views.NewView(streams)

	c := &TestCommand{
		Meta: meta,
	}

	code := c.Run([]string{"-no-color"})
	output = done(t)

	if code != 0 {
		t.Errorf("expected status code 0 but got %d: %s", code, output.All())
	}

	// Run the cleanup command.

	// Reset the streams for the next command.
	streams, done = terminal.StreamsForTesting(t)
	meta.Streams = streams
	meta.View = views.NewView(streams)

	cleanup := &TestCleanupCommand{
		Meta: meta,
	}

	code = cleanup.Run([]string{"-no-color"})
	output = done(t)

	if code != 0 {
		t.Errorf("expected status code 0 but got %d: %s", code, output.All())
	}

	// Reset the streams for the next command.
	// Running the test again should now work, because we cleaned everything
	// up.

	// Reset the streams for the next command.
	streams, done = terminal.StreamsForTesting(t)
	meta.Streams = streams
	meta.View = views.NewView(streams)

	c = &TestCommand{
		Meta: meta,
	}

	code = c.Run([]string{"-no-color"})
	output = done(t)

	if code != 0 {
		t.Errorf("expected status code 0 but got %d: %s", code, output.All())
	}
}

func TestTest_SkipCleanup_ConsecutiveTestsFail(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath(path.Join("test", "skip_cleanup_simple")), td)
	t.Chdir(td)

	provider := testing_command.NewProvider(nil)
	providerSource, close := newMockProviderSource(t, map[string][]string{
		"test": {"1.0.0"},
	})
	defer close()

	streams, done := terminal.StreamsForTesting(t)
	view := views.NewView(streams)
	ui := new(cli.MockUi)

	meta := Meta{
		testingOverrides:          metaOverridesForProvider(provider.Provider),
		Ui:                        ui,
		View:                      view,
		Streams:                   streams,
		ProviderSource:            providerSource,
		AllowExperimentalFeatures: true,
	}

	init := &InitCommand{
		Meta: meta,
	}

	output := done(t)

	if code := init.Run(nil); code != 0 {
		t.Fatalf("expected status code 0 but got %d: %s", code, output.All())
	}

	// Reset the streams for the next command.
	streams, done = terminal.StreamsForTesting(t)
	meta.Streams = streams
	meta.View = views.NewView(streams)

	c := &TestCommand{
		Meta: meta,
	}

	code := c.Run([]string{"-no-color"})
	output = done(t)

	if code != 0 {
		t.Errorf("expected status code 0 but got %d: %s", code, output.All())
	}

	// Reset the streams for the next command.
	// Running the test again should fail because of the skip cleanup.

	// Reset the streams for the next command.
	streams, done = terminal.StreamsForTesting(t)
	meta.Streams = streams
	meta.View = views.NewView(streams)

	c = &TestCommand{
		Meta: meta,
	}

	code = c.Run([]string{"-no-color"})
	output = done(t)

	if code == 0 {
		t.Errorf("expected status code 0 but got %d", code)
	}

	expectedOut := "main.tftest.hcl... in progress\nmain.tftest.hcl... tearing down\nmain.tftest.hcl... fail\n\nFailure! 0 passed, 0 failed.\n"
	expectedErr := "\nError: State manifest not empty\n\nThe state manifest for main.tftest.hcl should be empty before running tests.\nThis could be due to a previous test run not cleaning up after itself. Please\nensure that all state files are cleaned up before running tests.\n"

	if diff := cmp.Diff(expectedOut, output.Stdout()); len(diff) > 0 {
		t.Error(diff)
	}
	if diff := cmp.Diff(expectedErr, output.Stderr()); len(diff) > 0 {
		t.Error(diff)
	}
}

func TestTest_SharedState_Order(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath(path.Join("test", "shared_state")), td)
	t.Chdir(td)

	provider := testing_command.NewProvider(nil)
	providerSource, close := newMockProviderSource(t, map[string][]string{
		"test": {"1.0.0"},
	})
	defer close()

	streams, done := terminal.StreamsForTesting(t)
	view := views.NewView(streams)
	ui := new(cli.MockUi)

	meta := Meta{
		testingOverrides: metaOverridesForProvider(provider.Provider),
		Ui:               ui,
		View:             view,
		Streams:          streams,
		ProviderSource:   providerSource,
	}

	init := &InitCommand{
		Meta: meta,
	}

	if code := init.Run(nil); code != 0 {
		output := done(t)
		t.Fatalf("expected status code %d but got %d: %s", 9, code, output.All())
	}

	c := &TestCommand{
		Meta: meta,
	}

	c.Run(nil)
	output := done(t).All()

	// Split the log into lines
	lines := strings.Split(output, "\n")

	var arr []string
	for _, line := range lines {
		if strings.Contains(line, "run \"") && strings.Contains(line, "\x1b[32mpass") {
			arr = append(arr, line)
		}
	}

	// Ensure the order of the tests is correct. Even though they share no state,
	// the order should be sequential.
	expectedOrder := []string{
		// main.tftest.hcl
		"run \"setup\"",
		"run \"test\"",

		// no-shared-state.tftest.hcl
		"run \"setup\"",
		"run \"test_a\"",
		"run \"test_b\"",
		"run \"test_c\"",
		"run \"test_d\"",
		"run \"test_e\"",
	}

	for i, line := range expectedOrder {
		if !strings.Contains(arr[i], line) {
			t.Errorf("unexpected test order: expected %q, got %q", line, arr[i])
		}
	}
}

func TestTest_Parallel_Divided_Order(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath(path.Join("test", "parallel_divided")), td)
	t.Chdir(td)

	provider := testing_command.NewProvider(nil)
	providerSource, close := newMockProviderSource(t, map[string][]string{
		"test": {"1.0.0"},
	})
	defer close()

	streams, done := terminal.StreamsForTesting(t)
	view := views.NewView(streams)
	ui := new(cli.MockUi)

	meta := Meta{
		testingOverrides: metaOverridesForProvider(provider.Provider),
		Ui:               ui,
		View:             view,
		Streams:          streams,
		ProviderSource:   providerSource,
	}

	init := &InitCommand{
		Meta: meta,
	}

	if code := init.Run(nil); code != 0 {
		output := done(t)
		t.Fatalf("expected status code %d but got %d: %s", 9, code, output.All())
	}

	c := &TestCommand{
		Meta: meta,
	}

	c.Run(nil)
	output := done(t).All()

	// Split the log into lines
	lines := strings.Split(output, "\n")

	// Find the positions of the tests in the log output
	var mainFirstIndex, mainSecondIndex, mainThirdIndex, mainFourthIndex, mainFifthIndex, mainSixthIndex int
	for i, line := range lines {
		if strings.Contains(line, "run \"main_first\"") {
			mainFirstIndex = i
		} else if strings.Contains(line, "run \"main_second\"") {
			mainSecondIndex = i
		} else if strings.Contains(line, "run \"main_third\"") {
			mainThirdIndex = i
		} else if strings.Contains(line, "run \"main_fourth\"") {
			mainFourthIndex = i
		} else if strings.Contains(line, "run \"main_fifth\"") {
			mainFifthIndex = i
		} else if strings.Contains(line, "run \"main_sixth\"") {
			mainSixthIndex = i
		}
	}
	if mainFirstIndex == 0 || mainSecondIndex == 0 || mainThirdIndex == 0 || mainFourthIndex == 0 || mainFifthIndex == 0 || mainSixthIndex == 0 {
		t.Fatalf("one or more tests not found in the log output")
	}

	// Ensure the order of the tests is correct. The runs before main_fourth can execute in parallel.
	if mainFirstIndex > mainFourthIndex || mainSecondIndex > mainFourthIndex || mainThirdIndex > mainFourthIndex {
		t.Errorf("main_first, main_second, or main_third appears after main_fourth in the log output")
	}

	// Ensure main_fifth and main_sixth do not execute before main_fourth
	if mainFifthIndex < mainFourthIndex {
		t.Errorf("main_fifth appears before main_fourth in the log output")
	}
	if mainSixthIndex < mainFourthIndex {
		t.Errorf("main_sixth appears before main_fourth in the log output")
	}
}

func TestTest_Parallel(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath(path.Join("test", "parallel")), td)
	t.Chdir(td)

	provider := testing_command.NewProvider(nil)
	providerSource, close := newMockProviderSource(t, map[string][]string{
		"test": {"1.0.0"},
	})
	defer close()

	streams, done := terminal.StreamsForTesting(t)
	view := views.NewView(streams)
	ui := new(cli.MockUi)

	meta := Meta{
		testingOverrides: metaOverridesForProvider(provider.Provider),
		Ui:               ui,
		View:             view,
		Streams:          streams,
		ProviderSource:   providerSource,
	}

	init := &InitCommand{
		Meta: meta,
	}

	if code := init.Run(nil); code != 0 {
		output := done(t)
		t.Fatalf("expected status code %d but got %d: %s", 9, code, output.All())
	}

	c := &TestCommand{
		Meta: meta,
	}

	c.Run(nil)
	output := done(t).All()

	if !strings.Contains(output, "40 passed, 0 failed") {
		t.Errorf("output didn't produce the right output:\n\n%s", output)
	}

	// Split the log into lines
	lines := strings.Split(output, "\n")

	// Find the positions of "test_d", "test_c", "test_setup" in the log output
	var testDIndex, testCIndex, testSetupIndex int
	for i, line := range lines {
		if strings.Contains(line, "run \"setup\"") {
			testSetupIndex = i
		} else if strings.Contains(line, "run \"test_d\"") {
			testDIndex = i
		} else if strings.Contains(line, "run \"test_c\"") {
			testCIndex = i
		}
	}
	if testDIndex == 0 || testCIndex == 0 || testSetupIndex == 0 {
		t.Fatalf("test_d, test_c, or test_setup not found in the log output")
	}

	// Ensure "test_d" appears before "test_c", because test_d has no dependencies,
	// and would therefore run in parallel to much earlier tests which test_c depends on.
	if testDIndex > testCIndex {
		t.Errorf("test_d appears after test_c in the log output")
	}

	// Ensure "test_d" appears after "test_setup", because they have the same state key
	if testDIndex < testSetupIndex {
		t.Errorf("test_d appears before test_setup in the log output")
	}
}

func TestTest_ParallelTeardown(t *testing.T) {
	tests := []struct {
		name       string
		sources    map[string]string
		assertFunc func(t *testing.T, output string, dur time.Duration)
	}{
		{
			name: "parallel teardown",
			sources: map[string]string{
				"main.tf": `
					variable "input" {
					type = string
					}

					resource "test_resource" "foo" {
					value = var.input
					destroy_wait_seconds = 3
					}

					output "value" {
					value = test_resource.foo.value
					}
					`,
				"parallel.tftest.hcl": `
					test {
					parallel = true
					}

					variables {
					foo = "foo"
					}

					provider "test" {
					}

					provider "test" {
					alias = "start"
					}

					run "test_a" {
					state_key = "state_foo"
					variables {
						input = "foo"
					}
					providers = {
						test = test
					}

					assert {
						condition     = output.value == var.foo
						error_message = "error in test_a"
					}
					}

					run "test_b" {
					state_key = "state_bar"
					variables {
						input = "bar"
					}

					providers = {
						test = test.start
					}

					assert {
						condition     = output.value == "bar"
						error_message = "error in test_b"
					}
					}
					`,
			},
			assertFunc: func(t *testing.T, output string, dur time.Duration) {
				if !strings.Contains(output, "2 passed, 0 failed") {
					t.Errorf("output didn't produce the right output:\n\n%s", output)
				}
				// Each teardown sleeps for 3 seconds, so we expect the total duration to be less than 6 seconds.
				if dur >= 6*time.Second {
					t.Fatalf("parallel.tftest.hcl duration took too long: %0.2f seconds", dur.Seconds())
				}
			},
		},
		{
			name: "reference prevents parallel teardown",
			sources: map[string]string{
				"main.tf": `
					variable "input" {
						type = string
					}

					resource "test_resource" "foo" {
						value = var.input
						destroy_wait_seconds = 5
					}

					output "value" {
						value = test_resource.foo.value
					}
					`,
				"parallel.tftest.hcl": `
					test {
						parallel = true
					}

					variables {
						foo = "foo"
					}

					provider "test" {
					}

					provider "test" {
						alias = "start"
					}

					run "test_a" {
						state_key = "state_foo"
						variables {
							input = "foo"
						}
						providers = {
							test = test
						}

						assert {
							condition     = output.value == var.foo
							error_message = "error in test_a"
						}
					}

					run "test_b" {
						state_key = "state_bar"
						variables {
							input = "bar"
						}

						providers = {
							test = test.start
						}

						assert {
							condition     = output.value != run.test_a.value
							error_message = "error in test_b"
						}
					}
					`,
			},
			assertFunc: func(t *testing.T, output string, dur time.Duration) {
				if !strings.Contains(output, "2 passed, 0 failed") {
					t.Errorf("output didn't produce the right output:\n\n%s", output)
				}
				// Each teardown sleeps for 5 seconds, so we expect the total duration to be at least 10 seconds.
				if dur < 10*time.Second {
					t.Fatalf("parallel.tftest.hcl duration took too short: %0.2f seconds", dur.Seconds())
				}
			},
		},
		{
			name: "possible cyclic state key reference: skip edge that would cause cycle",
			sources: map[string]string{
				"main.tf": `
					variable "foo" {
						type = string
					}

					resource "test_resource" "foo" {
						value = var.foo
						// destroy_wait_seconds = 5
					}

					output "value" {
						value = test_resource.foo.value
					}
					`,
				// c2 => a1, b1 => a1, a2 => b1, b2 => c1
				"parallel.tftest.hcl": `
					test {
						parallel = true
					}

					variables {
						foo = "foo"
						indirect = run.c1.value
					}

					provider "test" {
					}

					provider "test" {
						alias = "start"
					}

					run "a1" {
						state_key = "a"
						variables {
							foo = "foo"
						}

						providers = {
							test = test
						}
					}

					run "b1" {
						state_key = "b"
						variables {
							foo = run.a1.value // no destroy edge here, because b2 owns the destroy node.
						}

						providers = {
							test = test
						}
					}

					run "a2" {
						state_key = "a"
						variables {
							foo = run.b1.value
						}

						providers = {
							test = test
						}
					}

					run "b2" {
						state_key = "b"
						variables {
							foo = var.indirect # This is an indirect reference to run.c1.value
							unused = run.b1.value
						}

						providers = {
							test = test
						}
					}

					run "c1" {
						state_key = "c"
						variables {
							foo = "foo"
						}
					}

					run "c2" {
						state_key = "c"
						variables {
							foo = run.a1.value
						}
					}
					`,
			},
			assertFunc: func(t *testing.T, output string, dur time.Duration) {
				if !strings.Contains(output, "6 passed, 0 failed") {
					t.Errorf("output didn't produce the right output:\n\n%s", output)
				}

				lines := strings.Split(output, "\n")
				aIdx, bIdx, cIdx := -1, -1, -1
				for idx, line := range lines {
					if strings.Contains(line, "tearing down") {
						if strings.Contains(line, "a2") {
							aIdx = idx
						}
						if strings.Contains(line, "b2") {
							bIdx = idx
						}
						if strings.Contains(line, "c2") {
							cIdx = idx
						}
					}
				}

				if cIdx > aIdx || aIdx > bIdx { // c => a => b
					t.Errorf("teardown order is incorrect: c2 (%d), a2 (%d), b2 (%d)", cIdx, aIdx, bIdx)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, td, closer := testModuleInline(t, tt.sources)
			defer closer()
			t.Chdir(td)

			providerSource, close := newMockProviderSource(t, map[string][]string{
				"test": {"1.0.0"},
			})
			defer close()

			streams, done := terminal.StreamsForTesting(t)
			view := views.NewView(streams)
			ui := new(cli.MockUi)

			// create a new provider instance for each test run, so that we can
			// ensure that the test provider locks do not interfere between runs.
			pInst := func() providers.Interface {
				return testing_command.NewProvider(nil).Provider
			}
			meta := Meta{
				testingOverrides: &testingOverrides{
					Providers: map[addrs.Provider]providers.Factory{
						addrs.NewDefaultProvider("test"): func() (providers.Interface, error) {
							return pInst(), nil
						},
					}},
				Ui:             ui,
				View:           view,
				Streams:        streams,
				ProviderSource: providerSource,
			}

			init := &InitCommand{Meta: meta}
			if code := init.Run(nil); code != 0 {
				output := done(t)
				t.Fatalf("expected status code %d but got %d: %s", 0, code, output.All())
			}

			c := &TestCommand{Meta: meta}
			c.Run([]string{"-json", "-no-color"})
			output := done(t).All()

			// Split the log into lines
			lines := strings.Split(output, "\n")

			// Find the start of the teardown and complete timestamps
			var startTimestamp, completeTimestamp string
			for _, line := range lines {
				if strings.Contains(line, `{"path":"parallel.tftest.hcl","progress":"teardown"`) {
					var obj map[string]interface{}
					if err := json.Unmarshal([]byte(line), &obj); err == nil {
						if ts, ok := obj["@timestamp"].(string); ok {
							startTimestamp = ts
						}
					}
				} else if strings.Contains(line, `{"path":"parallel.tftest.hcl","progress":"complete"`) {
					var obj map[string]interface{}
					if err := json.Unmarshal([]byte(line), &obj); err == nil {
						if ts, ok := obj["@timestamp"].(string); ok {
							completeTimestamp = ts
						}
					}
				}
			}

			if startTimestamp == "" || completeTimestamp == "" {
				t.Fatalf("could not find start or complete timestamp in log output")
			}

			startTime, err := time.Parse(time.RFC3339Nano, startTimestamp)
			if err != nil {
				t.Fatalf("failed to parse start timestamp: %v", err)
			}
			completeTime, err := time.Parse(time.RFC3339Nano, completeTimestamp)
			if err != nil {
				t.Fatalf("failed to parse complete timestamp: %v", err)
			}
			dur := completeTime.Sub(startTime)
			if tt.assertFunc != nil {
				tt.assertFunc(t, output, dur)
			}
		})
	}
}

func TestTest_InterruptSkipsRemaining(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath(path.Join("test", "with_interrupt_and_additional_file")), td)
	t.Chdir(td)

	provider := testing_command.NewProvider(nil)
	view, done := testView(t)

	interrupt := make(chan struct{})
	provider.Interrupt = interrupt

	c := &TestCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(provider.Provider),
			View:             view,
			ShutdownCh:       interrupt,
		},
	}

	c.Run([]string{"-no-color"})
	output := done(t).All()

	if !strings.Contains(output, "skip_me.tftest.hcl... skip") {
		t.Errorf("output didn't produce the right output:\n\n%s", output)
	}

	if provider.ResourceCount() > 0 {
		// we asked for a nice stop in this one, so it should still have tidied everything up.
		t.Errorf("should have deleted all resources on completion but left %v", provider.ResourceString())
	}
}

func TestTest_DoubleInterrupt(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath(path.Join("test", "with_double_interrupt")), td)
	t.Chdir(td)

	provider := testing_command.NewProvider(nil)
	view, done := testView(t)

	interrupt := make(chan struct{})
	provider.Interrupt = interrupt

	c := &TestCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(provider.Provider),
			View:             view,
			ShutdownCh:       interrupt,
		},
	}

	c.Run(nil)
	output := done(t).All()

	if !strings.Contains(output, "Two interrupts received") {
		t.Errorf("output didn't produce the right output:\n\n%s", output)
	}

	cleanupMessage := `Terraform was interrupted while executing main.tftest.hcl, and may not have
performed the expected cleanup operations.

Terraform has already created the following resources from the module under
test:
  - test_resource.primary
  - test_resource.secondary
  - test_resource.tertiary`

	// It's really important that the above message is printed, so we're testing
	// for it specifically and making sure it contains all the resources.
	if !strings.Contains(output, cleanupMessage) {
		t.Errorf("output didn't produce the right output:\n\n%s", output)
	}

	// This time the test command shouldn't have cleaned up the resource because
	// of the hard interrupt.
	if provider.ResourceCount() != 3 {
		// we asked for a nice stop in this one, so it should still have tidied everything up.
		t.Errorf("should not have deleted all resources on completion but left %v", provider.ResourceString())
	}
}

func TestTest_ProviderAlias(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath(path.Join("test", "with_provider_alias")), td)
	t.Chdir(td)

	store := &testing_command.ResourceStore{
		Data: make(map[string]cty.Value),
	}

	providerSource, close := newMockProviderSource(t, map[string][]string{
		"test": {"1.0.0"},
	})
	defer close()

	streams, done := terminal.StreamsForTesting(t)
	view := views.NewView(streams)
	ui := new(cli.MockUi)

	meta := Meta{
		testingOverrides: &testingOverrides{
			Providers: map[addrs.Provider]providers.Factory{
				addrs.NewDefaultProvider("test"): func() (providers.Interface, error) {
					return testing_command.NewProvider(store).Provider, nil
				},
			},
		},
		Ui:             ui,
		View:           view,
		Streams:        streams,
		ProviderSource: providerSource,
	}

	init := &InitCommand{
		Meta: meta,
	}

	output := done(t)

	if code := init.Run([]string{"-no-color"}); code != 0 {
		t.Fatalf("expected status code 0 but got %d: %s", code, output.All())
	}

	// Reset the streams for the next command.
	streams, done = terminal.StreamsForTesting(t)
	meta.Streams = streams
	meta.View = views.NewView(streams)

	command := &TestCommand{
		Meta: meta,
	}

	code := command.Run([]string{"-no-color"})
	output = done(t)

	printedOutput := false

	if code != 0 {
		printedOutput = true
		t.Errorf("expected status code 0 but got %d: %s", code, output.All())
	}

	if len(store.Data) > 0 {
		if !printedOutput {
			t.Errorf("should have deleted all resources on completion but left %d\n\n%s", len(store.Data), output.All())
		} else {
			t.Errorf("should have deleted all resources on completion but left %d", len(store.Data))
		}
	}
}

func TestTest_ComplexCondition(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath(path.Join("test", "complex_condition")), td)
	t.Chdir(td)

	provider := testing_command.NewProvider(nil)

	providerSource, close := newMockProviderSource(t, map[string][]string{"test": {"1.0.0"}})
	defer close()

	streams, done := terminal.StreamsForTesting(t)
	view := views.NewView(streams)
	ui := new(cli.MockUi)

	meta := Meta{
		testingOverrides: metaOverridesForProvider(provider.Provider),
		Ui:               ui,
		View:             view,
		Streams:          streams,
		ProviderSource:   providerSource,
	}

	init := &InitCommand{
		Meta: meta,
	}

	output := done(t)

	if code := init.Run([]string{"-no-color"}); code != 0 {
		t.Fatalf("expected status code 0 but got %d: %s", code, output.All())
	}

	// Reset the streams for the next command.
	streams, done = terminal.StreamsForTesting(t)
	meta.Streams = streams
	meta.View = views.NewView(streams)

	command := &TestCommand{
		Meta: meta,
	}

	code := command.Run([]string{"-no-color"})
	output = done(t)

	printedOutput := false

	if code != 1 {
		printedOutput = true
		t.Errorf("expected status code 1 but got %d: %s", code, output.All())
	}

	expectedOut := `main.tftest.hcl... in progress
  run "validate_diff_types"... fail
  run "validate_output"... fail
  run "validate_complex_output"... fail
  run "validate_complex_output_sensitive"... fail
  run "validate_complex_output_pass"... pass
main.tftest.hcl... tearing down
main.tftest.hcl... fail

Failure! 1 passed, 4 failed.
`

	expectedErr := `
Error: Test assertion failed

  on main.tftest.hcl line 37, in run "validate_diff_types":
  37:     condition = var.tr1 == var.tr2 
    ├────────────────
    │ Warning: LHS and RHS values are of different types


expected to fail

Error: Test assertion failed

  on main.tftest.hcl line 44, in run "validate_output":
  44:     condition = output.foo == var.foo
    ├────────────────
    │ Diff:
    │ --- actual
    │ +++ expected
    │   {
    │ -   "bar": "notbaz",
    │ +   "bar": "baz",
    │     "matches": "matches",
    │ -   "qux": "quux",
    │ -   "xuq": "xuq"
    │ +   "qux": "qux",
    │ +   "xuq": "nope"
    │   }


expected to fail due to different values

Error: Test assertion failed

  on main.tftest.hcl line 52, in run "validate_complex_output":
  52:     condition = output.complex == var.bar
    ├────────────────
    │ Warning: LHS and RHS values are of different types
    │ Diff:
    │ --- actual
    │ +++ expected
    │   {
    │     "root": [
    │       {
    │         "bar": [
    │           1
    │         ],
    │ -       "qux": "quux"
    │ +       "qux": "qux"
    │       },
    │       {
    │         "bar": [
    │           2
    │         ],
    │         "qux": "quux"
    │       }
    │     ]
    │   }


expected to fail

Error: Test assertion failed

  on main.tftest.hcl line 60, in run "validate_complex_output_sensitive":
  60:     condition = output.complex == output.complex_sensitive
    ├────────────────
    │ Diff:
    │ --- actual
    │ +++ expected
    │ - {
    │ -   "root": [
    │ -     {
    │ -       "bar": [
    │ -         1
    │ -       ],
    │ -       "qux": "quux"
    │ -     },
    │ -     {
    │ -       "bar": [
    │ -         2
    │ -       ],
    │ -       "qux": "quux"
    │ -     }
    │ -   ]
    │ - }
    │ + "(sensitive value)"


expected to fail
`
	if diff := cmp.Diff(output.Stdout(), expectedOut); len(diff) > 0 {
		t.Errorf("\nexpected: \n%s\ngot: %s\ndiff: %s", expectedOut, output.All(), diff)
	}
	if diff := cmp.Diff(output.Stderr(), expectedErr); len(diff) > 0 {
		t.Errorf("\nexpected stderr: \n%s\ngot: %s\ndiff: %s", expectedErr, output.Stderr(), diff)
	}

	if provider.ResourceCount() > 0 {
		if !printedOutput {
			t.Errorf("should have deleted all resources on completion but left %s\n\n%s", provider.ResourceString(), output.All())
		} else {
			t.Errorf("should have deleted all resources on completion but left %s", provider.ResourceString())
		}
	}
}

func TestTest_ComplexConditionVerbose(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath(path.Join("test", "complex_condition")), td)
	t.Chdir(td)

	provider := testing_command.NewProvider(nil)

	providerSource, close := newMockProviderSource(t, map[string][]string{"test": {"1.0.0"}})
	defer close()

	streams, done := terminal.StreamsForTesting(t)
	view := views.NewView(streams)
	ui := new(cli.MockUi)

	meta := Meta{
		testingOverrides: metaOverridesForProvider(provider.Provider),
		Ui:               ui,
		View:             view,
		Streams:          streams,
		ProviderSource:   providerSource,
	}

	init := &InitCommand{
		Meta: meta,
	}

	output := done(t)

	if code := init.Run([]string{"-no-color"}); code != 0 {
		t.Fatalf("expected status code 0 but got %d: %s", code, output.All())
	}

	// Reset the streams for the next command.
	streams, done = terminal.StreamsForTesting(t)
	meta.Streams = streams
	meta.View = views.NewView(streams)

	command := &TestCommand{
		Meta: meta,
	}

	code := command.Run([]string{"-no-color", "-verbose"})
	output = done(t)

	printedOutput := false

	if code != 1 {
		printedOutput = true
		t.Errorf("expected status code 1 but got %d: %s", code, output.All())
	}

	expectedErr := `
Error: Test assertion failed

  on main.tftest.hcl line 37, in run "validate_diff_types":
  37:     condition = var.tr1 == var.tr2 
    ├────────────────
    │ LHS:
    │   {
    │     "iops": null,
    │     "size": 60
    │   }
    │ RHS:
    │   {
    │     "iops": null,
    │     "size": 60
    │   }
    │ Warning: LHS and RHS values are of different types

    │ var.tr1 is {
    │     "iops": null,
    │     "size": 60
    │   }
    │ var.tr2 is {
    │     "iops": null,
    │     "size": 60
    │   }

expected to fail

Error: Test assertion failed

  on main.tftest.hcl line 44, in run "validate_output":
  44:     condition = output.foo == var.foo
    ├────────────────
    │ LHS:
    │   {
    │     "bar": "notbaz",
    │     "matches": "matches",
    │     "qux": "quux",
    │     "xuq": "xuq"
    │   }
    │ RHS:
    │   {
    │     "bar": "baz",
    │     "matches": "matches",
    │     "qux": "qux",
    │     "xuq": "nope"
    │   }
    │ Diff:
    │ --- actual
    │ +++ expected
    │   {
    │ -   "bar": "notbaz",
    │ +   "bar": "baz",
    │     "matches": "matches",
    │ -   "qux": "quux",
    │ -   "xuq": "xuq"
    │ +   "qux": "qux",
    │ +   "xuq": "nope"
    │   }

    │ output.foo is {
    │     "bar": "notbaz",
    │     "matches": "matches",
    │     "qux": "quux",
    │     "xuq": "xuq"
    │   }
    │ var.foo is {
    │     "bar": "baz",
    │     "matches": "matches",
    │     "qux": "qux",
    │     "xuq": "nope"
    │   }

expected to fail due to different values

Error: Test assertion failed

  on main.tftest.hcl line 52, in run "validate_complex_output":
  52:     condition = output.complex == var.bar
    ├────────────────
    │ LHS:
    │   {
    │     "root": [
    │       {
    │         "bar": [
    │           1
    │         ],
    │         "qux": "quux"
    │       },
    │       {
    │         "bar": [
    │           2
    │         ],
    │         "qux": "quux"
    │       }
    │     ]
    │   }
    │ RHS:
    │   {
    │     "root": [
    │       {
    │         "bar": [
    │           1
    │         ],
    │         "qux": "qux"
    │       },
    │       {
    │         "bar": [
    │           2
    │         ],
    │         "qux": "quux"
    │       }
    │     ]
    │   }
    │ Warning: LHS and RHS values are of different types
    │ Diff:
    │ --- actual
    │ +++ expected
    │   {
    │     "root": [
    │       {
    │         "bar": [
    │           1
    │         ],
    │ -       "qux": "quux"
    │ +       "qux": "qux"
    │       },
    │       {
    │         "bar": [
    │           2
    │         ],
    │         "qux": "quux"
    │       }
    │     ]
    │   }

    │ output.complex is {
    │     "root": [
    │       {
    │         "bar": [
    │           1
    │         ],
    │         "qux": "quux"
    │       },
    │       {
    │         "bar": [
    │           2
    │         ],
    │         "qux": "quux"
    │       }
    │     ]
    │   }
    │ var.bar is {
    │     "root": [
    │       {
    │         "bar": [
    │           1
    │         ],
    │         "qux": "qux"
    │       },
    │       {
    │         "bar": [
    │           2
    │         ],
    │         "qux": "quux"
    │       }
    │     ]
    │   }

expected to fail

Error: Test assertion failed

  on main.tftest.hcl line 60, in run "validate_complex_output_sensitive":
  60:     condition = output.complex == output.complex_sensitive
    ├────────────────
    │ LHS:
    │   {
    │     "root": [
    │       {
    │         "bar": [
    │           1
    │         ],
    │         "qux": "quux"
    │       },
    │       {
    │         "bar": [
    │           2
    │         ],
    │         "qux": "quux"
    │       }
    │     ]
    │   }
    │ RHS:
    │   "(sensitive value)"
    │ Diff:
    │ --- actual
    │ +++ expected
    │ - {
    │ -   "root": [
    │ -     {
    │ -       "bar": [
    │ -         1
    │ -       ],
    │ -       "qux": "quux"
    │ -     },
    │ -     {
    │ -       "bar": [
    │ -         2
    │ -       ],
    │ -       "qux": "quux"
    │ -     }
    │ -   ]
    │ - }
    │ + "(sensitive value)"

    │ output.complex is {
    │     "root": [
    │       {
    │         "bar": [
    │           1
    │         ],
    │         "qux": "quux"
    │       },
    │       {
    │         "bar": [
    │           2
    │         ],
    │         "qux": "quux"
    │       }
    │     ]
    │   }
    │ output.complex_sensitive is "(sensitive value)"

expected to fail
`
	outputs := []string{
		"main.tftest.hcl... in progress",
		"  run \"validate_diff_types\"... fail",
		"  run \"validate_output\"... fail",
		"  run \"validate_complex_output\"... fail",
		"  run \"validate_complex_output_sensitive\"... fail",
		"  run \"validate_complex_output_pass\"... pass",
		"main.tftest.hcl... tearing down",
		"main.tftest.hcl... fail",
		"Failure! 1 passed, 4 failed.",
	}
	stdout := output.Stdout()
	for _, expected := range outputs {
		if !strings.Contains(stdout, expected) {
			t.Errorf("output didn't contain expected output %q", expected)
		}
	}

	if diff := cmp.Diff(output.Stderr(), expectedErr); len(diff) > 0 {
		t.Errorf("\nexpected stderr: \n%s\ngot: %s\ndiff: %s", expectedErr, output.Stderr(), diff)
	}

	if provider.ResourceCount() > 0 {
		if !printedOutput {
			t.Errorf("should have deleted all resources on completion but left %s\n\n%s", provider.ResourceString(), output.All())
		} else {
			t.Errorf("should have deleted all resources on completion but left %s", provider.ResourceString())
		}
	}
}

func TestTest_ModuleDependencies(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath(path.Join("test", "with_setup_module")), td)
	t.Chdir(td)

	// Our two providers will share a common set of values to make things
	// easier.
	store := &testing_command.ResourceStore{
		Data: make(map[string]cty.Value),
	}

	// We set it up so the module provider will update the data sources
	// available to the core mock provider.
	test := testing_command.NewProvider(store)
	setup := testing_command.NewProvider(store)

	test.SetDataPrefix("data")
	test.SetResourcePrefix("resource")

	// Let's make the setup provider write into the data for test provider.
	setup.SetResourcePrefix("data")

	providerSource, close := newMockProviderSource(t, map[string][]string{
		"test":  {"1.0.0"},
		"setup": {"1.0.0"},
	})
	defer close()

	streams, done := terminal.StreamsForTesting(t)
	view := views.NewView(streams)
	ui := new(cli.MockUi)

	meta := Meta{
		testingOverrides: &testingOverrides{
			Providers: map[addrs.Provider]providers.Factory{
				addrs.NewDefaultProvider("test"):  providers.FactoryFixed(test.Provider),
				addrs.NewDefaultProvider("setup"): providers.FactoryFixed(setup.Provider),
			},
		},
		Ui:             ui,
		View:           view,
		Streams:        streams,
		ProviderSource: providerSource,
	}

	init := &InitCommand{
		Meta: meta,
	}

	if code := init.Run(nil); code != 0 {
		output := done(t)
		t.Fatalf("expected status code 0 but got %d: %s", code, output.All())
	}

	// Reset the streams for the next command.
	streams, done = terminal.StreamsForTesting(t)
	meta.Streams = streams
	meta.View = views.NewView(streams)

	command := &TestCommand{
		Meta: meta,
	}

	code := command.Run(nil)
	output := done(t)

	printedOutput := false

	if code != 0 {
		printedOutput = true
		t.Errorf("expected status code 0 but got %d: %s", code, output.All())
	}

	if test.ResourceCount() > 0 {
		if !printedOutput {
			printedOutput = true
			t.Errorf("should have deleted all resources on completion but left %s\n\n%s", test.ResourceString(), output.All())
		} else {
			t.Errorf("should have deleted all resources on completion but left %s", test.ResourceString())
		}
	}

	if setup.ResourceCount() > 0 {
		if !printedOutput {
			t.Errorf("should have deleted all resources on completion but left %s\n\n%s", setup.ResourceString(), output.All())
		} else {
			t.Errorf("should have deleted all resources on completion but left %s", setup.ResourceString())
		}
	}
}

func TestTest_CatchesErrorsBeforeDestroy(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath(path.Join("test", "invalid_default_state")), td)
	t.Chdir(td)

	provider := testing_command.NewProvider(nil)
	view, done := testView(t)

	c := &TestCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(provider.Provider),
			View:             view,
		},
	}

	code := c.Run([]string{"-no-color"})
	output := done(t)

	if code != 1 {
		t.Errorf("expected status code 0 but got %d", code)
	}

	expectedOut := `main.tftest.hcl... in progress
  run "test"... fail
main.tftest.hcl... tearing down
main.tftest.hcl... fail

Failure! 0 passed, 1 failed.
`

	expectedErr := `
Error: No value for required variable

  on main.tf line 2:
   2: variable "input" {

The module under test for run block "test" has a required variable "input"
with no set value. Use a -var or -var-file command line argument or add this
variable into a "variables" block within the test file or run block.
`

	actualOut := output.Stdout()
	actualErr := output.Stderr()

	if diff := cmp.Diff(actualOut, expectedOut); len(diff) > 0 {
		t.Errorf("std out didn't match expected:\nexpected:\n%s\nactual:\n%s\ndiff:\n%s", expectedOut, actualOut, diff)
	}

	if diff := cmp.Diff(actualErr, expectedErr); len(diff) > 0 {
		t.Errorf("std err didn't match expected:\nexpected:\n%s\nactual:\n%s\ndiff:\n%s", expectedErr, actualErr, diff)
	}

	if provider.ResourceCount() > 0 {
		t.Errorf("should have deleted all resources on completion but left %v", provider.ResourceString())
	}
}

func TestTest_Verbose(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath(path.Join("test", "plan_then_apply")), td)
	t.Chdir(td)

	provider := testing_command.NewProvider(nil)
	view, done := testView(t)

	c := &TestCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(provider.Provider),
			View:             view,
		},
	}

	code := c.Run([]string{"-verbose", "-no-color"})
	output := done(t)

	if code != 0 {
		t.Errorf("expected status code 0 but got %d", code)
	}

	expected := `main.tftest.hcl... in progress
  run "validate_test_resource"... pass

Terraform used the selected providers to generate the following execution
plan. Resource actions are indicated with the following symbols:
  + create

Terraform will perform the following actions:

  # test_resource.foo will be created
  + resource "test_resource" "foo" {
      + destroy_fail = (known after apply)
      + id           = "constant_value"
      + value        = "bar"
      + write_only   = (write-only attribute)
    }

Plan: 1 to add, 0 to change, 0 to destroy.

  run "apply_test_resource"... pass

# test_resource.foo:
resource "test_resource" "foo" {
    destroy_fail = false
    id           = "constant_value"
    value        = "bar"
    write_only   = (write-only attribute)
}

main.tftest.hcl... tearing down
main.tftest.hcl... pass

Success! 2 passed, 0 failed.
`

	actual := output.All()

	if diff := cmp.Diff(actual, expected); len(diff) > 0 {
		t.Errorf("output didn't match expected:\nexpected:\n%s\nactual:\n%s\ndiff:\n%s", expected, actual, diff)
	}

	if provider.ResourceCount() > 0 {
		t.Errorf("should have deleted all resources on completion but left %v", provider.ResourceString())
	}
}

func TestTest_ValidatesBeforeExecution(t *testing.T) {
	tcs := map[string]struct {
		expectedOut string
		expectedErr string
	}{
		"invalid": {
			expectedOut: `main.tftest.hcl... in progress
  run "invalid"... fail
main.tftest.hcl... tearing down
main.tftest.hcl... fail

Failure! 0 passed, 1 failed.
`,
			expectedErr: `
Error: Invalid ` + "`expect_failures`" + ` reference

  on main.tftest.hcl line 5, in run "invalid":
   5:         local.my_value,

You cannot expect failures from local.my_value. You can only expect failures
from checkable objects such as input variables, output values, check blocks,
managed resources and data sources.
`,
		},
		"invalid-module": {
			expectedOut: `main.tftest.hcl... in progress
  run "invalid"... fail
  run "test"... skip
main.tftest.hcl... tearing down
main.tftest.hcl... fail

Failure! 0 passed, 1 failed, 1 skipped.
`,
			expectedErr: `
Error: Reference to undeclared input variable

  on setup/main.tf line 3, in resource "test_resource" "setup":
   3:     value = var.not_real // Oh no!

An input variable with the name "not_real" has not been declared. This
variable can be declared with a variable "not_real" {} block.
`,
		},
		"missing-provider": {
			expectedOut: `main.tftest.hcl... in progress
  run "passes_validation"... fail
main.tftest.hcl... tearing down
main.tftest.hcl... fail

Failure! 0 passed, 1 failed.
`,
			expectedErr: `
Error: Provider configuration not present

To work with test_resource.secondary its original provider configuration at
provider["registry.terraform.io/hashicorp/test"].secondary is required, but
it has been removed. This occurs when a provider configuration is removed
while objects created by that provider still exist in the state. Re-add the
provider configuration to destroy test_resource.secondary, after which you
can remove the provider configuration again.
`,
		},
		"missing-provider-in-run-block": {
			expectedOut: `main.tftest.hcl... in progress
  run "passes_validation"... fail
main.tftest.hcl... tearing down
main.tftest.hcl... fail

Failure! 0 passed, 1 failed.
`,
			expectedErr: `
Error: Provider configuration not present

To work with test_resource.secondary its original provider configuration at
provider["registry.terraform.io/hashicorp/test"].secondary is required, but
it has been removed. This occurs when a provider configuration is removed
while objects created by that provider still exist in the state. Re-add the
provider configuration to destroy test_resource.secondary, after which you
can remove the provider configuration again.
`,
		},
		"missing-provider-definition-in-file": {
			expectedOut: `main.tftest.hcl... in progress
  run "passes_validation"... fail
main.tftest.hcl... tearing down
main.tftest.hcl... fail

Failure! 0 passed, 1 failed.
`,
			expectedErr: `
Error: Missing provider definition for test

  on main.tftest.hcl line 12, in run "passes_validation":
  12:     test = test

This provider block references a provider definition that does not exist.
`,
		},
		"missing-provider-in-test-module": {
			expectedOut: `main.tftest.hcl... in progress
  run "passes_validation_primary"... pass
  run "passes_validation_secondary"... fail
main.tftest.hcl... tearing down
main.tftest.hcl... fail

Failure! 1 passed, 1 failed.
`,
			expectedErr: `
Error: Provider configuration not present

To work with test_resource.secondary its original provider configuration at
provider["registry.terraform.io/hashicorp/test"].secondary is required, but
it has been removed. This occurs when a provider configuration is removed
while objects created by that provider still exist in the state. Re-add the
provider configuration to destroy test_resource.secondary, after which you
can remove the provider configuration again.
`,
		},
	}

	for file, tc := range tcs {
		t.Run(file, func(t *testing.T) {

			td := t.TempDir()
			testCopyDir(t, testFixturePath(path.Join("test", file)), td)
			t.Chdir(td)

			provider := testing_command.NewProvider(nil)

			providerSource, close := newMockProviderSource(t, map[string][]string{
				"test": {"1.0.0"},
			})
			defer close()

			streams, done := terminal.StreamsForTesting(t)
			view := views.NewView(streams)
			ui := new(cli.MockUi)

			meta := Meta{
				testingOverrides: metaOverridesForProvider(provider.Provider),
				Ui:               ui,
				View:             view,
				Streams:          streams,
				ProviderSource:   providerSource,
			}

			init := &InitCommand{
				Meta: meta,
			}

			if code := init.Run(nil); code != 0 {
				output := done(t)
				t.Fatalf("expected status code 0 but got %d: %s", code, output.All())
			}

			// Reset the streams for the next command.
			streams, done = terminal.StreamsForTesting(t)
			meta.Streams = streams
			meta.View = views.NewView(streams)

			c := &TestCommand{
				Meta: meta,
			}

			code := c.Run([]string{"-no-color"})
			output := done(t)

			if code != 1 {
				t.Errorf("expected status code 1 but got %d", code)
			}

			actualOut, expectedOut := output.Stdout(), tc.expectedOut
			actualErr, expectedErr := output.Stderr(), tc.expectedErr

			if !strings.Contains(actualOut, expectedOut) {
				t.Errorf("output didn't match expected:\nexpected:\n%s\nactual:\n%s", expectedOut, actualOut)
			}

			if diff := cmp.Diff(actualErr, expectedErr); len(diff) > 0 {
				t.Errorf("error didn't match expected:\nexpected:\n%s\nactual:\n%s\ndiff:\n%s", expectedErr, actualErr, diff)
			}

			if provider.ResourceCount() > 0 {
				t.Errorf("should have deleted all resources on completion but left %v", provider.ResourceString())
			}
		})
	}
}

func TestTest_NestedSetupModules(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath(path.Join("test", "with_nested_setup_modules")), td)
	t.Chdir(td)

	provider := testing_command.NewProvider(nil)

	providerSource, close := newMockProviderSource(t, map[string][]string{
		"test": {"1.0.0"},
	})
	defer close()

	streams, done := terminal.StreamsForTesting(t)
	view := views.NewView(streams)
	ui := new(cli.MockUi)

	meta := Meta{
		testingOverrides: metaOverridesForProvider(provider.Provider),
		Ui:               ui,
		View:             view,
		Streams:          streams,
		ProviderSource:   providerSource,
	}

	init := &InitCommand{
		Meta: meta,
	}

	if code := init.Run(nil); code != 0 {
		t.Fatalf("expected status code 0 but got %d: %s", code, ui.ErrorWriter)
	}

	command := &TestCommand{
		Meta: meta,
	}

	code := command.Run(nil)
	output := done(t)

	printedOutput := false

	if code != 0 {
		printedOutput = true
		t.Errorf("expected status code 0 but got %d: %s", code, output.All())
	}

	if provider.ResourceCount() > 0 {
		if !printedOutput {
			t.Errorf("should have deleted all resources on completion but left %s\n\n%s", provider.ResourceString(), output.All())
		} else {
			t.Errorf("should have deleted all resources on completion but left %s", provider.ResourceString())
		}
	}
}

func TestTest_StatePropagation(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath(path.Join("test", "state_propagation")), td)
	t.Chdir(td)

	provider := testing_command.NewProvider(nil)

	providerSource, close := newMockProviderSource(t, map[string][]string{
		"test": {"1.0.0"},
	})
	defer close()

	streams, done := terminal.StreamsForTesting(t)
	view := views.NewView(streams)
	ui := new(cli.MockUi)

	meta := Meta{
		testingOverrides: metaOverridesForProvider(provider.Provider),
		Ui:               ui,
		View:             view,
		Streams:          streams,
		ProviderSource:   providerSource,
	}

	init := &InitCommand{
		Meta: meta,
	}

	if code := init.Run(nil); code != 0 {
		output := done(t)
		t.Fatalf("expected status code 0 but got %d: %s", code, output.All())
	}

	// Reset the streams for the next command.
	streams, done = terminal.StreamsForTesting(t)
	meta.Streams = streams
	meta.View = views.NewView(streams)

	c := &TestCommand{
		Meta: meta,
	}

	code := c.Run([]string{"-verbose", "-no-color"})
	output := done(t)

	if code != 0 {
		t.Errorf("expected status code 0 but got %d", code)
	}

	expected := `main.tftest.hcl... in progress
  run "initial_apply_example"... pass

# test_resource.module_resource:
resource "test_resource" "module_resource" {
    destroy_fail = false
    id           = "df6h8as9"
    value        = "start"
    write_only   = (write-only attribute)
}

  run "initial_apply"... pass

# test_resource.resource:
resource "test_resource" "resource" {
    destroy_fail = false
    id           = "598318e0"
    value        = "start"
    write_only   = (write-only attribute)
}

  run "plan_second_example"... pass

Terraform used the selected providers to generate the following execution
plan. Resource actions are indicated with the following symbols:
  + create

Terraform will perform the following actions:

  # test_resource.second_module_resource will be created
  + resource "test_resource" "second_module_resource" {
      + destroy_fail = (known after apply)
      + id           = "b6a1d8cb"
      + value        = "start"
      + write_only   = (write-only attribute)
    }

Plan: 1 to add, 0 to change, 0 to destroy.

  run "plan_update"... pass

Terraform used the selected providers to generate the following execution
plan. Resource actions are indicated with the following symbols:
  ~ update in-place

Terraform will perform the following actions:

  # test_resource.resource will be updated in-place
  ~ resource "test_resource" "resource" {
        id           = "598318e0"
      ~ value        = "start" -> "update"
        # (2 unchanged attributes hidden)
    }

Plan: 0 to add, 1 to change, 0 to destroy.

  run "plan_update_example"... pass

Terraform used the selected providers to generate the following execution
plan. Resource actions are indicated with the following symbols:
  ~ update in-place

Terraform will perform the following actions:

  # test_resource.module_resource will be updated in-place
  ~ resource "test_resource" "module_resource" {
        id           = "df6h8as9"
      ~ value        = "start" -> "update"
        # (2 unchanged attributes hidden)
    }

Plan: 0 to add, 1 to change, 0 to destroy.

main.tftest.hcl... tearing down
main.tftest.hcl... pass

Success! 5 passed, 0 failed.
`

	actual := output.All()

	if diff := cmp.Diff(expected, actual); diff != "" {
		t.Errorf("output didn't match expected:\nexpected:\n%s\nactual:\n%s\ndiff:\n%s", expected, actual, diff)
	}

	if provider.ResourceCount() > 0 {
		t.Errorf("should have deleted all resources on completion but left %v", provider.ResourceString())
	}
}

func TestTest_SkipCleanup(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath(path.Join("test", "skip_cleanup")), td)
	t.Chdir(td)

	provider := testing_command.NewProvider(nil)
	providerSource, close := newMockProviderSource(t, map[string][]string{
		"test": {"1.0.0"},
	})
	defer close()

	streams, done := terminal.StreamsForTesting(t)
	view := views.NewView(streams)
	ui := new(cli.MockUi)

	meta := Meta{
		testingOverrides:          metaOverridesForProvider(provider.Provider),
		Ui:                        ui,
		View:                      view,
		Streams:                   streams,
		ProviderSource:            providerSource,
		AllowExperimentalFeatures: true,
	}

	init := &InitCommand{
		Meta: meta,
	}

	output := done(t)

	if code := init.Run(nil); code != 0 {
		t.Fatalf("expected status code 0 but got %d: %s", code, output.All())
	}

	// Reset the streams for the next command.
	streams, done = terminal.StreamsForTesting(t)
	meta.Streams = streams
	meta.View = views.NewView(streams)

	c := &TestCommand{
		Meta: meta,
	}

	code := c.Run([]string{"-no-color"})
	output = done(t)

	if code != 0 {
		t.Errorf("expected status code 0 but got %d", code)
	}

	t.Run("skipped resources should not be deleted", func(t *testing.T) {

		expected := `
Warning: Duplicate "skip_cleanup" block

  on main.tftest.hcl line 15, in run "test_three":
  15:   skip_cleanup = true

The run "test_three" has a skip_cleanup attribute set, but shares state with
an earlier run "test_two" that also has skip_cleanup set. The later run takes
precedence, and this attribute is ignored for the earlier run.
main.tftest.hcl... in progress
  run "test"... pass
  run "test_two"... pass
  run "test_three"... pass
  run "test_four"... pass
  run "test_five"... pass
main.tftest.hcl... tearing down
main.tftest.hcl... pass

Success! 5 passed, 0 failed.
`

		actual := output.All()
		if !strings.Contains(actual, expected) {
			t.Errorf("output didn't match expected:\nexpected:\n%s\nactual:\n%s\ndiff:\n%s", expected, actual, cmp.Diff(expected, actual))
		}

		if provider.ResourceCount() != 1 {
			t.Errorf("should have deleted all resources on completion but left %v", provider.ResourceString())
		}

		val := provider.Store.Get(provider.ResourceString())

		if val.GetAttr("value").AsString() != "test_three" {
			t.Errorf("expected resource to have value 'test_three' but got %s", val.GetAttr("value").AsString())
		}
	})

	t.Run("state should be persisted", func(t *testing.T) {
		expectedStates := map[string][]string{
			"main.": {"test_resource.resource"},
		}
		actualStates := removeOutputs(statesFromManifest(t, td))

		if diff := cmp.Diff(expectedStates, actualStates); diff != "" {
			t.Fatalf("unexpected states: %s", diff)
		}
	})
}

func TestTest_SkipCleanupWithRunDependencies(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath(path.Join("test", "skip_cleanup_with_run_deps")), td)
	t.Chdir(td)

	provider := testing_command.NewProvider(nil)
	providerSource, close := newMockProviderSource(t, map[string][]string{
		"test": {"1.0.0"},
	})
	defer close()

	streams, done := terminal.StreamsForTesting(t)
	view := views.NewView(streams)
	ui := new(cli.MockUi)

	meta := Meta{
		testingOverrides:          metaOverridesForProvider(provider.Provider),
		Ui:                        ui,
		View:                      view,
		Streams:                   streams,
		ProviderSource:            providerSource,
		AllowExperimentalFeatures: true,
	}

	init := &InitCommand{
		Meta: meta,
	}

	output := done(t)

	if code := init.Run(nil); code != 0 {
		t.Fatalf("expected status code 0 but got %d: %s", code, output.All())
	}

	// Reset the streams for the next command.
	streams, done = terminal.StreamsForTesting(t)
	meta.Streams = streams
	meta.View = views.NewView(streams)

	c := &TestCommand{
		Meta: meta,
	}

	code := c.Run([]string{"-no-color"})
	output = done(t)

	if code != 0 {
		t.Errorf("expected status code 0 but got %d", code)
	}

	t.Run("skipped resources should not be deleted", func(t *testing.T) {

		expected := `main.tftest.hcl... in progress
  run "test"... pass
  run "test_two"... pass
  run "test_three"... pass
main.tftest.hcl... tearing down
main.tftest.hcl... pass

Success! 3 passed, 0 failed.
`

		actual := output.All()
		if !strings.Contains(actual, expected) {
			t.Errorf("output didn't match expected:\nexpected:\n%s\nactual:\n%s\ndiff:\n%s", expected, actual, cmp.Diff(expected, actual))
		}

		if provider.ResourceCount() != 1 {
			t.Errorf("should have deleted all resources on completion but left %v", provider.ResourceString())
		}

		val := provider.Store.Get(provider.ResourceString())

		if val.GetAttr("value").AsString() != "test_two" {
			t.Errorf("expected resource to have value 'test_two' but got %s", val.GetAttr("value").AsString())
		}
	})

	// we want to check that we leave behind the state that was skipped
	// and the states that it depends on
	t.Run("state should be persisted", func(t *testing.T) {
		expectedStates := map[string][]string{
			"main.":      {"output.id", "output.unused"},
			"main.state": {"test_resource.resource", "output.id", "output.unused"},
		}
		actualStates := statesFromManifest(t, td)

		if diff := cmp.Diff(expectedStates, actualStates, equalIgnoreOrder()); diff != "" {
			t.Fatalf("unexpected states: %s", diff)
		}
	})

	t.Run("cleanup all left-over state", func(t *testing.T) {
		view, done := testView(t)
		c := &TestCleanupCommand{
			Meta: Meta{
				testingOverrides: metaOverridesForProvider(provider.Provider),
				View:             view,

				Ui:                        ui,
				ProviderSource:            providerSource,
				AllowExperimentalFeatures: true,
			},
		}

		c.Run([]string{"-no-color"})
		output := done(t)

		expectedCleanup := `main.tftest.hcl... in progress
main.tftest.hcl... tearing down
main.tftest.hcl... pass

Success!
`
		if diff := cmp.Diff(expectedCleanup, output.Stdout()); diff != "" {
			t.Errorf("unexpected cleanup output: expected %s\n, got %s\n, diff: %s", expectedCleanup, output.Stdout(), diff)
		}
		if err := output.Stderr(); len(err) != 0 {
			t.Errorf("unexpected error: %s", err)
		}

		expectedStates := map[string][]string{}
		actualStates := removeOutputs(statesFromManifest(t, td))

		if diff := cmp.Diff(expectedStates, actualStates); diff != "" {
			t.Fatalf("unexpected states after cleanup: %s", diff)
		}
	})
}

func TestTest_SkipCleanup_JSON(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath(path.Join("test", "skip_cleanup")), td)
	t.Chdir(td)

	provider := testing_command.NewProvider(nil)
	providerSource, close := newMockProviderSource(t, map[string][]string{
		"test": {"1.0.0"},
	})
	defer close()

	streams, done := terminal.StreamsForTesting(t)
	view := views.NewView(streams)
	ui := new(cli.MockUi)

	meta := Meta{
		testingOverrides:          metaOverridesForProvider(provider.Provider),
		Ui:                        ui,
		View:                      view,
		Streams:                   streams,
		ProviderSource:            providerSource,
		AllowExperimentalFeatures: true,
	}

	init := &InitCommand{
		Meta: meta,
	}

	output := done(t)

	if code := init.Run(nil); code != 0 {
		t.Fatalf("expected status code 0 but got %d: %s", code, output.All())
	}

	// Reset the streams for the next command.
	streams, done = terminal.StreamsForTesting(t)
	meta.Streams = streams
	meta.View = views.NewView(streams)

	c := &TestCommand{
		Meta: meta,
	}

	code := c.Run([]string{"-no-color", "-json"})
	output = done(t)

	if code != 0 {
		t.Errorf("expected status code 0 but got %d", code)
	}

	var messages []string
	for ix, line := range strings.Split(output.All(), "\n") {
		if len(line) == 0 {
			// Skip empty lines.
			continue
		}

		if ix == 0 {
			// skip the first one, it's version information
			continue
		}

		var obj map[string]interface{}

		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			t.Errorf("failed to unmarshal returned line: %s", line)
			continue
		}

		// Remove the timestamp as it changes every time.
		delete(obj, "@timestamp")

		if obj["type"].(string) == "test_run" {
			// Then we need to delete the `elapsed` field from within the run
			// as it'll cause flaky tests.

			run := obj["test_run"].(map[string]interface{})
			if run["progress"].(string) != "complete" {
				delete(run, "elapsed")
			}
		}

		message, err := json.Marshal(obj)
		if err != nil {
			t.Errorf("failed to remarshal returned line: %s", line)
			continue
		}

		messages = append(messages, string(message))
	}

	t.Run("skipped resources should not be deleted", func(t *testing.T) {

		expected := []string{
			`{"@level":"warn","@message":"Warning: Duplicate \"skip_cleanup\" block","@module":"terraform.ui","diagnostic":{"detail":"The run \"test_three\" has a skip_cleanup attribute set, but shares state with an earlier run \"test_two\" that also has skip_cleanup set. The later run takes precedence, and this attribute is ignored for the earlier run.","range":{"end":{"byte":163,"column":15,"line":15},"filename":"main.tftest.hcl","start":{"byte":151,"column":3,"line":15}},"severity":"warning","snippet":{"code":"  skip_cleanup = true","context":"run \"test_three\"","highlight_end_offset":14,"highlight_start_offset":2,"start_line":15,"values":[]},"summary":"Duplicate \"skip_cleanup\" block"},"type":"diagnostic"}`,
			`{"@level":"info","@message":"Found 1 file and 5 run blocks","@module":"terraform.ui","test_abstract":{"main.tftest.hcl":["test","test_two","test_three","test_four","test_five"]},"type":"test_abstract"}`,
			`{"@level":"info","@message":"main.tftest.hcl... in progress","@module":"terraform.ui","@testfile":"main.tftest.hcl","test_file":{"path":"main.tftest.hcl","progress":"starting"},"type":"test_file"}`,
			`{"@level":"info","@message":"  \"test\"... in progress","@module":"terraform.ui","@testfile":"main.tftest.hcl","@testrun":"test","test_run":{"path":"main.tftest.hcl","progress":"starting","run":"test"},"type":"test_run"}`,
			`{"@level":"info","@message":"  \"test\"... pass","@module":"terraform.ui","@testfile":"main.tftest.hcl","@testrun":"test","test_run":{"path":"main.tftest.hcl","progress":"complete","run":"test","status":"pass"},"type":"test_run"}`,
			`{"@level":"info","@message":"  \"test_two\"... in progress","@module":"terraform.ui","@testfile":"main.tftest.hcl","@testrun":"test_two","test_run":{"path":"main.tftest.hcl","progress":"starting","run":"test_two"},"type":"test_run"}`,
			`{"@level":"info","@message":"  \"test_two\"... pass","@module":"terraform.ui","@testfile":"main.tftest.hcl","@testrun":"test_two","test_run":{"path":"main.tftest.hcl","progress":"complete","run":"test_two","status":"pass"},"type":"test_run"}`,
			`{"@level":"info","@message":"  \"test_three\"... in progress","@module":"terraform.ui","@testfile":"main.tftest.hcl","@testrun":"test_three","test_run":{"path":"main.tftest.hcl","progress":"starting","run":"test_three"},"type":"test_run"}`,
			`{"@level":"info","@message":"  \"test_three\"... pass","@module":"terraform.ui","@testfile":"main.tftest.hcl","@testrun":"test_three","test_run":{"path":"main.tftest.hcl","progress":"complete","run":"test_three","status":"pass"},"type":"test_run"}`,
			`{"@level":"info","@message":"  \"test_four\"... in progress","@module":"terraform.ui","@testfile":"main.tftest.hcl","@testrun":"test_four","test_run":{"path":"main.tftest.hcl","progress":"starting","run":"test_four"},"type":"test_run"}`,
			`{"@level":"info","@message":"  \"test_four\"... pass","@module":"terraform.ui","@testfile":"main.tftest.hcl","@testrun":"test_four","test_run":{"path":"main.tftest.hcl","progress":"complete","run":"test_four","status":"pass"},"type":"test_run"}`,
			`{"@level":"info","@message":"  \"test_five\"... in progress","@module":"terraform.ui","@testfile":"main.tftest.hcl","@testrun":"test_five","test_run":{"path":"main.tftest.hcl","progress":"starting","run":"test_five"},"type":"test_run"}`,
			`{"@level":"info","@message":"  \"test_five\"... pass","@module":"terraform.ui","@testfile":"main.tftest.hcl","@testrun":"test_five","test_run":{"path":"main.tftest.hcl","progress":"complete","run":"test_five","status":"pass"},"type":"test_run"}`,
			`{"@level":"info","@message":"main.tftest.hcl... tearing down","@module":"terraform.ui","@testfile":"main.tftest.hcl","test_file":{"path":"main.tftest.hcl","progress":"teardown"},"type":"test_file"}`,
			`{"@level":"info","@message":"  \"test_three\"... tearing down","@module":"terraform.ui","@testfile":"main.tftest.hcl","@testrun":"test_three","test_run":{"path":"main.tftest.hcl","progress":"teardown","run":"test_three"},"type":"test_run"}`,
			`{"@level":"info","@message":"main.tftest.hcl... pass","@module":"terraform.ui","@testfile":"main.tftest.hcl","test_file":{"path":"main.tftest.hcl","progress":"complete","status":"pass"},"type":"test_file"}`,
			`{"@level":"info","@message":"Success! 5 passed, 0 failed.","@module":"terraform.ui","test_summary":{"errored":0,"failed":0,"passed":5,"skipped":0,"status":"pass"},"type":"test_summary"}`,
		}

		trimmedActual := strings.Join(messages, "\n")
		trimmedExpected := strings.Join(expected, "\n")
		if diff := cmp.Diff(trimmedExpected, trimmedActual); diff != "" {
			t.Errorf("output didn't match expected:\nexpected:\n%s\nactual:\n%s\ndiff:\n%s", trimmedExpected, trimmedActual, diff)
		}

		if provider.ResourceCount() != 1 {
			t.Errorf("should have deleted all resources on completion but left %v", provider.ResourceString())
		}

		val := provider.Store.Get(provider.ResourceString())

		if val.GetAttr("value").AsString() != "test_three" {
			t.Errorf("expected resource to have value 'test_three' but got %s", val.GetAttr("value").AsString())
		}
	})
}

func TestTest_SkipCleanup_FileLevelFlag(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath(path.Join("test", "skip_file_cleanup")), td)
	t.Chdir(td)

	provider := testing_command.NewProvider(nil)
	providerSource, close := newMockProviderSource(t, map[string][]string{
		"test": {"1.0.0"},
	})
	defer close()

	streams, done := terminal.StreamsForTesting(t)
	view := views.NewView(streams)
	ui := new(cli.MockUi)

	meta := Meta{
		testingOverrides:          metaOverridesForProvider(provider.Provider),
		Ui:                        ui,
		View:                      view,
		Streams:                   streams,
		ProviderSource:            providerSource,
		AllowExperimentalFeatures: true,
	}

	init := &InitCommand{
		Meta: meta,
	}

	output := done(t)

	if code := init.Run(nil); code != 0 {
		t.Fatalf("expected status code 0 but got %d: %s", code, output.All())
	}

	// Reset the streams for the next command.
	streams, done = terminal.StreamsForTesting(t)
	meta.Streams = streams
	meta.View = views.NewView(streams)

	c := &TestCommand{
		Meta: meta,
	}

	code := c.Run([]string{"-no-color"})
	output = done(t)

	if code != 0 {
		t.Errorf("expected status code 0 but got %d", code)
	}

	t.Run("skipped resources should not be deleted", func(t *testing.T) {

		expected := `main.tftest.hcl... in progress
  run "test"... pass
  run "test_two"... pass
  run "test_three"... pass
  run "test_four"... pass
  run "test_five"... pass
main.tftest.hcl... tearing down
main.tftest.hcl... pass

Success! 5 passed, 0 failed.
`

		actual := output.All()
		if !strings.Contains(actual, expected) {
			t.Errorf("output didn't match expected:\nexpected:\n%s\nactual:\n%s", expected, actual)
		}

		if provider.ResourceCount() != 1 {
			t.Errorf("should have deleted all resources on completion but left %v", provider.ResourceString())
		}

		val := provider.Store.Get(provider.ResourceString())

		if val.GetAttr("value").AsString() != "test_four" {
			t.Errorf("expected resource to have value 'test_four' but got %s", val.GetAttr("value").AsString())
		}
	})

	t.Run("state should be persisted with valid reason", func(t *testing.T) {
		manifest, err := teststates.LoadManifest(td, true)
		if err != nil {
			t.Fatal(err)
		}

		expectedStates := map[string][]string{
			"main.": {"test_resource.resource"},
		}
		actualStates := make(map[string][]string)

		var reason teststates.StateReason
		// Verify the states in the manifest
		for fileName, file := range manifest.Files {
			for name, state := range file.States {
				sm := statemgr.NewFilesystem(manifest.StateFilePath(state.ID))
				if err := sm.RefreshState(); err != nil {
					t.Fatalf("error when reading state file: %s", err)
				}
				reason = state.Reason
				state := sm.State()

				// If the state is nil, then the test cleaned up the state
				if state == nil {
					t.Fatalf("state is nil")
				}

				var resources []string
				for _, module := range state.Modules {
					for _, resource := range module.Resources {
						resources = append(resources, resource.Addr.String())
					}
				}
				sort.Strings(resources)
				actualStates[strings.TrimSuffix(fileName, ".tftest.hcl")+"."+name] = resources
			}
		}

		if diff := cmp.Diff(expectedStates, actualStates); diff != "" {
			t.Fatalf("unexpected states: %s", diff)
		}
		if reason != teststates.StateReasonSkip {
			t.Fatalf("expected reason to be skip but got %s", reason)
		}
	})
}

func TestTest_OnlyExternalModules(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath(path.Join("test", "only_modules")), td)
	t.Chdir(td)

	provider := testing_command.NewProvider(nil)

	providerSource, close := newMockProviderSource(t, map[string][]string{
		"test": {"1.0.0"},
	})
	defer close()

	streams, done := terminal.StreamsForTesting(t)
	view := views.NewView(streams)
	ui := new(cli.MockUi)

	meta := Meta{
		testingOverrides: metaOverridesForProvider(provider.Provider),
		Ui:               ui,
		View:             view,
		Streams:          streams,
		ProviderSource:   providerSource,
	}

	init := &InitCommand{
		Meta: meta,
	}

	if code := init.Run(nil); code != 0 {
		output := done(t)
		t.Fatalf("expected status code 0 but got %d: %s", code, output.All())
	}

	// Reset the streams for the next command.
	streams, done = terminal.StreamsForTesting(t)
	meta.Streams = streams
	meta.View = views.NewView(streams)

	c := &TestCommand{
		Meta: meta,
	}

	code := c.Run([]string{"-no-color"})
	output := done(t)

	if code != 0 {
		t.Errorf("expected status code 0 but got %d", code)
	}

	expected := `main.tftest.hcl... in progress
  run "first"... pass
  run "second"... pass
main.tftest.hcl... tearing down
main.tftest.hcl... pass

Success! 2 passed, 0 failed.
`

	actual := output.Stdout()

	if !strings.Contains(actual, expected) {
		t.Errorf("output didn't match expected:\nexpected:\n%s\nactual:\n%s", expected, actual)
	}

	if provider.ResourceCount() > 0 {
		t.Errorf("should have deleted all resources on completion but left %v", provider.ResourceString())
	}
}

func TestTest_PartialUpdates(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath(path.Join("test", "partial_updates")), td)
	t.Chdir(td)

	provider := testing_command.NewProvider(nil)
	view, done := testView(t)

	c := &TestCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(provider.Provider),
			View:             view,
		},
	}

	code := c.Run([]string{"-no-color"})
	output := done(t)

	if code != 0 {
		t.Errorf("expected status code 0 but got %d", code)
	}

	expected := `main.tftest.hcl... in progress
  run "first"... pass

Warning: Resource targeting is in effect

You are creating a plan with the -target option, which means that the result
of this plan may not represent all of the changes requested by the current
configuration.

The -target option is not for routine use, and is provided only for
exceptional situations such as recovering from errors or mistakes, or when
Terraform specifically suggests to use it as part of an error message.

Warning: Applied changes may be incomplete

The plan was created with the -target option in effect, so some changes
requested in the configuration may have been ignored and the output values
may not be fully updated. Run the following command to verify that no other
changes are pending:
    terraform plan

Note that the -target option is not suitable for routine use, and is provided
only for exceptional situations such as recovering from errors or mistakes,
or when Terraform specifically suggests to use it as part of an error
message.

  run "second"... pass
main.tftest.hcl... tearing down
main.tftest.hcl... pass

Success! 2 passed, 0 failed.
`

	actual := output.All()

	if diff := cmp.Diff(actual, expected); len(diff) > 0 {
		t.Errorf("output didn't match expected:\nexpected:\n%s\nactual:\n%s\ndiff:\n%s", expected, actual, diff)
	}

	if provider.ResourceCount() > 0 {
		t.Errorf("should have deleted all resources on completion but left %v", provider.ResourceString())
	}
}

// There should not be warnings in clean-up
func TestTest_InvalidWarningsInCleanup(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath(path.Join("test", "invalid-cleanup-warnings")), td)
	t.Chdir(td)

	provider := testing_command.NewProvider(nil)
	providerSource, close := newMockProviderSource(t, map[string][]string{
		"test": {"1.0.0"},
	})
	defer close()

	streams, done := terminal.StreamsForTesting(t)
	view := views.NewView(streams)
	ui := new(cli.MockUi)

	meta := Meta{
		testingOverrides: metaOverridesForProvider(provider.Provider),
		Ui:               ui,
		View:             view,
		Streams:          streams,
		ProviderSource:   providerSource,
	}

	init := &InitCommand{
		Meta: meta,
	}

	if code := init.Run(nil); code != 0 {
		output := done(t)
		t.Fatalf("expected status code 0 but got %d: %s", code, output.All())
	}

	// Reset the streams for the next command.
	streams, done = terminal.StreamsForTesting(t)
	meta.Streams = streams
	meta.View = views.NewView(streams)

	c := &TestCommand{
		Meta: meta,
	}

	code := c.Run([]string{"-no-color"})
	output := done(t)

	if code != 0 {
		t.Errorf("expected status code 0 but got %d", code)
	}

	expected := `main.tftest.hcl... in progress
  run "test"... pass

Warning: Value for undeclared variable

  on main.tftest.hcl line 6, in run "test":
   6:     validation = "Hello, world!"

The module under test does not declare a variable named "validation", but it
is declared in run block "test".

main.tftest.hcl... tearing down
main.tftest.hcl... pass

Success! 1 passed, 0 failed.
`

	actual := output.All()

	if diff := cmp.Diff(actual, expected); len(diff) > 0 {
		t.Errorf("output didn't match expected:\nexpected:\n%s\nactual:\n%s\ndiff:\n%s", expected, actual, diff)
	}

	if provider.ResourceCount() > 0 {
		t.Errorf("should have deleted all resources on completion but left %v", provider.ResourceString())
	}
}

func TestTest_BadReferences(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath(path.Join("test", "bad-references")), td)
	t.Chdir(td)

	provider := testing_command.NewProvider(nil)
	view, done := testView(t)

	c := &TestCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(provider.Provider),
			View:             view,
		},
	}

	code := c.Run([]string{"-no-color"})
	output := done(t)

	if code == 0 {
		t.Errorf("expected status code 0 but got %d", code)
	}

	expectedOut := `main.tftest.hcl... in progress
  run "setup"... pass
  run "test"... fail
  run "finalise"... skip
main.tftest.hcl... tearing down
main.tftest.hcl... fail
providers.tftest.hcl... in progress
  run "test"... skip
providers.tftest.hcl... tearing down
providers.tftest.hcl... fail

Failure! 1 passed, 1 failed, 2 skipped.
`
	actualOut := output.Stdout()
	if diff := cmp.Diff(actualOut, expectedOut); len(diff) > 0 {
		t.Errorf("output didn't match expected:\nexpected:\n%s\nactual:\n%s\ndiff:\n%s", expectedOut, actualOut, diff)
	}

	expectedErr := `
Error: Reference to unavailable variable

  on main.tftest.hcl line 15, in run "test":
  15:     input_one = var.notreal

The input variable "notreal" does not exist within this test file.

Error: Reference to unknown run block

  on main.tftest.hcl line 16, in run "test":
  16:     input_two = run.madeup.response

The run block "madeup" does not exist within this test file.

Error: Reference to unavailable variable

  on providers.tftest.hcl line 3, in provider "test":
   3:   resource_prefix = var.default

The input variable "default" does not exist within this test file.
`
	actualErr := output.Stderr()
	if diff := cmp.Diff(actualErr, expectedErr); len(diff) > 0 {
		t.Errorf("output didn't match expected:\nexpected:\n%s\nactual:\n%s\ndiff:\n%s", expectedErr, actualErr, diff)
	}

	if provider.ResourceCount() > 0 {
		t.Errorf("should have deleted all resources on completion but left %v", provider.ResourceString())
	}
}

func TestTest_UndefinedVariables(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath(path.Join("test", "variables_undefined_in_config")), td)
	t.Chdir(td)

	provider := testing_command.NewProvider(nil)
	view, done := testView(t)

	c := &TestCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(provider.Provider),
			View:             view,
		},
	}

	code := c.Run([]string{"-no-color"})
	output := done(t)

	if code == 0 {
		t.Errorf("expected status code 0 but got %d", code)
	}

	expectedOut := `main.tftest.hcl... in progress
  run "test"... fail
main.tftest.hcl... tearing down
main.tftest.hcl... fail

Failure! 0 passed, 1 failed.
`
	actualOut := output.Stdout()
	if diff := cmp.Diff(actualOut, expectedOut); len(diff) > 0 {
		t.Errorf("output didn't match expected:\nexpected:\n%s\nactual:\n%s\ndiff:\n%s", expectedOut, actualOut, diff)
	}

	expectedErr := `
Error: Reference to undeclared input variable

  on main.tf line 2, in resource "test_resource" "foo":
   2:   value = var.input

An input variable with the name "input" has not been declared. This variable
can be declared with a variable "input" {} block.
`
	actualErr := output.Stderr()
	if diff := cmp.Diff(actualErr, expectedErr); len(diff) > 0 {
		t.Errorf("output didn't match expected:\nexpected:\n%s\nactual:\n%s\ndiff:\n%s", expectedErr, actualErr, diff)
	}

	if provider.ResourceCount() > 0 {
		t.Errorf("should have deleted all resources on completion but left %v", provider.ResourceString())
	}
}

func TestTest_VariablesInProviders(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath(path.Join("test", "provider_vars")), td)
	t.Chdir(td)

	provider := testing_command.NewProvider(nil)
	view, done := testView(t)

	c := &TestCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(provider.Provider),
			View:             view,
		},
	}

	code := c.Run([]string{"-no-color"})
	output := done(t)

	if code != 0 {
		t.Errorf("expected status code 0 but got %d", code)
	}

	expected := `main.tftest.hcl... in progress
  run "test"... pass
main.tftest.hcl... tearing down
main.tftest.hcl... pass

Success! 1 passed, 0 failed.
`
	actual := output.All()
	if diff := cmp.Diff(actual, expected); len(diff) > 0 {
		t.Errorf("output didn't match expected:\nexpected:\n%s\nactual:\n%s\ndiff:\n%s", expected, actual, diff)
	}

	if provider.ResourceCount() > 0 {
		t.Errorf("should have deleted all resources on completion but left %v", provider.ResourceString())
	}
}

func TestTest_ExpectedFailuresDuringPlanning(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath(path.Join("test", "expected_failures_during_planning")), td)
	t.Chdir(td)

	provider := testing_command.NewProvider(nil)
	view, done := testView(t)

	c := &TestCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(provider.Provider),
			View:             view,
		},
	}

	code := c.Run([]string{"-no-color"})
	output := done(t)

	if code == 0 {
		t.Errorf("expected status code 0 but got %d", code)
	}

	expectedOut := `check.tftest.hcl... in progress
  run "check_passes"... pass
check.tftest.hcl... tearing down
check.tftest.hcl... pass
input.tftest.hcl... in progress
  run "input_failure"... fail

Warning: Expected failure while planning

A custom condition within var.input failed during the planning stage and
prevented the requested apply operation. While this was an expected failure,
the apply operation could not be executed and so the overall test case will
be marked as a failure and the original diagnostic included in the test
report.

  run "no_run"... skip
input.tftest.hcl... tearing down
input.tftest.hcl... fail
output.tftest.hcl... in progress
  run "output_failure"... fail

Warning: Expected failure while planning

  on output.tftest.hcl line 13, in run "output_failure":
  13:     output.output,

A custom condition within output.output failed during the planning stage and
prevented the requested apply operation. While this was an expected failure,
the apply operation could not be executed and so the overall test case will
be marked as a failure and the original diagnostic included in the test
report.

output.tftest.hcl... tearing down
output.tftest.hcl... fail
resource.tftest.hcl... in progress
  run "resource_failure"... fail

Warning: Expected failure while planning

A custom condition within test_resource.resource failed during the planning
stage and prevented the requested apply operation. While this was an expected
failure, the apply operation could not be executed and so the overall test
case will be marked as a failure and the original diagnostic included in the
test report.

resource.tftest.hcl... tearing down
resource.tftest.hcl... fail

Failure! 1 passed, 3 failed, 1 skipped.
`
	actualOut := output.Stdout()
	if diff := cmp.Diff(expectedOut, actualOut); len(diff) > 0 {
		t.Errorf("output didn't match expected:\nexpected:\n%s\nactual:\n%s\ndiff:\n%s", expectedOut, actualOut, diff)
	}

	expectedErr := `
Error: Invalid value for variable

  on input.tftest.hcl line 5, in run "input_failure":
   5:     input = "bcd"
    ├────────────────
    │ var.input is "bcd"

input must contain the character 'a'

This was checked by the validation rule at main.tf:5,3-13.

Error: Module output value precondition failed

  on main.tf line 33, in output "output":
  33:     condition = strcontains(test_resource.resource.value, "d")
    ├────────────────
    │ test_resource.resource.value is "abc"

input must contain the character 'd'

Error: Resource postcondition failed

  on main.tf line 16, in resource "test_resource" "resource":
  16:       condition = strcontains(self.value, "b")
    ├────────────────
    │ self.value is "acd"

input must contain the character 'b'
`
	actualErr := output.Stderr()
	if diff := cmp.Diff(actualErr, expectedErr); len(diff) > 0 {
		t.Errorf("output didn't match expected:\nexpected:\n%s\nactual:\n%s\ndiff:\n%s", expectedErr, actualErr, diff)
	}

	if provider.ResourceCount() > 0 {
		t.Errorf("should have deleted all resources on completion but left %v", provider.ResourceString())
	}
}

func TestTest_MissingExpectedFailuresDuringApply(t *testing.T) {
	// Test asserting that the test run fails, but not errors out, when expected failures are not present during apply.
	// This lets subsequent runs continue to execute and the file to be marked as failed.
	td := t.TempDir()
	testCopyDir(t, testFixturePath(path.Join("test", "expect_failures_during_apply")), td)
	t.Chdir(td)

	provider := testing_command.NewProvider(nil)
	view, done := testView(t)

	c := &TestCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(provider.Provider),
			View:             view,
		},
	}

	code := c.Run([]string{"-no-color"})
	output := done(t)

	if code == 0 {
		t.Errorf("expected status code 0 but got %d", code)
	}

	expectedOut := `main.tftest.hcl... in progress
  run "test"... fail
  run "follow-up"... pass

Warning: Value for undeclared variable

  on main.tftest.hcl line 16, in run "follow-up":
  16:     input = "does not matter"

The module under test does not declare a variable named "input", but it is
declared in run block "follow-up".

main.tftest.hcl... tearing down
main.tftest.hcl... fail

Failure! 1 passed, 1 failed.
`
	actualOut := output.Stdout()
	if diff := cmp.Diff(expectedOut, actualOut); len(diff) > 0 {
		t.Errorf("output didn't match expected:\nexpected:\n%s\nactual:\n%s\ndiff:\n%s", expectedOut, actualOut, diff)
	}

	expectedErr := `
Error: Missing expected failure

  on main.tftest.hcl line 7, in run "test":
   7:     output.output

The checkable object, output.output, was expected to report an error but did
not.
`
	actualErr := output.Stderr()
	if diff := cmp.Diff(actualErr, expectedErr); len(diff) > 0 {
		t.Errorf("output didn't match expected:\nexpected:\n%s\nactual:\n%s\ndiff:\n%s", expectedErr, actualErr, diff)
	}

	if provider.ResourceCount() > 0 {
		t.Errorf("should have deleted all resources on completion but left %v", provider.ResourceString())
	}
}

func TestTest_UnknownAndNulls(t *testing.T) {

	tcs := map[string]struct {
		code   int
		stdout string
		stderr string
	}{
		"null_value_in_assert": {
			code: 1,
			stdout: `main.tftest.hcl... in progress
  run "first"... fail
main.tftest.hcl... tearing down
main.tftest.hcl... fail

Failure! 0 passed, 1 failed.
`,
			stderr: `
Error: Test assertion failed

  on main.tftest.hcl line 8, in run "first":
   8:     condition     = test_resource.resource.value == output.null_output
    ├────────────────
    │ Diff:
    │ --- actual
    │ +++ expected
    │ - "bar"
    │ + null


this is always going to fail
`,
		},
		"null_value_in_vars": {
			code: 1,
			stdout: `fail.tftest.hcl... in progress
  run "first"... pass
  run "second"... fail
fail.tftest.hcl... tearing down
fail.tftest.hcl... fail
pass.tftest.hcl... in progress
  run "first"... pass
  run "second"... pass
pass.tftest.hcl... tearing down
pass.tftest.hcl... pass

Failure! 3 passed, 1 failed.
`,
			stderr: `
Error: Required variable not set

  on fail.tftest.hcl line 11, in run "second":
  11:     interesting_input = run.first.null_output

The given value is not suitable for var.interesting_input defined at
main.tf:7,1-29: required variable may not be set to null.
`,
		},
		"unknown_value_in_assert": {
			code: 1,
			stdout: `main.tftest.hcl... in progress
  run "one"... pass
  run "two"... fail
main.tftest.hcl... tearing down
main.tftest.hcl... fail

Failure! 1 passed, 1 failed.
`,
			stderr: fmt.Sprintf(`
Error: Unknown condition value

  on main.tftest.hcl line 8, in run "two":
   8:     condition = output.destroy_fail == run.one.destroy_fail
    ├────────────────
    │ output.destroy_fail is false
    │ run.one.destroy_fail is a bool

Condition expression could not be evaluated at this time. This means you have
executed a %s block with %s and one of the values your
condition depended on is not known until after the plan has been applied.
Either remove this value from your condition, or execute an %s command
from this %s block. Alternatively, if there is an override for this value,
you can make it available during the plan phase by setting %s in the %s block.
`, "`run`", "`command = plan`", "`apply`", "`run`", "`override_during =\nplan`", "`override_`"),
		},
		"unknown_value_in_vars": {
			code: 1,
			stdout: `main.tftest.hcl... in progress
  run "one"... pass
  run "two"... fail
main.tftest.hcl... tearing down
main.tftest.hcl... fail

Failure! 1 passed, 1 failed.
`,
			stderr: `
Error: Reference to unknown value

  on main.tftest.hcl line 8, in run "two":
   8:     destroy_fail = run.one.destroy_fail

The value for run.one.destroy_fail is unknown. Run block "one" is executing a
"plan" operation, and the specified output value is only known after apply.
`,
		},
		"nested_unknown_values": {
			code: 1,
			stdout: `main.tftest.hcl... in progress
  run "first"... pass
  run "second"... pass
  run "third"... fail
main.tftest.hcl... tearing down
main.tftest.hcl... fail

Failure! 2 passed, 1 failed.
`,
			stderr: `
Error: Reference to unknown value

  on main.tftest.hcl line 31, in run "third":
  31:     input = run.second

The value for run.second is unknown. Run block "second" is executing a "plan"
operation, and the specified output value is only known after apply.
`,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			td := t.TempDir()
			testCopyDir(t, testFixturePath(path.Join("test", name)), td)
			t.Chdir(td)

			provider := testing_command.NewProvider(nil)
			view, done := testView(t)

			c := &TestCommand{
				Meta: Meta{
					testingOverrides: metaOverridesForProvider(provider.Provider),
					View:             view,
				},
			}

			code := c.Run([]string{"-no-color"})
			output := done(t)

			if code != tc.code {
				t.Errorf("expected return code %d but got %d", tc.code, code)
			}

			expectedOut := tc.stdout
			actualOut := output.Stdout()
			if diff := cmp.Diff(expectedOut, actualOut); len(diff) > 0 {
				t.Errorf("unexpected output\n\nexpected:\n%s\nactual:\n%s\ndiff:\n%s", expectedOut, actualOut, diff)
			}

			expectedErr := tc.stderr
			actualErr := output.Stderr()
			if diff := cmp.Diff(expectedErr, actualErr); len(diff) > 0 {
				t.Errorf("unexpected output\n\nexpected:\n%s\nactual:\n%s\ndiff:\n%s", expectedErr, actualErr, diff)
			}
		})
	}

}

func TestTest_SensitiveInputValues(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath(path.Join("test", "sensitive_input_values")), td)
	t.Chdir(td)

	provider := testing_command.NewProvider(nil)

	providerSource, close := newMockProviderSource(t, map[string][]string{
		"test": {"1.0.0"},
	})
	defer close()

	streams, done := terminal.StreamsForTesting(t)
	view := views.NewView(streams)

	meta := Meta{
		testingOverrides: metaOverridesForProvider(provider.Provider),
		View:             view,
		Streams:          streams,
		ProviderSource:   providerSource,
	}

	init := &InitCommand{
		Meta: meta,
	}

	if code := init.Run(nil); code != 0 {
		output := done(t)
		t.Fatalf("expected status code 0 but got %d: %s", code, output.All())
	}

	// Reset the streams for the next command.
	streams, done = terminal.StreamsForTesting(t)
	meta.Streams = streams
	meta.View = views.NewView(streams)

	c := &TestCommand{
		Meta: meta,
	}

	code := c.Run([]string{"-no-color", "-verbose"})
	output := done(t)

	if code != 1 {
		t.Errorf("expected status code 1 but got %d", code)
	}

	expected := `main.tftest.hcl... in progress
  run "setup"... pass



Outputs:

password = (sensitive value)

  run "test"... pass

# test_resource.resource:
resource "test_resource" "resource" {
    destroy_fail = false
    id           = "9ddca5a9"
    value        = (sensitive value)
    write_only   = (write-only attribute)
}


Outputs:

password = (sensitive value)

  run "test_failed"... fail

# test_resource.resource:
resource "test_resource" "resource" {
    destroy_fail = false
    id           = "9ddca5a9"
    value        = (sensitive value)
    write_only   = (write-only attribute)
}


Outputs:

password = (sensitive value)

main.tftest.hcl... tearing down
main.tftest.hcl... fail

Failure! 2 passed, 1 failed.
`

	expectedErr := `
Error: Test assertion failed

  on main.tftest.hcl line 27, in run "test_failed":
  27:     condition = var.complex == {
  28:       foo = "bar"
  29:       baz = test_resource.resource.id
  30:     }
    ├────────────────
    │ LHS:
    │   {
    │     "baz": "(sensitive value)",
    │     "foo": "bar"
    │   }
    │ RHS:
    │   {
    │     "baz": "9ddca5a9",
    │     "foo": "bar"
    │   }
    │ Diff:
    │ --- actual
    │ +++ expected
    │   {
    │ -   "baz": "(sensitive value)",
    │ +   "baz": "9ddca5a9",
    │     "foo": "bar"
    │   }

    │ test_resource.resource.id is "9ddca5a9"
    │ var.complex is {
    │     "baz": "(sensitive value)",
    │     "foo": "bar"
    │   }

expected to fail
`

	actual := output.Stdout()

	if diff := cmp.Diff(actual, expected); len(diff) > 0 {
		t.Errorf("output didn't match expected:\nexpected:\n%s\nactual:\n%s\ndiff:\n%s", expected, actual, diff)
	}

	if diff := cmp.Diff(output.Stderr(), expectedErr); len(diff) > 0 {
		t.Errorf("stderr didn't match expected:\nexpected:\n%s\nactual:\n%s\ndiff:\n%s", expectedErr, output.Stderr(), diff)
	}

	if provider.ResourceCount() > 0 {
		t.Errorf("should have deleted all resources on completion but left %v", provider.ResourceString())
	}
}

// This test takes around 10 seconds to complete, as we're testing the progress
// updates that are printed every 2 seconds. Sorry!
func TestTest_LongRunningTest(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath(path.Join("test", "long_running")), td)
	t.Chdir(td)

	provider := testing_command.NewProvider(nil)
	view, done := testView(t)

	c := &TestCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(provider.Provider),
			View:             view,
		},
	}

	code := c.Run([]string{"-no-color"})
	output := done(t)

	if code != 0 {
		t.Errorf("expected status code 0 but got %d", code)
	}

	actual := output.All()
	expected := `main.tftest.hcl... in progress
  run "test"... pass
main.tftest.hcl... tearing down
main.tftest.hcl... pass

Success! 1 passed, 0 failed.
`

	if code != 0 {
		t.Errorf("expected return code %d but got %d", 0, code)
	}

	if diff := cmp.Diff(expected, actual); len(diff) > 0 {
		t.Errorf("unexpected output\n\nexpected:\n%s\nactual:\n%s\ndiff:\n%s", expected, actual, diff)
	}
}

// This test takes around 10 seconds to complete, as we're testing the progress
// updates that are printed every 2 seconds. Sorry!
func TestTest_LongRunningTestJSON(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath(path.Join("test", "long_running")), td)
	t.Chdir(td)

	provider := testing_command.NewProvider(nil)
	view, done := testView(t)

	c := &TestCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(provider.Provider),
			View:             view,
		},
	}

	code := c.Run([]string{"-json"})
	output := done(t)

	if code != 0 {
		t.Errorf("expected status code 0 but got %d", code)
	}

	actual := output.All()
	var messages []string
	for ix, line := range strings.Split(actual, "\n") {
		if len(line) == 0 {
			// Skip empty lines.
			continue
		}

		if ix == 0 {
			// skip the first one, it's version information
			continue
		}

		var obj map[string]interface{}

		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			t.Errorf("failed to unmarshal returned line: %s", line)
			continue
		}

		// Remove the timestamp as it changes every time.
		delete(obj, "@timestamp")

		if obj["type"].(string) == "test_run" {
			// Then we need to delete the `elapsed` field from within the run
			// as it'll cause flaky tests.

			run := obj["test_run"].(map[string]interface{})
			if run["progress"].(string) != "complete" {
				delete(run, "elapsed")
			}
		}

		message, err := json.Marshal(obj)
		if err != nil {
			t.Errorf("failed to remarshal returned line: %s", line)
			continue
		}

		messages = append(messages, string(message))
	}

	expected := []string{
		`{"@level":"info","@message":"Found 1 file and 1 run block","@module":"terraform.ui","test_abstract":{"main.tftest.hcl":["test"]},"type":"test_abstract"}`,
		`{"@level":"info","@message":"main.tftest.hcl... in progress","@module":"terraform.ui","@testfile":"main.tftest.hcl","test_file":{"path":"main.tftest.hcl","progress":"starting"},"type":"test_file"}`,
		`{"@level":"info","@message":"  \"test\"... in progress","@module":"terraform.ui","@testfile":"main.tftest.hcl","@testrun":"test","test_run":{"path":"main.tftest.hcl","progress":"starting","run":"test"},"type":"test_run"}`,
		`{"@level":"info","@message":"  \"test\"... in progress","@module":"terraform.ui","@testfile":"main.tftest.hcl","@testrun":"test","test_run":{"path":"main.tftest.hcl","progress":"running","run":"test"},"type":"test_run"}`,
		`{"@level":"info","@message":"  \"test\"... in progress","@module":"terraform.ui","@testfile":"main.tftest.hcl","@testrun":"test","test_run":{"path":"main.tftest.hcl","progress":"running","run":"test"},"type":"test_run"}`,
		`{"@level":"info","@message":"  \"test\"... pass","@module":"terraform.ui","@testfile":"main.tftest.hcl","@testrun":"test","test_run":{"path":"main.tftest.hcl","progress":"complete","run":"test","status":"pass"},"type":"test_run"}`,
		`{"@level":"info","@message":"main.tftest.hcl... tearing down","@module":"terraform.ui","@testfile":"main.tftest.hcl","test_file":{"path":"main.tftest.hcl","progress":"teardown"},"type":"test_file"}`,
		`{"@level":"info","@message":"  \"test\"... tearing down","@module":"terraform.ui","@testfile":"main.tftest.hcl","@testrun":"test","test_run":{"path":"main.tftest.hcl","progress":"teardown","run":"test"},"type":"test_run"}`,
		`{"@level":"info","@message":"  \"test\"... tearing down","@module":"terraform.ui","@testfile":"main.tftest.hcl","@testrun":"test","test_run":{"path":"main.tftest.hcl","progress":"teardown","run":"test"},"type":"test_run"}`,
		`{"@level":"info","@message":"main.tftest.hcl... pass","@module":"terraform.ui","@testfile":"main.tftest.hcl","test_file":{"path":"main.tftest.hcl","progress":"complete","status":"pass"},"type":"test_file"}`,
		`{"@level":"info","@message":"Success! 1 passed, 0 failed.","@module":"terraform.ui","test_summary":{"errored":0,"failed":0,"passed":1,"skipped":0,"status":"pass"},"type":"test_summary"}`,
	}

	if code != 0 {
		t.Errorf("expected return code %d but got %d", 0, code)
	}

	if diff := cmp.Diff(expected, messages); len(diff) > 0 {
		t.Errorf("unexpected output\n\nexpected:\n%s\nactual:\n%s\ndiff:\n%s", strings.Join(expected, "\n"), strings.Join(messages, "\n"), diff)
	}
}

func TestTest_InvalidOverrides(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath(path.Join("test", "invalid-overrides")), td)
	t.Chdir(td)

	provider := testing_command.NewProvider(nil)

	providerSource, close := newMockProviderSource(t, map[string][]string{
		"test": {"1.0.0"},
	})
	defer close()

	streams, done := terminal.StreamsForTesting(t)
	view := views.NewView(streams)
	ui := new(cli.MockUi)

	meta := Meta{
		testingOverrides: metaOverridesForProvider(provider.Provider),
		Ui:               ui,
		View:             view,
		Streams:          streams,
		ProviderSource:   providerSource,
	}

	init := &InitCommand{
		Meta: meta,
	}

	if code := init.Run(nil); code != 0 {
		output := done(t)
		t.Fatalf("expected status code 0 but got %d: %s", code, output.All())
	}

	// Reset the streams for the next command.
	streams, done = terminal.StreamsForTesting(t)
	meta.Streams = streams
	meta.View = views.NewView(streams)

	c := &TestCommand{
		Meta: meta,
	}

	code := c.Run([]string{"-no-color"})
	output := done(t)

	if code != 0 {
		t.Errorf("expected status code 0 but got %d", code)
	}

	expected := `main.tftest.hcl... in progress
  run "setup"... pass

Warning: Invalid override target

  on main.tftest.hcl line 39, in run "setup":
  39:     target = test_resource.absent_five

The override target test_resource.absent_five does not exist within the
configuration under test. This could indicate a typo in the target address or
an unnecessary override.

  run "test"... pass

Warning: Invalid override target

  on main.tftest.hcl line 45, in run "test":
  45:     target = module.setup.test_resource.absent_six

The override target module.setup.test_resource.absent_six does not exist
within the configuration under test. This could indicate a typo in the target
address or an unnecessary override.

main.tftest.hcl... tearing down
main.tftest.hcl... pass

Warning: Invalid override target

  on main.tftest.hcl line 4, in mock_provider "test":
   4:     target = test_resource.absent_one

The override target test_resource.absent_one does not exist within the
configuration under test. This could indicate a typo in the target address or
an unnecessary override.

(and 3 more similar warnings elsewhere)

Success! 2 passed, 0 failed.
`

	actual := output.All()

	if diff := cmp.Diff(actual, expected); len(diff) > 0 {
		t.Errorf("output didn't match expected:\nexpected:\n%s\nactual:\n%s\ndiff:\n%s", expected, actual, diff)
	}

	if provider.ResourceCount() > 0 {
		t.Errorf("should have deleted all resources on completion but left %v", provider.ResourceString())
	}
}

func TestTest_InvalidConfig(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath(path.Join("test", "invalid_config")), td)
	t.Chdir(td)

	provider := testing_command.NewProvider(nil)

	providerSource, close := newMockProviderSource(t, map[string][]string{
		"test": {"1.0.0"},
	})
	defer close()

	streams, done := terminal.StreamsForTesting(t)
	view := views.NewView(streams)
	ui := new(cli.MockUi)

	meta := Meta{
		Ui:             ui,
		View:           view,
		Streams:        streams,
		ProviderSource: providerSource,
	}

	init := &InitCommand{
		Meta: meta,
	}

	if code := init.Run(nil); code != 0 {
		output := done(t)
		t.Fatalf("expected status code 0 but got %d: %s", code, output.All())
	}

	// Reset the streams for the next command.
	streams, done = terminal.StreamsForTesting(t)
	meta.Streams = streams
	meta.View = views.NewView(streams)

	c := &TestCommand{
		Meta: meta,
	}

	code := c.Run([]string{"-no-color"})
	output := done(t)

	if code != 1 {
		t.Errorf("expected status code ! but got %d", code)
	}

	expectedOut := `main.tftest.hcl... in progress
  run "test"... fail
main.tftest.hcl... tearing down
main.tftest.hcl... fail

Failure! 0 passed, 1 failed.
`
	expectedErr := `
Error: Failed to load plugin schemas

Error while loading schemas for plugin components: Failed to obtain provider
schema: Could not load the schema for provider
registry.terraform.io/hashicorp/test: failed to instantiate provider
"registry.terraform.io/hashicorp/test" to obtain schema: fork/exec
.terraform/providers/registry.terraform.io/hashicorp/test/1.0.0/%s/terraform-provider-test_1.0.0:
permission denied..
`
	expectedErr = fmt.Sprintf(expectedErr, runtime.GOOS+"_"+runtime.GOARCH)
	out := output.Stdout()
	err := output.Stderr()

	if diff := cmp.Diff(out, expectedOut); len(diff) > 0 {
		t.Errorf("output didn't match expected:\nexpected:\n%s\nactual:\n%s\ndiff:\n%s", expectedErr, out, diff)
	}
	if diff := cmp.Diff(err, expectedErr); len(diff) > 0 {
		t.Errorf("error didn't match expected:\nexpected:\n%s\nactual:\n%s\ndiff:\n%s", expectedErr, err, diff)
	}

	if provider.ResourceCount() > 0 {
		t.Errorf("should have deleted all resources on completion but left %v", provider.ResourceString())
	}
}

// TestTest_ValidateBackendConfiguration tests validation of how backends are declared in test files:
// * it's not valid to re-use the same backend config (i.e the same state file)
// * it's not valid to use a deprecated backend type
// * it's not valid to use a non-existent backend type
//
// Backend validation performed in the command package is dependent on the internal/backend/init package,
// which cannot be imported in configuration parsing packages without creating an import cycle.
func TestTest_ReusedBackendConfiguration(t *testing.T) {
	testCases := map[string]struct {
		dirName   string
		expectErr string
	}{
		"validation detects when backend config is reused by runs using different user-supplied state key value": {
			dirName: "reused-backend-config",
			expectErr: `
Error: Repeat use of the same backend block

  on main.tftest.hcl line 12, in run "test_2":
  12:   backend "local" {

The run "test_2" contains a backend configuration that's already been used in
run "test_1". Sharing the same backend configuration between separate runs
will result in conflicting state updates.
`,
		},
		"validation detects when backend config is reused by runs using different implicit state key (corresponding to root and a child module) ": {
			dirName: "reused-backend-config-child-modules",
			expectErr: `
Error: Repeat use of the same backend block

  on main.tftest.hcl line 19, in run "test_2":
  19:   backend "local" {

The run "test_2" contains a backend configuration that's already been used in
run "test_1". Sharing the same backend configuration between separate runs
will result in conflicting state updates.
`,
		},
		"validation detects when a deprecated backend type is used": {
			dirName: "removed-backend-type",
			expectErr: `
Error: Unsupported backend type

  on main.tftest.hcl line 7, in run "test_removed_backend":
   7:   backend "etcd" {

The "etcd" backend is not supported in Terraform v1.3 or later.
`,
		},
		"validation detects when a non-existent backend type": {
			dirName: "non-existent-backend-type",
			expectErr: `
Error: Unsupported backend type

  on main.tftest.hcl line 7, in run "test_invalid_backend":
   7:   backend "foobar" {

There is no backend type named "foobar".
`,
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			td := t.TempDir()
			testCopyDir(t, testFixturePath(path.Join("test", tc.dirName)), td)
			t.Chdir(td)

			provider := testing_command.NewProvider(nil)

			providerSource, close := newMockProviderSource(t, map[string][]string{
				"test": {"1.0.0"},
			})
			defer close()

			streams, done := terminal.StreamsForTesting(t)
			view := views.NewView(streams)
			ui := new(cli.MockUi)

			meta := Meta{
				testingOverrides:          metaOverridesForProvider(provider.Provider),
				Ui:                        ui,
				View:                      view,
				Streams:                   streams,
				ProviderSource:            providerSource,
				AllowExperimentalFeatures: true,
			}

			init := &InitCommand{
				Meta: meta,
			}

			output := done(t)

			if code := init.Run(nil); code != 0 {
				t.Fatalf("expected status code 0 but got %d: %s", code, output.All())
			}

			// Reset the streams for the next command.
			streams, done = terminal.StreamsForTesting(t)
			meta.Streams = streams
			meta.View = views.NewView(streams)

			c := &TestCommand{
				Meta: meta,
			}

			code := c.Run([]string{"-no-color"})
			output = done(t)

			// Assertions
			if code != 1 {
				t.Errorf("expected status code 1 but got %d", code)
			}

			if diff := cmp.Diff(output.All(), tc.expectErr); len(diff) > 0 {
				t.Errorf("output didn't match expected:\nexpected:\n%s\nactual:\n%s\ndiff:\n%s", tc.expectErr, output.All(), diff)
			}

			if provider.ResourceCount() > 0 {
				t.Errorf("should have deleted all resources on completion but left %v", provider.ResourceString())
			}
		})
	}
}

// When there is no starting state, state is created by the run containing the backend block
func TestTest_UseOfBackends_stateCreatedByBackend(t *testing.T) {
	dirName := "valid-use-local-backend/no-prior-state"

	resourceId := "12345"
	expectedState := `test_resource.foobar:
  ID = 12345
  provider = provider["registry.terraform.io/hashicorp/test"]
  destroy_fail = false
  value = value-from-run-that-controls-backend

Outputs:

supplied_input_value = value-from-run-that-controls-backend
test_resource_id = 12345`

	// SETUP
	td := t.TempDir()
	testCopyDir(t, testFixturePath(path.Join("test", dirName)), td)
	t.Chdir(td)
	localStatePath := filepath.Join(td, DefaultStateFilename)

	provider := testing_command.NewProvider(nil)

	providerSource, close := newMockProviderSource(t, map[string][]string{
		"test": {"1.0.0"},
	})
	defer close()

	streams, done := terminal.StreamsForTesting(t)
	view := views.NewView(streams)
	ui := new(cli.MockUi)

	meta := Meta{
		testingOverrides:          metaOverridesForProvider(provider.Provider),
		Ui:                        ui,
		View:                      view,
		Streams:                   streams,
		ProviderSource:            providerSource,
		AllowExperimentalFeatures: true,
	}

	// INIT
	init := &InitCommand{
		Meta: meta,
	}

	if code := init.Run(nil); code != 0 {
		output := done(t)
		t.Fatalf("expected status code 0 but got %d: %s", code, output.All())
	}

	// TEST
	// Reset the streams for the next command.
	streams, done = terminal.StreamsForTesting(t)
	meta.Streams = streams
	meta.View = views.NewView(streams)

	c := &TestCommand{
		Meta: meta,
	}

	code := c.Run([]string{"-no-color"})
	output := done(t)

	// ASSERTIONS
	if code != 0 {
		t.Errorf("expected status code 0 but got %d", code)
	}
	stdErr := output.Stderr()
	if len(stdErr) > 0 {
		t.Fatalf("unexpected error output:\n%s", stdErr)
	}

	// State is stored according to the backend block
	actualState := testStateRead(t, localStatePath)
	if diff := cmp.Diff(actualState.String(), expectedState); len(diff) > 0 {
		t.Fatalf("state didn't match expected:\nexpected:\n%s\nactual:\n%s\ndiff:\n%s", expectedState, actualState, diff)
	}

	if provider.ResourceCount() != 1 {
		t.Fatalf("should have deleted all resources on completion except test_resource.a. Instead state contained %b resources: %v", provider.ResourceCount(), provider.ResourceString())
	}

	val := provider.Store.Get(resourceId)

	if val.GetAttr("id").AsString() != resourceId {
		t.Errorf("expected resource to have value %q but got %s", resourceId, val.GetAttr("id").AsString())
	}
	if val.GetAttr("value").AsString() != "value-from-run-that-controls-backend" {
		t.Errorf("expected resource to have value 'value-from-run-that-controls-backend' but got %s", val.GetAttr("value").AsString())
	}
}

// When there is pre-existing state, the state is used by the run containing the backend block
func TestTest_UseOfBackends_priorStateUsedByBackend(t *testing.T) {
	dirName := "valid-use-local-backend/with-prior-state"
	resourceId := "53d69028-477d-7ba0-83c3-ff3807e3756f" // This value needs to match the state file in the test fixtures
	expectedState := fmt.Sprintf(`test_resource.foobar:
  ID = %s
  provider = provider["registry.terraform.io/hashicorp/test"]
  destroy_fail = false
  value = value-from-run-that-controls-backend

Outputs:

supplied_input_value = value-from-run-that-controls-backend
test_resource_id = %s`, resourceId, resourceId)

	// SETUP
	td := t.TempDir()
	testCopyDir(t, testFixturePath(path.Join("test", dirName)), td)
	t.Chdir(td)
	localStatePath := filepath.Join(td, DefaultStateFilename)

	// resource store is like the remote object - needs to be assembled
	// to resemble what's in state / the remote object being reused by the test.
	resourceStore := &testing_command.ResourceStore{
		Data: map[string]cty.Value{
			resourceId: cty.ObjectVal(map[string]cty.Value{
				"id":                   cty.StringVal(resourceId),
				"interrupt_count":      cty.NullVal(cty.Number),
				"value":                cty.StringVal("value-from-run-that-controls-backend"),
				"write_only":           cty.NullVal(cty.String),
				"create_wait_seconds":  cty.NullVal(cty.Number),
				"destroy_fail":         cty.False,
				"destroy_wait_seconds": cty.NullVal(cty.Number),
				"defer":                cty.NullVal(cty.Bool),
			})},
	}
	provider := testing_command.NewProvider(resourceStore)

	providerSource, close := newMockProviderSource(t, map[string][]string{
		"test": {"1.0.0"},
	})
	defer close()

	streams, done := terminal.StreamsForTesting(t)
	view := views.NewView(streams)
	ui := new(cli.MockUi)

	meta := Meta{
		testingOverrides:          metaOverridesForProvider(provider.Provider),
		Ui:                        ui,
		View:                      view,
		Streams:                   streams,
		ProviderSource:            providerSource,
		AllowExperimentalFeatures: true,
	}

	// INIT
	init := &InitCommand{
		Meta: meta,
	}

	if code := init.Run(nil); code != 0 {
		output := done(t)
		t.Fatalf("expected status code 0 but got %d: %s", code, output.All())
	}

	// TEST
	// Reset the streams for the next command.
	streams, done = terminal.StreamsForTesting(t)
	meta.Streams = streams
	meta.View = views.NewView(streams)

	c := &TestCommand{
		Meta: meta,
	}

	code := c.Run([]string{"-no-color"})
	output := done(t)

	// ASSERTIONS
	if code != 0 {
		t.Errorf("expected status code 0 but got %d", code)
	}
	stdErr := output.Stderr()
	if len(stdErr) > 0 {
		t.Fatalf("unexpected error output:\n%s", stdErr)
	}

	// State is stored according to the backend block
	actualState := testStateRead(t, localStatePath)
	if diff := cmp.Diff(actualState.String(), expectedState); len(diff) > 0 {
		t.Fatalf("state didn't match expected:\nexpected:\n%s\nactual:\n%s\ndiff:\n%s", expectedState, actualState, diff)
	}

	if provider.ResourceCount() != 1 {
		t.Fatalf("should have deleted all resources on completion except test_resource.a. Instead state contained %b resources: %v", provider.ResourceCount(), provider.ResourceString())
	}

	val := provider.Store.Get(resourceId)

	// If the ID hasn't changed then we've used the pre-existing state, instead of remaking the resource
	if val.GetAttr("id").AsString() != resourceId {
		t.Errorf("expected resource to have value %q but got %s", resourceId, val.GetAttr("id").AsString())
	}
	if val.GetAttr("value").AsString() != "value-from-run-that-controls-backend" {
		t.Errorf("expected resource to have value 'value-from-run-that-controls-backend' but got %s", val.GetAttr("value").AsString())
	}
}

// Testing whether a state artifact is made for a run block with a backend or not
//
// Artifacts are made when the cleanup operation errors.
func TestTest_UseOfBackends_whenStateArtifactsAreMade(t *testing.T) {
	cases := map[string]struct {
		forceError          bool
		expectedCode        int
		expectStateManifest bool
	}{
		"no artifact made when there are no cleanup errors when processing a run block with a backend": {
			forceError:          false,
			expectedCode:        0,
			expectStateManifest: false,
		},
		"artifact made when a cleanup error is forced when processing a run block with a backend": {
			forceError:          true,
			expectedCode:        1,
			expectStateManifest: true,
		},
	}

	for tn, tc := range cases {
		t.Run(tn, func(t *testing.T) {

			// SETUP
			td := t.TempDir()
			testCopyDir(t, testFixturePath(path.Join("test", "valid-use-local-backend/no-prior-state")), td)
			t.Chdir(td)

			provider := testing_command.NewProvider(nil)
			erroringInvocationNum := 3
			applyResourceChangeCount := 0
			if tc.forceError {
				oldFunc := provider.Provider.ApplyResourceChangeFn
				newFunc := func(req providers.ApplyResourceChangeRequest) providers.ApplyResourceChangeResponse {
					applyResourceChangeCount++
					if applyResourceChangeCount < erroringInvocationNum {
						return oldFunc(req)
					}
					// Given the config in the test fixture used, the 5th call to this function is during cleanup
					// Return error to force error diagnostics during cleanup
					var diags tfdiags.Diagnostics
					return providers.ApplyResourceChangeResponse{
						Diagnostics: diags.Append(errors.New("error forced by mock provider")),
					}
				}
				provider.Provider.ApplyResourceChangeFn = newFunc
			}

			providerSource, close := newMockProviderSource(t, map[string][]string{
				"test": {"1.0.0"},
			})
			defer close()

			streams, done := terminal.StreamsForTesting(t)
			view := views.NewView(streams)
			ui := new(cli.MockUi)

			meta := Meta{
				testingOverrides:          metaOverridesForProvider(provider.Provider),
				Ui:                        ui,
				View:                      view,
				Streams:                   streams,
				ProviderSource:            providerSource,
				AllowExperimentalFeatures: true,
			}

			// INIT
			init := &InitCommand{
				Meta: meta,
			}

			if code := init.Run(nil); code != 0 {
				output := done(t)
				t.Fatalf("expected status code 0 but got %d: %s", code, output.All())
			}

			// TEST
			// Reset the streams for the next command.
			streams, done = terminal.StreamsForTesting(t)
			meta.Streams = streams
			meta.View = views.NewView(streams)

			c := &TestCommand{
				Meta: meta,
			}

			code := c.Run([]string{"-no-color"})

			// ASSERTIONS

			if tc.forceError && (applyResourceChangeCount != erroringInvocationNum) {
				t.Fatalf(`Test did not force error as expected. This is because a magic number in the test setup is coupled to the config.
The apply resource change function was invoked %d times but we trigger an error on the %dth time`, applyResourceChangeCount, erroringInvocationNum)
			}

			if code != tc.expectedCode {
				output := done(t)
				t.Errorf("expected status code %d but got %d: %s", tc.expectedCode, code, output.All())
			}

			// State is NOT stored in .terraform/test as a state artifact because
			// there haven't been any failures or errors in the tests
			manifest, err := teststates.LoadManifest(td, true)
			if err != nil {
				t.Fatal(err)
			}
			foundIds := []string{}
			for _, file := range manifest.Files {
				for _, state := range file.States {
					foundIds = append(foundIds, state.ID)
				}
			}
			if len(foundIds) > 0 && !tc.expectStateManifest {
				t.Fatalf("found %d state files in .terraform/test when none were expected", len(foundIds))
			}
			if len(foundIds) == 0 && tc.expectStateManifest {
				t.Fatalf("found 0 state files in .terraform/test when they were were expected")
			}
		})
	}
}

func TestTest_UseOfBackends_validatesUseOfSkipCleanup(t *testing.T) {
	cases := map[string]struct {
		testDir    string
		expectCode int
		expectErr  bool
	}{
		"cannot set skip_cleanup=false alongside a backend block": {
			testDir:    "backend-with-skip-cleanup/false",
			expectCode: 1,
			expectErr:  true,
		},
		"can set skip_cleanup=true alongside a backend block": {
			testDir:    "backend-with-skip-cleanup/true",
			expectCode: 0,
			expectErr:  false,
		},
	}

	for tn, tc := range cases {
		t.Run(tn, func(t *testing.T) {
			// SETUP
			td := t.TempDir()
			testCopyDir(t, testFixturePath(path.Join("test", tc.testDir)), td)
			t.Chdir(td)

			provider := testing_command.NewProvider(nil)
			providerSource, close := newMockProviderSource(t, map[string][]string{
				"test": {"1.0.0"},
			})
			defer close()

			streams, done := terminal.StreamsForTesting(t)
			view := views.NewView(streams)
			ui := new(cli.MockUi)

			meta := Meta{
				testingOverrides:          metaOverridesForProvider(provider.Provider),
				Ui:                        ui,
				View:                      view,
				Streams:                   streams,
				ProviderSource:            providerSource,
				AllowExperimentalFeatures: true,
			}

			// INIT
			init := &InitCommand{
				Meta: meta,
			}

			code := init.Run([]string{"-no-color"})
			output := done(t)

			// ASSERTIONS
			if code != tc.expectCode {
				t.Errorf("expected status code %d but got %d", tc.expectCode, code)
			}
			stdErr := output.Stderr()
			if len(stdErr) == 0 && tc.expectErr {
				t.Fatal("expected error output but got none")
			}
			if len(stdErr) != 0 && !tc.expectErr {
				t.Fatalf("did not expect error output but got: %s", stdErr)
			}

			if provider.ResourceCount() > 0 {
				t.Fatalf("should have deleted all resources on completion but left %v", provider.ResourceString())
			}

		})
	}
}

func TestTest_UseOfBackends_failureDuringApply(t *testing.T) {
	// SETUP
	td := t.TempDir()
	testCopyDir(t, testFixturePath(path.Join("test", "valid-use-local-backend/no-prior-state")), td)
	t.Chdir(td)
	localStatePath := filepath.Join(td, DefaultStateFilename)

	provider := testing_command.NewProvider(nil)
	// Force a failure during apply
	provider.Provider.ApplyResourceChangeFn = func(req providers.ApplyResourceChangeRequest) providers.ApplyResourceChangeResponse {
		resp := providers.ApplyResourceChangeResponse{}
		resp.Diagnostics = resp.Diagnostics.Append(fmt.Errorf("forced error"))
		return resp
	}

	providerSource, close := newMockProviderSource(t, map[string][]string{
		"test": {"1.0.0"},
	})
	defer close()

	streams, done := terminal.StreamsForTesting(t)
	view := views.NewView(streams)
	ui := new(cli.MockUi)

	meta := Meta{
		testingOverrides:          metaOverridesForProvider(provider.Provider),
		Ui:                        ui,
		View:                      view,
		Streams:                   streams,
		ProviderSource:            providerSource,
		AllowExperimentalFeatures: true,
	}

	// INIT
	init := &InitCommand{
		Meta: meta,
	}

	output := done(t)
	if code := init.Run(nil); code != 0 {
		t.Fatalf("expected status code 0 but got %d: %s", code, output.All())
	}

	// TEST
	// Reset the streams for the next command.
	streams, done = terminal.StreamsForTesting(t)
	meta.Streams = streams
	meta.View = views.NewView(streams)

	c := &TestCommand{
		Meta: meta,
	}

	code := c.Run([]string{"-no-color"})
	output = done(t)

	// ASSERTIONS
	if code != 1 {
		t.Errorf("expected status code 1 but got %d", code)
	}
	stdErr := output.Stderr()
	if len(stdErr) == 0 {
		t.Fatal("expected error output but got none")
	}

	// Resource was not provisioned
	if provider.ResourceCount() > 0 {
		t.Fatalf("no resources should have been provisioned successfully but got %v", provider.ResourceString())
	}

	// When there is a failure to apply changes to the test_resource, the resulting state saved via the backend
	// only includes the output and lacks any information about the test_resource
	actualBackendState := testStateRead(t, localStatePath)
	expectedBackendState := `<no state>
Outputs:

supplied_input_value = value-from-run-that-controls-backend
test_resource_id = 12345`
	if diff := cmp.Diff(actualBackendState.String(), expectedBackendState); len(diff) > 0 {
		t.Fatalf("state didn't match expected:\nexpected:\n%s\nactual:\n%s\ndiff:\n%s", expectedBackendState, actualBackendState, diff)
	}

	expectedStates := map[string][]string{} // empty
	actualStates := statesFromManifest(t, td)

	// No state artifacts are made: Verify the states in the manifest
	if diff := cmp.Diff(expectedStates, removeOutputs(actualStates)); diff != "" {
		t.Fatalf("unexpected states: %s", diff)
	}
}

func TestTest_RunBlocksInProviders(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath(path.Join("test", "provider_runs")), td)
	t.Chdir(td)

	provider := testing_command.NewProvider(nil)

	providerSource, close := newMockProviderSource(t, map[string][]string{
		"test": {"1.0.0"},
	})
	defer close()

	streams, done := terminal.StreamsForTesting(t)
	view := views.NewView(streams)
	ui := new(cli.MockUi)

	meta := Meta{
		testingOverrides: metaOverridesForProvider(provider.Provider),
		Ui:               ui,
		View:             view,
		Streams:          streams,
		ProviderSource:   providerSource,
	}

	init := &InitCommand{
		Meta: meta,
	}

	if code := init.Run(nil); code != 0 {
		output := done(t)
		t.Fatalf("expected status code 0 but got %d: %s", code, output.All())
	}

	// Reset the streams for the next command.
	streams, done = terminal.StreamsForTesting(t)
	meta.Streams = streams
	meta.View = views.NewView(streams)

	test := &TestCommand{
		Meta: meta,
	}

	code := test.Run([]string{"-no-color"})
	output := done(t)

	if code != 0 {
		t.Errorf("expected status code 0 but got %d", code)
	}

	expected := `main.tftest.hcl... in progress
  run "setup"... pass
  run "main"... pass
main.tftest.hcl... tearing down
main.tftest.hcl... pass

Success! 2 passed, 0 failed.
`
	actual := output.All()
	if diff := cmp.Diff(actual, expected); len(diff) > 0 {
		t.Errorf("output didn't match expected:\nexpected:\n%s\nactual:\n%s\ndiff:\n%s", expected, actual, diff)
	}

	if provider.ResourceCount() > 0 {
		t.Errorf("should have deleted all resources on completion but left %v", provider.ResourceString())
	}
}

func TestTest_RunBlocksInProviders_BadReferences(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath(path.Join("test", "provider_runs_invalid")), td)
	t.Chdir(td)

	store := &testing_command.ResourceStore{
		Data: make(map[string]cty.Value),
	}

	providerSource, close := newMockProviderSource(t, map[string][]string{
		"test": {"1.0.0"},
	})
	defer close()

	streams, done := terminal.StreamsForTesting(t)
	view := views.NewView(streams)
	ui := new(cli.MockUi)

	meta := Meta{
		testingOverrides: &testingOverrides{
			Providers: map[addrs.Provider]providers.Factory{
				addrs.NewDefaultProvider("test"): func() (providers.Interface, error) {
					return testing_command.NewProvider(store).Provider, nil
				},
			},
		},
		Ui:             ui,
		View:           view,
		Streams:        streams,
		ProviderSource: providerSource,
	}

	init := &InitCommand{
		Meta: meta,
	}

	if code := init.Run(nil); code != 0 {
		output := done(t)
		t.Fatalf("expected status code 0 but got %d: %s", code, output.All())
	}

	// Reset the streams for the next command.
	streams, done = terminal.StreamsForTesting(t)
	meta.Streams = streams
	meta.View = views.NewView(streams)

	test := &TestCommand{
		Meta: meta,
	}

	code := test.Run([]string{"-no-color"})
	output := done(t)

	if code != 1 {
		t.Errorf("expected status code 1 but got %d", code)
	}

	expectedOut := `missing_run_block.tftest.hcl... in progress
  run "main"... skip
missing_run_block.tftest.hcl... tearing down
missing_run_block.tftest.hcl... fail
unused_provider.tftest.hcl... in progress
  run "main"... pass
unused_provider.tftest.hcl... tearing down
unused_provider.tftest.hcl... pass

Failure! 1 passed, 0 failed, 1 skipped.
`
	actualOut := output.Stdout()
	if diff := cmp.Diff(actualOut, expectedOut); len(diff) > 0 {
		t.Errorf("output didn't match expected:\nexpected:\n%s\nactual:\n%s\ndiff:\n%s", expectedOut, actualOut, diff)
	}

	expectedErr := `
Error: Reference to unknown run block

  on missing_run_block.tftest.hcl line 2, in provider "test":
   2:   resource_prefix = run.missing.resource_directory

The run block "missing" does not exist within this test file.
`
	actualErr := output.Stderr()
	if diff := cmp.Diff(actualErr, expectedErr); len(diff) > 0 {
		t.Errorf("output didn't match expected:\nexpected:\n%s\nactual:\n%s\ndiff:\n%s", expectedErr, actualErr, diff)
	}

	if len(store.Data) > 0 {
		t.Errorf("should have deleted all resources on completion but left %v", len(store.Data))
	}
}

func TestTest_JUnitOutput(t *testing.T) {
	tcs := map[string]struct {
		path         string
		code         int
		wantFilename string
	}{
		"can create XML for a single file with 1 pass, 1 fail": {
			path:         "junit-output/1pass-1fail",
			wantFilename: "expected-output.xml",
			code:         1, // Test failure
		},
		"can create XML for multiple files with 1 pass each": {
			path:         "junit-output/multiple-files",
			wantFilename: "expected-output.xml",
			code:         0,
		},
		"can display a test run's errors under the equivalent test case element": {
			path:         "junit-output/missing-provider",
			wantFilename: "expected-output.xml",
			code:         1, // Test error
		},
	}

	for tn, tc := range tcs {
		t.Run(tn, func(t *testing.T) {
			// Setup test
			td := t.TempDir()
			testPath := path.Join("test", tc.path)
			testCopyDir(t, testFixturePath(testPath), td)
			t.Chdir(td)

			provider := testing_command.NewProvider(nil)
			view, done := testView(t)

			c := &TestCommand{
				Meta: Meta{
					testingOverrides: metaOverridesForProvider(provider.Provider),
					View:             view,
				},
			}

			// Run command with -junit-xml=./output.xml flag
			outputFile := fmt.Sprintf("%s/output.xml", td)
			code := c.Run([]string{fmt.Sprintf("-junit-xml=%s", outputFile), "-no-color"})
			done(t)

			// Assertions
			if code != tc.code {
				t.Errorf("expected status code %d but got %d", tc.code, code)
			}

			actualOut, err := os.ReadFile(outputFile)
			if err != nil {
				t.Fatalf("error opening XML file: %s", err)
			}
			expectedOutputFile := fmt.Sprintf("%s/%s", td, tc.wantFilename)
			expectedOutput, err := os.ReadFile(expectedOutputFile)
			if err != nil {
				t.Fatalf("error opening XML file: %s", err)
			}

			// actual output will include timestamps and test duration data, which isn't deterministic; redact it for comparison
			timeRegexp := regexp.MustCompile(`time="[^"]+"`)
			actualOut = timeRegexp.ReplaceAll(actualOut, []byte("time=\"TIME_REDACTED\""))
			timestampRegexp := regexp.MustCompile(`timestamp="[^"]+"`)
			actualOut = timestampRegexp.ReplaceAll(actualOut, []byte("timestamp=\"TIMESTAMP_REDACTED\""))

			if !bytes.Equal(actualOut, expectedOutput) {
				t.Fatalf("wanted XML:\n%s\n got XML:\n%s\ndiff:%s\n", string(expectedOutput), string(actualOut), cmp.Diff(expectedOutput, actualOut))
			}

			if provider.ResourceCount() > 0 {
				t.Errorf("should have deleted all resources on completion but left %v", provider.ResourceString())
			}
		})
	}
}

func TestTest_ReferencesIntoIncompletePlan(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath(path.Join("test", "expect-failures-assertions")), td)
	t.Chdir(td)

	provider := testing_command.NewProvider(nil)
	providerSource, close := newMockProviderSource(t, map[string][]string{
		"test": {"1.0.0"},
	})
	defer close()

	streams, done := terminal.StreamsForTesting(t)
	view := views.NewView(streams)
	ui := new(cli.MockUi)

	meta := Meta{
		testingOverrides: metaOverridesForProvider(provider.Provider),
		Ui:               ui,
		View:             view,
		Streams:          streams,
		ProviderSource:   providerSource,
	}

	init := &InitCommand{
		Meta: meta,
	}

	if code := init.Run(nil); code != 0 {
		output := done(t)
		t.Fatalf("expected status code %d but got %d: %s", 0, code, output.All())
	}

	// Reset the streams for the next command.
	streams, done = terminal.StreamsForTesting(t)
	meta.Streams = streams
	meta.View = views.NewView(streams)

	c := &TestCommand{
		Meta: meta,
	}

	code := c.Run([]string{"-no-color"})
	if code != 1 {
		t.Errorf("expected status code %d but got %d", 0, code)
	}
	output := done(t)

	out, err := output.Stdout(), output.Stderr()

	expectedOut := `main.tftest.hcl... in progress
  run "fail"... fail
main.tftest.hcl... tearing down
main.tftest.hcl... fail

Failure! 0 passed, 1 failed.
`

	if diff := cmp.Diff(out, expectedOut); len(diff) > 0 {
		t.Errorf("expected:\n%s\nactual:\n%s\ndiff:\n%s", expectedOut, out, diff)
	}

	if !strings.Contains(err, "Reference to uninitialized resource") {
		t.Errorf("missing reference to uninitialized resource error: \n%s", err)
	}

	if !strings.Contains(err, "Reference to uninitialized local") {
		t.Errorf("missing reference to uninitialized local error: \n%s", err)
	}
}

func TestTest_ReferencesIntoTargetedPlan(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath(path.Join("test", "invalid-reference-with-target")), td)
	t.Chdir(td)

	provider := testing_command.NewProvider(nil)
	providerSource, close := newMockProviderSource(t, map[string][]string{
		"test": {"1.0.0"},
	})
	defer close()

	streams, done := terminal.StreamsForTesting(t)
	view := views.NewView(streams)
	ui := new(cli.MockUi)

	meta := Meta{
		testingOverrides: metaOverridesForProvider(provider.Provider),
		Ui:               ui,
		View:             view,
		Streams:          streams,
		ProviderSource:   providerSource,
	}

	init := &InitCommand{
		Meta: meta,
	}

	if code := init.Run(nil); code != 0 {
		output := done(t)
		t.Fatalf("expected status code %d but got %d: %s", 0, code, output.All())
	}

	// Reset the streams for the next command.
	streams, done = terminal.StreamsForTesting(t)
	meta.Streams = streams
	meta.View = views.NewView(streams)

	c := &TestCommand{
		Meta: meta,
	}

	code := c.Run([]string{"-no-color"})
	if code != 1 {
		t.Errorf("expected status code %d but got %d", 0, code)
	}
	output := done(t)

	err := output.Stderr()

	if !strings.Contains(err, "Reference to uninitialized variable") {
		t.Errorf("missing reference to uninitialized variable error: \n%s", err)
	}
}

// https://github.com/hashicorp/terraform/issues/37546
func TestTest_TeardownOrder(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath(path.Join("test", "rds_shared_subnet")), td)
	t.Chdir(td)

	provider := testing_command.NewProvider(nil)
	providerSource, close := newMockProviderSource(t, map[string][]string{
		"test": {"1.0.0"},
	})
	defer close()

	streams, done := terminal.StreamsForTesting(t)
	view := views.NewView(streams)
	ui := new(cli.MockUi)

	meta := Meta{
		testingOverrides: metaOverridesForProvider(provider.Provider),
		Ui:               ui,
		View:             view,
		Streams:          streams,
		ProviderSource:   providerSource,
	}

	init := &InitCommand{
		Meta: meta,
	}

	if code := init.Run(nil); code != 0 {
		output := done(t)
		t.Fatalf("expected status code %d but got %d: %s", 0, code, output.All())
	}

	c := &TestCommand{
		Meta: meta,
	}

	code := c.Run([]string{"-no-color", "-json"})
	if code != 0 {
		t.Errorf("expected status code %d but got %d", 0, code)
	}
	output := done(t).All()

	// Parse the JSON output to check teardown order
	var setupTeardownStart time.Time
	var lastRunTeardownStart time.Time

	for line := range strings.SplitSeq(output, "\n") {
		if strings.Contains(line, `"progress":"teardown"`) {
			var obj map[string]any
			if err := json.Unmarshal([]byte(line), &obj); err != nil {
				t.Fatal(err)
			}
			if strings.Contains(line, `"setup_tests"`) {
				if ts, ok := obj["@timestamp"].(string); ok {
					// record the first time that the setup teardown appears in the output
					if setupTeardownStart.IsZero() {
						parsedTime, _ := time.Parse(time.RFC3339, ts)
						setupTeardownStart = parsedTime
					}
				}

			} else {
				if ts, ok := obj["@timestamp"].(string); ok {
					parsedTime, _ := time.Parse(time.RFC3339, ts)
					// record the last time that a run's teardown appears in the output
					if parsedTime.After(lastRunTeardownStart) {
						lastRunTeardownStart = parsedTime
					}
				}
			}
		}
	}

	// all runs should have been down with teardown before the setup starts
	if lastRunTeardownStart.After(setupTeardownStart) {
		t.Fatalf("setup is tearing down before dependants are done: \n %s", output)
	}

	if provider.ResourceCount() > 0 {
		t.Logf("Resources remaining after test completion (this might indicate the teardown issue): %v", provider.ResourceString())
	}
}

// testModuleInline takes a map of path -> config strings and yields a config
// structure with those files loaded from disk
func testModuleInline(t *testing.T, sources map[string]string) (*configs.Config, string, func()) {
	t.Helper()

	cfgPath := t.TempDir()

	for path, configStr := range sources {
		dir := filepath.Dir(path)
		if dir != "." {
			err := os.MkdirAll(filepath.Join(cfgPath, dir), os.FileMode(0777))
			if err != nil {
				t.Fatalf("Error creating subdir: %s", err)
			}
		}
		// Write the configuration
		cfgF, err := os.Create(filepath.Join(cfgPath, path))
		if err != nil {
			t.Fatalf("Error creating temporary file for config: %s", err)
		}

		_, err = io.Copy(cfgF, strings.NewReader(configStr))
		cfgF.Close()
		if err != nil {
			t.Fatalf("Error creating temporary file for config: %s", err)
		}
	}

	loader, cleanup := configload.NewLoaderForTests(t)

	// Test modules usually do not refer to remote sources, and for local
	// sources only this ultimately just records all of the module paths
	// in a JSON file so that we can load them below.
	inst := initwd.NewModuleInstaller(loader.ModulesDir(), loader, registry.NewClient(nil, nil))
	_, instDiags := inst.InstallModules(context.Background(), cfgPath, "tests", true, false, initwd.ModuleInstallHooksImpl{})
	if instDiags.HasErrors() {
		t.Fatal(instDiags.Err())
	}

	// Since module installer has modified the module manifest on disk, we need
	// to refresh the cache of it in the loader.
	if err := loader.RefreshModules(); err != nil {
		t.Fatalf("failed to refresh modules after installation: %s", err)
	}

	config, diags := loader.LoadConfigWithTests(cfgPath, "tests")
	if diags.HasErrors() {
		t.Fatal(diags.Error())
	}

	return config, cfgPath, func() {
		cleanup()
	}
}

func statesFromManifest(t *testing.T, td string) map[string][]string {
	manifest, err := teststates.LoadManifest(td, true)
	if err != nil {
		t.Fatal(err)
	}

	states := make(map[string][]string)

	// Verify the states in the manifest
	for fileName, file := range manifest.Files {
		for name, state := range file.States {
			sm := statemgr.NewFilesystem(manifest.StateFilePath(state.ID))
			if err := sm.RefreshState(); err != nil {
				t.Fatalf("error when reading state file: %s", err)
			}
			state := sm.State()

			// If the state is nil, then the test cleaned up the state
			if state == nil || state.Empty() {
				continue
			}

			var resources []string
			for _, module := range state.Modules {
				for _, resource := range module.Resources {
					resources = append(resources, resource.Addr.String())
				}
			}
			for _, output := range state.RootOutputValues {
				resources = append(resources, output.Addr.String())
			}
			if len(resources) == 0 {
				continue
			}
			sort.Strings(resources)
			states[strings.TrimSuffix(fileName, ".tftest.hcl")+"."+name] = resources
		}
	}

	return states
}

func equalIgnoreOrder() cmp.Option {
	less := func(a, b string) bool { return a < b }
	return cmpopts.SortSlices(less)
}

func removeOutputs(states map[string][]string) map[string][]string {
	for k, v := range states {
		new_v := make([]string, 0, len(v))
		for _, s := range v {
			if !strings.HasPrefix(s, "output.") {
				new_v = append(new_v, s)
			}
		}
		if len(new_v) == 0 {
			delete(states, k)
			continue
		}
		states[k] = new_v
	}

	return states
}
