package client

const (
	GLOBAL_LOAD_BALANCER_POLICY_TYPE = "globalLoadBalancerPolicy"
)

type GlobalLoadBalancerPolicy struct {
	Resource

	Name string `json:"name,omitempty" yaml:"name,omitempty"`
}

type GlobalLoadBalancerPolicyCollection struct {
	Collection
	Data []GlobalLoadBalancerPolicy `json:"data,omitempty"`
}

type GlobalLoadBalancerPolicyClient struct {
	rancherClient *RancherClient
}

type GlobalLoadBalancerPolicyOperations interface {
	List(opts *ListOpts) (*GlobalLoadBalancerPolicyCollection, error)
	Create(opts *GlobalLoadBalancerPolicy) (*GlobalLoadBalancerPolicy, error)
	Update(existing *GlobalLoadBalancerPolicy, updates interface{}) (*GlobalLoadBalancerPolicy, error)
	ById(id string) (*GlobalLoadBalancerPolicy, error)
	Delete(container *GlobalLoadBalancerPolicy) error
}

func newGlobalLoadBalancerPolicyClient(rancherClient *RancherClient) *GlobalLoadBalancerPolicyClient {
	return &GlobalLoadBalancerPolicyClient{
		rancherClient: rancherClient,
	}
}

func (c *GlobalLoadBalancerPolicyClient) Create(container *GlobalLoadBalancerPolicy) (*GlobalLoadBalancerPolicy, error) {
	resp := &GlobalLoadBalancerPolicy{}
	err := c.rancherClient.doCreate(GLOBAL_LOAD_BALANCER_POLICY_TYPE, container, resp)
	return resp, err
}

func (c *GlobalLoadBalancerPolicyClient) Update(existing *GlobalLoadBalancerPolicy, updates interface{}) (*GlobalLoadBalancerPolicy, error) {
	resp := &GlobalLoadBalancerPolicy{}
	err := c.rancherClient.doUpdate(GLOBAL_LOAD_BALANCER_POLICY_TYPE, &existing.Resource, updates, resp)
	return resp, err
}

func (c *GlobalLoadBalancerPolicyClient) List(opts *ListOpts) (*GlobalLoadBalancerPolicyCollection, error) {
	resp := &GlobalLoadBalancerPolicyCollection{}
	err := c.rancherClient.doList(GLOBAL_LOAD_BALANCER_POLICY_TYPE, opts, resp)
	return resp, err
}

func (c *GlobalLoadBalancerPolicyClient) ById(id string) (*GlobalLoadBalancerPolicy, error) {
	resp := &GlobalLoadBalancerPolicy{}
	err := c.rancherClient.doById(GLOBAL_LOAD_BALANCER_POLICY_TYPE, id, resp)
	if apiError, ok := err.(*ApiError); ok {
		if apiError.StatusCode == 404 {
			return nil, nil
		}
	}
	return resp, err
}

func (c *GlobalLoadBalancerPolicyClient) Delete(container *GlobalLoadBalancerPolicy) error {
	return c.rancherClient.doResourceDelete(GLOBAL_LOAD_BALANCER_POLICY_TYPE, &container.Resource)
}
