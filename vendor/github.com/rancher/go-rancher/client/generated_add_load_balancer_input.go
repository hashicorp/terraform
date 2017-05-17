package client

const (
	ADD_LOAD_BALANCER_INPUT_TYPE = "addLoadBalancerInput"
)

type AddLoadBalancerInput struct {
	Resource

	LoadBalancerId string `json:"loadBalancerId,omitempty" yaml:"load_balancer_id,omitempty"`

	Weight int64 `json:"weight,omitempty" yaml:"weight,omitempty"`
}

type AddLoadBalancerInputCollection struct {
	Collection
	Data []AddLoadBalancerInput `json:"data,omitempty"`
}

type AddLoadBalancerInputClient struct {
	rancherClient *RancherClient
}

type AddLoadBalancerInputOperations interface {
	List(opts *ListOpts) (*AddLoadBalancerInputCollection, error)
	Create(opts *AddLoadBalancerInput) (*AddLoadBalancerInput, error)
	Update(existing *AddLoadBalancerInput, updates interface{}) (*AddLoadBalancerInput, error)
	ById(id string) (*AddLoadBalancerInput, error)
	Delete(container *AddLoadBalancerInput) error
}

func newAddLoadBalancerInputClient(rancherClient *RancherClient) *AddLoadBalancerInputClient {
	return &AddLoadBalancerInputClient{
		rancherClient: rancherClient,
	}
}

func (c *AddLoadBalancerInputClient) Create(container *AddLoadBalancerInput) (*AddLoadBalancerInput, error) {
	resp := &AddLoadBalancerInput{}
	err := c.rancherClient.doCreate(ADD_LOAD_BALANCER_INPUT_TYPE, container, resp)
	return resp, err
}

func (c *AddLoadBalancerInputClient) Update(existing *AddLoadBalancerInput, updates interface{}) (*AddLoadBalancerInput, error) {
	resp := &AddLoadBalancerInput{}
	err := c.rancherClient.doUpdate(ADD_LOAD_BALANCER_INPUT_TYPE, &existing.Resource, updates, resp)
	return resp, err
}

func (c *AddLoadBalancerInputClient) List(opts *ListOpts) (*AddLoadBalancerInputCollection, error) {
	resp := &AddLoadBalancerInputCollection{}
	err := c.rancherClient.doList(ADD_LOAD_BALANCER_INPUT_TYPE, opts, resp)
	return resp, err
}

func (c *AddLoadBalancerInputClient) ById(id string) (*AddLoadBalancerInput, error) {
	resp := &AddLoadBalancerInput{}
	err := c.rancherClient.doById(ADD_LOAD_BALANCER_INPUT_TYPE, id, resp)
	if apiError, ok := err.(*ApiError); ok {
		if apiError.StatusCode == 404 {
			return nil, nil
		}
	}
	return resp, err
}

func (c *AddLoadBalancerInputClient) Delete(container *AddLoadBalancerInput) error {
	return c.rancherClient.doResourceDelete(ADD_LOAD_BALANCER_INPUT_TYPE, &container.Resource)
}
