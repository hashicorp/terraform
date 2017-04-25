package client

const (
	RACKSPACE_CONFIG_TYPE = "rackspaceConfig"
)

type RackspaceConfig struct {
	Resource

	ApiKey string `json:"apiKey,omitempty" yaml:"api_key,omitempty"`

	DockerInstall string `json:"dockerInstall,omitempty" yaml:"docker_install,omitempty"`

	EndpointType string `json:"endpointType,omitempty" yaml:"endpoint_type,omitempty"`

	FlavorId string `json:"flavorId,omitempty" yaml:"flavor_id,omitempty"`

	ImageId string `json:"imageId,omitempty" yaml:"image_id,omitempty"`

	Region string `json:"region,omitempty" yaml:"region,omitempty"`

	SshPort string `json:"sshPort,omitempty" yaml:"ssh_port,omitempty"`

	SshUser string `json:"sshUser,omitempty" yaml:"ssh_user,omitempty"`

	Username string `json:"username,omitempty" yaml:"username,omitempty"`
}

type RackspaceConfigCollection struct {
	Collection
	Data []RackspaceConfig `json:"data,omitempty"`
}

type RackspaceConfigClient struct {
	rancherClient *RancherClient
}

type RackspaceConfigOperations interface {
	List(opts *ListOpts) (*RackspaceConfigCollection, error)
	Create(opts *RackspaceConfig) (*RackspaceConfig, error)
	Update(existing *RackspaceConfig, updates interface{}) (*RackspaceConfig, error)
	ById(id string) (*RackspaceConfig, error)
	Delete(container *RackspaceConfig) error
}

func newRackspaceConfigClient(rancherClient *RancherClient) *RackspaceConfigClient {
	return &RackspaceConfigClient{
		rancherClient: rancherClient,
	}
}

func (c *RackspaceConfigClient) Create(container *RackspaceConfig) (*RackspaceConfig, error) {
	resp := &RackspaceConfig{}
	err := c.rancherClient.doCreate(RACKSPACE_CONFIG_TYPE, container, resp)
	return resp, err
}

func (c *RackspaceConfigClient) Update(existing *RackspaceConfig, updates interface{}) (*RackspaceConfig, error) {
	resp := &RackspaceConfig{}
	err := c.rancherClient.doUpdate(RACKSPACE_CONFIG_TYPE, &existing.Resource, updates, resp)
	return resp, err
}

func (c *RackspaceConfigClient) List(opts *ListOpts) (*RackspaceConfigCollection, error) {
	resp := &RackspaceConfigCollection{}
	err := c.rancherClient.doList(RACKSPACE_CONFIG_TYPE, opts, resp)
	return resp, err
}

func (c *RackspaceConfigClient) ById(id string) (*RackspaceConfig, error) {
	resp := &RackspaceConfig{}
	err := c.rancherClient.doById(RACKSPACE_CONFIG_TYPE, id, resp)
	if apiError, ok := err.(*ApiError); ok {
		if apiError.StatusCode == 404 {
			return nil, nil
		}
	}
	return resp, err
}

func (c *RackspaceConfigClient) Delete(container *RackspaceConfig) error {
	return c.rancherClient.doResourceDelete(RACKSPACE_CONFIG_TYPE, &container.Resource)
}
