package command

import (
	"fmt"
	"strings"
)

// ZeroThirteenUpgradeCommand upgrades configuration files for a module
// to include explicit provider source settings
type ZeroThirteenUpgradeCommand struct {
	Meta
}

func (c *ZeroThirteenUpgradeCommand) Run(args []string) int {
	c.Ui.Output(fmt.Sprintf(`
The 0.13upgrade command has been removed. You must run this command with
Terraform v0.13 to upgrade your provider requirements before upgrading to the
current version.`))
	return 0
}

func (c *ZeroThirteenUpgradeCommand) Help() string {
	helpText := `
Usage: terraform 0.13upgrade

  The 0.13upgrade command has been removed. You must run this command with
  Terraform v0.13 to upgrade your provider requirements before upgrading to
  the current version.
`
	return strings.TrimSpace(helpText)
}

func (c *ZeroThirteenUpgradeCommand) Synopsis() string {
	return "Rewrites pre-0.13 module source code for v0.13"
}
