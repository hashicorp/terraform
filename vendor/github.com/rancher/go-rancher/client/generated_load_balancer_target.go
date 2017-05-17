package client

const (
	LOAD_BALANCER_TARGET_TYPE = "loadBalancerTarget"
)

type LoadBalancerTarget struct {
	Resource

	AccountId string `json:"accountId,omitempty" yaml:"account_id,omitempty"`

	Created string `json:"created,omitempty" yaml:"created,omitempty"`

	Data map[string]interface{} `json:"data,omitempty" yaml:"data,omitempty"`

	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	InstanceId string `json:"instanceId,omitempty" yaml:"instance_id,omitempty"`

	IpAddress string `json:"ipAddress,omitempty" yaml:"ip_address,omitempty"`

	Kind string `json:"kind,omitempty" yaml:"kind,omitempty"`

	LoadBalancerId string `json:"loadBalancerId,omitempty" yaml:"load_balancer_id,omitempty"`

	Name string `json:"name,omitempty" yaml:"name,omitempty"`

	Ports []string `json:"ports,omitempty" yaml:"ports,omitempty"`

	RemoveTime string `json:"removeTime,omitempty" yaml:"remove_time,omitempty"`

	Removed string `json:"removed,omitempty" yaml:"removed,omitempty"`

	State string `json:"state,omitempty" yaml:"state,omitempty"`

	Transitioning string `json:"transitioning,omitempty" yaml:"transitioning,omitempty"`

	TransitioningMessage string `json:"transitioningMessage,omitempty" yaml:"transitioning_message,omitempty"`

	TransitioningProgress int64 `json:"transitioningProgress,omitempty" yaml:"transitioning_progress,omitempty"`

	Uuid string `json:"uuid,omitempty" yaml:"uuid,omitempty"`
}

type LoadBalancerTargetCollection struct {
	Collection
	Data []LoadBalancerTarget `json:"data,omitempty"`
}

type LoadBalancerTargetClient struct {
	rancherClient *RancherClient
}

type LoadBalancerTargetOperations interface {
	List(opts *ListOpts) (*LoadBalancerTargetCollection, error)
	Create(opts *LoadBalancerTarget) (*LoadBalancerTarget, error)
	Update(existing *LoadBalancerTarget, updates interface{}) (*LoadBalancerTarget, error)
	ById(id string) (*LoadBalancerTarget, error)
	Delete(container *LoadBalancerTarget) error

	ActionCreate(*LoadBalancerTarget) (*LoadBalancerTarget, error)

	ActionRemove(*LoadBalancerTarget) (*LoadBalancerTarget, error)

	ActionUpdate(*LoadBalancerTarget) (*LoadBalancerTarget, error)
}

func newLoadBalancerTargetClient(rancherClient *RancherClient) *LoadBalancerTargetClient {
	return &LoadBalancerTargetClient{
		rancherClient: rancherClient,
	}
}

func (c *LoadBalancerTargetClient) Create(container *LoadBalancerTarget) (*LoadBalancerTarget, error) {
	resp := &LoadBalancerTarget{}
	err := c.rancherClient.doCreate(LOAD_BALANCER_TARGET_TYPE, container, resp)
	return resp, err
}

func (c *LoadBalancerTargetClient) Update(existing *LoadBalancerTarget, updates interface{}) (*LoadBalancerTarget, error) {
	resp := &LoadBalancerTarget{}
	err := c.rancherClient.doUpdate(LOAD_BALANCER_TARGET_TYPE, &existing.Resource, updates, resp)
	return resp, err
}

func (c *LoadBalancerTargetClient) List(opts *ListOpts) (*LoadBalancerTargetCollection, error) {
	resp := &LoadBalancerTargetCollection{}
	err := c.rancherClient.doList(LOAD_BALANCER_TARGET_TYPE, opts, resp)
	return resp, err
}

func (c *LoadBalancerTargetClient) ById(id string) (*LoadBalancerTarget, error) {
	resp := &LoadBalancerTarget{}
	err := c.rancherClient.doById(LOAD_BALANCER_TARGET_TYPE, id, resp)
	if apiError, ok := err.(*ApiError); ok {
		if apiError.StatusCode == 404 {
			return nil, nil
		}
	}
	return resp, err
}

func (c *LoadBalancerTargetClient) Delete(container *LoadBalancerTarget) error {
	return c.rancherClient.doResourceDelete(LOAD_BALANCER_TARGET_TYPE, &container.Resource)
}

func (c *LoadBalancerTargetClient) ActionCreate(resource *LoadBalancerTarget) (*LoadBalancerTarget, error) {

	resp := &LoadBalancerTarget{}

	err := c.rancherClient.doAction(LOAD_BALANCER_TARGET_TYPE, "create", &resource.Resource, nil, resp)

	return resp, err
}

func (c *LoadBalancerTargetClient) ActionRemove(resource *LoadBalancerTarget) (*LoadBalancerTarget, error) {

	resp := &LoadBalancerTarget{}

	err := c.rancherClient.doAction(LOAD_BALANCER_TARGET_TYPE, "remove", &resource.Resource, nil, resp)

	return resp, err
}

func (c *LoadBalancerTargetClient) ActionUpdate(resource *LoadBalancerTarget) (*LoadBalancerTarget, error) {

	resp := &LoadBalancerTarget{}

	err := c.rancherClient.doAction(LOAD_BALANCER_TARGET_TYPE, "update", &resource.Resource, nil, resp)

	return resp, err
}
