package command

import (
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/plans/planfile"
	"github.com/hashicorp/terraform/internal/states/statefile"
	"github.com/hashicorp/terraform/internal/states/statemgr"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// ShowCommand is a Command implementation that reads and outputs the
// contents of a Terraform plan or state file.
type ShowCommand struct {
	Meta
}

func (c *ShowCommand) Run(rawArgs []string) int {
	// Parse and apply global view arguments
	common, rawArgs := arguments.ParseView(rawArgs)
	c.View.Configure(common)

	// Parse and validate flags
	args, diags := arguments.ParseShow(rawArgs)
	if diags.HasErrors() {
		c.View.Diagnostics(diags)
		c.View.HelpPrompt("show")
		return 1
	}

	// Set up view
	view := views.NewShow(args.ViewType, c.View)

	// Check for user-supplied plugin path
	var err error
	if c.pluginPath, err = c.loadPluginPath(); err != nil {
		diags = diags.Append(fmt.Errorf("error loading plugin path: %s", err))
		view.Diagnostics(diags)
		return 1
	}

	// Get the data we need to display
	plan, stateFile, config, schemas, showDiags := c.show(args.Path)
	diags = diags.Append(showDiags)
	if showDiags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	// Display the data
	return view.Display(config, plan, stateFile, schemas)
}

func (c *ShowCommand) Help() string {
	helpText := `
Usage: terraform [global options] show [options] [path]

  Reads and outputs a Terraform state or plan file in a human-readable
  form. If no path is specified, the current state will be shown.

Options:

  -no-color           If specified, output won't contain any color.
  -json               If specified, output the Terraform plan or state in
                      a machine-readable form.

`
	return strings.TrimSpace(helpText)
}

func (c *ShowCommand) Synopsis() string {
	return "Show the current state or a saved plan"
}

func (c *ShowCommand) show(path string) (*plans.Plan, *statefile.File, *configs.Config, *terraform.Schemas, tfdiags.Diagnostics) {
	var diags, showDiags tfdiags.Diagnostics
	var plan *plans.Plan
	var stateFile *statefile.File
	var config *configs.Config
	var schemas *terraform.Schemas

	// No plan file or state file argument provided,
	// so get the latest state snapshot
	if path == "" {
		stateFile, showDiags = c.showFromLatestStateSnapshot()
		diags = diags.Append(showDiags)
		if showDiags.HasErrors() {
			return plan, stateFile, config, schemas, diags
		}
	}

	// Plan file or state file argument provided,
	// so try to load the argument as a plan file first.
	// If that fails, try to load it as a statefile.
	if path != "" {
		plan, stateFile, config, showDiags = c.showFromPath(path)
		diags = diags.Append(showDiags)
		if showDiags.HasErrors() {
			return plan, stateFile, config, schemas, diags
		}
	}

	// Get schemas, if possible
	if config != nil || stateFile != nil {
		opts, err := c.contextOpts()
		if err != nil {
			diags = diags.Append(err)
			return plan, stateFile, config, schemas, diags
		}
		tfCtx, ctxDiags := terraform.NewContext(opts)
		diags = diags.Append(ctxDiags)
		if ctxDiags.HasErrors() {
			return plan, stateFile, config, schemas, diags
		}
		var schemaDiags tfdiags.Diagnostics
		schemas, schemaDiags = tfCtx.Schemas(config, stateFile.State)
		diags = diags.Append(schemaDiags)
		if schemaDiags.HasErrors() {
			return plan, stateFile, config, schemas, diags
		}
	}

	return plan, stateFile, config, schemas, diags
}
func (c *ShowCommand) showFromLatestStateSnapshot() (*statefile.File, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// Load the backend
	b, backendDiags := c.Backend(nil)
	diags = diags.Append(backendDiags)
	if backendDiags.HasErrors() {
		return nil, diags
	}
	c.ignoreRemoteVersionConflict(b)

	// Load the workspace
	workspace, err := c.Workspace()
	if err != nil {
		diags = diags.Append(fmt.Errorf("error selecting workspace: %s", err))
		return nil, diags
	}

	// Get the latest state snapshot from the backend for the current workspace
	stateFile, stateErr := getStateFromBackend(b, workspace)
	if stateErr != nil {
		diags = diags.Append(stateErr.Error())
		return nil, diags
	}

	return stateFile, diags
}

func (c *ShowCommand) showFromPath(path string) (*plans.Plan, *statefile.File, *configs.Config, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	var planErr, stateErr error
	var plan *plans.Plan
	var stateFile *statefile.File
	var config *configs.Config

	// Try to get the plan file and associated data from
	// the path argument. If that fails, try to get the
	// statefile from the path argument.
	plan, stateFile, config, planErr = getPlanFromPath(path)
	if planErr != nil {
		stateFile, stateErr = getStateFromPath(path)
		if stateErr != nil {
			diags = diags.Append(
				tfdiags.Sourceless(
					tfdiags.Error,
					"Failed to read the given file as a state or plan file",
					fmt.Sprintf("State read error: %s\n\nPlan read error: %s", stateErr, planErr),
				),
			)
			return nil, nil, nil, diags
		}
	}
	return plan, stateFile, config, diags
}

// getPlanFromPath returns a plan, statefile, and config if the user-supplied
// path points to a plan file. If both plan and error are nil, the path is likely
// a directory. An error could suggest that the given path points to a statefile.
func getPlanFromPath(path string) (*plans.Plan, *statefile.File, *configs.Config, error) {
	planReader, err := planfile.Open(path)
	if err != nil {
		return nil, nil, nil, err
	}

	// Get plan
	plan, err := planReader.ReadPlan()
	if err != nil {
		return nil, nil, nil, err
	}

	// Get statefile
	stateFile, err := planReader.ReadStateFile()
	if err != nil {
		return nil, nil, nil, err
	}

	// Get config
	config, diags := planReader.ReadConfig()
	if diags.HasErrors() {
		return nil, nil, nil, diags.Err()
	}

	return plan, stateFile, config, err
}

// getStateFromPath returns a statefile if the user-supplied path points to a statefile.
func getStateFromPath(path string) (*statefile.File, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("Error loading statefile: %s", err)
	}
	defer file.Close()

	var stateFile *statefile.File
	stateFile, err = statefile.Read(file)
	if err != nil {
		return nil, fmt.Errorf("Error reading %s as a statefile: %s", path, err)
	}
	return stateFile, nil
}

// getStateFromBackend returns the State for the current workspace, if available.
func getStateFromBackend(b backend.Backend, workspace string) (*statefile.File, error) {
	// Get the state store for the given workspace
	stateStore, err := b.StateMgr(workspace)
	if err != nil {
		return nil, fmt.Errorf("Failed to load state manager: %s", err)
	}

	// Refresh the state store with the latest state snapshot from persistent storage
	if err := stateStore.RefreshState(); err != nil {
		return nil, fmt.Errorf("Failed to load state: %s", err)
	}

	// Get the latest state snapshot and return it
	stateFile := statemgr.Export(stateStore)
	return stateFile, nil
}
