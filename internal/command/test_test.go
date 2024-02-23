// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"encoding/json"
	"fmt"
	"path"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/cli"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	testing_command "github.com/hashicorp/terraform/internal/command/testing"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/terminal"
)

func TestTest_Runs(t *testing.T) {
	tcs := map[string]struct {
		override              string
		args                  []string
		expectedOut           string
		expectedErr           []string
		expectedResourceCount int
		code                  int
		initCode              int
		skip                  bool
	}{
		"simple_pass": {
			expectedOut: "1 passed, 0 failed.",
			code:        0,
		},
		"simple_pass_nested": {
			expectedOut: "1 passed, 0 failed.",
			code:        0,
		},
		"simple_pass_nested_alternate": {
			args:        []string{"-test-directory", "other"},
			expectedOut: "1 passed, 0 failed.",
			code:        0,
		},
		"simple_pass_very_nested": {
			args:        []string{"-test-directory", "tests/subdir"},
			expectedOut: "1 passed, 0 failed.",
			code:        0,
		},
		"simple_pass_very_nested_alternate": {
			override:    "simple_pass_very_nested",
			args:        []string{"-test-directory", "./tests/subdir"},
			expectedOut: "1 passed, 0 failed.",
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
			expectedOut: "1 passed, 0 failed.",
			code:        0,
		},
		"pass_with_outputs": {
			expectedOut: "1 passed, 0 failed.",
			code:        0,
		},
		"pass_with_variables": {
			expectedOut: "2 passed, 0 failed.",
			code:        0,
		},
		"plan_then_apply": {
			expectedOut: "2 passed, 0 failed.",
			code:        0,
		},
		"expect_failures_checks": {
			expectedOut: "1 passed, 0 failed.",
			code:        0,
		},
		"expect_failures_inputs": {
			expectedOut: "1 passed, 0 failed.",
			code:        0,
		},
		"expect_failures_outputs": {
			expectedOut: "1 passed, 0 failed.",
			code:        0,
		},
		"expect_failures_resources": {
			expectedOut: "1 passed, 0 failed.",
			code:        0,
		},
		"multiple_files": {
			expectedOut: "2 passed, 0 failed",
			code:        0,
		},
		"multiple_files_with_filter": {
			override:    "multiple_files",
			args:        []string{"-filter=one.tftest.hcl"},
			expectedOut: "1 passed, 0 failed",
			code:        0,
		},
		"variables": {
			expectedOut: "2 passed, 0 failed",
			code:        0,
		},
		"variables_overridden": {
			override:    "variables",
			args:        []string{"-var=input=foo"},
			expectedOut: "1 passed, 1 failed",
			expectedErr: []string{`invalid value`},
			code:        1,
		},
		"simple_fail": {
			expectedOut: "0 passed, 1 failed.",
			expectedErr: []string{"invalid value"},
			code:        1,
		},
		"custom_condition_checks": {
			expectedOut: "0 passed, 1 failed.",
			expectedErr: []string{"this really should fail"},
			code:        1,
		},
		"custom_condition_inputs": {
			expectedOut: "0 passed, 1 failed.",
			expectedErr: []string{"this should definitely fail"},
			code:        1,
		},
		"custom_condition_outputs": {
			expectedOut: "0 passed, 1 failed.",
			expectedErr: []string{"this should fail"},
			code:        1,
		},
		"custom_condition_resources": {
			expectedOut: "0 passed, 1 failed.",
			expectedErr: []string{"this really should fail"},
			code:        1,
		},
		"no_providers_in_main": {
			expectedOut: "1 passed, 0 failed",
			code:        0,
		},
		"default_variables": {
			expectedOut: "1 passed, 0 failed.",
			code:        0,
		},
		"undefined_variables": {
			expectedOut: "1 passed, 0 failed.",
			code:        0,
		},
		"shared_state": {
			expectedOut: "2 passed, 0 failed.",
			code:        0,
		},
		"shared_state_object": {
			expectedOut: "2 passed, 0 failed.",
			code:        0,
		},
		"variable_references": {
			expectedOut: "2 passed, 0 failed.",
			args:        []string{"-var=global=\"triple\""},
			code:        0,
		},
		"unreferenced_global_variable": {
			override:    "variable_references",
			expectedOut: "2 passed, 0 failed.",
			// The other variable shouldn't pass validation, but it won't be
			// referenced anywhere so should just be ignored.
			args: []string{"-var=global=\"triple\"", "-var=other=bad"},
			code: 0,
		},
		"variables_types": {
			expectedOut: "1 passed, 0 failed.",
			args:        []string{"-var=number_input=0", "-var=string_input=Hello, world!", "-var=list_input=[\"Hello\",\"world\"]"},
			code:        0,
		},
		"null-outputs": {
			expectedOut: "2 passed, 0 failed.",
			code:        0,
		},
		"destroy_fail": {
			expectedOut:           "1 passed, 0 failed.",
			expectedErr:           []string{`Terraform left the following resources in state`},
			code:                  1,
			expectedResourceCount: 1,
		},
		"default_optional_values": {
			expectedOut: "4 passed, 0 failed.",
			code:        0,
		},
		"tfvars_in_test_dir": {
			expectedOut: "2 passed, 0 failed.",
			code:        0,
		},
		"auto_tfvars_in_test_dir": {
			override:    "tfvars_in_test_dir",
			args:        []string{"-test-directory=alternate"},
			expectedOut: "2 passed, 0 failed.",
			code:        0,
		},
		"functions_available": {
			expectedOut: "2 passed, 0 failed.",
			code:        0,
		},
		"mocking": {
			expectedOut: "6 passed, 0 failed.",
			code:        0,
		},
		"mocking-invalid": {
			expectedErr: []string{"Invalid outputs attribute"},
			initCode:    1,
		},
		"dangling_data_block": {
			expectedOut: "2 passed, 0 failed.",
			code:        0,
		},
		"skip_destroy_on_empty": {
			expectedOut: "3 passed, 0 failed.",
			code:        0,
		},
		"empty_module_with_output": {
			expectedOut: "1 passed, 0 failed.",
			code:        0,
		},
		"global_var_refs": {
			expectedOut: "2 failed, 1 skipped.",
			expectedErr: []string{"The input variable \"env_var_input\" is not available to the current context", "The input variable \"setup\" is not available to the current context"},
			code:        1,
		},
		"global_var_ref_in_suite_var": {
			expectedOut: "1 passed, 0 failed.",
			code:        0,
		},
	}
	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			if tc.skip {
				t.Skip()
			}

			file := name
			if len(tc.override) > 0 {
				file = tc.override
			}

			td := t.TempDir()
			testCopyDir(t, testFixturePath(path.Join("test", file)), td)
			defer testChdir(t, td)()

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

			if code := init.Run(nil); code != tc.initCode {
				t.Fatalf("expected status code %d but got %d: %s", tc.initCode, code, ui.ErrorWriter)
			}

			if tc.initCode > 0 {
				// Then we don't expect the init step to succeed. So we'll check
				// the init output for our expected error messages and outputs.

				stdout, stderr := ui.ErrorWriter.String(), ui.ErrorWriter.String()

				if !strings.Contains(stdout, tc.expectedOut) {
					t.Errorf("output didn't contain expected string:\n\n%s", stdout)
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

			c := &TestCommand{
				Meta: meta,
			}

			code := c.Run(tc.args)
			output := done(t)

			if code != tc.code {
				t.Errorf("expected status code %d but got %d:\n\n%s", tc.code, code, output.All())
			}

			if !strings.Contains(output.Stdout(), tc.expectedOut) {
				t.Errorf("output didn't contain expected string:\n\n%s", output.Stdout())
			}

			if len(tc.expectedErr) > 0 {
				for _, expectedErr := range tc.expectedErr {
					if !strings.Contains(output.Stderr(), expectedErr) {
						t.Errorf("error didn't contain expected string:\n\n%s", output.Stderr())
					}
				}
			} else if output.Stderr() != "" {
				t.Errorf("unexpected stderr output\n%s", output.Stderr())
			}

			if provider.ResourceCount() != tc.expectedResourceCount {
				t.Errorf("should have left %d resources on completion but left %v", tc.expectedResourceCount, provider.ResourceString())
			}
		})
	}
}

func TestTest_Interrupt(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath(path.Join("test", "with_interrupt")), td)
	defer testChdir(t, td)()

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

func TestTest_InterruptSkipsRemaining(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath(path.Join("test", "with_interrupt_and_additional_file")), td)
	defer testChdir(t, td)()

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
	defer testChdir(t, td)()

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
	defer testChdir(t, td)()

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

func TestTest_ModuleDependencies(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath(path.Join("test", "with_setup_module")), td)
	defer testChdir(t, td)()

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
	defer testChdir(t, td)()

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
	defer testChdir(t, td)()

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
    }

Plan: 1 to add, 0 to change, 0 to destroy.

  run "apply_test_resource"... pass

# test_resource.foo:
resource "test_resource" "foo" {
    destroy_fail = false
    id           = "constant_value"
    value        = "bar"
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
			defer testChdir(t, td)()

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

			if diff := cmp.Diff(actualOut, expectedOut); len(diff) > 0 {
				t.Errorf("output didn't match expected:\nexpected:\n%s\nactual:\n%s\ndiff:\n%s", expectedOut, actualOut, diff)
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
	defer testChdir(t, td)()

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
	defer testChdir(t, td)()

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
}

  run "initial_apply"... pass

# test_resource.resource:
resource "test_resource" "resource" {
    destroy_fail = false
    id           = "598318e0"
    value        = "start"
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
        # (1 unchanged attribute hidden)
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
        # (1 unchanged attribute hidden)
    }

Plan: 0 to add, 1 to change, 0 to destroy.

main.tftest.hcl... tearing down
main.tftest.hcl... pass

Success! 5 passed, 0 failed.
`

	actual := output.All()

	if diff := cmp.Diff(actual, expected); len(diff) > 0 {
		t.Errorf("output didn't match expected:\nexpected:\n%s\nactual:\n%s\ndiff:\n%s", expected, actual, diff)
	}

	if provider.ResourceCount() > 0 {
		t.Errorf("should have deleted all resources on completion but left %v", provider.ResourceString())
	}
}

func TestTest_OnlyExternalModules(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath(path.Join("test", "only_modules")), td)
	defer testChdir(t, td)()

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

	actual := output.All()

	if diff := cmp.Diff(actual, expected); len(diff) > 0 {
		t.Errorf("output didn't match expected:\nexpected:\n%s\nactual:\n%s\ndiff:\n%s", expected, actual, diff)
	}

	if provider.ResourceCount() > 0 {
		t.Errorf("should have deleted all resources on completion but left %v", provider.ResourceString())
	}
}

func TestTest_PartialUpdates(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath(path.Join("test", "partial_updates")), td)
	defer testChdir(t, td)()

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

func TestTest_BadReferences(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath(path.Join("test", "bad-references")), td)
	defer testChdir(t, td)()

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

Warning: Value for undeclared variable

  on main.tftest.hcl line 17, in run "test":
  17:     input_three = run.madeup.response

The module under test does not declare a variable named "input_three", but it
is declared in run block "test".

  run "finalise"... skip
main.tftest.hcl... tearing down
main.tftest.hcl... fail
providers.tftest.hcl... in progress
  run "test"... fail
providers.tftest.hcl... tearing down
providers.tftest.hcl... fail

Failure! 1 passed, 2 failed, 1 skipped.
`
	actualOut := output.Stdout()
	if diff := cmp.Diff(actualOut, expectedOut); len(diff) > 0 {
		t.Errorf("output didn't match expected:\nexpected:\n%s\nactual:\n%s\ndiff:\n%s", expectedOut, actualOut, diff)
	}

	expectedErr := `
Error: Reference to unavailable variable

  on main.tftest.hcl line 15, in run "test":
  15:     input_one = var.notreal

The input variable "notreal" is not available to the current context. Within
the variables block of a run block you can only reference variables defined
at the file or global levels; within the variables block of a suite you can
only reference variables defined at the global levels.

Error: Reference to unavailable run block

  on main.tftest.hcl line 16, in run "test":
  16:     input_two = run.finalise.response

The run block "finalise" has not executed yet. You can only reference run
blocks that are in the same test file and will execute before the current run
block.

Error: Reference to unknown run block

  on main.tftest.hcl line 17, in run "test":
  17:     input_three = run.madeup.response

The run block "madeup" does not exist within this test file. You can only
reference run blocks that are in the same test file and will execute before
the current run block.

Error: Reference to unavailable variable

  on providers.tftest.hcl line 3, in provider "test":
   3:   resource_prefix = var.default

The input variable "default" is not available to the current context. Within
the variables block of a run block you can only reference variables defined
at the file or global levels; within the variables block of a suite you can
only reference variables defined at the global levels.
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
	defer testChdir(t, td)()

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
	defer testChdir(t, td)()

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
	defer testChdir(t, td)()

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

Failure! 1 passed, 3 failed.
`
	actualOut := output.Stdout()
	if diff := cmp.Diff(actualOut, expectedOut); len(diff) > 0 {
		t.Errorf("output didn't match expected:\nexpected:\n%s\nactual:\n%s\ndiff:\n%s", expectedOut, actualOut, diff)
	}

	expectedErr := `
Error: Invalid value for variable

  on main.tf line 2:
   2: variable "input" {
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
    │ output.null_output is null
    │ test_resource.resource.value is "bar"

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

Condition expression could not be evaluated at this time. This means you have
executed a %s block with %s and one of the values your
condition depended on is not known until after the plan has been applied.
Either remove this value from your condition, or execute an %s command
from this %s block.
`, "`run`", "`command = plan`", "`apply`", "`run`"),
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
			defer testChdir(t, td)()

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
	defer testChdir(t, td)()

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
  run "test"... pass

Warning: Sensitive metadata on variable lost

  on main.tftest.hcl line 13, in run "test":
  13:     password = run.setup.password

The input variable is marked as sensitive, while the receiving configuration
is not. The underlying sensitive information may be exposed when var.password
is referenced. Mark the variable block in the configuration as sensitive to
resolve this warning.

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

// This test takes around 10 seconds to complete, as we're testing the progress
// updates that are printed every 2 seconds. Sorry!
func TestTest_LongRunningTest(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath(path.Join("test", "long_running")), td)
	defer testChdir(t, td)()

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
	defer testChdir(t, td)()

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
	defer testChdir(t, td)()

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

func TestTest_RunBlocksInProviders(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath(path.Join("test", "provider_runs")), td)
	defer testChdir(t, td)()

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
	defer testChdir(t, td)()

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

	test := &TestCommand{
		Meta: meta,
	}

	code := test.Run([]string{"-no-color"})
	output := done(t)

	if code != 1 {
		t.Errorf("expected status code 1 but got %d", code)
	}

	expectedOut := `missing_run_block.tftest.hcl... in progress
  run "main"... fail
missing_run_block.tftest.hcl... tearing down
missing_run_block.tftest.hcl... fail
unavailable_run_block.tftest.hcl... in progress
  run "main"... fail
unavailable_run_block.tftest.hcl... tearing down
unavailable_run_block.tftest.hcl... fail
unused_provider.tftest.hcl... in progress
  run "main"... pass
unused_provider.tftest.hcl... tearing down
unused_provider.tftest.hcl... pass

Failure! 1 passed, 2 failed.
`
	actualOut := output.Stdout()
	if diff := cmp.Diff(actualOut, expectedOut); len(diff) > 0 {
		t.Errorf("output didn't match expected:\nexpected:\n%s\nactual:\n%s\ndiff:\n%s", expectedOut, actualOut, diff)
	}

	expectedErr := `
Error: Reference to unknown run block

  on missing_run_block.tftest.hcl line 2, in provider "test":
   2:   resource_prefix = run.missing.resource_directory

The run block "missing" does not exist within this test file. You can only
reference run blocks that are in the same test file and will execute before
the provider is required.

Error: Reference to unavailable run block

  on unavailable_run_block.tftest.hcl line 2, in provider "test":
   2:   resource_prefix = run.main.resource_directory

The run block "main" has not executed yet. You can only reference run blocks
that are in the same test file and will execute before the provider is
required.
`
	actualErr := output.Stderr()
	if diff := cmp.Diff(actualErr, expectedErr); len(diff) > 0 {
		t.Errorf("output didn't match expected:\nexpected:\n%s\nactual:\n%s\ndiff:\n%s", expectedErr, actualErr, diff)
	}

	if provider.ResourceCount() > 0 {
		t.Errorf("should have deleted all resources on completion but left %v", provider.ResourceString())
	}
}
