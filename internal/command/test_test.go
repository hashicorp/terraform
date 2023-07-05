package command

import (
	"path"
	"strings"
	"testing"

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
			// TODO(liamcervante): Enable this when support for expect_failures
			//   has been added.
			skip: true,
		},
		"expect_failures_inputs": {
			expected: "1 passed, 0 failed.",
			code:     0,
			// TODO(liamcervante): Enable this when support for expect_failures
			//   has been added.
			skip: true,
		},
		"expect_failures_outputs": {
			expected: "1 passed, 0 failed.",
			code:     0,
			// TODO(liamcervante): Enable this when support for expect_failures
			//   has been added.
			skip: true,
		},
		"expect_failures_resources": {
			expected: "1 passed, 0 failed.",
			code:     0,
			// TODO(liamcervante): Enable this when support for expect_failures
			//   has been added.
			skip: true,
		},
		"simple_fail": {
			expected: "0 passed, 1 failed.",
			code:     1,
		},
		"custom_condition_checks": {
			expected: "0 passed, 1 failed.",
			code:     1,
			// TODO(liamcervante): Enable this, at the moment checks aren't
			//   causing the tests to fail when they should. Also, it's not
			//   skipping warnings during the plan when it should.
			skip: true,
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
	}
	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			if tc.skip {
				t.Skip()
			}

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

	if !strings.Contains(output, "Terraform Test Interrupted") {
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
