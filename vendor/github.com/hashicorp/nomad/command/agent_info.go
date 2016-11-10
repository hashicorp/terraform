package command

import (
	"fmt"
	"sort"
	"strings"
)

type AgentInfoCommand struct {
	Meta
}

func (c *AgentInfoCommand) Help() string {
	helpText := `
Usage: nomad agent-info [options]

  Display status information about the local agent.

General Options:

  ` + generalOptionsUsage()
	return strings.TrimSpace(helpText)
}

func (c *AgentInfoCommand) Synopsis() string {
	return "Display status information about the local agent"
}

func (c *AgentInfoCommand) Run(args []string) int {
	flags := c.Meta.FlagSet("agent-info", FlagSetClient)
	flags.Usage = func() { c.Ui.Output(c.Help()) }
	if err := flags.Parse(args); err != nil {
		return 1
	}

	// Check that we either got no jobs or exactly one.
	args = flags.Args()
	if len(args) > 0 {
		c.Ui.Error(c.Help())
		return 1
	}

	// Get the HTTP client
	client, err := c.Meta.Client()
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error initializing client: %s", err))
		return 1
	}

	// Query the agent info
	info, err := client.Agent().Self()
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error querying agent info: %s", err))
		return 1
	}

	// Sort and output agent info
	var stats map[string]interface{}
	stats, _ = info["stats"]
	statsKeys := make([]string, 0, len(stats))
	for key := range stats {
		statsKeys = append(statsKeys, key)
	}
	sort.Strings(statsKeys)

	for _, key := range statsKeys {
		c.Ui.Output(key)
		statsData, _ := stats[key].(map[string]interface{})
		statsDataKeys := make([]string, len(statsData))
		i := 0
		for key := range statsData {
			statsDataKeys[i] = key
			i++
		}
		sort.Strings(statsDataKeys)

		for _, key := range statsDataKeys {
			c.Ui.Output(fmt.Sprintf("  %s = %v", key, statsData[key]))
		}
	}

	return 0
}
