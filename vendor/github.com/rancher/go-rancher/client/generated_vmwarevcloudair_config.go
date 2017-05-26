package client

const (
	VMWAREVCLOUDAIR_CONFIG_TYPE = "vmwarevcloudairConfig"
)

type VmwarevcloudairConfig struct {
	Resource

	Catalog string `json:"catalog,omitempty" yaml:"catalog,omitempty"`

	Catalogitem string `json:"catalogitem,omitempty" yaml:"catalogitem,omitempty"`

	Computeid string `json:"computeid,omitempty" yaml:"computeid,omitempty"`

	CpuCount string `json:"cpuCount,omitempty" yaml:"cpu_count,omitempty"`

	DockerPort string `json:"dockerPort,omitempty" yaml:"docker_port,omitempty"`

	Edgegateway string `json:"edgegateway,omitempty" yaml:"edgegateway,omitempty"`

	MemorySize string `json:"memorySize,omitempty" yaml:"memory_size,omitempty"`

	Orgvdcnetwork string `json:"orgvdcnetwork,omitempty" yaml:"orgvdcnetwork,omitempty"`

	Password string `json:"password,omitempty" yaml:"password,omitempty"`

	Provision bool `json:"provision,omitempty" yaml:"provision,omitempty"`

	Publicip string `json:"publicip,omitempty" yaml:"publicip,omitempty"`

	SshPort string `json:"sshPort,omitempty" yaml:"ssh_port,omitempty"`

	Username string `json:"username,omitempty" yaml:"username,omitempty"`

	Vdcid string `json:"vdcid,omitempty" yaml:"vdcid,omitempty"`
}

type VmwarevcloudairConfigCollection struct {
	Collection
	Data []VmwarevcloudairConfig `json:"data,omitempty"`
}

type VmwarevcloudairConfigClient struct {
	rancherClient *RancherClient
}

type VmwarevcloudairConfigOperations interface {
	List(opts *ListOpts) (*VmwarevcloudairConfigCollection, error)
	Create(opts *VmwarevcloudairConfig) (*VmwarevcloudairConfig, error)
	Update(existing *VmwarevcloudairConfig, updates interface{}) (*VmwarevcloudairConfig, error)
	ById(id string) (*VmwarevcloudairConfig, error)
	Delete(container *VmwarevcloudairConfig) error
}

func newVmwarevcloudairConfigClient(rancherClient *RancherClient) *VmwarevcloudairConfigClient {
	return &VmwarevcloudairConfigClient{
		rancherClient: rancherClient,
	}
}

func (c *VmwarevcloudairConfigClient) Create(container *VmwarevcloudairConfig) (*VmwarevcloudairConfig, error) {
	resp := &VmwarevcloudairConfig{}
	err := c.rancherClient.doCreate(VMWAREVCLOUDAIR_CONFIG_TYPE, container, resp)
	return resp, err
}

func (c *VmwarevcloudairConfigClient) Update(existing *VmwarevcloudairConfig, updates interface{}) (*VmwarevcloudairConfig, error) {
	resp := &VmwarevcloudairConfig{}
	err := c.rancherClient.doUpdate(VMWAREVCLOUDAIR_CONFIG_TYPE, &existing.Resource, updates, resp)
	return resp, err
}

func (c *VmwarevcloudairConfigClient) List(opts *ListOpts) (*VmwarevcloudairConfigCollection, error) {
	resp := &VmwarevcloudairConfigCollection{}
	err := c.rancherClient.doList(VMWAREVCLOUDAIR_CONFIG_TYPE, opts, resp)
	return resp, err
}

func (c *VmwarevcloudairConfigClient) ById(id string) (*VmwarevcloudairConfig, error) {
	resp := &VmwarevcloudairConfig{}
	err := c.rancherClient.doById(VMWAREVCLOUDAIR_CONFIG_TYPE, id, resp)
	if apiError, ok := err.(*ApiError); ok {
		if apiError.StatusCode == 404 {
			return nil, nil
		}
	}
	return resp, err
}

func (c *VmwarevcloudairConfigClient) Delete(container *VmwarevcloudairConfig) error {
	return c.rancherClient.doResourceDelete(VMWAREVCLOUDAIR_CONFIG_TYPE, &container.Resource)
}
