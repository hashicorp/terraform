package command

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform/internal/checks"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/moduletest/testconfigs"
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

	ctx, cancel := c.InterruptibleContext()
	defer cancel()

	results, moreDiags := c.run(ctx, args)
	diags = diags.Append(moreDiags)

	initFailed := diags.HasErrors()
	view.Diagnostics(diags)
	diags = view.Results(results)
	resultsFailed := diags.HasErrors()
	view.Diagnostics(diags) // possible additional errors from saving the results

	var testsFailed bool
	for _, scenario := range results {
		if scenario.Status == checks.StatusFail || scenario.Status == checks.StatusError {
			testsFailed = true
			break
		}
	}

	// Lots of things can possibly have failed
	if initFailed || resultsFailed || testsFailed {
		return 1
	}
	return 0
}

func (c *TestCommand) run(ctx context.Context, args arguments.Test) (results map[string]*moduletest.ScenarioResult, diags tfdiags.Diagnostics) {
	suite, diags := testconfigs.LoadSuiteForModule(".")
	if diags.HasErrors() {
		return nil, diags
	}

	// TEMP: For now we'll just stub out fake results for all of the scenarios
	// as a placeholder. In a real system (as opposed to this prototype)
	// it should be the moduletest package's responsiblity to run tests and
	// return results for them.
	ret := make(map[string]*moduletest.ScenarioResult, len(suite.Scenarios))
	for name, scenario := range suite.Scenarios {
		ret[name] = &moduletest.ScenarioResult{
			Name:   name,
			Status: checks.StatusUnknown,
		}
		for _, stepName := range scenario.StepsOrder {
			ret[name].Steps = append(ret[name].Steps, moduletest.StepResult{
				Name:   stepName,
				Status: checks.StatusUnknown,
			})
		}
	}

	return ret, diags
}

func (c *TestCommand) Help() string {
	helpText := `
Usage: terraform test [options]

  This is an experimental command to help with automated integration
  testing of shared modules. The usage and behavior of this command is
  likely to change in breaking ways in subsequent releases, as we
  are currently using this command primarily for research purposes.

Options:

  -compact-warnings  Use a more compact representation for warnings, if
                     this command produces only warnings and no errors.

  -junit-xml=FILE    In addition to the usual output, also write test
                     results to the given file path in JUnit XML format.
                     This format is commonly supported by CI systems, and
                     they typically expect to be given a filename to search
                     for in the test workspace after the test run finishes.

  -no-color          Don't include virtual terminal formatting sequences in
                     the output.
`
	return strings.TrimSpace(helpText)
}

func (c *TestCommand) Synopsis() string {
	return "Experimental support for module integration testing"
}
