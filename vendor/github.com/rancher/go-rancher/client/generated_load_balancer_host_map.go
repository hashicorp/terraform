package client

const (
	LOAD_BALANCER_HOST_MAP_TYPE = "loadBalancerHostMap"
)

type LoadBalancerHostMap struct {
	Resource

	AccountId string `json:"accountId,omitempty" yaml:"account_id,omitempty"`

	Created string `json:"created,omitempty" yaml:"created,omitempty"`

	Data map[string]interface{} `json:"data,omitempty" yaml:"data,omitempty"`

	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	HostId string `json:"hostId,omitempty" yaml:"host_id,omitempty"`

	Kind string `json:"kind,omitempty" yaml:"kind,omitempty"`

	LoadBalancerId string `json:"loadBalancerId,omitempty" yaml:"load_balancer_id,omitempty"`

	Name string `json:"name,omitempty" yaml:"name,omitempty"`

	RemoveTime string `json:"removeTime,omitempty" yaml:"remove_time,omitempty"`

	Removed string `json:"removed,omitempty" yaml:"removed,omitempty"`

	State string `json:"state,omitempty" yaml:"state,omitempty"`

	Uuid string `json:"uuid,omitempty" yaml:"uuid,omitempty"`
}

type LoadBalancerHostMapCollection struct {
	Collection
	Data []LoadBalancerHostMap `json:"data,omitempty"`
}

type LoadBalancerHostMapClient struct {
	rancherClient *RancherClient
}

type LoadBalancerHostMapOperations interface {
	List(opts *ListOpts) (*LoadBalancerHostMapCollection, error)
	Create(opts *LoadBalancerHostMap) (*LoadBalancerHostMap, error)
	Update(existing *LoadBalancerHostMap, updates interface{}) (*LoadBalancerHostMap, error)
	ById(id string) (*LoadBalancerHostMap, error)
	Delete(container *LoadBalancerHostMap) error
}

func newLoadBalancerHostMapClient(rancherClient *RancherClient) *LoadBalancerHostMapClient {
	return &LoadBalancerHostMapClient{
		rancherClient: rancherClient,
	}
}

func (c *LoadBalancerHostMapClient) Create(container *LoadBalancerHostMap) (*LoadBalancerHostMap, error) {
	resp := &LoadBalancerHostMap{}
	err := c.rancherClient.doCreate(LOAD_BALANCER_HOST_MAP_TYPE, container, resp)
	return resp, err
}

func (c *LoadBalancerHostMapClient) Update(existing *LoadBalancerHostMap, updates interface{}) (*LoadBalancerHostMap, error) {
	resp := &LoadBalancerHostMap{}
	err := c.rancherClient.doUpdate(LOAD_BALANCER_HOST_MAP_TYPE, &existing.Resource, updates, resp)
	return resp, err
}

func (c *LoadBalancerHostMapClient) List(opts *ListOpts) (*LoadBalancerHostMapCollection, error) {
	resp := &LoadBalancerHostMapCollection{}
	err := c.rancherClient.doList(LOAD_BALANCER_HOST_MAP_TYPE, opts, resp)
	return resp, err
}

func (c *LoadBalancerHostMapClient) ById(id string) (*LoadBalancerHostMap, error) {
	resp := &LoadBalancerHostMap{}
	err := c.rancherClient.doById(LOAD_BALANCER_HOST_MAP_TYPE, id, resp)
	if apiError, ok := err.(*ApiError); ok {
		if apiError.StatusCode == 404 {
			return nil, nil
		}
	}
	return resp, err
}

func (c *LoadBalancerHostMapClient) Delete(container *LoadBalancerHostMap) error {
	return c.rancherClient.doResourceDelete(LOAD_BALANCER_HOST_MAP_TYPE, &container.Resource)
}
