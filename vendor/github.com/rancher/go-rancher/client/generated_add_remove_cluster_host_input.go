package client

const (
	ADD_REMOVE_CLUSTER_HOST_INPUT_TYPE = "addRemoveClusterHostInput"
)

type AddRemoveClusterHostInput struct {
	Resource

	HostId string `json:"hostId,omitempty" yaml:"host_id,omitempty"`
}

type AddRemoveClusterHostInputCollection struct {
	Collection
	Data []AddRemoveClusterHostInput `json:"data,omitempty"`
}

type AddRemoveClusterHostInputClient struct {
	rancherClient *RancherClient
}

type AddRemoveClusterHostInputOperations interface {
	List(opts *ListOpts) (*AddRemoveClusterHostInputCollection, error)
	Create(opts *AddRemoveClusterHostInput) (*AddRemoveClusterHostInput, error)
	Update(existing *AddRemoveClusterHostInput, updates interface{}) (*AddRemoveClusterHostInput, error)
	ById(id string) (*AddRemoveClusterHostInput, error)
	Delete(container *AddRemoveClusterHostInput) error
}

func newAddRemoveClusterHostInputClient(rancherClient *RancherClient) *AddRemoveClusterHostInputClient {
	return &AddRemoveClusterHostInputClient{
		rancherClient: rancherClient,
	}
}

func (c *AddRemoveClusterHostInputClient) Create(container *AddRemoveClusterHostInput) (*AddRemoveClusterHostInput, error) {
	resp := &AddRemoveClusterHostInput{}
	err := c.rancherClient.doCreate(ADD_REMOVE_CLUSTER_HOST_INPUT_TYPE, container, resp)
	return resp, err
}

func (c *AddRemoveClusterHostInputClient) Update(existing *AddRemoveClusterHostInput, updates interface{}) (*AddRemoveClusterHostInput, error) {
	resp := &AddRemoveClusterHostInput{}
	err := c.rancherClient.doUpdate(ADD_REMOVE_CLUSTER_HOST_INPUT_TYPE, &existing.Resource, updates, resp)
	return resp, err
}

func (c *AddRemoveClusterHostInputClient) List(opts *ListOpts) (*AddRemoveClusterHostInputCollection, error) {
	resp := &AddRemoveClusterHostInputCollection{}
	err := c.rancherClient.doList(ADD_REMOVE_CLUSTER_HOST_INPUT_TYPE, opts, resp)
	return resp, err
}

func (c *AddRemoveClusterHostInputClient) ById(id string) (*AddRemoveClusterHostInput, error) {
	resp := &AddRemoveClusterHostInput{}
	err := c.rancherClient.doById(ADD_REMOVE_CLUSTER_HOST_INPUT_TYPE, id, resp)
	if apiError, ok := err.(*ApiError); ok {
		if apiError.StatusCode == 404 {
			return nil, nil
		}
	}
	return resp, err
}

func (c *AddRemoveClusterHostInputClient) Delete(container *AddRemoveClusterHostInput) error {
	return c.rancherClient.doResourceDelete(ADD_REMOVE_CLUSTER_HOST_INPUT_TYPE, &container.Resource)
}
