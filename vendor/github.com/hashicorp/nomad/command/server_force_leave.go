package command

import (
	"fmt"
	"strings"
)

type ServerForceLeaveCommand struct {
	Meta
}

func (c *ServerForceLeaveCommand) Help() string {
	helpText := `
Usage: nomad server-force-leave [options] <node>

  Forces an server to enter the "left" state. This can be used to
  eject nodes which have failed and will not rejoin the cluster.
  Note that if the member is actually still alive, it will
  eventually rejoin the cluster again.

General Options:

  ` + generalOptionsUsage()
	return strings.TrimSpace(helpText)
}

func (c *ServerForceLeaveCommand) Synopsis() string {
	return "Force a server into the 'left' state"
}

func (c *ServerForceLeaveCommand) Run(args []string) int {
	flags := c.Meta.FlagSet("server-force-leave", FlagSetClient)
	flags.Usage = func() { c.Ui.Output(c.Help()) }
	if err := flags.Parse(args); err != nil {
		return 1
	}

	// Check that we got exactly one node
	args = flags.Args()
	if len(args) != 1 {
		c.Ui.Error(c.Help())
		return 1
	}
	node := args[0]

	// Get the HTTP client
	client, err := c.Meta.Client()
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error initializing client: %s", err))
		return 1
	}

	// Call force-leave on the node
	if err := client.Agent().ForceLeave(node); err != nil {
		c.Ui.Error(fmt.Sprintf("Error force-leaving server %s: %s", node, err))
		return 1
	}

	return 0
}
