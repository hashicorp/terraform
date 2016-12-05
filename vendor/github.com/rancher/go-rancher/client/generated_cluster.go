package client

const (
	CLUSTER_TYPE = "cluster"
)

type Cluster struct {
	Resource

	AccountId string `json:"accountId,omitempty" yaml:"account_id,omitempty"`

	AgentId string `json:"agentId,omitempty" yaml:"agent_id,omitempty"`

	AgentState string `json:"agentState,omitempty" yaml:"agent_state,omitempty"`

	ApiProxy string `json:"apiProxy,omitempty" yaml:"api_proxy,omitempty"`

	ComputeTotal int64 `json:"computeTotal,omitempty" yaml:"compute_total,omitempty"`

	Created string `json:"created,omitempty" yaml:"created,omitempty"`

	Data map[string]interface{} `json:"data,omitempty" yaml:"data,omitempty"`

	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	DiscoverySpec string `json:"discoverySpec,omitempty" yaml:"discovery_spec,omitempty"`

	Hostname string `json:"hostname,omitempty" yaml:"hostname,omitempty"`

	Info interface{} `json:"info,omitempty" yaml:"info,omitempty"`

	Kind string `json:"kind,omitempty" yaml:"kind,omitempty"`

	Labels map[string]interface{} `json:"labels,omitempty" yaml:"labels,omitempty"`

	Name string `json:"name,omitempty" yaml:"name,omitempty"`

	PhysicalHostId string `json:"physicalHostId,omitempty" yaml:"physical_host_id,omitempty"`

	Port int64 `json:"port,omitempty" yaml:"port,omitempty"`

	PublicEndpoints []interface{} `json:"publicEndpoints,omitempty" yaml:"public_endpoints,omitempty"`

	RemoveTime string `json:"removeTime,omitempty" yaml:"remove_time,omitempty"`

	Removed string `json:"removed,omitempty" yaml:"removed,omitempty"`

	State string `json:"state,omitempty" yaml:"state,omitempty"`

	Transitioning string `json:"transitioning,omitempty" yaml:"transitioning,omitempty"`

	TransitioningMessage string `json:"transitioningMessage,omitempty" yaml:"transitioning_message,omitempty"`

	TransitioningProgress int64 `json:"transitioningProgress,omitempty" yaml:"transitioning_progress,omitempty"`

	Uuid string `json:"uuid,omitempty" yaml:"uuid,omitempty"`
}

type ClusterCollection struct {
	Collection
	Data []Cluster `json:"data,omitempty"`
}

type ClusterClient struct {
	rancherClient *RancherClient
}

type ClusterOperations interface {
	List(opts *ListOpts) (*ClusterCollection, error)
	Create(opts *Cluster) (*Cluster, error)
	Update(existing *Cluster, updates interface{}) (*Cluster, error)
	ById(id string) (*Cluster, error)
	Delete(container *Cluster) error

	ActionActivate(*Cluster) (*Host, error)

	ActionAddhost(*Cluster, *AddRemoveClusterHostInput) (*Cluster, error)

	ActionCreate(*Cluster) (*Host, error)

	ActionDeactivate(*Cluster) (*Host, error)

	ActionDockersocket(*Cluster) (*HostAccess, error)

	ActionPurge(*Cluster) (*Host, error)

	ActionRemove(*Cluster) (*Host, error)

	ActionRemovehost(*Cluster, *AddRemoveClusterHostInput) (*Cluster, error)

	ActionRestore(*Cluster) (*Host, error)

	ActionUpdate(*Cluster) (*Host, error)
}

func newClusterClient(rancherClient *RancherClient) *ClusterClient {
	return &ClusterClient{
		rancherClient: rancherClient,
	}
}

func (c *ClusterClient) Create(container *Cluster) (*Cluster, error) {
	resp := &Cluster{}
	err := c.rancherClient.doCreate(CLUSTER_TYPE, container, resp)
	return resp, err
}

func (c *ClusterClient) Update(existing *Cluster, updates interface{}) (*Cluster, error) {
	resp := &Cluster{}
	err := c.rancherClient.doUpdate(CLUSTER_TYPE, &existing.Resource, updates, resp)
	return resp, err
}

func (c *ClusterClient) List(opts *ListOpts) (*ClusterCollection, error) {
	resp := &ClusterCollection{}
	err := c.rancherClient.doList(CLUSTER_TYPE, opts, resp)
	return resp, err
}

func (c *ClusterClient) ById(id string) (*Cluster, error) {
	resp := &Cluster{}
	err := c.rancherClient.doById(CLUSTER_TYPE, id, resp)
	if apiError, ok := err.(*ApiError); ok {
		if apiError.StatusCode == 404 {
			return nil, nil
		}
	}
	return resp, err
}

func (c *ClusterClient) Delete(container *Cluster) error {
	return c.rancherClient.doResourceDelete(CLUSTER_TYPE, &container.Resource)
}

func (c *ClusterClient) ActionActivate(resource *Cluster) (*Host, error) {

	resp := &Host{}

	err := c.rancherClient.doAction(CLUSTER_TYPE, "activate", &resource.Resource, nil, resp)

	return resp, err
}

func (c *ClusterClient) ActionAddhost(resource *Cluster, input *AddRemoveClusterHostInput) (*Cluster, error) {

	resp := &Cluster{}

	err := c.rancherClient.doAction(CLUSTER_TYPE, "addhost", &resource.Resource, input, resp)

	return resp, err
}

func (c *ClusterClient) ActionCreate(resource *Cluster) (*Host, error) {

	resp := &Host{}

	err := c.rancherClient.doAction(CLUSTER_TYPE, "create", &resource.Resource, nil, resp)

	return resp, err
}

func (c *ClusterClient) ActionDeactivate(resource *Cluster) (*Host, error) {

	resp := &Host{}

	err := c.rancherClient.doAction(CLUSTER_TYPE, "deactivate", &resource.Resource, nil, resp)

	return resp, err
}

func (c *ClusterClient) ActionDockersocket(resource *Cluster) (*HostAccess, error) {

	resp := &HostAccess{}

	err := c.rancherClient.doAction(CLUSTER_TYPE, "dockersocket", &resource.Resource, nil, resp)

	return resp, err
}

func (c *ClusterClient) ActionPurge(resource *Cluster) (*Host, error) {

	resp := &Host{}

	err := c.rancherClient.doAction(CLUSTER_TYPE, "purge", &resource.Resource, nil, resp)

	return resp, err
}

func (c *ClusterClient) ActionRemove(resource *Cluster) (*Host, error) {

	resp := &Host{}

	err := c.rancherClient.doAction(CLUSTER_TYPE, "remove", &resource.Resource, nil, resp)

	return resp, err
}

func (c *ClusterClient) ActionRemovehost(resource *Cluster, input *AddRemoveClusterHostInput) (*Cluster, error) {

	resp := &Cluster{}

	err := c.rancherClient.doAction(CLUSTER_TYPE, "removehost", &resource.Resource, input, resp)

	return resp, err
}

func (c *ClusterClient) ActionRestore(resource *Cluster) (*Host, error) {

	resp := &Host{}

	err := c.rancherClient.doAction(CLUSTER_TYPE, "restore", &resource.Resource, nil, resp)

	return resp, err
}

func (c *ClusterClient) ActionUpdate(resource *Cluster) (*Host, error) {

	resp := &Host{}

	err := c.rancherClient.doAction(CLUSTER_TYPE, "update", &resource.Resource, nil, resp)

	return resp, err
}
