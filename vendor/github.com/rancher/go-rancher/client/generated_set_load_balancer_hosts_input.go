package client

const (
	SET_LOAD_BALANCER_HOSTS_INPUT_TYPE = "setLoadBalancerHostsInput"
)

type SetLoadBalancerHostsInput struct {
	Resource

	HostIds []string `json:"hostIds,omitempty" yaml:"host_ids,omitempty"`
}

type SetLoadBalancerHostsInputCollection struct {
	Collection
	Data []SetLoadBalancerHostsInput `json:"data,omitempty"`
}

type SetLoadBalancerHostsInputClient struct {
	rancherClient *RancherClient
}

type SetLoadBalancerHostsInputOperations interface {
	List(opts *ListOpts) (*SetLoadBalancerHostsInputCollection, error)
	Create(opts *SetLoadBalancerHostsInput) (*SetLoadBalancerHostsInput, error)
	Update(existing *SetLoadBalancerHostsInput, updates interface{}) (*SetLoadBalancerHostsInput, error)
	ById(id string) (*SetLoadBalancerHostsInput, error)
	Delete(container *SetLoadBalancerHostsInput) error
}

func newSetLoadBalancerHostsInputClient(rancherClient *RancherClient) *SetLoadBalancerHostsInputClient {
	return &SetLoadBalancerHostsInputClient{
		rancherClient: rancherClient,
	}
}

func (c *SetLoadBalancerHostsInputClient) Create(container *SetLoadBalancerHostsInput) (*SetLoadBalancerHostsInput, error) {
	resp := &SetLoadBalancerHostsInput{}
	err := c.rancherClient.doCreate(SET_LOAD_BALANCER_HOSTS_INPUT_TYPE, container, resp)
	return resp, err
}

func (c *SetLoadBalancerHostsInputClient) Update(existing *SetLoadBalancerHostsInput, updates interface{}) (*SetLoadBalancerHostsInput, error) {
	resp := &SetLoadBalancerHostsInput{}
	err := c.rancherClient.doUpdate(SET_LOAD_BALANCER_HOSTS_INPUT_TYPE, &existing.Resource, updates, resp)
	return resp, err
}

func (c *SetLoadBalancerHostsInputClient) List(opts *ListOpts) (*SetLoadBalancerHostsInputCollection, error) {
	resp := &SetLoadBalancerHostsInputCollection{}
	err := c.rancherClient.doList(SET_LOAD_BALANCER_HOSTS_INPUT_TYPE, opts, resp)
	return resp, err
}

func (c *SetLoadBalancerHostsInputClient) ById(id string) (*SetLoadBalancerHostsInput, error) {
	resp := &SetLoadBalancerHostsInput{}
	err := c.rancherClient.doById(SET_LOAD_BALANCER_HOSTS_INPUT_TYPE, id, resp)
	if apiError, ok := err.(*ApiError); ok {
		if apiError.StatusCode == 404 {
			return nil, nil
		}
	}
	return resp, err
}

func (c *SetLoadBalancerHostsInputClient) Delete(container *SetLoadBalancerHostsInput) error {
	return c.rancherClient.doResourceDelete(SET_LOAD_BALANCER_HOSTS_INPUT_TYPE, &container.Resource)
}
