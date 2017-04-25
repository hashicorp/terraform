package client

const (
	LOAD_BALANCER_LISTENER_TYPE = "loadBalancerListener"
)

type LoadBalancerListener struct {
	Resource

	AccountId string `json:"accountId,omitempty" yaml:"account_id,omitempty"`

	Algorithm string `json:"algorithm,omitempty" yaml:"algorithm,omitempty"`

	Created string `json:"created,omitempty" yaml:"created,omitempty"`

	Data map[string]interface{} `json:"data,omitempty" yaml:"data,omitempty"`

	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	Kind string `json:"kind,omitempty" yaml:"kind,omitempty"`

	Name string `json:"name,omitempty" yaml:"name,omitempty"`

	PrivatePort int64 `json:"privatePort,omitempty" yaml:"private_port,omitempty"`

	RemoveTime string `json:"removeTime,omitempty" yaml:"remove_time,omitempty"`

	Removed string `json:"removed,omitempty" yaml:"removed,omitempty"`

	ServiceId string `json:"serviceId,omitempty" yaml:"service_id,omitempty"`

	SourcePort int64 `json:"sourcePort,omitempty" yaml:"source_port,omitempty"`

	SourceProtocol string `json:"sourceProtocol,omitempty" yaml:"source_protocol,omitempty"`

	State string `json:"state,omitempty" yaml:"state,omitempty"`

	TargetPort int64 `json:"targetPort,omitempty" yaml:"target_port,omitempty"`

	TargetProtocol string `json:"targetProtocol,omitempty" yaml:"target_protocol,omitempty"`

	Transitioning string `json:"transitioning,omitempty" yaml:"transitioning,omitempty"`

	TransitioningMessage string `json:"transitioningMessage,omitempty" yaml:"transitioning_message,omitempty"`

	TransitioningProgress int64 `json:"transitioningProgress,omitempty" yaml:"transitioning_progress,omitempty"`

	Uuid string `json:"uuid,omitempty" yaml:"uuid,omitempty"`
}

type LoadBalancerListenerCollection struct {
	Collection
	Data []LoadBalancerListener `json:"data,omitempty"`
}

type LoadBalancerListenerClient struct {
	rancherClient *RancherClient
}

type LoadBalancerListenerOperations interface {
	List(opts *ListOpts) (*LoadBalancerListenerCollection, error)
	Create(opts *LoadBalancerListener) (*LoadBalancerListener, error)
	Update(existing *LoadBalancerListener, updates interface{}) (*LoadBalancerListener, error)
	ById(id string) (*LoadBalancerListener, error)
	Delete(container *LoadBalancerListener) error

	ActionCreate(*LoadBalancerListener) (*LoadBalancerListener, error)

	ActionRemove(*LoadBalancerListener) (*LoadBalancerListener, error)
}

func newLoadBalancerListenerClient(rancherClient *RancherClient) *LoadBalancerListenerClient {
	return &LoadBalancerListenerClient{
		rancherClient: rancherClient,
	}
}

func (c *LoadBalancerListenerClient) Create(container *LoadBalancerListener) (*LoadBalancerListener, error) {
	resp := &LoadBalancerListener{}
	err := c.rancherClient.doCreate(LOAD_BALANCER_LISTENER_TYPE, container, resp)
	return resp, err
}

func (c *LoadBalancerListenerClient) Update(existing *LoadBalancerListener, updates interface{}) (*LoadBalancerListener, error) {
	resp := &LoadBalancerListener{}
	err := c.rancherClient.doUpdate(LOAD_BALANCER_LISTENER_TYPE, &existing.Resource, updates, resp)
	return resp, err
}

func (c *LoadBalancerListenerClient) List(opts *ListOpts) (*LoadBalancerListenerCollection, error) {
	resp := &LoadBalancerListenerCollection{}
	err := c.rancherClient.doList(LOAD_BALANCER_LISTENER_TYPE, opts, resp)
	return resp, err
}

func (c *LoadBalancerListenerClient) ById(id string) (*LoadBalancerListener, error) {
	resp := &LoadBalancerListener{}
	err := c.rancherClient.doById(LOAD_BALANCER_LISTENER_TYPE, id, resp)
	if apiError, ok := err.(*ApiError); ok {
		if apiError.StatusCode == 404 {
			return nil, nil
		}
	}
	return resp, err
}

func (c *LoadBalancerListenerClient) Delete(container *LoadBalancerListener) error {
	return c.rancherClient.doResourceDelete(LOAD_BALANCER_LISTENER_TYPE, &container.Resource)
}

func (c *LoadBalancerListenerClient) ActionCreate(resource *LoadBalancerListener) (*LoadBalancerListener, error) {

	resp := &LoadBalancerListener{}

	err := c.rancherClient.doAction(LOAD_BALANCER_LISTENER_TYPE, "create", &resource.Resource, nil, resp)

	return resp, err
}

func (c *LoadBalancerListenerClient) ActionRemove(resource *LoadBalancerListener) (*LoadBalancerListener, error) {

	resp := &LoadBalancerListener{}

	err := c.rancherClient.doAction(LOAD_BALANCER_LISTENER_TYPE, "remove", &resource.Resource, nil, resp)

	return resp, err
}
