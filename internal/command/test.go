package command

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// TestCommand is the implementation of "terraform test".
type TestCommand struct {
	Meta
}

func (c *TestCommand) Run(rawArgs []string) int {
	// Parse and apply global view arguments
	common, rawArgs := arguments.ParseView(rawArgs)
	c.View.Configure(common)

	args, diags := arguments.ParseTest(rawArgs)
	view := views.NewTest(c.View, args.Output)
	if diags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	diags = diags.Append(tfdiags.Sourceless(
		tfdiags.Warning,
		`The "terraform test" command is experimental`,
		"We'd like to invite adventurous module authors to write integration tests for their modules using this command, but all of the behaviors of this command are currently experimental and may change based on feedback.\n\nFor more information on the testing experiment, including ongoing research goals and avenues for feedback, see:\n    https://www.terraform.io/docs/language/modules/testing-experiment.html",
	))
	view.Diagnostics(diags)
	diags = nil // reset because we've already emitted everything from so far

	ctx, cancel := c.InterruptibleContext()
	defer cancel()

	// NOTE: We intentionally don't add resultDiags to our main diags because
	// we consider those ones to be a part of the test result report, to be
	// printed by the view.Results method.
	results, resultDiags := c.run(ctx, args)
	moreDiags := view.Results(results, resultDiags, c.configSources())
	diags = diags.Append(moreDiags)
	view.Diagnostics(diags)

	if diags.HasErrors() {
		return 1
	}
	return 0
}

func (c *TestCommand) run(ctx context.Context, args arguments.Test) (results map[addrs.ModuleTestScenario]*moduletest.ScenarioResult, diags tfdiags.Diagnostics) {
	scenarios, diags := moduletest.LoadScenarios(".")
	if diags.HasErrors() {
		return nil, diags
	}

	if len(scenarios) == 0 {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"No test scenarios for this module",
			"Terraform did not find any test scenarios for the current module.\n\nTest scenarios are Terraform configurations placed in subdirectories of a \"tests\" subdirectory under the current working directory. Create at least one test scenario configuration before running tests.",
		))
		return nil, diags
	}

	results = make(map[addrs.ModuleTestScenario]*moduletest.ScenarioResult, len(scenarios))

	mainConfigLoader, err := c.initConfigLoader()
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to initialize configuration loader",
			fmt.Sprintf("Unable to initialize the main configuration loader: %s.", err),
		))
		return results, diags
	}

	// TODO: Figure out what this actually needs to contain so that the
	// test runner can get access to development environment resources it
	// needs, such as provider plugin factories.
	env := &moduletest.RunEnvironment{
		ConfigParser:       mainConfigLoader.Parser(),
		Services:           c.Services,
		ExperimentsAllowed: c.AllowExperimentalFeatures,
	}

	// TODO: Consider running scenarios concurrently. Probably that would need
	// to be an opt-in/opt-out in a scenario configuration file so that
	// authors will be able to test modules that interact with "singleton"
	// objects that would not be possible to use concurrently from multiple
	// test configurations.

	// We're intentionally letting Go's pseudorandom map iteration order
	// happen here, because test scenarios are supposed to all be independent
	// of one another and so randomizing the execution order should draw
	// attention to accidental dependencies.
	for _, scenario := range scenarios {
		if ctx.Err() != nil {
			// If the context has already failed in some way then we'll
			// halt early and report whatever's already happened.
			// (Something downstream should have already emitted the error
			// into a diagnostics somewhere in the results.)
			break
		}

		result, moreDiags := scenario.RunTests(ctx, env)
		diags = diags.Append(moreDiags)
		results[scenario.Addr()] = result

		// TODO: If result.FinalState is non-empty, write it out as a local
		// state file in the scenario directory and emit an error to tell
		// the user it's there.
	}

	return results, diags
}

func (c *TestCommand) Help() string {
	helpText := `
Usage: terraform test [options]

  This is an experimental command to help with automated integration
  testing of shared modules. The usage and behavior of this command is
  likely to change in breaking ways in subsequent releases, as we
  are currently using this command primarily for research purposes.

  In its current experimental form, "test" will look under the current
  working directory for a subdirectory called "tests", and then within
  that directory search for one or more subdirectories that contain
  ".tf" or ".tf.json" files. For any that it finds, it will perform
  Terraform operations similar to the following sequence of commands
  in each of those directories:
      terraform validate
      terraform apply
      terraform destroy

  The test configurations should not declare any input variables and
  should at least contain a call to the module being tested, which
  will always be available at the path ../.. due to the expected
  filesystem layout.

  The tests are considered to be successful if all of the above steps
  succeed.

  Test configurations may optionally include uses of the special
  built-in test provider terraform.io/builtin/test, which allows
  writing explicit test assertions which must also all pass in order
  for the test run to be considered successful.

  This initial implementation is intended as a minimally-viable
  product to use for further research and experimentation, and in
  particular it currently lacks the following capabilities that we
  expect to consider in later iterations, based on feedback:
    - Testing of subsequent updates to existing infrastructure,
      where currently it only supports initial creation and
      then destruction.
    - Testing top-level modules that are intended to be used for
      "real" environments, which typically have hard-coded values
      that don't permit creating a separate "copy" for testing.
    - Some sort of support for unit test runs that don't interact
      with remote systems at all, e.g. for use in checking pull
      requests from untrusted contributors.

  In the meantime, we'd like to hear feedback from module authors
  who have tried writing some experimental tests for their modules
  about what sorts of tests you were able to write, what sorts of
  tests you weren't able to write, and any tests that you were
  able to write but that were difficult to model in some way.

Options:

  -compact-warnings  Use a more compact representation for warnings, if
                     this command produces only warnings and no errors.

  -junit-xml=FILE    In addition to the usual output, also write test
                     results to the given file path in JUnit XML format.
                     This format is commonly supported by CI systems, and
                     they typically expect to be given a filename to search
                     for in the test workspace after the test run finishes.

  -show-all          Force showing all of the test cases, including those
                     that passed or were skipped altogether. Normally the
                     human-oriented output summarizes successful test
                     scenarios to just an aggregate result, making no direct
                     mention of the individual test cases that succeeded.

  -no-color          Don't include virtual terminal formatting sequences in
                     the output.
`
	return strings.TrimSpace(helpText)
}

func (c *TestCommand) Synopsis() string {
	return "Experimental support for module integration testing"
}
