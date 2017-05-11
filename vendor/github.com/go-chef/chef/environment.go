package chef

import "fmt"
import "sort"

// Environment has a Reader, hey presto
type EnvironmentService struct {
	client *Client
}

type EnvironmentResult map[string]string

// Environment represents the native Go version of the deserialized Environment type
type Environment struct {
	Name               string            `json:"name"`
	Description        string            `json:"description"`
	ChefType           string            `json:"chef_type"`
	Attributes         interface{}       `json:"attributes,omitempty"`
	DefaultAttributes  interface{}       `json:"default_attributes,omitempty"`
	OverrideAttributes interface{}       `json:"override_attributes,omitempty"`
	JsonClass          string            `json:"json_class,omitempty"`
	CookbookVersions   map[string]string `json:"cookbook_versions"`
}

type EnvironmentCookbookResult map[string]CookbookVersions

func strMapToStr(e map[string]string) (out string) {
	keys := make([]string, len(e))
	for k, _ := range e {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if k == "" {
			continue
		}
		out += fmt.Sprintf("%s => %s\n", k, e[k])
	}
	return
}

// String makes EnvironmentResult implement the string result
func (e EnvironmentResult) String() (out string) {
	return strMapToStr(e)
}

// List lists the environments in the Chef server.
//
// Chef API docs: http://docs.getchef.com/api_chef_server.html#id14
func (e *EnvironmentService) List() (data *EnvironmentResult, err error) {
	err = e.client.magicRequestDecoder("GET", "environments", nil, &data)
	return
}

// Create an environment in the Chef server.
//
// Chef API docs: http://docs.getchef.com/api_chef_server.html#id15
func (e *EnvironmentService) Create(environment *Environment) (data *EnvironmentResult, err error) {
	body, err := JSONReader(environment)
	if err != nil {
		return
	}

	err = e.client.magicRequestDecoder("POST", "environments", body, &data)
	return
}

// Delete an environment from the Chef server.
//
// Chef API docs: http://docs.getchef.com/api_chef_server.html#id16

// Get gets an environment from the Chef server.
//
// Chef API docs: http://docs.getchef.com/api_chef_server.html#id17
func (e *EnvironmentService) Get(name string) (data *Environment, err error) {
	path := fmt.Sprintf("environments/%s", name)
	err = e.client.magicRequestDecoder("GET", path, nil, &data)
	return
}

// Write an environment to the Chef server.
//
// Chef API docs: http://docs.getchef.com/api_chef_server.html#id18
func (e *EnvironmentService) Put(environment *Environment) (data *Environment, err error) {
	path := fmt.Sprintf("environments/%s", environment.Name)
	body, err := JSONReader(environment)
	if err != nil {
		return
	}

	err = e.client.magicRequestDecoder("PUT", path, body, &data)
	return
}

// Get the versions of a cookbook for this environment from the Chef server.
//
// Chef API docs: http://docs.getchef.com/api_chef_server.html#id19
func (e *EnvironmentService) ListCookbooks(name string, numVersions string) (data EnvironmentCookbookResult, err error) {
  path := versionParams(fmt.Sprintf("environments/%s/cookbooks", name), numVersions)
	err = e.client.magicRequestDecoder("GET", path, nil, &data)
	return
}

// Get a hash of cookbooks and cookbook versions (including all dependencies) that
// are required by the run_list array. Version constraints may be specified using
// the @ symbol after the cookbook name as a delimiter. Version constraints may also
// be present when the cookbook_versions attributes is specified for an environment
// or when dependencies are specified by a cookbook.
//
// Chef API docs: http://docs.getchef.com/api_chef_server.html#id20

// Get a list of cookbooks and cookbook versions that are available to the specified environment.
//
// Chef API docs: http://docs.getchef.com/api_chef_server.html#id21
