package client

const (
	ADD_REMOVE_LOAD_BALANCER_TARGET_INPUT_TYPE = "addRemoveLoadBalancerTargetInput"
)

type AddRemoveLoadBalancerTargetInput struct {
	Resource

	LoadBalancerTarget LoadBalancerTarget `json:"loadBalancerTarget,omitempty" yaml:"load_balancer_target,omitempty"`
}

type AddRemoveLoadBalancerTargetInputCollection struct {
	Collection
	Data []AddRemoveLoadBalancerTargetInput `json:"data,omitempty"`
}

type AddRemoveLoadBalancerTargetInputClient struct {
	rancherClient *RancherClient
}

type AddRemoveLoadBalancerTargetInputOperations interface {
	List(opts *ListOpts) (*AddRemoveLoadBalancerTargetInputCollection, error)
	Create(opts *AddRemoveLoadBalancerTargetInput) (*AddRemoveLoadBalancerTargetInput, error)
	Update(existing *AddRemoveLoadBalancerTargetInput, updates interface{}) (*AddRemoveLoadBalancerTargetInput, error)
	ById(id string) (*AddRemoveLoadBalancerTargetInput, error)
	Delete(container *AddRemoveLoadBalancerTargetInput) error
}

func newAddRemoveLoadBalancerTargetInputClient(rancherClient *RancherClient) *AddRemoveLoadBalancerTargetInputClient {
	return &AddRemoveLoadBalancerTargetInputClient{
		rancherClient: rancherClient,
	}
}

func (c *AddRemoveLoadBalancerTargetInputClient) Create(container *AddRemoveLoadBalancerTargetInput) (*AddRemoveLoadBalancerTargetInput, error) {
	resp := &AddRemoveLoadBalancerTargetInput{}
	err := c.rancherClient.doCreate(ADD_REMOVE_LOAD_BALANCER_TARGET_INPUT_TYPE, container, resp)
	return resp, err
}

func (c *AddRemoveLoadBalancerTargetInputClient) Update(existing *AddRemoveLoadBalancerTargetInput, updates interface{}) (*AddRemoveLoadBalancerTargetInput, error) {
	resp := &AddRemoveLoadBalancerTargetInput{}
	err := c.rancherClient.doUpdate(ADD_REMOVE_LOAD_BALANCER_TARGET_INPUT_TYPE, &existing.Resource, updates, resp)
	return resp, err
}

func (c *AddRemoveLoadBalancerTargetInputClient) List(opts *ListOpts) (*AddRemoveLoadBalancerTargetInputCollection, error) {
	resp := &AddRemoveLoadBalancerTargetInputCollection{}
	err := c.rancherClient.doList(ADD_REMOVE_LOAD_BALANCER_TARGET_INPUT_TYPE, opts, resp)
	return resp, err
}

func (c *AddRemoveLoadBalancerTargetInputClient) ById(id string) (*AddRemoveLoadBalancerTargetInput, error) {
	resp := &AddRemoveLoadBalancerTargetInput{}
	err := c.rancherClient.doById(ADD_REMOVE_LOAD_BALANCER_TARGET_INPUT_TYPE, id, resp)
	if apiError, ok := err.(*ApiError); ok {
		if apiError.StatusCode == 404 {
			return nil, nil
		}
	}
	return resp, err
}

func (c *AddRemoveLoadBalancerTargetInputClient) Delete(container *AddRemoveLoadBalancerTargetInput) error {
	return c.rancherClient.doResourceDelete(ADD_REMOVE_LOAD_BALANCER_TARGET_INPUT_TYPE, &container.Resource)
}
