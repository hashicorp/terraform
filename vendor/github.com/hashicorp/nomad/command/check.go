package command

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

const (
	HealthCritical = 2
	HealthWarn     = 1
	HealthPass     = 0
	HealthUnknown  = 3
)

type AgentCheckCommand struct {
	Meta
}

func (c *AgentCheckCommand) Help() string {
	helpText := `
Usage: nomad check

  Display state of the Nomad agent. The exit code of the command is Nagios
  compatible and could be used with alerting systems.

General Options:

  ` + generalOptionsUsage() + `

Agent Check Options:

  -min-peers
     Minimum number of peers that a server is expected to know.

  -min-servers
     Minumum number of servers that a client is expected to know.
`

	return strings.TrimSpace(helpText)
}

func (c *AgentCheckCommand) Synopsis() string {
	return "Displays health of the local Nomad agent"
}

func (c *AgentCheckCommand) Run(args []string) int {
	var minPeers, minServers int

	flags := c.Meta.FlagSet("check", FlagSetClient)
	flags.Usage = func() { c.Ui.Output(c.Help()) }
	flags.IntVar(&minPeers, "min-peers", 0, "")
	flags.IntVar(&minServers, "min-servers", 1, "")

	if err := flags.Parse(args); err != nil {
		return 1
	}

	client, err := c.Meta.Client()
	if err != nil {
		c.Ui.Error(fmt.Sprintf("error initializing client: %s", err))
		return HealthCritical
	}

	info, err := client.Agent().Self()
	if err != nil {
		c.Ui.Output(fmt.Sprintf("unable to query agent info: %v", err))
		return HealthCritical
	}
	if stats, ok := info["stats"]; !ok && (reflect.TypeOf(stats).Kind() == reflect.Map) {
		c.Ui.Error("error getting stats from the agent api")
		return 1
	}
	if _, ok := info["stats"]["nomad"]; ok {
		return c.checkServerHealth(info["stats"], minPeers)
	}

	if _, ok := info["stats"]["client"]; ok {
		return c.checkClientHealth(info["stats"], minServers)
	}
	return HealthWarn
}

// checkServerHealth returns the health of a server.
// TODO Add more rules for determining server health
func (c *AgentCheckCommand) checkServerHealth(info map[string]interface{}, minPeers int) int {
	raft := info["raft"].(map[string]interface{})
	knownPeers, err := strconv.Atoi(raft["num_peers"].(string))
	if err != nil {
		c.Ui.Output(fmt.Sprintf("unable to get known peers: %v", err))
		return HealthCritical
	}

	if knownPeers < minPeers {
		c.Ui.Output(fmt.Sprintf("known peers: %v, is less than expected number of peers: %v", knownPeers, minPeers))
		return HealthCritical
	}
	return HealthPass
}

// checkClientHealth returns the health of a client
func (c *AgentCheckCommand) checkClientHealth(info map[string]interface{}, minServers int) int {
	clientStats := info["client"].(map[string]interface{})
	knownServers, err := strconv.Atoi(clientStats["known_servers"].(string))
	if err != nil {
		c.Ui.Output(fmt.Sprintf("unable to get known servers: %v", err))
		return HealthCritical
	}

	heartbeatTTL, err := time.ParseDuration(clientStats["heartbeat_ttl"].(string))
	if err != nil {
		c.Ui.Output(fmt.Sprintf("unable to parse heartbeat TTL: %v", err))
		return HealthCritical
	}

	lastHeartbeat, err := time.ParseDuration(clientStats["last_heartbeat"].(string))
	if err != nil {
		c.Ui.Output(fmt.Sprintf("unable to parse last heartbeat: %v", err))
		return HealthCritical
	}

	if lastHeartbeat > heartbeatTTL {
		c.Ui.Output(fmt.Sprintf("last heartbeat was %q time ago, expected heartbeat ttl: %q", lastHeartbeat, heartbeatTTL))
		return HealthCritical
	}

	if knownServers < minServers {
		c.Ui.Output(fmt.Sprintf("known servers: %v, is less than expected number of servers: %v", knownServers, minServers))
		return HealthCritical
	}

	return HealthPass
}
