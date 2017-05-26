package client

const (
	LOAD_BALANCER_TYPE = "loadBalancer"
)

type LoadBalancer struct {
	Resource

	AccountId string `json:"accountId,omitempty" yaml:"account_id,omitempty"`

	CertificateIds []string `json:"certificateIds,omitempty" yaml:"certificate_ids,omitempty"`

	Created string `json:"created,omitempty" yaml:"created,omitempty"`

	Data map[string]interface{} `json:"data,omitempty" yaml:"data,omitempty"`

	DefaultCertificateId string `json:"defaultCertificateId,omitempty" yaml:"default_certificate_id,omitempty"`

	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	GlobalLoadBalancerId string `json:"globalLoadBalancerId,omitempty" yaml:"global_load_balancer_id,omitempty"`

	Kind string `json:"kind,omitempty" yaml:"kind,omitempty"`

	LoadBalancerConfigId string `json:"loadBalancerConfigId,omitempty" yaml:"load_balancer_config_id,omitempty"`

	Name string `json:"name,omitempty" yaml:"name,omitempty"`

	RemoveTime string `json:"removeTime,omitempty" yaml:"remove_time,omitempty"`

	Removed string `json:"removed,omitempty" yaml:"removed,omitempty"`

	ServiceId string `json:"serviceId,omitempty" yaml:"service_id,omitempty"`

	State string `json:"state,omitempty" yaml:"state,omitempty"`

	Transitioning string `json:"transitioning,omitempty" yaml:"transitioning,omitempty"`

	TransitioningMessage string `json:"transitioningMessage,omitempty" yaml:"transitioning_message,omitempty"`

	TransitioningProgress int64 `json:"transitioningProgress,omitempty" yaml:"transitioning_progress,omitempty"`

	Uuid string `json:"uuid,omitempty" yaml:"uuid,omitempty"`

	Weight int64 `json:"weight,omitempty" yaml:"weight,omitempty"`
}

type LoadBalancerCollection struct {
	Collection
	Data []LoadBalancer `json:"data,omitempty"`
}

type LoadBalancerClient struct {
	rancherClient *RancherClient
}

type LoadBalancerOperations interface {
	List(opts *ListOpts) (*LoadBalancerCollection, error)
	Create(opts *LoadBalancer) (*LoadBalancer, error)
	Update(existing *LoadBalancer, updates interface{}) (*LoadBalancer, error)
	ById(id string) (*LoadBalancer, error)
	Delete(container *LoadBalancer) error

	ActionActivate(*LoadBalancer) (*LoadBalancer, error)

	ActionAddhost(*LoadBalancer, *AddRemoveLoadBalancerHostInput) (*LoadBalancer, error)

	ActionAddtarget(*LoadBalancer, *AddRemoveLoadBalancerTargetInput) (*LoadBalancer, error)

	ActionCreate(*LoadBalancer) (*LoadBalancer, error)

	ActionDeactivate(*LoadBalancer) (*LoadBalancer, error)

	ActionRemove(*LoadBalancer) (*LoadBalancer, error)

	ActionRemovehost(*LoadBalancer, *AddRemoveLoadBalancerHostInput) (*LoadBalancer, error)

	ActionRemovetarget(*LoadBalancer, *AddRemoveLoadBalancerTargetInput) (*LoadBalancer, error)

	ActionSethosts(*LoadBalancer, *SetLoadBalancerHostsInput) (*LoadBalancer, error)

	ActionSettargets(*LoadBalancer, *SetLoadBalancerTargetsInput) (*LoadBalancer, error)

	ActionUpdate(*LoadBalancer) (*LoadBalancer, error)
}

func newLoadBalancerClient(rancherClient *RancherClient) *LoadBalancerClient {
	return &LoadBalancerClient{
		rancherClient: rancherClient,
	}
}

func (c *LoadBalancerClient) Create(container *LoadBalancer) (*LoadBalancer, error) {
	resp := &LoadBalancer{}
	err := c.rancherClient.doCreate(LOAD_BALANCER_TYPE, container, resp)
	return resp, err
}

func (c *LoadBalancerClient) Update(existing *LoadBalancer, updates interface{}) (*LoadBalancer, error) {
	resp := &LoadBalancer{}
	err := c.rancherClient.doUpdate(LOAD_BALANCER_TYPE, &existing.Resource, updates, resp)
	return resp, err
}

func (c *LoadBalancerClient) List(opts *ListOpts) (*LoadBalancerCollection, error) {
	resp := &LoadBalancerCollection{}
	err := c.rancherClient.doList(LOAD_BALANCER_TYPE, opts, resp)
	return resp, err
}

func (c *LoadBalancerClient) ById(id string) (*LoadBalancer, error) {
	resp := &LoadBalancer{}
	err := c.rancherClient.doById(LOAD_BALANCER_TYPE, id, resp)
	if apiError, ok := err.(*ApiError); ok {
		if apiError.StatusCode == 404 {
			return nil, nil
		}
	}
	return resp, err
}

func (c *LoadBalancerClient) Delete(container *LoadBalancer) error {
	return c.rancherClient.doResourceDelete(LOAD_BALANCER_TYPE, &container.Resource)
}

func (c *LoadBalancerClient) ActionActivate(resource *LoadBalancer) (*LoadBalancer, error) {

	resp := &LoadBalancer{}

	err := c.rancherClient.doAction(LOAD_BALANCER_TYPE, "activate", &resource.Resource, nil, resp)

	return resp, err
}

func (c *LoadBalancerClient) ActionAddhost(resource *LoadBalancer, input *AddRemoveLoadBalancerHostInput) (*LoadBalancer, error) {

	resp := &LoadBalancer{}

	err := c.rancherClient.doAction(LOAD_BALANCER_TYPE, "addhost", &resource.Resource, input, resp)

	return resp, err
}

func (c *LoadBalancerClient) ActionAddtarget(resource *LoadBalancer, input *AddRemoveLoadBalancerTargetInput) (*LoadBalancer, error) {

	resp := &LoadBalancer{}

	err := c.rancherClient.doAction(LOAD_BALANCER_TYPE, "addtarget", &resource.Resource, input, resp)

	return resp, err
}

func (c *LoadBalancerClient) ActionCreate(resource *LoadBalancer) (*LoadBalancer, error) {

	resp := &LoadBalancer{}

	err := c.rancherClient.doAction(LOAD_BALANCER_TYPE, "create", &resource.Resource, nil, resp)

	return resp, err
}

func (c *LoadBalancerClient) ActionDeactivate(resource *LoadBalancer) (*LoadBalancer, error) {

	resp := &LoadBalancer{}

	err := c.rancherClient.doAction(LOAD_BALANCER_TYPE, "deactivate", &resource.Resource, nil, resp)

	return resp, err
}

func (c *LoadBalancerClient) ActionRemove(resource *LoadBalancer) (*LoadBalancer, error) {

	resp := &LoadBalancer{}

	err := c.rancherClient.doAction(LOAD_BALANCER_TYPE, "remove", &resource.Resource, nil, resp)

	return resp, err
}

func (c *LoadBalancerClient) ActionRemovehost(resource *LoadBalancer, input *AddRemoveLoadBalancerHostInput) (*LoadBalancer, error) {

	resp := &LoadBalancer{}

	err := c.rancherClient.doAction(LOAD_BALANCER_TYPE, "removehost", &resource.Resource, input, resp)

	return resp, err
}

func (c *LoadBalancerClient) ActionRemovetarget(resource *LoadBalancer, input *AddRemoveLoadBalancerTargetInput) (*LoadBalancer, error) {

	resp := &LoadBalancer{}

	err := c.rancherClient.doAction(LOAD_BALANCER_TYPE, "removetarget", &resource.Resource, input, resp)

	return resp, err
}

func (c *LoadBalancerClient) ActionSethosts(resource *LoadBalancer, input *SetLoadBalancerHostsInput) (*LoadBalancer, error) {

	resp := &LoadBalancer{}

	err := c.rancherClient.doAction(LOAD_BALANCER_TYPE, "sethosts", &resource.Resource, input, resp)

	return resp, err
}

func (c *LoadBalancerClient) ActionSettargets(resource *LoadBalancer, input *SetLoadBalancerTargetsInput) (*LoadBalancer, error) {

	resp := &LoadBalancer{}

	err := c.rancherClient.doAction(LOAD_BALANCER_TYPE, "settargets", &resource.Resource, input, resp)

	return resp, err
}

func (c *LoadBalancerClient) ActionUpdate(resource *LoadBalancer) (*LoadBalancer, error) {

	resp := &LoadBalancer{}

	err := c.rancherClient.doAction(LOAD_BALANCER_TYPE, "update", &resource.Resource, nil, resp)

	return resp, err
}
