package client

const (
	VMWAREVSPHERE_CONFIG_TYPE = "vmwarevsphereConfig"
)

type VmwarevsphereConfig struct {
	Resource

	Boot2dockerUrl string `json:"boot2dockerUrl,omitempty" yaml:"boot2docker_url,omitempty"`

	ComputeIp string `json:"computeIp,omitempty" yaml:"compute_ip,omitempty"`

	CpuCount string `json:"cpuCount,omitempty" yaml:"cpu_count,omitempty"`

	Datacenter string `json:"datacenter,omitempty" yaml:"datacenter,omitempty"`

	Datastore string `json:"datastore,omitempty" yaml:"datastore,omitempty"`

	DiskSize string `json:"diskSize,omitempty" yaml:"disk_size,omitempty"`

	MemorySize string `json:"memorySize,omitempty" yaml:"memory_size,omitempty"`

	Network string `json:"network,omitempty" yaml:"network,omitempty"`

	Password string `json:"password,omitempty" yaml:"password,omitempty"`

	Pool string `json:"pool,omitempty" yaml:"pool,omitempty"`

	Username string `json:"username,omitempty" yaml:"username,omitempty"`

	Vcenter string `json:"vcenter,omitempty" yaml:"vcenter,omitempty"`
}

type VmwarevsphereConfigCollection struct {
	Collection
	Data []VmwarevsphereConfig `json:"data,omitempty"`
}

type VmwarevsphereConfigClient struct {
	rancherClient *RancherClient
}

type VmwarevsphereConfigOperations interface {
	List(opts *ListOpts) (*VmwarevsphereConfigCollection, error)
	Create(opts *VmwarevsphereConfig) (*VmwarevsphereConfig, error)
	Update(existing *VmwarevsphereConfig, updates interface{}) (*VmwarevsphereConfig, error)
	ById(id string) (*VmwarevsphereConfig, error)
	Delete(container *VmwarevsphereConfig) error
}

func newVmwarevsphereConfigClient(rancherClient *RancherClient) *VmwarevsphereConfigClient {
	return &VmwarevsphereConfigClient{
		rancherClient: rancherClient,
	}
}

func (c *VmwarevsphereConfigClient) Create(container *VmwarevsphereConfig) (*VmwarevsphereConfig, error) {
	resp := &VmwarevsphereConfig{}
	err := c.rancherClient.doCreate(VMWAREVSPHERE_CONFIG_TYPE, container, resp)
	return resp, err
}

func (c *VmwarevsphereConfigClient) Update(existing *VmwarevsphereConfig, updates interface{}) (*VmwarevsphereConfig, error) {
	resp := &VmwarevsphereConfig{}
	err := c.rancherClient.doUpdate(VMWAREVSPHERE_CONFIG_TYPE, &existing.Resource, updates, resp)
	return resp, err
}

func (c *VmwarevsphereConfigClient) List(opts *ListOpts) (*VmwarevsphereConfigCollection, error) {
	resp := &VmwarevsphereConfigCollection{}
	err := c.rancherClient.doList(VMWAREVSPHERE_CONFIG_TYPE, opts, resp)
	return resp, err
}

func (c *VmwarevsphereConfigClient) ById(id string) (*VmwarevsphereConfig, error) {
	resp := &VmwarevsphereConfig{}
	err := c.rancherClient.doById(VMWAREVSPHERE_CONFIG_TYPE, id, resp)
	if apiError, ok := err.(*ApiError); ok {
		if apiError.StatusCode == 404 {
			return nil, nil
		}
	}
	return resp, err
}

func (c *VmwarevsphereConfigClient) Delete(container *VmwarevsphereConfig) error {
	return c.rancherClient.doResourceDelete(VMWAREVSPHERE_CONFIG_TYPE, &container.Resource)
}
