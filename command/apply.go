package command

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/command/arguments"
	"github.com/hashicorp/terraform/command/views"
	"github.com/hashicorp/terraform/plans/planfile"
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

func (c *ApplyCommand) Run(rawArgs []string) int {
	// Parse and apply global view arguments
	common, rawArgs := arguments.ParseView(rawArgs)
	c.View.Configure(common)

	// Parse and validate flags
	args, diags := arguments.ParseApply(rawArgs)

	// Instantiate the view, even if there are flag errors, so that we render
	// diagnostics according to the desired view
	var view views.Apply
	view = views.NewApply(args.ViewType, c.Destroy, c.RunningInAutomation, c.View)

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
	planFile, diags := c.LoadPlanFile(args.PlanPath)
	if diags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	// Check for invalid combination of plan file and variable overrides
	if planFile != nil && !args.Vars.Empty() {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Can't set variables when applying a saved plan",
			"The -var and -var-file options cannot be used when applying a saved plan file, because a saved plan includes the variable values that were set when it was created.",
		))
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
	be, beDiags := c.PrepareBackend(planFile, args.State)
	diags = diags.Append(beDiags)
	if diags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	// Build the operation request
	opReq, opDiags := c.OperationRequest(be, view, planFile, args.Operation)
	diags = diags.Append(opDiags)

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

	if op.Result != backend.OperationSuccess {
		return op.Result.ExitStatus()
	}

	// // Render the resource count and outputs
	view.ResourceCount(args.State.StateOutPath)
	if !c.Destroy && op.State != nil {
		view.Outputs(op.State.RootModule().OutputValues)
	}

	view.Diagnostics(diags)

	if diags.HasErrors() {
		return 1
	}

	return 0
}

func (c *ApplyCommand) LoadPlanFile(path string) (*planfile.Reader, tfdiags.Diagnostics) {
	var planFile *planfile.Reader
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

func (c *ApplyCommand) PrepareBackend(planFile *planfile.Reader, args *arguments.State) (backend.Enhanced, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// FIXME: we need to apply the state arguments to the meta object here
	// because they are later used when initializing the backend. Carving a
	// path to pass these arguments to the functions that need them is
	// difficult but would make their use easier to understand.
	c.Meta.applyStateArguments(args)

	// Load the backend
	var be backend.Enhanced
	var beDiags tfdiags.Diagnostics
	if planFile == nil {
		backendConfig, configDiags := c.loadBackendConfig(".")
		diags = diags.Append(configDiags)
		if configDiags.HasErrors() {
			return nil, diags
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
		be, beDiags = c.BackendForPlan(plan.Backend)
	}

	diags = diags.Append(beDiags)
	if beDiags.HasErrors() {
		return nil, diags
	}
	return be, diags
}

func (c *ApplyCommand) OperationRequest(be backend.Enhanced, view views.Apply, planFile *planfile.Reader, args *arguments.Operation,
) (*backend.Operation, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// Applying changes with dev overrides in effect could make it impossible
	// to switch back to a release version if the schema isn't compatible,
	// so we'll warn about it.
	diags = diags.Append(c.providerDevOverrideRuntimeWarnings())

	// Build the operation
	opReq := c.Operation(be)
	opReq.AutoApprove = args.AutoApprove
	opReq.ConfigDir = "."
	opReq.Destroy = c.Destroy
	opReq.Hooks = view.Hooks()
	opReq.PlanFile = planFile
	opReq.PlanRefresh = args.Refresh
	opReq.Targets = args.Targets
	opReq.Type = backend.OperationTypeApply
	opReq.View = view.Operation()

	// FIXME: To allow errors to be easily rendered, the showDiagnostics method
	// accepts ...interface{}. The backend no longer needs this, as only
	// tfdiags.Diagnostics values are used. Once we have migrated plan and refresh
	// to use views, we can remove ShowDiagnostics and update the backend code to
	// call view.Diagnostics() instead.
	opReq.ShowDiagnostics = func(vals ...interface{}) {
		var diags tfdiags.Diagnostics
		diags = diags.Append(vals...)
		view.Diagnostics(diags)
	}

	var err error
	opReq.ConfigLoader, err = c.initConfigLoader()
	if err != nil {
		diags = diags.Append(fmt.Errorf("Failed to initialize config loader: %s", err))
		return nil, diags
	}

	return opReq, diags
}

func (c *ApplyCommand) GatherVariables(opReq *backend.Operation, args *arguments.Vars) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	// FIXME the arguments package currently trivially gathers variable related
	// arguments in a heterogenous slice, in order to minimize the number of
	// code paths gathering variables during the transition to this structure.
	// Once all commands that gather variables have been converted to this
	// structure, we could move the variable gathering code to the arguments
	// package directly, removing this shim layer.

	varArgs := args.All()
	items := make([]rawFlag, len(varArgs))
	for i := range varArgs {
		items[i].Name = varArgs[i].Name
		items[i].Value = varArgs[i].Value
	}
	c.Meta.variableArgs = rawFlags{items: &items}
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
