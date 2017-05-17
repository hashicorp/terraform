package client

const (
	OPENSTACK_CONFIG_TYPE = "openstackConfig"
)

type OpenstackConfig struct {
	Resource

	AuthUrl string `json:"authUrl,omitempty" yaml:"auth_url,omitempty"`

	AvailabilityZone string `json:"availabilityZone,omitempty" yaml:"availability_zone,omitempty"`

	DomainId string `json:"domainId,omitempty" yaml:"domain_id,omitempty"`

	DomainName string `json:"domainName,omitempty" yaml:"domain_name,omitempty"`

	EndpointType string `json:"endpointType,omitempty" yaml:"endpoint_type,omitempty"`

	FlavorId string `json:"flavorId,omitempty" yaml:"flavor_id,omitempty"`

	FlavorName string `json:"flavorName,omitempty" yaml:"flavor_name,omitempty"`

	FloatingipPool string `json:"floatingipPool,omitempty" yaml:"floatingip_pool,omitempty"`

	ImageId string `json:"imageId,omitempty" yaml:"image_id,omitempty"`

	ImageName string `json:"imageName,omitempty" yaml:"image_name,omitempty"`

	Insecure bool `json:"insecure,omitempty" yaml:"insecure,omitempty"`

	NetId string `json:"netId,omitempty" yaml:"net_id,omitempty"`

	NetName string `json:"netName,omitempty" yaml:"net_name,omitempty"`

	Password string `json:"password,omitempty" yaml:"password,omitempty"`

	Region string `json:"region,omitempty" yaml:"region,omitempty"`

	SecGroups string `json:"secGroups,omitempty" yaml:"sec_groups,omitempty"`

	SshPort string `json:"sshPort,omitempty" yaml:"ssh_port,omitempty"`

	SshUser string `json:"sshUser,omitempty" yaml:"ssh_user,omitempty"`

	TenantId string `json:"tenantId,omitempty" yaml:"tenant_id,omitempty"`

	TenantName string `json:"tenantName,omitempty" yaml:"tenant_name,omitempty"`

	Username string `json:"username,omitempty" yaml:"username,omitempty"`
}

type OpenstackConfigCollection struct {
	Collection
	Data []OpenstackConfig `json:"data,omitempty"`
}

type OpenstackConfigClient struct {
	rancherClient *RancherClient
}

type OpenstackConfigOperations interface {
	List(opts *ListOpts) (*OpenstackConfigCollection, error)
	Create(opts *OpenstackConfig) (*OpenstackConfig, error)
	Update(existing *OpenstackConfig, updates interface{}) (*OpenstackConfig, error)
	ById(id string) (*OpenstackConfig, error)
	Delete(container *OpenstackConfig) error
}

func newOpenstackConfigClient(rancherClient *RancherClient) *OpenstackConfigClient {
	return &OpenstackConfigClient{
		rancherClient: rancherClient,
	}
}

func (c *OpenstackConfigClient) Create(container *OpenstackConfig) (*OpenstackConfig, error) {
	resp := &OpenstackConfig{}
	err := c.rancherClient.doCreate(OPENSTACK_CONFIG_TYPE, container, resp)
	return resp, err
}

func (c *OpenstackConfigClient) Update(existing *OpenstackConfig, updates interface{}) (*OpenstackConfig, error) {
	resp := &OpenstackConfig{}
	err := c.rancherClient.doUpdate(OPENSTACK_CONFIG_TYPE, &existing.Resource, updates, resp)
	return resp, err
}

func (c *OpenstackConfigClient) List(opts *ListOpts) (*OpenstackConfigCollection, error) {
	resp := &OpenstackConfigCollection{}
	err := c.rancherClient.doList(OPENSTACK_CONFIG_TYPE, opts, resp)
	return resp, err
}

func (c *OpenstackConfigClient) ById(id string) (*OpenstackConfig, error) {
	resp := &OpenstackConfig{}
	err := c.rancherClient.doById(OPENSTACK_CONFIG_TYPE, id, resp)
	if apiError, ok := err.(*ApiError); ok {
		if apiError.StatusCode == 404 {
			return nil, nil
		}
	}
	return resp, err
}

func (c *OpenstackConfigClient) Delete(container *OpenstackConfig) error {
	return c.rancherClient.doResourceDelete(OPENSTACK_CONFIG_TYPE, &container.Resource)
}
