package client

const (
	UBIQUITY_CONFIG_TYPE = "ubiquityConfig"
)

type UbiquityConfig struct {
	Resource

	ApiToken string `json:"apiToken,omitempty" yaml:"api_token,omitempty"`

	ApiUsername string `json:"apiUsername,omitempty" yaml:"api_username,omitempty"`

	ClientId string `json:"clientId,omitempty" yaml:"client_id,omitempty"`

	FlavorId string `json:"flavorId,omitempty" yaml:"flavor_id,omitempty"`

	ImageId string `json:"imageId,omitempty" yaml:"image_id,omitempty"`

	ZoneId string `json:"zoneId,omitempty" yaml:"zone_id,omitempty"`
}

type UbiquityConfigCollection struct {
	Collection
	Data []UbiquityConfig `json:"data,omitempty"`
}

type UbiquityConfigClient struct {
	rancherClient *RancherClient
}

type UbiquityConfigOperations interface {
	List(opts *ListOpts) (*UbiquityConfigCollection, error)
	Create(opts *UbiquityConfig) (*UbiquityConfig, error)
	Update(existing *UbiquityConfig, updates interface{}) (*UbiquityConfig, error)
	ById(id string) (*UbiquityConfig, error)
	Delete(container *UbiquityConfig) error
}

func newUbiquityConfigClient(rancherClient *RancherClient) *UbiquityConfigClient {
	return &UbiquityConfigClient{
		rancherClient: rancherClient,
	}
}

func (c *UbiquityConfigClient) Create(container *UbiquityConfig) (*UbiquityConfig, error) {
	resp := &UbiquityConfig{}
	err := c.rancherClient.doCreate(UBIQUITY_CONFIG_TYPE, container, resp)
	return resp, err
}

func (c *UbiquityConfigClient) Update(existing *UbiquityConfig, updates interface{}) (*UbiquityConfig, error) {
	resp := &UbiquityConfig{}
	err := c.rancherClient.doUpdate(UBIQUITY_CONFIG_TYPE, &existing.Resource, updates, resp)
	return resp, err
}

func (c *UbiquityConfigClient) List(opts *ListOpts) (*UbiquityConfigCollection, error) {
	resp := &UbiquityConfigCollection{}
	err := c.rancherClient.doList(UBIQUITY_CONFIG_TYPE, opts, resp)
	return resp, err
}

func (c *UbiquityConfigClient) ById(id string) (*UbiquityConfig, error) {
	resp := &UbiquityConfig{}
	err := c.rancherClient.doById(UBIQUITY_CONFIG_TYPE, id, resp)
	if apiError, ok := err.(*ApiError); ok {
		if apiError.StatusCode == 404 {
			return nil, nil
		}
	}
	return resp, err
}

func (c *UbiquityConfigClient) Delete(container *UbiquityConfig) error {
	return c.rancherClient.doResourceDelete(UBIQUITY_CONFIG_TYPE, &container.Resource)
}
