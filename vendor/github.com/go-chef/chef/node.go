package chef

import "fmt"

type NodeService struct {
	client *Client
}

// Node represents the native Go version of the deserialized Node type
type Node struct {
	Name                string                 `json:"name"`
	Environment         string                 `json:"chef_environment,omitempty"`
	ChefType            string                 `json:"chef_type,omitempty"`
	AutomaticAttributes map[string]interface{} `json:"automatic,omitempty"`
	NormalAttributes    map[string]interface{} `json:"normal,omitempty"`
	DefaultAttributes   map[string]interface{} `json:"default,omitempty"`
	OverrideAttributes  map[string]interface{} `json:"override,omitempty"`
	JsonClass           string                 `json:"json_class,omitempty"`
	RunList             []string               `json:"run_list,omitempty"`
}

type NodeResult struct {
	Uri string `json:"uri"`
}

// NewNode is the Node constructor method
func NewNode(name string) (node Node) {
	node = Node{
		Name:        name,
		Environment: "_default",
		ChefType:    "node",
		JsonClass:   "Chef::Node",
	}
	return
}

// List lists the nodes in the Chef server.
//
// Chef API docs: http://docs.opscode.com/api_chef_server.html#id25
func (e *NodeService) List() (data map[string]string, err error) {
	err = e.client.magicRequestDecoder("GET", "nodes", nil, &data)
	return
}

// Get gets a node from the Chef server.
//
// Chef API docs: http://docs.opscode.com/api_chef_server.html#id28
func (e *NodeService) Get(name string) (node Node, err error) {
	url := fmt.Sprintf("nodes/%s", name)
	err = e.client.magicRequestDecoder("GET", url, nil, &node)
	return
}

// Post creates a Node on the chef server
//
// Chef API docs: https://docs.getchef.com/api_chef_server.html#id39
func (e *NodeService) Post(node Node) (data *NodeResult, err error) {
	body, err := JSONReader(node)
	if err != nil {
		return
	}

	err = e.client.magicRequestDecoder("POST", "nodes", body, &data)
	return
}

// Put updates a node on the Chef server.
//
// Chef API docs: http://docs.getchef.com/api_chef_server.html#id42
func (e *NodeService) Put(n Node) (node Node, err error) {
	url := fmt.Sprintf("nodes/%s", n.Name)
	body, err := JSONReader(n)
	if err != nil {
		return
	}

	err = e.client.magicRequestDecoder("PUT", url, body, &node)
	return
}

// Delete removes a node on the Chef server
//
// Chef API docs: https://docs.getchef.com/api_chef_server.html#id40
func (e *NodeService) Delete(name string) (err error) {
	err = e.client.magicRequestDecoder("DELETE", "nodes/"+name, nil, nil)
	return
}
