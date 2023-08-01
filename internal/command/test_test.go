package command

import (
	"path"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/mitchellh/cli"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	testing_command "github.com/hashicorp/terraform/internal/command/testing"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/terminal"
)

func TestTest(t *testing.T) {
	tcs := map[string]struct {
		override string
		args     []string
		expected string
		code     int
		skip     bool
	}{
		"simple_pass": {
			expected: "1 passed, 0 failed.",
			code:     0,
		},
		"simple_pass_nested": {
			expected: "1 passed, 0 failed.",
			code:     0,
		},
		"simple_pass_nested_alternate": {
			args:     []string{"-test-directory", "other"},
			expected: "1 passed, 0 failed.",
			code:     0,
		},
		"simple_pass_very_nested": {
			args:     []string{"-test-directory", "tests/subdir"},
			expected: "1 passed, 0 failed.",
			code:     0,
		},
		"simple_pass_very_nested_alternate": {
			override: "simple_pass_very_nested",
			args:     []string{"-test-directory", "./tests/subdir"},
			expected: "1 passed, 0 failed.",
			code:     0,
		},
		"pass_with_locals": {
			expected: "1 passed, 0 failed.",
			code:     0,
		},
		"pass_with_outputs": {
			expected: "1 passed, 0 failed.",
			code:     0,
		},
		"pass_with_variables": {
			expected: "2 passed, 0 failed.",
			code:     0,
		},
		"plan_then_apply": {
			expected: "2 passed, 0 failed.",
			code:     0,
		},
		"expect_failures_checks": {
			expected: "1 passed, 0 failed.",
			code:     0,
		},
		"expect_failures_inputs": {
			expected: "1 passed, 0 failed.",
			code:     0,
		},
		"expect_failures_outputs": {
			expected: "1 passed, 0 failed.",
			code:     0,
		},
		"expect_failures_resources": {
			expected: "1 passed, 0 failed.",
			code:     0,
		},
		"multiple_files": {
			expected: "2 passed, 0 failed",
			code:     0,
		},
		"multiple_files_with_filter": {
			override: "multiple_files",
			args:     []string{"-filter=one.tftest.hcl"},
			expected: "1 passed, 0 failed",
			code:     0,
		},
		"variables": {
			expected: "2 passed, 0 failed",
			code:     0,
		},
		"variables_overridden": {
			override: "variables",
			args:     []string{"-var=input=foo"},
			expected: "1 passed, 1 failed",
			code:     1,
		},
		"simple_fail": {
			expected: "0 passed, 1 failed.",
			code:     1,
		},
		"custom_condition_checks": {
			expected: "0 passed, 1 failed.",
			code:     1,
		},
		"custom_condition_inputs": {
			expected: "0 passed, 1 failed.",
			code:     1,
		},
		"custom_condition_outputs": {
			expected: "0 passed, 1 failed.",
			code:     1,
		},
		"custom_condition_resources": {
			expected: "0 passed, 1 failed.",
			code:     1,
		},
		"no_providers_in_main": {
			expected: "1 passed, 0 failed",
			code:     0,
		},
		"default_variables": {
			expected: "1 passed, 0 failed.",
			code:     0,
		},
		"undefined_variables": {
			expected: "1 passed, 0 failed.",
			code:     0,
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
			view, done := testView(t)

			c := &TestCommand{
				Meta: Meta{
					testingOverrides: metaOverridesForProvider(provider.Provider),
					View:             view,
				},
			}

			code := c.Run(tc.args)
			output := done(t)

			if code != tc.code {
				t.Errorf("expected status code %d but got %d", tc.code, code)
			}

			if !strings.Contains(output.Stdout(), tc.expected) {
				t.Errorf("output didn't contain expected string:\n\n%s", output.All())
			}

			if provider.ResourceCount() > 0 {
				t.Errorf("should have deleted all resources on completion but left %v", provider.ResourceString())
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

	expectedOut := `main.tftest.hcl... fail
  run "test"... skip

Failure! 0 passed, 0 failed, 1 skipped.
`

	expectedErr := `
Error: No value for required variable

  on main.tf line 2:
   2: variable "input" {

The root module input variable "input" is not set, and has no default value.
Use a -var or -var-file command line argument to provide a value for this
variable.
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

	expected := `main.tftest.hcl... pass
  run "validate_test_resource"... pass

Terraform used the selected providers to generate the following execution
plan. Resource actions are indicated with the following symbols:
  + create

Terraform will perform the following actions:

  # test_resource.foo will be created
  + resource "test_resource" "foo" {
      + id    = "constant_value"
      + value = "bar"
    }

Plan: 1 to add, 0 to change, 0 to destroy.
  run "validate_test_resource"... pass
# test_resource.foo:
resource "test_resource" "foo" {
    id    = "constant_value"
    value = "bar"
}

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
			expectedOut: `main.tftest.hcl... fail
  run "invalid"... fail

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
			expectedOut: `main.tftest.hcl... fail
  run "invalid"... fail
  run "test"... skip

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
			expectedOut: `main.tftest.hcl... fail
  run "passes_validation"... skip

Failure! 0 passed, 0 failed, 1 skipped.
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
			expectedOut: `main.tftest.hcl... fail
  run "passes_validation"... fail

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
			expectedOut: `main.tftest.hcl... fail
  run "passes_validation_primary"... pass
  run "passes_validation_secondary"... fail

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
