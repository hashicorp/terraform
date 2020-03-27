package command

import "fmt"

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
