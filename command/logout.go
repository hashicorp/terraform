package command

import (
	"fmt"
	"path/filepath"
	"strings"

	svchost "github.com/hashicorp/terraform-svchost"
	"github.com/hashicorp/terraform/command/cliconfig"
	"github.com/hashicorp/terraform/tfdiags"
)

// LogoutCommand is a Command implementation which removes stored credentials
// for a remote service host.
type LogoutCommand struct {
	Meta
}

// Run implements cli.Command.
func (c *LogoutCommand) Run(args []string) int {
	args, err := c.Meta.process(args, false)
	if err != nil {
		return 1
	}

	cmdFlags := c.Meta.defaultFlagSet("logout")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	args = cmdFlags.Args()
	if len(args) > 1 {
		c.Ui.Error(
			"The logout command expects at most one argument: the host to log out of.")
		cmdFlags.Usage()
		return 1
	}

	var diags tfdiags.Diagnostics

	givenHostname := "app.terraform.io"
	if len(args) != 0 {
		givenHostname = args[0]
	}

	hostname, err := svchost.ForComparison(givenHostname)
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Invalid hostname",
			fmt.Sprintf("The given hostname %q is not valid: %s.", givenHostname, err.Error()),
		))
		c.showDiagnostics(diags)
		return 1
	}

	// From now on, since we've validated the given hostname, we should use
	// dispHostname in the UI to ensure we're presenting it in the canonical
	// form, in case that helps users with debugging when things aren't
	// working as expected. (Perhaps the normalization is part of the cause.)
	dispHostname := hostname.ForDisplay()

	creds := c.Services.CredentialsSource().(*cliconfig.CredentialsSource)
	filename, _ := creds.CredentialsFilePath()
	credsCtx := &loginCredentialsContext{
		Location:      creds.HostCredentialsLocation(hostname),
		LocalFilename: filename, // empty in the very unlikely event that we can't select a config directory for this user
		HelperType:    creds.CredentialsHelperType(),
	}

	if credsCtx.Location == cliconfig.CredentialsInOtherFile {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			fmt.Sprintf("Credentials for %s are manually configured", dispHostname),
			"The \"terraform logout\" command cannot log out because credentials for this host are manually configured in a CLI configuration file.\n\nTo log out, revoke the existing credentials and remove that block from the CLI configuration.",
		))
	}

	if diags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	// credsCtx might not be set if we're using a mock credentials source
	// in a test, but it should always be set in normal use.
	if credsCtx != nil {
		switch credsCtx.Location {
		case cliconfig.CredentialsNotAvailable:
			c.Ui.Output(fmt.Sprintf("No credentials for %s are stored.\n", dispHostname))
			return 0
		case cliconfig.CredentialsViaHelper:
			c.Ui.Output(fmt.Sprintf("Removing the stored credentials for %s from the configured\n%q credentials helper.\n", dispHostname, credsCtx.HelperType))
		case cliconfig.CredentialsInPrimaryFile:
			c.Ui.Output(fmt.Sprintf("Removing the stored credentials for %s from the following file:\n    %s\n", dispHostname, credsCtx.LocalFilename))
		}
	}

	err = creds.ForgetForHost(hostname)
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to remove API token",
			fmt.Sprintf("Unable to remove stored API token: %s", err),
		))
	}

	c.showDiagnostics(diags)
	if diags.HasErrors() {
		return 1
	}

	c.Ui.Output(
		fmt.Sprintf(
			c.Colorize().Color(strings.TrimSpace(`
[green][bold]Success![reset] [bold]Terraform has removed the stored API token for %s.[reset]
`)),
			dispHostname,
		) + "\n",
	)

	return 0
}

// Help implements cli.Command.
func (c *LogoutCommand) Help() string {
	defaultFile := c.defaultOutputFile()
	if defaultFile == "" {
		// Because this is just for the help message and it's very unlikely
		// that a user wouldn't have a functioning home directory anyway,
		// we'll just use a placeholder here. The real command has some
		// more complex behavior for this case. This result is not correct
		// on all platforms, but given how unlikely we are to hit this case
		// that seems okay.
		defaultFile = "~/.terraform/credentials.tfrc.json"
	}

	helpText := `
Usage: terraform logout [hostname]

  Removes locally-stored credentials for specified hostname.

  Note: the API token is only removed from local storage, not destroyed on the
  remote server, so it will remain valid until manually revoked.

  If no hostname is provided, the default hostname is app.terraform.io.
      %s
`
	return strings.TrimSpace(helpText)
}

// Synopsis implements cli.Command.
func (c *LogoutCommand) Synopsis() string {
	return "Remove locally-stored credentials for a remote host"
}

func (c *LogoutCommand) defaultOutputFile() string {
	if c.CLIConfigDir == "" {
		return "" // no default available
	}
	return filepath.Join(c.CLIConfigDir, "credentials.tfrc.json")
}
