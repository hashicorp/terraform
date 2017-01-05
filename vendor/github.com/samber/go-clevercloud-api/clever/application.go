package clever

import (
	"fmt"
	"strings"
)


//
// TYPES
//
type InstanceSize struct {
	Name   string `json:"name"`
	Memory int    `json:"mem"`
	Cpus   int    `json:"cpus"`
}
type InstanceRuntime struct {
	Id         string `json:"id"`
	Slug       string `json:"slug"`
	Name       string `json:"name"`
	DeployType string `json:"deployType"`
}

type ApplicationInput struct {
	Name             string `json:"name"`
	Description      string `json:"description,omitempty"`
	Region           string `json:"zone"`
	Deploy           string `json:"deploy"`
	CancelOnPush     bool   `json:"cancelOnPush,omitempty"`
	SeparateBuild    bool   `json:"separateBuild,omitempty"`
	StickySessions   bool   `json:"stickySessions,omitempty"`
	Homogeneous      bool   `json:"homogeneous,omitempty"`
	InstanceRuntime  string `json:"instanceType"`
	InstanceVariant  string `json:"instanceVariant"`
	InstanceVersion  string `json:"instanceVersion"`
	InstanceSizeMin  string `json:"minFlavor,omitempty"`
	InstanceSizeMax  string `json:"maxFlavor,omitempty"`
	InstanceCountMin int    `json:"minInstances"`
	InstanceCountMax int    `json:"maxInstances"`
}

type ApplicationOutput struct {
	Id             string                      `json:"id"`
	Name           string                      `json:"name"`
	Description    string                      `json:"description,omitempty"`
	Region         string                      `json:"zone"`
	Instance       ApplicationOutputInstance   `json:"instance"`
	Deployment     ApplicationOutputDeployment `json:"deployment"`
	Fqdns          []Fqdn                      `json:"vhosts"`
	StickySessions bool                        `json:"stickySessions,omitempty"`
	CancelOnPush   bool                        `json:"cancelOnPush,omitempty"`
	SeparateBuild  bool                        `json:"separateBuild,omitempty"`
	Homogeneous    bool                        `json:"homogeneous,omitempty"`
	OwnerId        string                      `json:"ownerId"`
	State          string                      `json:"state"`
	Branch         string                      `json:"branch,omitempty"`
	CommitId       string                      `json:"commitId,omitempty"`
}
type ApplicationOutputInstance struct {
	Version            string          `json:"version"`
	InstanceRuntime    InstanceRuntime `json:"variant"`
	InstanceCountMin   int             `json:"minInstances"`
	InstanceCountMax   int             `json:"maxInstances"`
	InstanceSizeMin    InstanceSize    `json:"minFlavor,omitempty"`
	InstanceSizeMax    InstanceSize    `json:"maxFlavor,omitempty"`
	InstanceAndVersion string          `json:"instanceAndVersion,omitempty"`
}
type ApplicationOutputDeployment struct {
	Type      string `json:"type"`
	RepoState string `json:"repoState"`
	SshUrl    string `json:"url"`
	HttpUrl   string `json:"httpUrl"`
}


//
// INIT
//
type AvailableInstance struct {
	Name            string          `json:"name"`
	Deployments     []string        `json:"deployments"`
	Size            []InstanceSize  `json:"flavors"`
	InstanceRuntime InstanceRuntime `json:"variant"`
	Version         string          `json:"version"`
	DefaultSize     InstanceSize    `json:"defaultFlavor"`
}

var AVAILABLE_INSTANCE []*AvailableInstance
var MATCHING_INSTANCE_FLAVOR map[string]map[string]string
var MATCHING_INSTANCE_RUNTIME map[string]*AvailableInstance

func (c *Client) loadApplicationInstances() error {
	if err := c.jsonRequest("GET", "/products/instances", nil, &AVAILABLE_INSTANCE); err != nil {
		return err
	}

	MATCHING_INSTANCE_FLAVOR = map[string]map[string]string{}
	MATCHING_INSTANCE_RUNTIME = map[string]*AvailableInstance{}
	for _, instance := range AVAILABLE_INSTANCE {
		MATCHING_INSTANCE_FLAVOR[instance.InstanceRuntime.Slug] = map[string]string{}
		MATCHING_INSTANCE_RUNTIME[instance.InstanceRuntime.Slug] = instance
		for _, size := range instance.Size {
			MATCHING_INSTANCE_FLAVOR[instance.InstanceRuntime.Slug][strings.ToLower(size.Name)] = size.Name
		}
	}

	return nil
}


//
// APPS
//
func (c *Client) GetApplicationById(app_id string) (*ApplicationOutput, error) {
	var appOutput ApplicationOutput
	err := c.get("/organisations/"+c.config.OrgId+"/applications/"+app_id, &appOutput)
	if err != nil {
		return nil, err
	}

	return &appOutput, nil
}

func (c *Client) CreateApplication(appInput *ApplicationInput) (*ApplicationOutput, error) {
	if err := prepareApplicationInput(appInput); err != nil {
		return nil, err
	}

	var appOutput ApplicationOutput
	err := c.post("/organisations/"+c.config.OrgId+"/applications", appInput, &appOutput)
	if err != nil {
		return nil, err
	}

	return &appOutput, nil
}

func (c *Client) UpdateApplication(app_id string, appInput *ApplicationInput) (*ApplicationOutput, error) {
	if err := prepareApplicationInput(appInput); err != nil {
		return nil, err
	}

	var appOutput ApplicationOutput
	err := c.put("/organisations/"+c.config.OrgId+"/applications/"+app_id, appInput, &appOutput)
	if err != nil {
		return nil, err
	}

	return &appOutput, nil
}

func (c *Client) DeleteApplication(app_id string) error {
	return c.delete("/organisations/" + c.config.OrgId + "/applications/" + app_id)
}

func prepareApplicationInput(appInput *ApplicationInput) error {
	// Set instance variant id and version according to InstanceRuntime field
	if instanceRuntime, ok := MATCHING_INSTANCE_RUNTIME[strings.ToLower(appInput.InstanceRuntime)]; ok == false {
		return fmt.Errorf("Incorrect instance type:" + appInput.InstanceRuntime)
	} else {
		appInput.InstanceVariant = instanceRuntime.InstanceRuntime.Id
		appInput.InstanceVersion = instanceRuntime.Version
	}

	// Get the instance size (go-clevercloud-api accept "XS" or "xs" params)
	// If instance size is not specified, then we use the default flavor
	if appInput.InstanceSizeMin != "" {
		appInput.InstanceSizeMin = MATCHING_INSTANCE_FLAVOR[strings.ToLower(appInput.InstanceRuntime)][strings.ToLower(appInput.InstanceSizeMin)]
	} else {
		appInput.InstanceSizeMin = MATCHING_INSTANCE_RUNTIME[strings.ToLower(appInput.InstanceRuntime)].DefaultSize.Name
	}
	if appInput.InstanceSizeMax != "" {
		appInput.InstanceSizeMax = MATCHING_INSTANCE_FLAVOR[strings.ToLower(appInput.InstanceRuntime)][strings.ToLower(appInput.InstanceSizeMax)]
	} else {
		appInput.InstanceSizeMax = MATCHING_INSTANCE_RUNTIME[strings.ToLower(appInput.InstanceRuntime)].DefaultSize.Name
	}

	// ex: play2 is a java instance actually
	appInput.InstanceRuntime = MATCHING_INSTANCE_RUNTIME[strings.ToLower(appInput.InstanceRuntime)].InstanceRuntime.DeployType

	return nil
}


//
// APP ENV VARS
//
func (c *Client) GetApplicationEnvById(app_id string) (map[string]string, error) {
	var envOutput []Env
	err := c.get("/organisations/"+c.config.OrgId+"/applications/"+app_id+"/env", &envOutput)
	if err != nil {
		return nil, err
	}

	returnedKv := map[string]string{}
	for _, output := range envOutput {
		returnedKv[output.Key] = output.Value
	}

	return returnedKv, nil
}

func (c *Client) CreateApplicationEnv(app_id string, kv map[string]string) (map[string]string, error) {
	return c.UpdateApplicationEnv(app_id, kv)
}

func (c *Client) UpdateApplicationEnv(app_id string, kv map[string]string) (map[string]string, error) {
	for k, v := range kv {
		env := Env{
			Key:   k,
			Value: v,
		}
		err := c.put("/organisations/"+c.config.OrgId+"/applications/"+app_id+"/env/"+k, &env, nil)
		if err != nil {
			return nil, err
		}
	}

	return kv, nil
}

func (c *Client) DeleteApplicationEnv(app_id string, key string) error {
	return c.delete("/organisations/" + c.config.OrgId + "/applications/" + app_id + "/env/" + key)
}


//
// APP DNS MANAGEMENT
//
func (c *Client) GetApplicationFqdnById(app_id string) ([]string, error) {
	var fqdns []Fqdn
	err := c.get("/organisations/"+c.config.OrgId+"/applications/"+app_id+"/vhosts", &fqdns)
	if err != nil {
		return nil, err
	}

	returnedFqdns := []string{}
	for _, output := range fqdns {
		returnedFqdns = append(returnedFqdns, output.Fqdn)
	}

	return returnedFqdns, nil
}

func (c *Client) CreateApplicationFqdn(app_id string, fqdns []string) ([]string, error) {
	return c.UpdateApplicationFqdn(app_id, fqdns)
}

func (c *Client) UpdateApplicationFqdn(app_id string, fqdns []string) ([]string, error) {
	for _, fqdn := range fqdns {
		err := c.put("/organisations/"+c.config.OrgId+"/applications/"+app_id+"/vhosts/"+fqdn, nil, nil)
		if err != nil {
			return nil, err
		}
	}

	return fqdns, nil
}

func (c *Client) DeleteApplicationFqdn(app_id string, fqdn string) error {
	return c.delete("/organisations/" + c.config.OrgId + "/applications/" + app_id + "/vhosts/" + fqdn)
}

/*
 * Deploy
 */
func (c *Client) RestartApplication(app_id string) error {
	return c.post("/organisations/"+c.config.OrgId+"/applications/"+app_id+"/instances", nil, nil)
}
