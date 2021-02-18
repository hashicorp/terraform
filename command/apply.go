package command

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/command/arguments"
	"github.com/hashicorp/terraform/command/views"
	"github.com/hashicorp/terraform/plans/planfile"
	"github.com/hashicorp/terraform/terraform"
	"github.com/hashicorp/terraform/tfdiags"
)

// ApplyCommand is a Command implementation that applies a Terraform
// configuration and actually builds or changes infrastructure.
type ApplyCommand struct {
	Meta

	// If true, then this apply command will become the "destroy"
	// command. It is just like apply but only processes a destroy.
	Destroy bool
}

func (c *ApplyCommand) Run(args []string) int {
	var refresh, autoApprove bool
	args = c.Meta.process(args)
	cmdName := "apply"
	if c.Destroy {
		cmdName = "destroy"
	}

	cmdFlags := c.Meta.extendedFlagSet(cmdName)
	cmdFlags.BoolVar(&autoApprove, "auto-approve", false, "skip interactive approval of plan before applying")
	cmdFlags.BoolVar(&refresh, "refresh", true, "refresh")
	cmdFlags.IntVar(&c.Meta.parallelism, "parallelism", DefaultParallelism, "parallelism")
	cmdFlags.StringVar(&c.Meta.statePath, "state", "", "path")
	cmdFlags.StringVar(&c.Meta.stateOutPath, "state-out", "", "path")
	cmdFlags.StringVar(&c.Meta.backupPath, "backup", "", "path")
	cmdFlags.BoolVar(&c.Meta.stateLock, "lock", true, "lock state")
	cmdFlags.DurationVar(&c.Meta.stateLockTimeout, "lock-timeout", 0, "lock timeout")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		c.Ui.Error(fmt.Sprintf("Error parsing command-line flags: %s\n", err.Error()))
		return 1
	}

	diags := c.parseTargetFlags()
	if diags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	args = cmdFlags.Args()
	var planPath string
	if len(args) > 0 {
		planPath = args[0]
		args = args[1:]
	}

	configPath, err := ModulePath(args)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	// Check for user-supplied plugin path
	if c.pluginPath, err = c.loadPluginPath(); err != nil {
		c.Ui.Error(fmt.Sprintf("Error loading plugin path: %s", err))
		return 1
	}

	// Try to load plan if path is specified
	var planFile *planfile.Reader
	if planPath != "" {
		planFile, err = c.PlanFile(planPath)
		if err != nil {
			c.Ui.Error(err.Error())
			return 1
		}

		// If the path doesn't look like a plan, both planFile and err will be
		// nil. In that case, the user is probably trying to use the positional
		// argument to specify a configuration path. Point them at -chdir.
		if planFile == nil {
			c.Ui.Error(fmt.Sprintf("Failed to load %q as a plan file. Did you mean to use -chdir?", planPath))
			return 1
		}

		// If we successfully loaded a plan but this is a destroy operation,
		// explain that this is not supported.
		if c.Destroy {
			c.Ui.Error("Destroy can't be called with a plan file.")
			return 1
		}
	}
	if planFile != nil {
		// Reset the config path for backend loading
		configPath = ""

		if !c.variableArgs.Empty() {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Can't set variables when applying a saved plan",
				"The -var and -var-file options cannot be used when applying a saved plan file, because a saved plan includes the variable values that were set when it was created.",
			))
			c.showDiagnostics(diags)
			return 1
		}
	}

	// Set up our count hook that keeps track of resource changes
	countHook := new(CountHook)

	// Load the backend
	var be backend.Enhanced
	var beDiags tfdiags.Diagnostics
	if planFile == nil {
		backendConfig, configDiags := c.loadBackendConfig(configPath)
		diags = diags.Append(configDiags)
		if configDiags.HasErrors() {
			c.showDiagnostics(diags)
			return 1
		}

		be, beDiags = c.Backend(&BackendOpts{
			Config: backendConfig,
		})
	} else {
		plan, err := planFile.ReadPlan()
		if err != nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Failed to read plan from plan file",
				fmt.Sprintf("Cannot read the plan from the given plan file: %s.", err),
			))
			c.showDiagnostics(diags)
			return 1
		}
		if plan.Backend.Config == nil {
			// Should never happen; always indicates a bug in the creation of the plan file
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Failed to read plan from plan file",
				"The given plan file does not have a valid backend configuration. This is a bug in the Terraform command that generated this plan file.",
			))
			c.showDiagnostics(diags)
			return 1
		}
		be, beDiags = c.BackendForPlan(plan.Backend)
	}
	diags = diags.Append(beDiags)
	if beDiags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	// Applying changes with dev overrides in effect could make it impossible
	// to switch back to a release version if the schema isn't compatible,
	// so we'll warn about it.
	diags = diags.Append(c.providerDevOverrideRuntimeWarnings())

	// Before we delegate to the backend, we'll print any warning diagnostics
	// we've accumulated here, since the backend will start fresh with its own
	// diagnostics.
	c.showDiagnostics(diags)
	diags = nil

	// Build the operation
	opReq := c.Operation(be)
	opReq.AutoApprove = autoApprove
	opReq.ConfigDir = configPath
	opReq.Destroy = c.Destroy
	opReq.Hooks = []terraform.Hook{countHook, c.uiHook()}
	opReq.PlanFile = planFile
	opReq.PlanRefresh = refresh
	opReq.ShowDiagnostics = c.showDiagnostics
	opReq.Type = backend.OperationTypeApply
	opReq.View = views.NewOperation(arguments.ViewHuman, c.RunningInAutomation, c.View)

	opReq.ConfigLoader, err = c.initConfigLoader()
	if err != nil {
		c.showDiagnostics(err)
		return 1
	}

	{
		var moreDiags tfdiags.Diagnostics
		opReq.Variables, moreDiags = c.collectVariableValues()
		diags = diags.Append(moreDiags)
		if moreDiags.HasErrors() {
			c.showDiagnostics(diags)
			return 1
		}
	}

	op, err := c.RunOperation(be, opReq)
	if err != nil {
		c.showDiagnostics(err)
		return 1
	}

	if op.Result != backend.OperationSuccess {
		return op.Result.ExitStatus()
	}

	// Show the count results from the operation
	if c.Destroy {
		c.Ui.Output(c.Colorize().Color(fmt.Sprintf(
			"[reset][bold][green]\n"+
				"Destroy complete! Resources: %d destroyed.",
			countHook.Removed)))
	} else {
		c.Ui.Output(c.Colorize().Color(fmt.Sprintf(
			"[reset][bold][green]\n"+
				"Apply complete! Resources: %d added, %d changed, %d destroyed.",
			countHook.Added,
			countHook.Changed,
			countHook.Removed)))
	}

	// only show the state file help message if the state is local.
	if (countHook.Added > 0 || countHook.Changed > 0) && c.Meta.stateOutPath != "" {
		c.Ui.Output(c.Colorize().Color(fmt.Sprintf(
			"[reset]\n"+
				"The state of your infrastructure has been saved to the path\n"+
				"below. This state is required to modify and destroy your\n"+
				"infrastructure, so keep it safe. To inspect the complete state\n"+
				"use the `terraform show` command.\n\n"+
				"State path: %s",
			c.Meta.stateOutPath)))
	}

	if !c.Destroy && op.State != nil {
		outputValues := op.State.RootModule().OutputValues
		if len(outputValues) > 0 {
			c.Ui.Output(c.Colorize().Color("[reset][bold][green]\nOutputs:\n\n"))
			view := views.NewOutput(arguments.ViewHuman, c.View)
			view.Output("", outputValues)
		}
	}

	return op.Result.ExitStatus()
}

func (c *ApplyCommand) Help() string {
	if c.Destroy {
		return c.helpDestroy()
	}

	return c.helpApply()
}

func (c *ApplyCommand) Synopsis() string {
	if c.Destroy {
		return "Destroy previously-created infrastructure"
	}

	return "Create or update infrastructure"
}

func (c *ApplyCommand) helpApply() string {
	helpText := `
Usage: terraform apply [options] [PLAN]

  Creates or updates infrastructure according to Terraform configuration
  files in the current directory.

  By default, Terraform will generate a new plan and present it for your
  approval before taking any action. You can optionally provide a plan
  file created by a previous call to "terraform plan", in which case
  Terraform will take the actions described in that plan without any
  confirmation prompt.

Options:

  -auto-approve          Skip interactive approval of plan before applying.

  -backup=path           Path to backup the existing state file before
                         modifying. Defaults to the "-state-out" path with
                         ".backup" extension. Set to "-" to disable backup.

  -compact-warnings      If Terraform produces any warnings that are not
                         accompanied by errors, show them in a more compact
                         form that includes only the summary messages.

  -lock=true             Lock the state file when locking is supported.

  -lock-timeout=0s       Duration to retry a state lock.

  -input=true            Ask for input for variables if not directly set.

  -no-color              If specified, output won't contain any color.

  -parallelism=n         Limit the number of parallel resource operations.
                         Defaults to 10.

  -refresh=true          Update state prior to checking for differences. This
                         has no effect if a plan file is given to apply.

  -state=path            Path to read and save state (unless state-out
                         is specified). Defaults to "terraform.tfstate".

  -state-out=path        Path to write state to that is different than
                         "-state". This can be used to preserve the old
                         state.

  -target=resource       Resource to target. Operation will be limited to this
                         resource and its dependencies. This flag can be used
                         multiple times.

  -var 'foo=bar'         Set a variable in the Terraform configuration. This
                         flag can be set multiple times.

  -var-file=foo          Set variables in the Terraform configuration from
                         a file. If "terraform.tfvars" or any ".auto.tfvars"
                         files are present, they will be automatically loaded.


`
	return strings.TrimSpace(helpText)
}

func (c *ApplyCommand) helpDestroy() string {
	helpText := `
Usage: terraform destroy [options]

  Destroy Terraform-managed infrastructure.

Options:

  -backup=path           Path to backup the existing state file before
                         modifying. Defaults to the "-state-out" path with
                         ".backup" extension. Set to "-" to disable backup.

  -auto-approve          Skip interactive approval before destroying.

  -lock=true             Lock the state file when locking is supported.

  -lock-timeout=0s       Duration to retry a state lock.

  -no-color              If specified, output won't contain any color.

  -parallelism=n         Limit the number of concurrent operations.
                         Defaults to 10.

  -refresh=true          Update state prior to checking for differences. This
                         has no effect if a plan file is given to apply.

  -state=path            Path to read and save state (unless state-out
                         is specified). Defaults to "terraform.tfstate".

  -state-out=path        Path to write state to that is different than
                         "-state". This can be used to preserve the old
                         state.

  -target=resource       Resource to target. Operation will be limited to this
                         resource and its dependencies. This flag can be used
                         multiple times.

  -var 'foo=bar'         Set a variable in the Terraform configuration. This
                         flag can be set multiple times.

  -var-file=foo          Set variables in the Terraform configuration from
                         a file. If "terraform.tfvars" or any ".auto.tfvars"
                         files are present, they will be automatically loaded.


`
	return strings.TrimSpace(helpText)
}

const outputInterrupt = `Interrupt received.
Please wait for Terraform to exit or data loss may occur.
Gracefully shutting down...`
