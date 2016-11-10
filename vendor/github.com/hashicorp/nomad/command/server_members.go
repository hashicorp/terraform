package command

import (
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/nomad/api"
	"github.com/ryanuber/columnize"
)

type ServerMembersCommand struct {
	Meta
}

func (c *ServerMembersCommand) Help() string {
	helpText := `
Usage: nomad server-members [options]

  Display a list of the known servers and their status.

General Options:

  ` + generalOptionsUsage() + `

Server Members Options:

  -detailed
    Show detailed information about each member. This dumps
    a raw set of tags which shows more information than the
    default output format.
`
	return strings.TrimSpace(helpText)
}

func (c *ServerMembersCommand) Synopsis() string {
	return "Display a list of known servers and their status"
}

func (c *ServerMembersCommand) Run(args []string) int {
	var detailed bool

	flags := c.Meta.FlagSet("server-members", FlagSetClient)
	flags.Usage = func() { c.Ui.Output(c.Help()) }
	flags.BoolVar(&detailed, "detailed", false, "Show detailed output")

	if err := flags.Parse(args); err != nil {
		return 1
	}

	// Check for extra arguments
	args = flags.Args()
	if len(args) != 0 {
		c.Ui.Error(c.Help())
		return 1
	}

	// Get the HTTP client
	client, err := c.Meta.Client()
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error initializing client: %s", err))
		return 1
	}

	// Query the members
	mem, err := client.Agent().Members()
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error querying servers: %s", err))
		return 1
	}

	// Sort the members
	sort.Sort(api.AgentMembersNameSort(mem))

	// Determine the leaders per region.
	leaders, err := regionLeaders(client, mem)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error determining leaders: %s", err))
		return 1
	}

	// Format the list
	var out []string
	if detailed {
		out = detailedOutput(mem)
	} else {
		out = standardOutput(mem, leaders)
	}

	// Dump the list
	c.Ui.Output(columnize.SimpleFormat(out))
	return 0
}

func standardOutput(mem []*api.AgentMember, leaders map[string]string) []string {
	// Format the members list
	members := make([]string, len(mem)+1)
	members[0] = "Name|Address|Port|Status|Leader|Protocol|Build|Datacenter|Region"
	for i, member := range mem {
		reg := member.Tags["region"]
		regLeader, ok := leaders[reg]
		isLeader := false
		if ok {
			if regLeader == fmt.Sprintf("%s:%s", member.Addr, member.Tags["port"]) {

				isLeader = true
			}
		}

		members[i+1] = fmt.Sprintf("%s|%s|%d|%s|%t|%d|%s|%s|%s",
			member.Name,
			member.Addr,
			member.Port,
			member.Status,
			isLeader,
			member.ProtocolCur,
			member.Tags["build"],
			member.Tags["dc"],
			member.Tags["region"])
	}
	return members
}

func detailedOutput(mem []*api.AgentMember) []string {
	// Format the members list
	members := make([]string, len(mem)+1)
	members[0] = "Name|Address|Port|Tags"
	for i, member := range mem {
		// Format the tags
		tagPairs := make([]string, 0, len(member.Tags))
		for k, v := range member.Tags {
			tagPairs = append(tagPairs, fmt.Sprintf("%s=%s", k, v))
		}
		tags := strings.Join(tagPairs, ",")

		members[i+1] = fmt.Sprintf("%s|%s|%d|%s",
			member.Name,
			member.Addr,
			member.Port,
			tags)
	}
	return members
}

// regionLeaders returns a map of regions to the IP of the member that is the
// leader.
func regionLeaders(client *api.Client, mem []*api.AgentMember) (map[string]string, error) {
	// Determine the unique regions.
	leaders := make(map[string]string)
	regions := make(map[string]struct{})
	for _, m := range mem {
		regions[m.Tags["region"]] = struct{}{}
	}

	if len(regions) == 0 {
		return leaders, nil
	}

	status := client.Status()
	for reg := range regions {
		l, err := status.RegionLeader(reg)
		if err != nil {
			// This error means that region has no leader.
			if strings.Contains(err.Error(), "No cluster leader") {
				continue
			}
			return nil, err
		}

		leaders[reg] = l
	}

	return leaders, nil
}
