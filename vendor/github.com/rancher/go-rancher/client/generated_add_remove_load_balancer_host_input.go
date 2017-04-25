package client

const (
	ADD_REMOVE_LOAD_BALANCER_HOST_INPUT_TYPE = "addRemoveLoadBalancerHostInput"
)

type AddRemoveLoadBalancerHostInput struct {
	Resource

	HostId string `json:"hostId,omitempty" yaml:"host_id,omitempty"`
}

type AddRemoveLoadBalancerHostInputCollection struct {
	Collection
	Data []AddRemoveLoadBalancerHostInput `json:"data,omitempty"`
}

type AddRemoveLoadBalancerHostInputClient struct {
	rancherClient *RancherClient
}

type AddRemoveLoadBalancerHostInputOperations interface {
	List(opts *ListOpts) (*AddRemoveLoadBalancerHostInputCollection, error)
	Create(opts *AddRemoveLoadBalancerHostInput) (*AddRemoveLoadBalancerHostInput, error)
	Update(existing *AddRemoveLoadBalancerHostInput, updates interface{}) (*AddRemoveLoadBalancerHostInput, error)
	ById(id string) (*AddRemoveLoadBalancerHostInput, error)
	Delete(container *AddRemoveLoadBalancerHostInput) error
}

func newAddRemoveLoadBalancerHostInputClient(rancherClient *RancherClient) *AddRemoveLoadBalancerHostInputClient {
	return &AddRemoveLoadBalancerHostInputClient{
		rancherClient: rancherClient,
	}
}

func (c *AddRemoveLoadBalancerHostInputClient) Create(container *AddRemoveLoadBalancerHostInput) (*AddRemoveLoadBalancerHostInput, error) {
	resp := &AddRemoveLoadBalancerHostInput{}
	err := c.rancherClient.doCreate(ADD_REMOVE_LOAD_BALANCER_HOST_INPUT_TYPE, container, resp)
	return resp, err
}

func (c *AddRemoveLoadBalancerHostInputClient) Update(existing *AddRemoveLoadBalancerHostInput, updates interface{}) (*AddRemoveLoadBalancerHostInput, error) {
	resp := &AddRemoveLoadBalancerHostInput{}
	err := c.rancherClient.doUpdate(ADD_REMOVE_LOAD_BALANCER_HOST_INPUT_TYPE, &existing.Resource, updates, resp)
	return resp, err
}

func (c *AddRemoveLoadBalancerHostInputClient) List(opts *ListOpts) (*AddRemoveLoadBalancerHostInputCollection, error) {
	resp := &AddRemoveLoadBalancerHostInputCollection{}
	err := c.rancherClient.doList(ADD_REMOVE_LOAD_BALANCER_HOST_INPUT_TYPE, opts, resp)
	return resp, err
}

func (c *AddRemoveLoadBalancerHostInputClient) ById(id string) (*AddRemoveLoadBalancerHostInput, error) {
	resp := &AddRemoveLoadBalancerHostInput{}
	err := c.rancherClient.doById(ADD_REMOVE_LOAD_BALANCER_HOST_INPUT_TYPE, id, resp)
	if apiError, ok := err.(*ApiError); ok {
		if apiError.StatusCode == 404 {
			return nil, nil
		}
	}
	return resp, err
}

func (c *AddRemoveLoadBalancerHostInputClient) Delete(container *AddRemoveLoadBalancerHostInput) error {
	return c.rancherClient.doResourceDelete(ADD_REMOVE_LOAD_BALANCER_HOST_INPUT_TYPE, &container.Resource)
}
