package client

const (
	ADD_REMOVE_LOAD_BALANCER_LISTENER_INPUT_TYPE = "addRemoveLoadBalancerListenerInput"
)

type AddRemoveLoadBalancerListenerInput struct {
	Resource

	LoadBalancerListenerId string `json:"loadBalancerListenerId,omitempty" yaml:"load_balancer_listener_id,omitempty"`
}

type AddRemoveLoadBalancerListenerInputCollection struct {
	Collection
	Data []AddRemoveLoadBalancerListenerInput `json:"data,omitempty"`
}

type AddRemoveLoadBalancerListenerInputClient struct {
	rancherClient *RancherClient
}

type AddRemoveLoadBalancerListenerInputOperations interface {
	List(opts *ListOpts) (*AddRemoveLoadBalancerListenerInputCollection, error)
	Create(opts *AddRemoveLoadBalancerListenerInput) (*AddRemoveLoadBalancerListenerInput, error)
	Update(existing *AddRemoveLoadBalancerListenerInput, updates interface{}) (*AddRemoveLoadBalancerListenerInput, error)
	ById(id string) (*AddRemoveLoadBalancerListenerInput, error)
	Delete(container *AddRemoveLoadBalancerListenerInput) error
}

func newAddRemoveLoadBalancerListenerInputClient(rancherClient *RancherClient) *AddRemoveLoadBalancerListenerInputClient {
	return &AddRemoveLoadBalancerListenerInputClient{
		rancherClient: rancherClient,
	}
}

func (c *AddRemoveLoadBalancerListenerInputClient) Create(container *AddRemoveLoadBalancerListenerInput) (*AddRemoveLoadBalancerListenerInput, error) {
	resp := &AddRemoveLoadBalancerListenerInput{}
	err := c.rancherClient.doCreate(ADD_REMOVE_LOAD_BALANCER_LISTENER_INPUT_TYPE, container, resp)
	return resp, err
}

func (c *AddRemoveLoadBalancerListenerInputClient) Update(existing *AddRemoveLoadBalancerListenerInput, updates interface{}) (*AddRemoveLoadBalancerListenerInput, error) {
	resp := &AddRemoveLoadBalancerListenerInput{}
	err := c.rancherClient.doUpdate(ADD_REMOVE_LOAD_BALANCER_LISTENER_INPUT_TYPE, &existing.Resource, updates, resp)
	return resp, err
}

func (c *AddRemoveLoadBalancerListenerInputClient) List(opts *ListOpts) (*AddRemoveLoadBalancerListenerInputCollection, error) {
	resp := &AddRemoveLoadBalancerListenerInputCollection{}
	err := c.rancherClient.doList(ADD_REMOVE_LOAD_BALANCER_LISTENER_INPUT_TYPE, opts, resp)
	return resp, err
}

func (c *AddRemoveLoadBalancerListenerInputClient) ById(id string) (*AddRemoveLoadBalancerListenerInput, error) {
	resp := &AddRemoveLoadBalancerListenerInput{}
	err := c.rancherClient.doById(ADD_REMOVE_LOAD_BALANCER_LISTENER_INPUT_TYPE, id, resp)
	if apiError, ok := err.(*ApiError); ok {
		if apiError.StatusCode == 404 {
			return nil, nil
		}
	}
	return resp, err
}

func (c *AddRemoveLoadBalancerListenerInputClient) Delete(container *AddRemoveLoadBalancerListenerInput) error {
	return c.rancherClient.doResourceDelete(ADD_REMOVE_LOAD_BALANCER_LISTENER_INPUT_TYPE, &container.Resource)
}
