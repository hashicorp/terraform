package client

const (
	VIRTUALBOX_CONFIG_TYPE = "virtualboxConfig"
)

type VirtualboxConfig struct {
	Resource

	Boot2dockerUrl string `json:"boot2dockerUrl,omitempty" yaml:"boot2docker_url,omitempty"`

	CpuCount string `json:"cpuCount,omitempty" yaml:"cpu_count,omitempty"`

	DiskSize string `json:"diskSize,omitempty" yaml:"disk_size,omitempty"`

	HostonlyCidr string `json:"hostonlyCidr,omitempty" yaml:"hostonly_cidr,omitempty"`

	ImportBoot2dockerVm string `json:"importBoot2dockerVm,omitempty" yaml:"import_boot2docker_vm,omitempty"`

	Memory string `json:"memory,omitempty" yaml:"memory,omitempty"`

	NoShare bool `json:"noShare,omitempty" yaml:"no_share,omitempty"`
}

type VirtualboxConfigCollection struct {
	Collection
	Data []VirtualboxConfig `json:"data,omitempty"`
}

type VirtualboxConfigClient struct {
	rancherClient *RancherClient
}

type VirtualboxConfigOperations interface {
	List(opts *ListOpts) (*VirtualboxConfigCollection, error)
	Create(opts *VirtualboxConfig) (*VirtualboxConfig, error)
	Update(existing *VirtualboxConfig, updates interface{}) (*VirtualboxConfig, error)
	ById(id string) (*VirtualboxConfig, error)
	Delete(container *VirtualboxConfig) error
}

func newVirtualboxConfigClient(rancherClient *RancherClient) *VirtualboxConfigClient {
	return &VirtualboxConfigClient{
		rancherClient: rancherClient,
	}
}

func (c *VirtualboxConfigClient) Create(container *VirtualboxConfig) (*VirtualboxConfig, error) {
	resp := &VirtualboxConfig{}
	err := c.rancherClient.doCreate(VIRTUALBOX_CONFIG_TYPE, container, resp)
	return resp, err
}

func (c *VirtualboxConfigClient) Update(existing *VirtualboxConfig, updates interface{}) (*VirtualboxConfig, error) {
	resp := &VirtualboxConfig{}
	err := c.rancherClient.doUpdate(VIRTUALBOX_CONFIG_TYPE, &existing.Resource, updates, resp)
	return resp, err
}

func (c *VirtualboxConfigClient) List(opts *ListOpts) (*VirtualboxConfigCollection, error) {
	resp := &VirtualboxConfigCollection{}
	err := c.rancherClient.doList(VIRTUALBOX_CONFIG_TYPE, opts, resp)
	return resp, err
}

func (c *VirtualboxConfigClient) ById(id string) (*VirtualboxConfig, error) {
	resp := &VirtualboxConfig{}
	err := c.rancherClient.doById(VIRTUALBOX_CONFIG_TYPE, id, resp)
	if apiError, ok := err.(*ApiError); ok {
		if apiError.StatusCode == 404 {
			return nil, nil
		}
	}
	return resp, err
}

func (c *VirtualboxConfigClient) Delete(container *VirtualboxConfig) error {
	return c.rancherClient.doResourceDelete(VIRTUALBOX_CONFIG_TYPE, &container.Resource)
}
