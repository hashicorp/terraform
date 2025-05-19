// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/cloud"
	"github.com/hashicorp/terraform/internal/cloud/cloudplan"
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

// Many of the methods we get data from can emit special error types if they're
// pretty sure about the file type but still can't use it. But they can't all do
// that! So, we have to do a couple ourselves if we want to preserve that data.
type errUnusableDataMisc struct {
	inner error
	kind  string
}

func errUnusable(err error, kind string) *errUnusableDataMisc {
	return &errUnusableDataMisc{inner: err, kind: kind}
}
func (e *errUnusableDataMisc) Error() string {
	return e.inner.Error()
}
func (e *errUnusableDataMisc) Unwrap() error {
	return e.inner
}

// ShowCommand is a Command implementation that reads and outputs the
// contents of a Terraform plan or state file.
type ShowCommand struct {
	Meta
	viewType arguments.ViewType
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
	c.viewType = args.ViewType

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
	plan, jsonPlan, stateFile, config, schemas, showDiags := c.show(args.Path)
	diags = diags.Append(showDiags)
	if showDiags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	// Display the data
	return view.Display(config, plan, jsonPlan, stateFile, schemas)
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

func (c *ShowCommand) show(path string) (*plans.Plan, *cloudplan.RemotePlanJSON, *statefile.File, *configs.Config, *terraform.Schemas, tfdiags.Diagnostics) {
	var diags, showDiags tfdiags.Diagnostics
	var plan *plans.Plan
	var jsonPlan *cloudplan.RemotePlanJSON
	var stateFile *statefile.File
	var config *configs.Config
	var schemas *terraform.Schemas

	// No plan file or state file argument provided,
	// so get the latest state snapshot
	if path == "" {
		stateFile, showDiags = c.showFromLatestStateSnapshot()
		diags = diags.Append(showDiags)
		if showDiags.HasErrors() {
			return plan, jsonPlan, stateFile, config, schemas, diags
		}
	}

	// Plan file or state file argument provided,
	// so try to load the argument as a plan file first.
	// If that fails, try to load it as a statefile.
	if path != "" {
		plan, jsonPlan, stateFile, config, showDiags = c.showFromPath(path)
		diags = diags.Append(showDiags)
		if showDiags.HasErrors() {
			return plan, jsonPlan, stateFile, config, schemas, diags
		}
	}

	// Get schemas, if possible
	if config != nil || stateFile != nil {
		schemas, diags = c.MaybeGetSchemas(stateFile.State, config)
		if diags.HasErrors() {
			return plan, jsonPlan, stateFile, config, schemas, diags
		}
	}

	return plan, jsonPlan, stateFile, config, schemas, diags
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
		diags = diags.Append(stateErr)
		return nil, diags
	}

	return stateFile, diags
}

func (c *ShowCommand) showFromPath(path string) (*plans.Plan, *cloudplan.RemotePlanJSON, *statefile.File, *configs.Config, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	var planErr, stateErr error
	var plan *plans.Plan
	var jsonPlan *cloudplan.RemotePlanJSON
	var stateFile *statefile.File
	var config *configs.Config

	// Path might be a local plan file, a bookmark to a saved cloud plan, or a
	// state file. First, try to get a plan and associated data from a local
	// plan file. If that fails, try to get a json plan from the path argument.
	// If that fails, try to get the statefile from the path argument.
	plan, jsonPlan, stateFile, config, planErr = c.getPlanFromPath(path)
	if planErr != nil {
		stateFile, stateErr = getStateFromPath(path)
		if stateErr != nil {
			// To avoid spamming the user with irrelevant errors, first check to
			// see if one of our errors happens to know for a fact what file
			// type we were dealing with. If so, then we can ignore the other
			// ones (which are likely to be something unhelpful like "not a
			// valid zip file"). If not, we can fall back to dumping whatever
			// we've got.
			var unLocal *planfile.ErrUnusableLocalPlan
			var unState *statefile.ErrUnusableState
			var unMisc *errUnusableDataMisc
			if errors.As(planErr, &unLocal) {
				diags = diags.Append(
					tfdiags.Sourceless(
						tfdiags.Error,
						"Couldn't show local plan",
						fmt.Sprintf("Plan read error: %s", unLocal),
					),
				)
			} else if errors.As(planErr, &unMisc) {
				diags = diags.Append(
					tfdiags.Sourceless(
						tfdiags.Error,
						fmt.Sprintf("Couldn't show %s", unMisc.kind),
						fmt.Sprintf("Plan read error: %s", unMisc),
					),
				)
			} else if errors.As(stateErr, &unState) {
				diags = diags.Append(
					tfdiags.Sourceless(
						tfdiags.Error,
						"Couldn't show state file",
						fmt.Sprintf("Plan read error: %s", unState),
					),
				)
			} else if errors.As(stateErr, &unMisc) {
				diags = diags.Append(
					tfdiags.Sourceless(
						tfdiags.Error,
						fmt.Sprintf("Couldn't show %s", unMisc.kind),
						fmt.Sprintf("Plan read error: %s", unMisc),
					),
				)
			} else {
				// Ok, give up and show the really big error
				diags = diags.Append(
					tfdiags.Sourceless(
						tfdiags.Error,
						"Failed to read the given file as a state or plan file",
						fmt.Sprintf("State read error: %s\n\nPlan read error: %s", stateErr, planErr),
					),
				)
			}

			return nil, nil, nil, nil, diags
		}
	}
	return plan, jsonPlan, stateFile, config, diags
}

// getPlanFromPath returns a plan, json plan, statefile, and config if the
// user-supplied path points to either a local or cloud plan file. Note that
// some of the return values will be nil no matter what; local plan files do not
// yield a json plan, and cloud plans do not yield real plan/state/config
// structs. An error generally suggests that the given path is either a
// directory or a statefile.
func (c *ShowCommand) getPlanFromPath(path string) (*plans.Plan, *cloudplan.RemotePlanJSON, *statefile.File, *configs.Config, error) {
	var err error
	var plan *plans.Plan
	var jsonPlan *cloudplan.RemotePlanJSON
	var stateFile *statefile.File
	var config *configs.Config

	pf, err := planfile.OpenWrapped(path)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	if lp, ok := pf.Local(); ok {
		plan, stateFile, config, err = getDataFromPlanfileReader(lp)
	} else if cp, ok := pf.Cloud(); ok {
		redacted := c.viewType != arguments.ViewJSON
		jsonPlan, err = c.getDataFromCloudPlan(cp, redacted)
	}

	return plan, jsonPlan, stateFile, config, err
}

func (c *ShowCommand) getDataFromCloudPlan(plan *cloudplan.SavedPlanBookmark, redacted bool) (*cloudplan.RemotePlanJSON, error) {
	// Set up the backend
	b, backendDiags := c.Backend(nil)
	if backendDiags.HasErrors() {
		return nil, errUnusable(backendDiags.Err(), "cloud plan")
	}
	// Cloud plans only work if we're cloud.
	cl, ok := b.(*cloud.Cloud)
	if !ok {
		errMessage := fmt.Sprintf("can't show a saved cloud plan unless the current root module is connected to %s", cl.AppName())
		return nil, errUnusable(errors.New(errMessage), "cloud plan")
	}

	result, err := cl.ShowPlanForRun(context.Background(), plan.RunID, plan.Hostname, redacted)
	if err != nil {
		err = errUnusable(err, "cloud plan")
	}
	return result, err
}

// getDataFromPlanfileReader returns a plan, statefile, and config, extracted from a local plan file.
func getDataFromPlanfileReader(planReader *planfile.Reader) (*plans.Plan, *statefile.File, *configs.Config, error) {
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
		return nil, nil, nil, errUnusable(diags.Err(), "local plan")
	}

	return plan, stateFile, config, err
}

// getStateFromPath returns a statefile if the user-supplied path points to a statefile.
func getStateFromPath(path string) (*statefile.File, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("Error loading statefile: %w", err)
	}
	defer file.Close()

	var stateFile *statefile.File
	stateFile, err = statefile.Read(file)
	if err != nil {
		return nil, fmt.Errorf("Error reading %s as a statefile: %w", path, err)
	}
	return stateFile, nil
}

// getStateFromBackend returns the State for the current workspace, if available.
func getStateFromBackend(b backend.Backend, workspace string) (*statefile.File, error) {
	// Get the state store for the given workspace
	stateStore, err := b.StateMgr(workspace)
	if err != nil {
		return nil, fmt.Errorf("Failed to load state manager: %w", err)
	}

	// Refresh the state store with the latest state snapshot from persistent storage
	if err := stateStore.RefreshState(); err != nil {
		return nil, fmt.Errorf("Failed to load state: %w", err)
	}

	// Get the latest state snapshot and return it
	stateFile := statemgr.Export(stateStore)
	return stateFile, nil
}
