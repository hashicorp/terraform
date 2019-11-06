package command

import (
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/terraform/backend"
	localBackend "github.com/hashicorp/terraform/backend/local"
	"github.com/hashicorp/terraform/command/format"
	"github.com/hashicorp/terraform/command/jsonplan"
	"github.com/hashicorp/terraform/command/jsonstate"
	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/plans/planfile"
	"github.com/hashicorp/terraform/states/statefile"
	"github.com/hashicorp/terraform/states/statemgr"
	"github.com/hashicorp/terraform/tfdiags"
)

// ShowCommand is a Command implementation that reads and outputs the
// contents of a Terraform plan or state file.
type ShowCommand struct {
	Meta
}

func (c *ShowCommand) Run(args []string) int {
	args, err := c.Meta.process(args, false)
	if err != nil {
		return 1
	}

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
	if c.pluginPath, err = c.loadPluginPath(); err != nil {
		c.Ui.Error(fmt.Sprintf("Error loading plugin path: %s", err))
		return 1
	}

	var diags tfdiags.Diagnostics

	// Load the backend
	b, backendDiags := c.Backend(nil)
	diags = diags.Append(backendDiags)
	if backendDiags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	// We require a local backend
	local, ok := b.(backend.Local)
	if !ok {
		c.showDiagnostics(diags) // in case of any warnings in here
		c.Ui.Error(ErrUnsupportedLocalOp)
		return 1
	}

	// the show command expects the config dir to always be the cwd
	cwd, err := os.Getwd()
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error getting cwd: %s", err))
		return 1
	}

	// Determine if a planfile was passed to the command
	var planFile *planfile.Reader
	if len(args) > 0 {
		// We will handle error checking later on - this is just required to
		// load the local context if the given path is successfully read as
		// a planfile.
		planFile, _ = c.PlanFile(args[0])
	}

	// Build the operation
	opReq := c.Operation(b)
	opReq.ConfigDir = cwd
	opReq.PlanFile = planFile
	opReq.ConfigLoader, err = c.initConfigLoader()
	opReq.AllowUnsetVariables = true
	if err != nil {
		diags = diags.Append(err)
		c.showDiagnostics(diags)
		return 1
	}

	// Get the context
	ctx, _, ctxDiags := local.Context(opReq)
	diags = diags.Append(ctxDiags)
	if ctxDiags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	// Get the schemas from the context
	schemas := ctx.Schemas()

	var planErr, stateErr error
	var plan *plans.Plan
	var stateFile *statefile.File

	// if a path was provided, try to read it as a path to a planfile
	// if that fails, try to read the cli argument as a path to a statefile
	if len(args) > 0 {
		path := args[0]
		plan, stateFile, planErr = getPlanFromPath(path)
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
		env := c.Workspace()
		stateFile, stateErr = getStateFromEnv(b, env)
		if stateErr != nil {
			c.Ui.Error(stateErr.Error())
			return 1
		}
	}

	if plan != nil {
		if jsonOutput == true {
			config := ctx.Config()
			jsonPlan, err := jsonplan.Marshal(config, plan, stateFile, schemas)

			if err != nil {
				c.Ui.Error(fmt.Sprintf("Failed to marshal plan to json: %s", err))
				return 1
			}
			c.Ui.Output(string(jsonPlan))
			return 0
		}

		// FIXME: We currently call into the local backend for this, since
		// the "terraform plan" logic lives there and our package call graph
		// means we can't orient this dependency the other way around. In
		// future we'll hopefully be able to refactor the backend architecture
		// a little so that CLI UI rendering always happens in this "command"
		// package rather than in the backends themselves, but for now we're
		// accepting this oddity because "terraform show" is a less commonly
		// used way to render a plan than "terraform plan" is.
		localBackend.RenderPlan(plan, stateFile.State, schemas, c.Ui, c.Colorize())
		return 0
	}

	if jsonOutput == true {
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
Usage: terraform show [options] [path]

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
	return "Inspect Terraform state or plan"
}

// getPlanFromPath returns a plan and statefile if the user-supplied path points
// to a planfile. If both plan and error are nil, the path is likely a
// directory. An error could suggest that the given path points to a statefile.
func getPlanFromPath(path string) (*plans.Plan, *statefile.File, error) {
	pr, err := planfile.Open(path)
	if err != nil {
		return nil, nil, err
	}
	plan, err := pr.ReadPlan()
	if err != nil {
		return nil, nil, err
	}

	stateFile, err := pr.ReadStateFile()
	return plan, stateFile, err
}

// getStateFromPath returns a statefile if the user-supplied path points to a statefile.
func getStateFromPath(path string) (*statefile.File, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("Error loading statefile: %s", err)
	}
	defer f.Close()

	var stateFile *statefile.File
	stateFile, err = statefile.Read(f)
	if err != nil {
		return nil, fmt.Errorf("Error reading %s as a statefile: %s", path, err)
	}
	return stateFile, nil
}

// getStateFromEnv returns the State for the current workspace, if available.
func getStateFromEnv(b backend.Backend, env string) (*statefile.File, error) {
	// Get the state
	stateStore, err := b.StateMgr(env)
	if err != nil {
		return nil, fmt.Errorf("Failed to load state manager: %s", err)
	}

	if err := stateStore.RefreshState(); err != nil {
		return nil, fmt.Errorf("Failed to load state: %s", err)
	}

	sf := statemgr.Export(stateStore)

	return sf, nil
}
