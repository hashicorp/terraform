package command

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/command/clistate"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/tfdiags"
	"github.com/mitchellh/cli"
)

// StateReplaceProviderCommand is a Command implementation that allows users
// to change the provider associated with existing resources. This is only
// likely to be useful if a provider is forked or changes its fully-qualified
// name.

type StateReplaceProviderCommand struct {
	StateMeta
}

func (c *StateReplaceProviderCommand) Run(args []string) int {
	args = c.Meta.process(args)

	var autoApprove bool
	cmdFlags := c.Meta.ignoreRemoteVersionFlagSet("state replace-provider")
	cmdFlags.BoolVar(&autoApprove, "auto-approve", false, "skip interactive approval of replacements")
	cmdFlags.StringVar(&c.backupPath, "backup", "-", "backup")
	cmdFlags.BoolVar(&c.Meta.stateLock, "lock", true, "lock states")
	cmdFlags.DurationVar(&c.Meta.stateLockTimeout, "lock-timeout", 0, "lock timeout")
	cmdFlags.StringVar(&c.statePath, "state", "", "path")
	if err := cmdFlags.Parse(args); err != nil {
		c.Ui.Error(fmt.Sprintf("Error parsing command-line flags: %s\n", err.Error()))
		return cli.RunResultHelp
	}
	args = cmdFlags.Args()
	if len(args) != 2 {
		c.Ui.Error("Exactly two arguments expected.\n")
		return cli.RunResultHelp
	}

	var diags tfdiags.Diagnostics

	// Parse from/to arguments into providers
	from, fromDiags := addrs.ParseProviderSourceString(args[0])
	if fromDiags.HasErrors() {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			fmt.Sprintf(`Invalid "from" provider %q`, args[0]),
			fromDiags.Err().Error(),
		))
	}
	to, toDiags := addrs.ParseProviderSourceString(args[1])
	if toDiags.HasErrors() {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			fmt.Sprintf(`Invalid "to" provider %q`, args[1]),
			toDiags.Err().Error(),
		))
	}
	if diags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	// Initialize the state manager as configured
	stateMgr, err := c.State()
	if err != nil {
		c.Ui.Error(fmt.Sprintf(errStateLoadingState, err))
		return 1
	}

	// Acquire lock if requested
	if c.stateLock {
		stateLocker := clistate.NewLocker(context.Background(), c.stateLockTimeout, c.Ui, c.Colorize())
		if err := stateLocker.Lock(stateMgr, "state-replace-provider"); err != nil {
			c.Ui.Error(fmt.Sprintf("Error locking source state: %s", err))
			return 1
		}
		defer stateLocker.Unlock(nil)
	}

	// Refresh and load state
	if err := stateMgr.RefreshState(); err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to refresh source state: %s", err))
		return 1
	}

	state := stateMgr.State()
	if state == nil {
		c.Ui.Error(errStateNotFound)
		return 1
	}

	// Fetch all resources from the state
	resources, diags := c.lookupAllResources(state)
	if diags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	var willReplace []*states.Resource

	// Update all matching resources with new provider
	for _, resource := range resources {
		if resource.ProviderConfig.Provider.Equals(from) {
			willReplace = append(willReplace, resource)
		}
	}
	c.showDiagnostics(diags)

	if len(willReplace) == 0 {
		c.Ui.Output("No matching resources found.")
		return 0
	}

	// Explain the changes
	colorize := c.Colorize()
	c.Ui.Output("Terraform will perform the following actions:\n")
	c.Ui.Output(colorize.Color("  [yellow]~[reset] Updating provider:"))
	c.Ui.Output(colorize.Color(fmt.Sprintf("    [red]-[reset] %s", from)))
	c.Ui.Output(colorize.Color(fmt.Sprintf("    [green]+[reset] %s\n", to)))

	c.Ui.Output(colorize.Color(fmt.Sprintf("[bold]Changing[reset] %d resources:\n", len(willReplace))))
	for _, resource := range willReplace {
		c.Ui.Output(colorize.Color(fmt.Sprintf("  %s", resource.Addr)))
	}

	// Confirm
	if !autoApprove {
		c.Ui.Output(colorize.Color(
			"\n[bold]Do you want to make these changes?[reset]\n" +
				"Only 'yes' will be accepted to continue.\n",
		))
		v, err := c.Ui.Ask("Enter a value:")
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Error asking for approval: %s", err))
			return 1
		}
		if v != "yes" {
			c.Ui.Output("Cancelled replacing providers.")
			return 0
		}
	}

	// Update the provider for each resource
	for _, resource := range willReplace {
		resource.ProviderConfig.Provider = to
	}

	// Write the updated state
	if err := stateMgr.WriteState(state); err != nil {
		c.Ui.Error(fmt.Sprintf(errStateRmPersist, err))
		return 1
	}
	if err := stateMgr.PersistState(); err != nil {
		c.Ui.Error(fmt.Sprintf(errStateRmPersist, err))
		return 1
	}

	c.Ui.Output(fmt.Sprintf("\nSuccessfully replaced provider for %d resources.", len(willReplace)))
	return 0
}

func (c *StateReplaceProviderCommand) Help() string {
	helpText := `
Usage: terraform state replace-provider [options] FROM_PROVIDER_FQN TO_PROVIDER_FQN

  Replace provider for resources in the Terraform state.

Options:

  -auto-approve           Skip interactive approval.

  -backup=PATH            Path where Terraform should write the backup for the
						  state file. This can't be disabled. If not set,
						  Terraform will write it to the same path as the state
						  file with a ".backup" extension.

  -lock=true              Lock the state files when locking is supported.

  -lock-timeout=0s        Duration to retry a state lock.

  -state=PATH             Path to the state file to update. Defaults to the
						  configured backend, or "terraform.tfstate"

  -ignore-remote-version  Continue even if remote and local Terraform versions
                          are incompatible. This may result in an unusable
                          workspace, and should be used with extreme caution.

`
	return strings.TrimSpace(helpText)
}

func (c *StateReplaceProviderCommand) Synopsis() string {
	return "Replace provider in the state"
}
