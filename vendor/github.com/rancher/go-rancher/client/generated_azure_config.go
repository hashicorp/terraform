package client

const (
	AZURE_CONFIG_TYPE = "azureConfig"
)

type AzureConfig struct {
	Resource

	DockerPort string `json:"dockerPort,omitempty" yaml:"docker_port,omitempty"`

	DockerSwarmMasterPort string `json:"dockerSwarmMasterPort,omitempty" yaml:"docker_swarm_master_port,omitempty"`

	Image string `json:"image,omitempty" yaml:"image,omitempty"`

	Location string `json:"location,omitempty" yaml:"location,omitempty"`

	Password string `json:"password,omitempty" yaml:"password,omitempty"`

	PublishSettingsFile string `json:"publishSettingsFile,omitempty" yaml:"publish_settings_file,omitempty"`

	Size string `json:"size,omitempty" yaml:"size,omitempty"`

	SshPort string `json:"sshPort,omitempty" yaml:"ssh_port,omitempty"`

	SubscriptionCert string `json:"subscriptionCert,omitempty" yaml:"subscription_cert,omitempty"`

	SubscriptionId string `json:"subscriptionId,omitempty" yaml:"subscription_id,omitempty"`

	Username string `json:"username,omitempty" yaml:"username,omitempty"`
}

type AzureConfigCollection struct {
	Collection
	Data []AzureConfig `json:"data,omitempty"`
}

type AzureConfigClient struct {
	rancherClient *RancherClient
}

type AzureConfigOperations interface {
	List(opts *ListOpts) (*AzureConfigCollection, error)
	Create(opts *AzureConfig) (*AzureConfig, error)
	Update(existing *AzureConfig, updates interface{}) (*AzureConfig, error)
	ById(id string) (*AzureConfig, error)
	Delete(container *AzureConfig) error
}

func newAzureConfigClient(rancherClient *RancherClient) *AzureConfigClient {
	return &AzureConfigClient{
		rancherClient: rancherClient,
	}
}

func (c *AzureConfigClient) Create(container *AzureConfig) (*AzureConfig, error) {
	resp := &AzureConfig{}
	err := c.rancherClient.doCreate(AZURE_CONFIG_TYPE, container, resp)
	return resp, err
}

func (c *AzureConfigClient) Update(existing *AzureConfig, updates interface{}) (*AzureConfig, error) {
	resp := &AzureConfig{}
	err := c.rancherClient.doUpdate(AZURE_CONFIG_TYPE, &existing.Resource, updates, resp)
	return resp, err
}

func (c *AzureConfigClient) List(opts *ListOpts) (*AzureConfigCollection, error) {
	resp := &AzureConfigCollection{}
	err := c.rancherClient.doList(AZURE_CONFIG_TYPE, opts, resp)
	return resp, err
}

func (c *AzureConfigClient) ById(id string) (*AzureConfig, error) {
	resp := &AzureConfig{}
	err := c.rancherClient.doById(AZURE_CONFIG_TYPE, id, resp)
	if apiError, ok := err.(*ApiError); ok {
		if apiError.StatusCode == 404 {
			return nil, nil
		}
	}
	return resp, err
}

func (c *AzureConfigClient) Delete(container *AzureConfig) error {
	return c.rancherClient.doResourceDelete(AZURE_CONFIG_TYPE, &container.Resource)
}
