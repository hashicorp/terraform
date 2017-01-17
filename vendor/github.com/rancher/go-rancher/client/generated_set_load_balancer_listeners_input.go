package client

const (
	SET_LOAD_BALANCER_LISTENERS_INPUT_TYPE = "setLoadBalancerListenersInput"
)

type SetLoadBalancerListenersInput struct {
	Resource

	LoadBalancerListenerIds []string `json:"loadBalancerListenerIds,omitempty" yaml:"load_balancer_listener_ids,omitempty"`
}

type SetLoadBalancerListenersInputCollection struct {
	Collection
	Data []SetLoadBalancerListenersInput `json:"data,omitempty"`
}

type SetLoadBalancerListenersInputClient struct {
	rancherClient *RancherClient
}

type SetLoadBalancerListenersInputOperations interface {
	List(opts *ListOpts) (*SetLoadBalancerListenersInputCollection, error)
	Create(opts *SetLoadBalancerListenersInput) (*SetLoadBalancerListenersInput, error)
	Update(existing *SetLoadBalancerListenersInput, updates interface{}) (*SetLoadBalancerListenersInput, error)
	ById(id string) (*SetLoadBalancerListenersInput, error)
	Delete(container *SetLoadBalancerListenersInput) error
}

func newSetLoadBalancerListenersInputClient(rancherClient *RancherClient) *SetLoadBalancerListenersInputClient {
	return &SetLoadBalancerListenersInputClient{
		rancherClient: rancherClient,
	}
}

func (c *SetLoadBalancerListenersInputClient) Create(container *SetLoadBalancerListenersInput) (*SetLoadBalancerListenersInput, error) {
	resp := &SetLoadBalancerListenersInput{}
	err := c.rancherClient.doCreate(SET_LOAD_BALANCER_LISTENERS_INPUT_TYPE, container, resp)
	return resp, err
}

func (c *SetLoadBalancerListenersInputClient) Update(existing *SetLoadBalancerListenersInput, updates interface{}) (*SetLoadBalancerListenersInput, error) {
	resp := &SetLoadBalancerListenersInput{}
	err := c.rancherClient.doUpdate(SET_LOAD_BALANCER_LISTENERS_INPUT_TYPE, &existing.Resource, updates, resp)
	return resp, err
}

func (c *SetLoadBalancerListenersInputClient) List(opts *ListOpts) (*SetLoadBalancerListenersInputCollection, error) {
	resp := &SetLoadBalancerListenersInputCollection{}
	err := c.rancherClient.doList(SET_LOAD_BALANCER_LISTENERS_INPUT_TYPE, opts, resp)
	return resp, err
}

func (c *SetLoadBalancerListenersInputClient) ById(id string) (*SetLoadBalancerListenersInput, error) {
	resp := &SetLoadBalancerListenersInput{}
	err := c.rancherClient.doById(SET_LOAD_BALANCER_LISTENERS_INPUT_TYPE, id, resp)
	if apiError, ok := err.(*ApiError); ok {
		if apiError.StatusCode == 404 {
			return nil, nil
		}
	}
	return resp, err
}

func (c *SetLoadBalancerListenersInputClient) Delete(container *SetLoadBalancerListenersInput) error {
	return c.rancherClient.doResourceDelete(SET_LOAD_BALANCER_LISTENERS_INPUT_TYPE, &container.Resource)
}
