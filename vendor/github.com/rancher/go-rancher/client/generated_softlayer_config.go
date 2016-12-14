package client

const (
	SOFTLAYER_CONFIG_TYPE = "softlayerConfig"
)

type SoftlayerConfig struct {
	Resource

	ApiEndpoint string `json:"apiEndpoint,omitempty" yaml:"api_endpoint,omitempty"`

	ApiKey string `json:"apiKey,omitempty" yaml:"api_key,omitempty"`

	Cpu string `json:"cpu,omitempty" yaml:"cpu,omitempty"`

	DiskSize string `json:"diskSize,omitempty" yaml:"disk_size,omitempty"`

	Domain string `json:"domain,omitempty" yaml:"domain,omitempty"`

	Hostname string `json:"hostname,omitempty" yaml:"hostname,omitempty"`

	HourlyBilling bool `json:"hourlyBilling,omitempty" yaml:"hourly_billing,omitempty"`

	Image string `json:"image,omitempty" yaml:"image,omitempty"`

	LocalDisk bool `json:"localDisk,omitempty" yaml:"local_disk,omitempty"`

	Memory string `json:"memory,omitempty" yaml:"memory,omitempty"`

	PrivateNetOnly bool `json:"privateNetOnly,omitempty" yaml:"private_net_only,omitempty"`

	PrivateVlanId string `json:"privateVlanId,omitempty" yaml:"private_vlan_id,omitempty"`

	PublicVlanId string `json:"publicVlanId,omitempty" yaml:"public_vlan_id,omitempty"`

	Region string `json:"region,omitempty" yaml:"region,omitempty"`

	User string `json:"user,omitempty" yaml:"user,omitempty"`
}

type SoftlayerConfigCollection struct {
	Collection
	Data []SoftlayerConfig `json:"data,omitempty"`
}

type SoftlayerConfigClient struct {
	rancherClient *RancherClient
}

type SoftlayerConfigOperations interface {
	List(opts *ListOpts) (*SoftlayerConfigCollection, error)
	Create(opts *SoftlayerConfig) (*SoftlayerConfig, error)
	Update(existing *SoftlayerConfig, updates interface{}) (*SoftlayerConfig, error)
	ById(id string) (*SoftlayerConfig, error)
	Delete(container *SoftlayerConfig) error
}

func newSoftlayerConfigClient(rancherClient *RancherClient) *SoftlayerConfigClient {
	return &SoftlayerConfigClient{
		rancherClient: rancherClient,
	}
}

func (c *SoftlayerConfigClient) Create(container *SoftlayerConfig) (*SoftlayerConfig, error) {
	resp := &SoftlayerConfig{}
	err := c.rancherClient.doCreate(SOFTLAYER_CONFIG_TYPE, container, resp)
	return resp, err
}

func (c *SoftlayerConfigClient) Update(existing *SoftlayerConfig, updates interface{}) (*SoftlayerConfig, error) {
	resp := &SoftlayerConfig{}
	err := c.rancherClient.doUpdate(SOFTLAYER_CONFIG_TYPE, &existing.Resource, updates, resp)
	return resp, err
}

func (c *SoftlayerConfigClient) List(opts *ListOpts) (*SoftlayerConfigCollection, error) {
	resp := &SoftlayerConfigCollection{}
	err := c.rancherClient.doList(SOFTLAYER_CONFIG_TYPE, opts, resp)
	return resp, err
}

func (c *SoftlayerConfigClient) ById(id string) (*SoftlayerConfig, error) {
	resp := &SoftlayerConfig{}
	err := c.rancherClient.doById(SOFTLAYER_CONFIG_TYPE, id, resp)
	if apiError, ok := err.(*ApiError); ok {
		if apiError.StatusCode == 404 {
			return nil, nil
		}
	}
	return resp, err
}

func (c *SoftlayerConfigClient) Delete(container *SoftlayerConfig) error {
	return c.rancherClient.doResourceDelete(SOFTLAYER_CONFIG_TYPE, &container.Resource)
}
