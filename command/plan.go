package command

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/config/module"
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

	// Check if the path is a plan
	plan, err := c.Plan(configPath)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}
	if plan != nil {
		// Disable refreshing no matter what since we only want to show the plan
		refresh = false

		// Set the config path to empty for backend loading
		configPath = ""
	}

	// Load the module if we don't have one yet (not running from plan)
	var mod *module.Tree
	if plan == nil {
		mod, err = c.Module(configPath)
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Failed to load root config module: %s", err))
			return 1
		}
	}

	var conf *config.Config
	if mod != nil {
		conf = mod.Config()
	}
	// Load the backend
	b, err := c.Backend(&BackendOpts{
		Config: conf,
		Plan:   plan,
	})
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to load backend: %s", err))
		return 1
	}

	// Build the operation
	opReq := c.Operation()
	opReq.Destroy = destroy
	opReq.Module = mod
	opReq.Plan = plan
	opReq.PlanRefresh = refresh
	opReq.PlanOutPath = outPath
	opReq.Type = backend.OperationTypePlan

	// Perform the operation
	op, err := b.Operation(context.Background(), opReq)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error starting operation: %s", err))
		return 1
	}

	// Wait for the operation to complete
	<-op.Done()
	if err := op.Err; err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	/*
		err = terraform.SetDebugInfo(DefaultDataDir)
		if err != nil {
			c.Ui.Error(err.Error())
			return 1
		}
	*/

	if detailed && !op.PlanEmpty {
		return 2
	}

	return 0
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
                      a file. If "terraform.tfvars" is present, it will be
                      automatically loaded if this flag is not specified.
`
	return strings.TrimSpace(helpText)
}

func (c *PlanCommand) Synopsis() string {
	return "Generate and show an execution plan"
}
