package command

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/tfdiags"
)

// PlanCommand is a Command implementation that compares a Terraform
// configuration to an actual infrastructure and shows the differences.
type PlanCommand struct {
	Meta
}

func (c *PlanCommand) Run(args []string) int {
	var destroy, refresh, detailed bool
	var outPath string
	var moduleDepth int

	args, err := c.Meta.process(args, true)
	if err != nil {
		return 1
	}

	cmdFlags := c.Meta.flagSet("plan")
	cmdFlags.BoolVar(&destroy, "destroy", false, "destroy")
	cmdFlags.BoolVar(&refresh, "refresh", true, "refresh")
	c.addModuleDepthFlag(cmdFlags, &moduleDepth)
	cmdFlags.StringVar(&outPath, "out", "", "path")
	cmdFlags.IntVar(
		&c.Meta.parallelism, "parallelism", DefaultParallelism, "parallelism")
	cmdFlags.StringVar(&c.Meta.statePath, "state", "", "path")
	cmdFlags.BoolVar(&detailed, "detailed-exitcode", false, "detailed-exitcode")
	cmdFlags.BoolVar(&c.Meta.stateLock, "lock", true, "lock state")
	cmdFlags.DurationVar(&c.Meta.stateLockTimeout, "lock-timeout", 0, "lock timeout")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	configPath, err := ModulePath(cmdFlags.Args())
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	// Check for user-supplied plugin path
	if c.pluginPath, err = c.loadPluginPath(); err != nil {
		c.Ui.Error(fmt.Sprintf("Error loading plugin path: %s", err))
		return 1
	}

	// Check if the path is a plan, which is not permitted
	planFileReader, err := c.PlanFile(configPath)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}
	if planFileReader != nil {
		c.showDiagnostics(tfdiags.Sourceless(
			tfdiags.Error,
			"Invalid configuration directory",
			fmt.Sprintf("Cannot pass a saved plan file to the 'terraform plan' command. To apply a saved plan, use: terraform apply %s", configPath),
		))
		return 1
	}

	var diags tfdiags.Diagnostics

	var backendConfig *configs.Backend
	var configDiags tfdiags.Diagnostics
	backendConfig, configDiags = c.loadBackendConfig(configPath)
	diags = diags.Append(configDiags)
	if configDiags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	// Load the backend
	b, backendDiags := c.Backend(&BackendOpts{
		Config: backendConfig,
	})
	diags = diags.Append(backendDiags)
	if backendDiags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	// Emit any diagnostics we've accumulated before we delegate to the
	// backend, since the backend will handle its own diagnostics internally.
	c.showDiagnostics(diags)
	diags = nil

	// Build the operation
	opReq := c.Operation(b)
	opReq.Destroy = destroy
	opReq.ConfigDir = configPath
	opReq.PlanRefresh = refresh
	opReq.PlanOutPath = outPath
	opReq.PlanRefresh = refresh
	opReq.Type = backend.OperationTypePlan
	opReq.ConfigLoader, err = c.initConfigLoader()
	if err != nil {
		c.showDiagnostics(err)
		return 1
	}

	// c.Backend above has a non-obvious side-effect of also populating
	// c.backendState, which is the state-shaped formulation of the effective
	// backend configuration after evaluation of the backend configuration.
	// We will in turn adapt that to a plans.Backend to include in a plan file
	// if opReq.PlanOutPath was set to a non-empty value above.
	//
	// FIXME: It's ugly to be doing this inline here, but it's also not really
	// clear where would be better to do it. In future we should find a better
	// home for this logic, and ideally also stop depending on the side-effect
	// of c.Backend setting c.backendState.
	{
		// This is not actually a state in the usual sense, but rather a
		// representation of part of the current working directory's
		// "configuration state".
		backendPseudoState := c.backendState
		if backendPseudoState == nil {
			// Should never happen if c.Backend is behaving properly.
			diags = diags.Append(fmt.Errorf("Backend initialization didn't produce resolved configuration (This is a bug in Terraform)"))
			c.showDiagnostics(diags)
			return 1
		}
		var backendForPlan plans.Backend
		backendForPlan.Type = backendPseudoState.Type
		backendForPlan.Workspace = c.Workspace()

		// Configuration is a little more awkward to handle here because it's
		// stored in state as raw JSON but we need it as a plans.DynamicValue
		// to save it in the state. To do that conversion we need to know the
		// configuration schema of the backend.
		configSchema := b.ConfigSchema()
		config, err := backendPseudoState.Config(configSchema)
		if err != nil {
			// This means that the stored settings don't conform to the current
			// schema, which could either be because we're reading something
			// created by an older version that is no longer compatible, or
			// because the user manually tampered with the stored config.
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Invalid backend initialization",
				fmt.Sprintf("The backend configuration for this working directory is not valid: %s.\n\nIf you have recently upgraded Terraform, you may need to re-run \"terraform init\" to re-initialize this working directory.", err),
			))
			c.showDiagnostics(diags)
			return 1
		}
		configForPlan, err := plans.NewDynamicValue(config, configSchema.ImpliedType())
		if err != nil {
			// This should never happen, since we've just decoded this value
			// using the same schema.
			diags = diags.Append(fmt.Errorf("Failed to encode backend configuration to store in plan: %s", err))
			c.showDiagnostics(diags)
			return 1
		}
		backendForPlan.Config = configForPlan
	}

	// Perform the operation
	op, err := c.RunOperation(b, opReq)
	if err != nil {
		c.showDiagnostics(err)
		return 1
	}

	if op.Result != backend.OperationSuccess {
		return op.Result.ExitStatus()
	}
	if detailed && !op.PlanEmpty {
		return 2
	}

	return op.Result.ExitStatus()
}

func (c *PlanCommand) Help() string {
	helpText := `
Usage: terraform plan [options] [DIR-OR-PLAN]

  Generates an execution plan for Terraform.

  This execution plan can be reviewed prior to running apply to get a
  sense for what Terraform will do. Optionally, the plan can be saved to
  a Terraform plan file, and apply can take this plan file to execute
  this plan exactly.

  If a saved plan is passed as an argument, this command will output
  the saved plan contents. It will not modify the given plan.

Options:

  -destroy            If set, a plan will be generated to destroy all resources
                      managed by the given configuration and state.

  -detailed-exitcode  Return detailed exit codes when the command exits. This
                      will change the meaning of exit codes to:
                      0 - Succeeded, diff is empty (no changes)
                      1 - Errored
                      2 - Succeeded, there is a diff

  -input=true         Ask for input for variables if not directly set.

  -lock=true          Lock the state file when locking is supported.

  -lock-timeout=0s    Duration to retry a state lock.

  -module-depth=n     Specifies the depth of modules to show in the output.
                      This does not affect the plan itself, only the output
                      shown. By default, this is -1, which will expand all.

  -no-color           If specified, output won't contain any color.

  -out=path           Write a plan file to the given path. This can be used as
                      input to the "apply" command.

  -parallelism=n      Limit the number of concurrent operations. Defaults to 10.

  -refresh=true       Update state prior to checking for differences.

  -state=statefile    Path to a Terraform state file to use to look
                      up Terraform-managed resources. By default it will
                      use the state "terraform.tfstate" if it exists.

  -target=resource    Resource to target. Operation will be limited to this
                      resource and its dependencies. This flag can be used
                      multiple times.

  -var 'foo=bar'      Set a variable in the Terraform configuration. This
                      flag can be set multiple times.

  -var-file=foo       Set variables in the Terraform configuration from
                      a file. If "terraform.tfvars" or any ".auto.tfvars"
                      files are present, they will be automatically loaded.
`
	return strings.TrimSpace(helpText)
}

func (c *PlanCommand) Synopsis() string {
	return "Generate and show an execution plan"
}
