package command

import (
	"fmt"
	"strings"
)

type ServerJoinCommand struct {
	Meta
}

func (c *ServerJoinCommand) Help() string {
	helpText := `
Usage: nomad server-join [options] <addr> [<addr>...]

  Joins the local server to one or more Nomad servers. Joining is
  only required for server nodes, and only needs to succeed
  against one or more of the provided addresses. Once joined, the
  gossip layer will handle discovery of the other server nodes in
  the cluster.

General Options:

  ` + generalOptionsUsage()
	return strings.TrimSpace(helpText)
}

func (c *ServerJoinCommand) Synopsis() string {
	return "Join server nodes together"
}

func (c *ServerJoinCommand) Run(args []string) int {
	flags := c.Meta.FlagSet("server-join", FlagSetClient)
	flags.Usage = func() { c.Ui.Output(c.Help()) }
	if err := flags.Parse(args); err != nil {
		return 1
	}

	// Check that we got at least one node
	args = flags.Args()
	if len(args) < 1 {
		c.Ui.Error(c.Help())
		return 1
	}
	nodes := args

	// Get the HTTP client
	client, err := c.Meta.Client()
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error initializing client: %s", err))
		return 1
	}

	// Attempt the join
	n, err := client.Agent().Join(nodes...)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error joining: %s", err))
		return 1
	}

	// Success
	c.Ui.Output(fmt.Sprintf("Joined %d servers successfully", n))
	return 0
}
