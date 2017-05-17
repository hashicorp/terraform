package client

const (
	EXOSCALE_CONFIG_TYPE = "exoscaleConfig"
)

type ExoscaleConfig struct {
	Resource

	ApiKey string `json:"apiKey,omitempty" yaml:"api_key,omitempty"`

	ApiSecretKey string `json:"apiSecretKey,omitempty" yaml:"api_secret_key,omitempty"`

	AvailabilityZone string `json:"availabilityZone,omitempty" yaml:"availability_zone,omitempty"`

	DiskSize string `json:"diskSize,omitempty" yaml:"disk_size,omitempty"`

	Image string `json:"image,omitempty" yaml:"image,omitempty"`

	InstanceProfile string `json:"instanceProfile,omitempty" yaml:"instance_profile,omitempty"`

	SecurityGroup []string `json:"securityGroup,omitempty" yaml:"security_group,omitempty"`

	Url string `json:"url,omitempty" yaml:"url,omitempty"`
}

type ExoscaleConfigCollection struct {
	Collection
	Data []ExoscaleConfig `json:"data,omitempty"`
}

type ExoscaleConfigClient struct {
	rancherClient *RancherClient
}

type ExoscaleConfigOperations interface {
	List(opts *ListOpts) (*ExoscaleConfigCollection, error)
	Create(opts *ExoscaleConfig) (*ExoscaleConfig, error)
	Update(existing *ExoscaleConfig, updates interface{}) (*ExoscaleConfig, error)
	ById(id string) (*ExoscaleConfig, error)
	Delete(container *ExoscaleConfig) error
}

func newExoscaleConfigClient(rancherClient *RancherClient) *ExoscaleConfigClient {
	return &ExoscaleConfigClient{
		rancherClient: rancherClient,
	}
}

func (c *ExoscaleConfigClient) Create(container *ExoscaleConfig) (*ExoscaleConfig, error) {
	resp := &ExoscaleConfig{}
	err := c.rancherClient.doCreate(EXOSCALE_CONFIG_TYPE, container, resp)
	return resp, err
}

func (c *ExoscaleConfigClient) Update(existing *ExoscaleConfig, updates interface{}) (*ExoscaleConfig, error) {
	resp := &ExoscaleConfig{}
	err := c.rancherClient.doUpdate(EXOSCALE_CONFIG_TYPE, &existing.Resource, updates, resp)
	return resp, err
}

func (c *ExoscaleConfigClient) List(opts *ListOpts) (*ExoscaleConfigCollection, error) {
	resp := &ExoscaleConfigCollection{}
	err := c.rancherClient.doList(EXOSCALE_CONFIG_TYPE, opts, resp)
	return resp, err
}

func (c *ExoscaleConfigClient) ById(id string) (*ExoscaleConfig, error) {
	resp := &ExoscaleConfig{}
	err := c.rancherClient.doById(EXOSCALE_CONFIG_TYPE, id, resp)
	if apiError, ok := err.(*ApiError); ok {
		if apiError.StatusCode == 404 {
			return nil, nil
		}
	}
	return resp, err
}

func (c *ExoscaleConfigClient) Delete(container *ExoscaleConfig) error {
	return c.rancherClient.doResourceDelete(EXOSCALE_CONFIG_TYPE, &container.Resource)
}
