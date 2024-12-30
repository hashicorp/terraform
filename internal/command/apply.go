// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/internal/backend/backendrun"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/plans/planfile"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// ApplyCommand is a Command implementation that applies a Terraform
// configuration and actually builds or changes infrastructure.
type ApplyCommand struct {
	Meta

	// If true, then this apply command will become the "destroy"
	// command. It is just like apply but only processes a destroy.
	Destroy bool
}

func (c *ApplyCommand) Run(rawArgs []string) int {
	var diags tfdiags.Diagnostics

	// Parse and apply global view arguments
	common, rawArgs := arguments.ParseView(rawArgs)
	c.View.Configure(common)

	// Propagate -no-color for legacy use of Ui.  The remote backend and
	// cloud package use this; it should be removed when/if they are
	// migrated to views.
	c.Meta.color = !common.NoColor
	c.Meta.Color = c.Meta.color

	// Parse and validate flags
	var args *arguments.Apply
	switch {
	case c.Destroy:
		args, diags = arguments.ParseApplyDestroy(rawArgs)
	default:
		args, diags = arguments.ParseApply(rawArgs)
	}

	// Instantiate the view, even if there are flag errors, so that we render
	// diagnostics according to the desired view
	view := views.NewApply(args.ViewType, c.Destroy, c.View)

	if diags.HasErrors() {
		view.Diagnostics(diags)
		view.HelpPrompt()
		return 1
	}

	// Check for user-supplied plugin path
	var err error
	if c.pluginPath, err = c.loadPluginPath(); err != nil {
		diags = diags.Append(err)
		view.Diagnostics(diags)
		return 1
	}

	// Attempt to load the plan file, if specified
	planFile, loadPlanFileDiags := c.LoadPlanFile(args.PlanPath)
	diags = diags.Append(loadPlanFileDiags)
	if diags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	// FIXME: the -input flag value is needed to initialize the backend and the
	// operation, but there is no clear path to pass this value down, so we
	// continue to mutate the Meta object state for now.
	c.Meta.input = args.InputEnabled

	// FIXME: the -parallelism flag is used to control the concurrency of
	// Terraform operations. At the moment, this value is used both to
	// initialize the backend via the ContextOpts field inside CLIOpts, and to
	// set a largely unused field on the Operation request. Again, there is no
	// clear path to pass this value down, so we continue to mutate the Meta
	// object state for now.
	c.Meta.parallelism = args.Operation.Parallelism

	// Prepare the backend, passing the plan file if present, and the
	// backend-specific arguments
	be, beDiags := c.PrepareBackend(planFile, args.State, args.ViewType)
	diags = diags.Append(beDiags)
	if diags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	// Build the operation request
	opReq, opDiags := c.OperationRequest(be, view, args.ViewType, planFile, args.Operation, args.AutoApprove)
	diags = diags.Append(opDiags)
	if diags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	// Collect variable value and add them to the operation request
	diags = diags.Append(c.GatherVariables(opReq, args.Vars))

	// Before we delegate to the backend, we'll print any warning diagnostics
	// we've accumulated here, since the backend will start fresh with its own
	// diagnostics.
	view.Diagnostics(diags)
	if diags.HasErrors() {
		return 1
	}
	diags = nil

	// Run the operation
	op, err := c.RunOperation(be, opReq)
	if err != nil {
		diags = diags.Append(err)
		view.Diagnostics(diags)
		return 1
	}

	if op.Result != backendrun.OperationSuccess {
		return op.Result.ExitStatus()
	}

	// Render the resource count and outputs, unless those counts are being
	// rendered already in a remote Terraform process.
	if rb, isRemoteBackend := be.(BackendWithRemoteTerraformVersion); !isRemoteBackend || rb.IsLocalOperations() {
		view.ResourceCount(args.State.StateOutPath)
		if !c.Destroy && op.State != nil {
			view.Outputs(op.State.RootOutputValues)
		}
	}

	view.Diagnostics(diags)

	if diags.HasErrors() {
		return 1
	}

	return 0
}

func (c *ApplyCommand) LoadPlanFile(path string) (*planfile.WrappedPlanFile, tfdiags.Diagnostics) {
	var planFile *planfile.WrappedPlanFile
	var diags tfdiags.Diagnostics

	// Try to load plan if path is specified
	if path != "" {
		var err error
		planFile, err = c.PlanFile(path)
		if err != nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				fmt.Sprintf("Failed to load %q as a plan file", path),
				fmt.Sprintf("Error: %s", err),
			))
			return nil, diags
		}

		// If the path doesn't look like a plan, both planFile and err will be
		// nil. In that case, the user is probably trying to use the positional
		// argument to specify a configuration path. Point them at -chdir.
		if planFile == nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				fmt.Sprintf("Failed to load %q as a plan file", path),
				"The specified path is a directory, not a plan file. You can use the global -chdir flag to use this directory as the configuration root.",
			))
			return nil, diags
		}

		// If we successfully loaded a plan but this is a destroy operation,
		// explain that this is not supported.
		if c.Destroy {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Destroy can't be called with a plan file",
				fmt.Sprintf("If this plan was created using plan -destroy, apply it using:\n  terraform apply %q", path),
			))
			return nil, diags
		}
	}

	return planFile, diags
}

func (c *ApplyCommand) PrepareBackend(planFile *planfile.WrappedPlanFile, args *arguments.State, viewType arguments.ViewType) (backendrun.OperationsBackend, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// FIXME: we need to apply the state arguments to the meta object here
	// because they are later used when initializing the backend. Carving a
	// path to pass these arguments to the functions that need them is
	// difficult but would make their use easier to understand.
	c.Meta.applyStateArguments(args)

	// Load the backend
	var be backendrun.OperationsBackend
	var beDiags tfdiags.Diagnostics
	if lp, ok := planFile.Local(); ok {
		plan, err := lp.ReadPlan()
		if err != nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Failed to read plan from plan file",
				fmt.Sprintf("Cannot read the plan from the given plan file: %s.", err),
			))
			return nil, diags
		}
		if plan.Backend.Config == nil {
			// Should never happen; always indicates a bug in the creation of the plan file
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Failed to read plan from plan file",
				"The given plan file does not have a valid backend configuration. This is a bug in the Terraform command that generated this plan file.",
			))
			return nil, diags
		}
		be, beDiags = c.BackendForLocalPlan(plan.Backend)
	} else {
		// Both new plans and saved cloud plans load their backend from config.
		backendConfig, configDiags := c.loadBackendConfig(".")
		diags = diags.Append(configDiags)
		if configDiags.HasErrors() {
			return nil, diags
		}

		be, beDiags = c.Backend(&BackendOpts{
			Config:   backendConfig,
			ViewType: viewType,
		})
	}

	diags = diags.Append(beDiags)
	if beDiags.HasErrors() {
		return nil, diags
	}
	return be, diags
}

func (c *ApplyCommand) OperationRequest(
	be backendrun.OperationsBackend,
	view views.Apply,
	viewType arguments.ViewType,
	planFile *planfile.WrappedPlanFile,
	args *arguments.Operation,
	autoApprove bool,
) (*backendrun.Operation, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// Applying changes with dev overrides in effect could make it impossible
	// to switch back to a release version if the schema isn't compatible,
	// so we'll warn about it.
	b, isRemoteBackend := be.(BackendWithRemoteTerraformVersion)
	if isRemoteBackend && !b.IsLocalOperations() {
		diags = diags.Append(c.providerDevOverrideRuntimeWarningsRemoteExecution())
	} else {
		diags = diags.Append(c.providerDevOverrideRuntimeWarnings())
	}

	// Build the operation
	opReq := c.Operation(be, viewType)
	opReq.AutoApprove = autoApprove
	opReq.ConfigDir = "."
	opReq.PlanMode = args.PlanMode
	opReq.Hooks = view.Hooks()
	opReq.PlanFile = planFile
	opReq.PlanRefresh = args.Refresh
	opReq.Targets = args.Targets
	opReq.ForceReplace = args.ForceReplace
	opReq.Type = backendrun.OperationTypeApply
	opReq.View = view.Operation()
	opReq.StatePersistInterval = c.Meta.StatePersistInterval()

	// EXPERIMENTAL: maybe enable deferred actions
	if c.AllowExperimentalFeatures {
		opReq.DeferralAllowed = args.DeferralAllowed
	} else if args.DeferralAllowed {
		// Belated flag parse error, since we don't know about experiments
		// support at actual parse time.
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to parse command-line flags",
			"The -allow-deferral flag is only valid in experimental builds of Terraform.",
		))
		return nil, diags
	}

	var err error
	opReq.ConfigLoader, err = c.initConfigLoader()
	if err != nil {
		diags = diags.Append(fmt.Errorf("Failed to initialize config loader: %s", err))
		return nil, diags
	}

	return opReq, diags
}

func (c *ApplyCommand) GatherVariables(opReq *backendrun.Operation, args *arguments.Vars) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	// FIXME the arguments package currently trivially gathers variable related
	// arguments in a heterogenous slice, in order to minimize the number of
	// code paths gathering variables during the transition to this structure.
	// Once all commands that gather variables have been converted to this
	// structure, we could move the variable gathering code to the arguments
	// package directly, removing this shim layer.

	varArgs := args.All()
	items := make([]arguments.FlagNameValue, len(varArgs))
	for i := range varArgs {
		items[i].Name = varArgs[i].Name
		items[i].Value = varArgs[i].Value
	}
	c.Meta.variableArgs = arguments.FlagNameValueSlice{Items: &items}
	opReq.Variables, diags = c.collectVariableValues()

	return diags
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
Usage: terraform [global options] apply [options] [PLAN]

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

  -destroy               Destroy Terraform-managed infrastructure.
                         The command "terraform destroy" is a convenience alias
                         for this option.

  -lock=false            Don't hold a state lock during the operation. This is
                         dangerous if others might concurrently run commands
                         against the same workspace.

  -lock-timeout=0s       Duration to retry a state lock.

  -input=true            Ask for input for variables if not directly set.

  -no-color              If specified, output won't contain any color.

  -parallelism=n         Limit the number of parallel resource operations.
                         Defaults to 10.

  -state=path            Path to read and save state (unless state-out
                         is specified). Defaults to "terraform.tfstate".

  -state-out=path        Path to write state to that is different than
                         "-state". This can be used to preserve the old
                         state.
                         
  -var 'foo=bar'         Set a value for one of the input variables in the root
                         module of the configuration. Use this option more than
                         once to set more than one variable.

  -var-file=filename     Load variable values from the given file, in addition
                         to the default files terraform.tfvars and *.auto.tfvars.
                         Use this option more than once to include more than one
                         variables file.

  If you don't provide a saved plan file then this command will also accept
  all of the plan-customization options accepted by the terraform plan command.
  For more information on those options, run:
      terraform plan -help
`
	return strings.TrimSpace(helpText)
}

func (c *ApplyCommand) helpDestroy() string {
	helpText := `
Usage: terraform [global options] destroy [options]

  Destroy Terraform-managed infrastructure.

  This command is a convenience alias for:
      terraform apply -destroy

  This command also accepts many of the plan-customization options accepted by
  the terraform plan command. For more information on those options, run:
      terraform plan -help
`
	return strings.TrimSpace(helpText)
}
