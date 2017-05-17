package client

const (
	GLOBAL_LOAD_BALANCER_HEALTH_CHECK_TYPE = "globalLoadBalancerHealthCheck"
)

type GlobalLoadBalancerHealthCheck struct {
	Resource

	Name string `json:"name,omitempty" yaml:"name,omitempty"`
}

type GlobalLoadBalancerHealthCheckCollection struct {
	Collection
	Data []GlobalLoadBalancerHealthCheck `json:"data,omitempty"`
}

type GlobalLoadBalancerHealthCheckClient struct {
	rancherClient *RancherClient
}

type GlobalLoadBalancerHealthCheckOperations interface {
	List(opts *ListOpts) (*GlobalLoadBalancerHealthCheckCollection, error)
	Create(opts *GlobalLoadBalancerHealthCheck) (*GlobalLoadBalancerHealthCheck, error)
	Update(existing *GlobalLoadBalancerHealthCheck, updates interface{}) (*GlobalLoadBalancerHealthCheck, error)
	ById(id string) (*GlobalLoadBalancerHealthCheck, error)
	Delete(container *GlobalLoadBalancerHealthCheck) error
}

func newGlobalLoadBalancerHealthCheckClient(rancherClient *RancherClient) *GlobalLoadBalancerHealthCheckClient {
	return &GlobalLoadBalancerHealthCheckClient{
		rancherClient: rancherClient,
	}
}

func (c *GlobalLoadBalancerHealthCheckClient) Create(container *GlobalLoadBalancerHealthCheck) (*GlobalLoadBalancerHealthCheck, error) {
	resp := &GlobalLoadBalancerHealthCheck{}
	err := c.rancherClient.doCreate(GLOBAL_LOAD_BALANCER_HEALTH_CHECK_TYPE, container, resp)
	return resp, err
}

func (c *GlobalLoadBalancerHealthCheckClient) Update(existing *GlobalLoadBalancerHealthCheck, updates interface{}) (*GlobalLoadBalancerHealthCheck, error) {
	resp := &GlobalLoadBalancerHealthCheck{}
	err := c.rancherClient.doUpdate(GLOBAL_LOAD_BALANCER_HEALTH_CHECK_TYPE, &existing.Resource, updates, resp)
	return resp, err
}

func (c *GlobalLoadBalancerHealthCheckClient) List(opts *ListOpts) (*GlobalLoadBalancerHealthCheckCollection, error) {
	resp := &GlobalLoadBalancerHealthCheckCollection{}
	err := c.rancherClient.doList(GLOBAL_LOAD_BALANCER_HEALTH_CHECK_TYPE, opts, resp)
	return resp, err
}

func (c *GlobalLoadBalancerHealthCheckClient) ById(id string) (*GlobalLoadBalancerHealthCheck, error) {
	resp := &GlobalLoadBalancerHealthCheck{}
	err := c.rancherClient.doById(GLOBAL_LOAD_BALANCER_HEALTH_CHECK_TYPE, id, resp)
	if apiError, ok := err.(*ApiError); ok {
		if apiError.StatusCode == 404 {
			return nil, nil
		}
	}
	return resp, err
}

func (c *GlobalLoadBalancerHealthCheckClient) Delete(container *GlobalLoadBalancerHealthCheck) error {
	return c.rancherClient.doResourceDelete(GLOBAL_LOAD_BALANCER_HEALTH_CHECK_TYPE, &container.Resource)
}
