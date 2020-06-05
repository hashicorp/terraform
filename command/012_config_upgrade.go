package command

import (
	"fmt"
	"strings"
)

type ZeroTwelveUpgradeCommand struct {
	Meta
}

func (c *ZeroTwelveUpgradeCommand) Run(args []string) int {
	c.Ui.Output(fmt.Sprintf(`
The 0.12upgrade command is deprecated. You must run this command with Terraform
v0.12 to upgrade your configuration syntax before upgrading to the current
version.`))
	return 0
}

func (c *ZeroTwelveUpgradeCommand) Help() string {
	helpText := `
Usage: terraform 0.12upgrade

  The 0.12upgrade command is deprecated. You must run this command with
  Terraform v0.12 to upgrade your configuration syntax before upgrading to
  the current version.
`
	return strings.TrimSpace(helpText)
}

func (c *ZeroTwelveUpgradeCommand) Synopsis() string {
	return "Rewrites pre-0.12 module source code for v0.12"
}
