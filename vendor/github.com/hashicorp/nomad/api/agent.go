package api

import (
	"fmt"
	"net/url"
)

// Agent encapsulates an API client which talks to Nomad's
// agent endpoints for a specific node.
type Agent struct {
	client *Client

	// Cache static agent info
	nodeName   string
	datacenter string
	region     string
}

// KeyringResponse is a unified key response and can be used for install,
// remove, use, as well as listing key queries.
type KeyringResponse struct {
	Messages map[string]string
	Keys     map[string]int
	NumNodes int
}

// KeyringRequest is request objects for serf key operations.
type KeyringRequest struct {
	Key string
}

// Agent returns a new agent which can be used to query
// the agent-specific endpoints.
func (c *Client) Agent() *Agent {
	return &Agent{client: c}
}

// Self is used to query the /v1/agent/self endpoint and
// returns information specific to the running agent.
func (a *Agent) Self() (map[string]map[string]interface{}, error) {
	var out map[string]map[string]interface{}

	// Query the self endpoint on the agent
	_, err := a.client.query("/v1/agent/self", &out, nil)
	if err != nil {
		return nil, fmt.Errorf("failed querying self endpoint: %s", err)
	}

	// Populate the cache for faster queries
	a.populateCache(out)

	return out, nil
}

// populateCache is used to insert various pieces of static
// data into the agent handle. This is used during subsequent
// lookups for the same data later on to save the round trip.
func (a *Agent) populateCache(info map[string]map[string]interface{}) {
	if a.nodeName == "" {
		a.nodeName, _ = info["member"]["Name"].(string)
	}
	if tags, ok := info["member"]["Tags"].(map[string]interface{}); ok {
		if a.datacenter == "" {
			a.datacenter, _ = tags["dc"].(string)
		}
		if a.region == "" {
			a.region, _ = tags["region"].(string)
		}
	}
}

// NodeName is used to query the Nomad agent for its node name.
func (a *Agent) NodeName() (string, error) {
	// Return from cache if we have it
	if a.nodeName != "" {
		return a.nodeName, nil
	}

	// Query the node name
	_, err := a.Self()
	return a.nodeName, err
}

// Datacenter is used to return the name of the datacenter which
// the agent is a member of.
func (a *Agent) Datacenter() (string, error) {
	// Return from cache if we have it
	if a.datacenter != "" {
		return a.datacenter, nil
	}

	// Query the agent for the DC
	_, err := a.Self()
	return a.datacenter, err
}

// Region is used to look up the region the agent is in.
func (a *Agent) Region() (string, error) {
	// Return from cache if we have it
	if a.region != "" {
		return a.region, nil
	}

	// Query the agent for the region
	_, err := a.Self()
	return a.region, err
}

// Join is used to instruct a server node to join another server
// via the gossip protocol. Multiple addresses may be specified.
// We attempt to join all of the hosts in the list. Returns the
// number of nodes successfully joined and any error. If one or
// more nodes have a successful result, no error is returned.
func (a *Agent) Join(addrs ...string) (int, error) {
	// Accumulate the addresses
	v := url.Values{}
	for _, addr := range addrs {
		v.Add("address", addr)
	}

	// Send the join request
	var resp joinResponse
	_, err := a.client.write("/v1/agent/join?"+v.Encode(), nil, &resp, nil)
	if err != nil {
		return 0, fmt.Errorf("failed joining: %s", err)
	}
	if resp.Error != "" {
		return 0, fmt.Errorf("failed joining: %s", resp.Error)
	}
	return resp.NumJoined, nil
}

// Members is used to query all of the known server members
func (a *Agent) Members() (*ServerMembers, error) {
	var resp *ServerMembers

	// Query the known members
	_, err := a.client.query("/v1/agent/members", &resp, nil)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// ForceLeave is used to eject an existing node from the cluster.
func (a *Agent) ForceLeave(node string) error {
	_, err := a.client.write("/v1/agent/force-leave?node="+node, nil, nil, nil)
	return err
}

// Servers is used to query the list of servers on a client node.
func (a *Agent) Servers() ([]string, error) {
	var resp []string
	_, err := a.client.query("/v1/agent/servers", &resp, nil)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// SetServers is used to update the list of servers on a client node.
func (a *Agent) SetServers(addrs []string) error {
	// Accumulate the addresses
	v := url.Values{}
	for _, addr := range addrs {
		v.Add("address", addr)
	}

	_, err := a.client.write("/v1/agent/servers?"+v.Encode(), nil, nil, nil)
	return err
}

// ListKeys returns the list of installed keys
func (a *Agent) ListKeys() (*KeyringResponse, error) {
	var resp KeyringResponse
	_, err := a.client.query("/v1/agent/keyring/list", &resp, nil)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// InstallKey installs a key in the keyrings of all the serf members
func (a *Agent) InstallKey(key string) (*KeyringResponse, error) {
	args := KeyringRequest{
		Key: key,
	}
	var resp KeyringResponse
	_, err := a.client.write("/v1/agent/keyring/install", &args, &resp, nil)
	return &resp, err
}

// UseKey uses a key from the keyring of serf members
func (a *Agent) UseKey(key string) (*KeyringResponse, error) {
	args := KeyringRequest{
		Key: key,
	}
	var resp KeyringResponse
	_, err := a.client.write("/v1/agent/keyring/use", &args, &resp, nil)
	return &resp, err
}

// RemoveKey removes a particular key from keyrings of serf members
func (a *Agent) RemoveKey(key string) (*KeyringResponse, error) {
	args := KeyringRequest{
		Key: key,
	}
	var resp KeyringResponse
	_, err := a.client.write("/v1/agent/keyring/remove", &args, &resp, nil)
	return &resp, err
}

// joinResponse is used to decode the response we get while
// sending a member join request.
type joinResponse struct {
	NumJoined int    `json:"num_joined"`
	Error     string `json:"error"`
}

type ServerMembers struct {
	ServerName string
	Region     string
	DC         string
	Members    []*AgentMember
}

// AgentMember represents a cluster member known to the agent
type AgentMember struct {
	Name        string
	Addr        string
	Port        uint16
	Tags        map[string]string
	Status      string
	ProtocolMin uint8
	ProtocolMax uint8
	ProtocolCur uint8
	DelegateMin uint8
	DelegateMax uint8
	DelegateCur uint8
}

// AgentMembersNameSort implements sort.Interface for []*AgentMembersNameSort
// based on the Name, DC and Region
type AgentMembersNameSort []*AgentMember

func (a AgentMembersNameSort) Len() int      { return len(a) }
func (a AgentMembersNameSort) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a AgentMembersNameSort) Less(i, j int) bool {
	if a[i].Tags["region"] != a[j].Tags["region"] {
		return a[i].Tags["region"] < a[j].Tags["region"]
	}

	if a[i].Tags["dc"] != a[j].Tags["dc"] {
		return a[i].Tags["dc"] < a[j].Tags["dc"]
	}

	return a[i].Name < a[j].Name

}
