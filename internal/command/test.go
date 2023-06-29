package command

import (
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configload"
	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type TestCommand struct {
	Meta

	loader *configload.Loader
}

func (c *TestCommand) Help() string {
	helpText := `
Usage: terraform [global options] test [options]

  Executes automated integration tests against the current Terraform 
  configuration.

  Terraform will search for .tftest files within the current configuration and 
  testing directories. Terraform will then execute the testing run blocks within
  any testing files in order, and verify conditional checks and assertions 
  against the created infrastructure. 

  This command creates real infrastructure and will attempt to clean up the
  testing infrastructure on completion. Monitor the output carefully to ensure
  this cleanup process is successful.

Options:

  TODO: implement optional arguments.
`
	return strings.TrimSpace(helpText)
}

func (c *TestCommand) Synopsis() string {
	return "Execute integration tests for Terraform modules"
}

func (c *TestCommand) Run(rawArgs []string) int {
	var diags tfdiags.Diagnostics

	common, _ := arguments.ParseView(rawArgs)
	c.View.Configure(common)

	view := views.NewTest(arguments.ViewHuman, c.View)

	loader, err := c.initConfigLoader()
	diags = diags.Append(err)
	if err != nil {
		view.Diagnostics(nil, nil, diags)
		return 1
	}
	c.loader = loader

	config, configDiags := loader.LoadConfigWithTests(".", "tests")
	diags = diags.Append(configDiags)
	if configDiags.HasErrors() {
		view.Diagnostics(nil, nil, diags)
		return 1
	}

	suite := moduletest.Suite{
		Files: func() map[string]*moduletest.File {
			files := make(map[string]*moduletest.File)
			for name, file := range config.Module.Tests {
				var runs []*moduletest.Run
				for _, run := range file.Runs {
					runs = append(runs, &moduletest.Run{
						Config: run,
						Name:   run.Name,
					})
				}
				files[name] = &moduletest.File{
					Config: file,
					Name:   name,
					Runs:   runs,
				}
			}
			return files
		}(),
	}

	view.Abstract(&suite)
	c.ExecuteTestSuite(&suite, config, view)
	view.Conclusion(&suite)

	if suite.Status != moduletest.Pass {
		return 1
	}
	return 0
}

func (c *TestCommand) ExecuteTestSuite(suite *moduletest.Suite, config *configs.Config, view views.Test) {
	var diags tfdiags.Diagnostics

	opts, err := c.contextOpts()
	diags = diags.Append(err)
	if err != nil {
		suite.Status = suite.Status.Merge(moduletest.Error)
		view.Diagnostics(nil, nil, diags)
		return
	}

	ctx, ctxDiags := terraform.NewContext(opts)
	diags = diags.Append(ctxDiags)
	if ctxDiags.HasErrors() {
		suite.Status = suite.Status.Merge(moduletest.Error)
		view.Diagnostics(nil, nil, diags)
		return
	}
	view.Diagnostics(nil, nil, diags) // Print out any warnings from the setup.

	var files []string
	for name := range suite.Files {
		files = append(files, name)
	}
	sort.Strings(files) // execute the files in alphabetical order

	suite.Status = moduletest.Pass
	for _, name := range files {
		file := suite.Files[name]
		c.ExecuteTestFile(ctx, file, config, view)

		suite.Status = suite.Status.Merge(file.Status)
	}
}

func (c *TestCommand) ExecuteTestFile(ctx *terraform.Context, file *moduletest.File, config *configs.Config, view views.Test) {
	var diags tfdiags.Diagnostics

	globalVariableValues, diags := c.CollectDefaultVariables(file.Config.Variables, config)
	if diags.HasErrors() {
		file.Status = file.Status.Merge(moduletest.Error)
		view.File(file)
		view.Diagnostics(nil, file, diags)
		return
	}

	state := states.NewState()
	defer func() {

		// Whatever happens, at the end of this test we don't want to leave
		// active resources behind. So we'll do a destroy action against the
		// state in a deferred function.

		plan, planDiags := ctx.Plan(config, state, &terraform.PlanOpts{
			Mode:         plans.DestroyMode,
			SetVariables: globalVariableValues,
		})
		if planDiags.HasErrors() {
			// This is bad, we need to tell the user that we couldn't clean up
			// and they need to go and manually delete some resources.
			view.DestroySummary(planDiags, file, state)
			return
		}
		view.Diagnostics(nil, file, planDiags) // Print out any warnings from the destroy plan.

		finalState, applyDiags := ctx.Apply(plan, config)
		view.DestroySummary(applyDiags, file, finalState)
	}()

	file.Status = file.Status.Merge(moduletest.Pass)
	for _, run := range file.Runs {
		if file.Status == moduletest.Error {
			run.Status = moduletest.Skip
			continue
		}

		state = c.ExecuteTestRun(ctx, run, state, config, globalVariableValues)
		file.Status = file.Status.Merge(run.Status)
	}

	view.File(file)
	view.Diagnostics(nil, file, diags)

	for _, run := range file.Runs {
		view.Run(run, file)
	}
}

func (c *TestCommand) ExecuteTestRun(ctx *terraform.Context, run *moduletest.Run, state *states.State, config *configs.Config, defaults terraform.InputValues) *states.State {

	var targets []addrs.Targetable
	for _, target := range run.Config.Options.Target {
		addr, diags := addrs.ParseTarget(target)
		run.Diagnostics = run.Diagnostics.Append(diags)
		if diags.HasErrors() {
			run.Status = moduletest.Error
			return state
		}

		targets = append(targets, addr.Subject)
	}

	var replaces []addrs.AbsResourceInstance
	for _, replace := range run.Config.Options.Replace {
		addr, diags := addrs.ParseAbsResourceInstance(replace)
		run.Diagnostics = run.Diagnostics.Append(diags)
		if diags.HasErrors() {
			run.Status = moduletest.Error
			return state
		}

		if addr.Resource.Resource.Mode != addrs.ManagedResourceMode {
			run.Diagnostics = run.Diagnostics.Append(hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "can only target managed resources for forced replacements",
				Detail:   addr.String(),
				Subject:  replace.SourceRange().Ptr(),
			})
			return state
		}

		replaces = append(replaces, addr)
	}

	variables, diags := c.OverrideDefaultVariables(run.Config.Variables, config, defaults)
	run.Diagnostics = run.Diagnostics.Append(diags)
	if diags.HasErrors() {
		run.Status = moduletest.Error
		return state
	}

	var references []*addrs.Reference
	for _, assert := range run.Config.CheckRules {
		for _, variable := range assert.Condition.Variables() {
			reference, diags := addrs.ParseRef(variable)
			run.Diagnostics = run.Diagnostics.Append(diags)
			references = append(references, reference)
		}
	}
	if run.Diagnostics.HasErrors() {
		run.Status = moduletest.Error
		return state
	}

	plan, diags := ctx.Plan(config, state, &terraform.PlanOpts{
		Mode: func() plans.Mode {
			switch run.Config.Options.Mode {
			case configs.RefreshOnlyTestMode:
				return plans.RefreshOnlyMode
			default:
				return plans.NormalMode
			}
		}(),
		SetVariables:       variables,
		Targets:            targets,
		ForceReplace:       replaces,
		SkipRefresh:        !run.Config.Options.Refresh,
		ExternalReferences: references,
	})
	run.Diagnostics = run.Diagnostics.Append(diags)
	if diags.HasErrors() {
		run.Status = moduletest.Error
		return state
	}

	if run.Config.Command == configs.ApplyTestCommand {
		state, diags = ctx.Apply(plan, config)
		run.Diagnostics = run.Diagnostics.Append(diags)
		if diags.HasErrors() {
			run.Status = moduletest.Error
			return state
		}

		ctx.TestContext(config, state, plan, variables).EvaluateAgainstState(run)
		return state
	}

	ctx.TestContext(config, plan.PlannedState, plan, variables).EvaluateAgainstPlan(run)
	return state
}

func (c *TestCommand) CollectDefaultVariables(exprs map[string]hcl.Expression, config *configs.Config) (terraform.InputValues, tfdiags.Diagnostics) {
	unparsed := make(map[string]backend.UnparsedVariableValue)
	for key, value := range exprs {
		unparsed[key] = unparsedVariableValueExpression{
			expr:       value,
			sourceType: terraform.ValueFromConfig,
		}
	}
	return backend.ParseVariableValues(unparsed, config.Module.Variables)
}

func (c *TestCommand) OverrideDefaultVariables(exprs map[string]hcl.Expression, config *configs.Config, existing terraform.InputValues) (terraform.InputValues, tfdiags.Diagnostics) {
	if len(exprs) == 0 {
		return existing, nil
	}

	decls := make(map[string]*configs.Variable)
	unparsed := make(map[string]backend.UnparsedVariableValue)
	for name, variable := range exprs {

		if config, ok := config.Module.Variables[name]; ok {
			decls[name] = config
		}

		unparsed[name] = unparsedVariableValueExpression{
			expr:       variable,
			sourceType: terraform.ValueFromConfig,
		}
	}

	overrides, diags := backend.ParseVariableValues(unparsed, decls)
	values := make(terraform.InputValues)
	for name, value := range existing {
		if override, ok := overrides[name]; ok {
			values[name] = override
			continue
		}
		values[name] = value
	}
	return values, diags
}
