package client

const (
	REMOVE_LOAD_BALANCER_INPUT_TYPE = "removeLoadBalancerInput"
)

type RemoveLoadBalancerInput struct {
	Resource

	LoadBalancerId string `json:"loadBalancerId,omitempty" yaml:"load_balancer_id,omitempty"`
}

type RemoveLoadBalancerInputCollection struct {
	Collection
	Data []RemoveLoadBalancerInput `json:"data,omitempty"`
}

type RemoveLoadBalancerInputClient struct {
	rancherClient *RancherClient
}

type RemoveLoadBalancerInputOperations interface {
	List(opts *ListOpts) (*RemoveLoadBalancerInputCollection, error)
	Create(opts *RemoveLoadBalancerInput) (*RemoveLoadBalancerInput, error)
	Update(existing *RemoveLoadBalancerInput, updates interface{}) (*RemoveLoadBalancerInput, error)
	ById(id string) (*RemoveLoadBalancerInput, error)
	Delete(container *RemoveLoadBalancerInput) error
}

func newRemoveLoadBalancerInputClient(rancherClient *RancherClient) *RemoveLoadBalancerInputClient {
	return &RemoveLoadBalancerInputClient{
		rancherClient: rancherClient,
	}
}

func (c *RemoveLoadBalancerInputClient) Create(container *RemoveLoadBalancerInput) (*RemoveLoadBalancerInput, error) {
	resp := &RemoveLoadBalancerInput{}
	err := c.rancherClient.doCreate(REMOVE_LOAD_BALANCER_INPUT_TYPE, container, resp)
	return resp, err
}

func (c *RemoveLoadBalancerInputClient) Update(existing *RemoveLoadBalancerInput, updates interface{}) (*RemoveLoadBalancerInput, error) {
	resp := &RemoveLoadBalancerInput{}
	err := c.rancherClient.doUpdate(REMOVE_LOAD_BALANCER_INPUT_TYPE, &existing.Resource, updates, resp)
	return resp, err
}

func (c *RemoveLoadBalancerInputClient) List(opts *ListOpts) (*RemoveLoadBalancerInputCollection, error) {
	resp := &RemoveLoadBalancerInputCollection{}
	err := c.rancherClient.doList(REMOVE_LOAD_BALANCER_INPUT_TYPE, opts, resp)
	return resp, err
}

func (c *RemoveLoadBalancerInputClient) ById(id string) (*RemoveLoadBalancerInput, error) {
	resp := &RemoveLoadBalancerInput{}
	err := c.rancherClient.doById(REMOVE_LOAD_BALANCER_INPUT_TYPE, id, resp)
	if apiError, ok := err.(*ApiError); ok {
		if apiError.StatusCode == 404 {
			return nil, nil
		}
	}
	return resp, err
}

func (c *RemoveLoadBalancerInputClient) Delete(container *RemoveLoadBalancerInput) error {
	return c.rancherClient.doResourceDelete(REMOVE_LOAD_BALANCER_INPUT_TYPE, &container.Resource)
}
