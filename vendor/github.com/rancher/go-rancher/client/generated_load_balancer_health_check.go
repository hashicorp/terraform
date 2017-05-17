package client

const (
	LOAD_BALANCER_HEALTH_CHECK_TYPE = "loadBalancerHealthCheck"
)

type LoadBalancerHealthCheck struct {
	Resource

	HealthyThreshold int64 `json:"healthyThreshold,omitempty" yaml:"healthy_threshold,omitempty"`

	Interval int64 `json:"interval,omitempty" yaml:"interval,omitempty"`

	Name string `json:"name,omitempty" yaml:"name,omitempty"`

	Port int64 `json:"port,omitempty" yaml:"port,omitempty"`

	RequestLine string `json:"requestLine,omitempty" yaml:"request_line,omitempty"`

	ResponseTimeout int64 `json:"responseTimeout,omitempty" yaml:"response_timeout,omitempty"`

	UnhealthyThreshold int64 `json:"unhealthyThreshold,omitempty" yaml:"unhealthy_threshold,omitempty"`
}

type LoadBalancerHealthCheckCollection struct {
	Collection
	Data []LoadBalancerHealthCheck `json:"data,omitempty"`
}

type LoadBalancerHealthCheckClient struct {
	rancherClient *RancherClient
}

type LoadBalancerHealthCheckOperations interface {
	List(opts *ListOpts) (*LoadBalancerHealthCheckCollection, error)
	Create(opts *LoadBalancerHealthCheck) (*LoadBalancerHealthCheck, error)
	Update(existing *LoadBalancerHealthCheck, updates interface{}) (*LoadBalancerHealthCheck, error)
	ById(id string) (*LoadBalancerHealthCheck, error)
	Delete(container *LoadBalancerHealthCheck) error
}

func newLoadBalancerHealthCheckClient(rancherClient *RancherClient) *LoadBalancerHealthCheckClient {
	return &LoadBalancerHealthCheckClient{
		rancherClient: rancherClient,
	}
}

func (c *LoadBalancerHealthCheckClient) Create(container *LoadBalancerHealthCheck) (*LoadBalancerHealthCheck, error) {
	resp := &LoadBalancerHealthCheck{}
	err := c.rancherClient.doCreate(LOAD_BALANCER_HEALTH_CHECK_TYPE, container, resp)
	return resp, err
}

func (c *LoadBalancerHealthCheckClient) Update(existing *LoadBalancerHealthCheck, updates interface{}) (*LoadBalancerHealthCheck, error) {
	resp := &LoadBalancerHealthCheck{}
	err := c.rancherClient.doUpdate(LOAD_BALANCER_HEALTH_CHECK_TYPE, &existing.Resource, updates, resp)
	return resp, err
}

func (c *LoadBalancerHealthCheckClient) List(opts *ListOpts) (*LoadBalancerHealthCheckCollection, error) {
	resp := &LoadBalancerHealthCheckCollection{}
	err := c.rancherClient.doList(LOAD_BALANCER_HEALTH_CHECK_TYPE, opts, resp)
	return resp, err
}

func (c *LoadBalancerHealthCheckClient) ById(id string) (*LoadBalancerHealthCheck, error) {
	resp := &LoadBalancerHealthCheck{}
	err := c.rancherClient.doById(LOAD_BALANCER_HEALTH_CHECK_TYPE, id, resp)
	if apiError, ok := err.(*ApiError); ok {
		if apiError.StatusCode == 404 {
			return nil, nil
		}
	}
	return resp, err
}

func (c *LoadBalancerHealthCheckClient) Delete(container *LoadBalancerHealthCheck) error {
	return c.rancherClient.doResourceDelete(LOAD_BALANCER_HEALTH_CHECK_TYPE, &container.Resource)
}
