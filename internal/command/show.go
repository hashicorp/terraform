package command

import (
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/format"
	"github.com/hashicorp/terraform/internal/command/jsonplan"
	"github.com/hashicorp/terraform/internal/command/jsonstate"
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

func (c *ShowCommand) Run(args []string) int {
	args = c.Meta.process(args)
	cmdFlags := c.Meta.defaultFlagSet("show")
	var jsonOutput bool
	cmdFlags.BoolVar(&jsonOutput, "json", false, "produce JSON output")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		c.Ui.Error(fmt.Sprintf("Error parsing command-line flags: %s\n", err.Error()))
		return 1
	}

	args = cmdFlags.Args()
	if len(args) > 2 {
		c.Ui.Error(
			"The show command expects at most two arguments.\n The path to a " +
				"Terraform state or plan file, and optionally -json for json output.\n")
		cmdFlags.Usage()
		return 1
	}

	// Check for user-supplied plugin path
	var err error
	if c.pluginPath, err = c.loadPluginPath(); err != nil {
		c.Ui.Error(fmt.Sprintf("Error loading plugin path: %s", err))
		return 1
	}

	var diags tfdiags.Diagnostics

	var planErr, stateErr error
	var plan *plans.Plan
	var stateFile *statefile.File
	var config *configs.Config
	var schemas *terraform.Schemas

	// if a path was provided, try to read it as a path to a planfile
	// if that fails, try to read the cli argument as a path to a statefile
	if len(args) > 0 {
		path := args[0]
		plan, stateFile, config, planErr = getPlanFromPath(path)
		if planErr != nil {
			stateFile, stateErr = getStateFromPath(path)
			if stateErr != nil {
				c.Ui.Error(fmt.Sprintf(
					"Terraform couldn't read the given file as a state or plan file.\n"+
						"The errors while attempting to read the file as each format are\n"+
						"shown below.\n\n"+
						"State read error: %s\n\nPlan read error: %s",
					stateErr,
					planErr))
				return 1
			}
		}
	} else {
		// Load the backend
		b, backendDiags := c.Backend(nil)
		diags = diags.Append(backendDiags)
		if backendDiags.HasErrors() {
			c.showDiagnostics(diags)
			return 1
		}
		c.ignoreRemoteVersionConflict(b)

		workspace, err := c.Workspace()
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Error selecting workspace: %s", err))
			return 1
		}
		stateFile, stateErr = getStateFromBackend(b, workspace)
		if stateErr != nil {
			c.Ui.Error(stateErr.Error())
			return 1
		}
	}

	if config != nil || stateFile != nil {
		opts, err := c.contextOpts()
		if err != nil {
			diags = diags.Append(err)
			c.showDiagnostics(diags)
			return 1
		}
		tfCtx, ctxDiags := terraform.NewContext(opts)
		diags = diags.Append(ctxDiags)
		if ctxDiags.HasErrors() {
			c.showDiagnostics(diags)
			return 1
		}
		var schemaDiags tfdiags.Diagnostics
		schemas, schemaDiags = tfCtx.Schemas(config, stateFile.State)
		diags = diags.Append(schemaDiags)
		if schemaDiags.HasErrors() {
			c.showDiagnostics(diags)
			return 1
		}
	}

	if plan != nil {
		if jsonOutput {
			jsonPlan, err := jsonplan.Marshal(config, plan, stateFile, schemas)

			if err != nil {
				c.Ui.Error(fmt.Sprintf("Failed to marshal plan to json: %s", err))
				return 1
			}
			c.Ui.Output(string(jsonPlan))
			return 0
		}

		view := views.NewShow(arguments.ViewHuman, c.View)
		view.Plan(plan, schemas)
		return 0
	}

	if jsonOutput {
		// At this point, it is possible that there is neither state nor a plan.
		// That's ok, we'll just return an empty object.
		jsonState, err := jsonstate.Marshal(stateFile, schemas)
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Failed to marshal state to json: %s", err))
			return 1
		}
		c.Ui.Output(string(jsonState))
	} else {
		if stateFile == nil {
			c.Ui.Output("No state.")
			return 0
		}
		c.Ui.Output(format.State(&format.StateOpts{
			State:   stateFile.State,
			Color:   c.Colorize(),
			Schemas: schemas,
		}))
	}

	return 0
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

// getPlanFromPath returns a plan, statefile, and config if the user-supplied
// path points to a planfile. If both plan and error are nil, the path is likely
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
