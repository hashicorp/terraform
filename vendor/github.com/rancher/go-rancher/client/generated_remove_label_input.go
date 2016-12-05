package client

const (
	REMOVE_LABEL_INPUT_TYPE = "removeLabelInput"
)

type RemoveLabelInput struct {
	Resource

	Label string `json:"label,omitempty"`
}

type RemoveLabelInputCollection struct {
	Collection
	Data []RemoveLabelInput `json:"data,omitempty"`
}

type RemoveLabelInputClient struct {
	rancherClient *RancherClient
}

type RemoveLabelInputOperations interface {
	List(opts *ListOpts) (*RemoveLabelInputCollection, error)
	Create(opts *RemoveLabelInput) (*RemoveLabelInput, error)
	Update(existing *RemoveLabelInput, updates interface{}) (*RemoveLabelInput, error)
	ById(id string) (*RemoveLabelInput, error)
	Delete(container *RemoveLabelInput) error
}

func newRemoveLabelInputClient(rancherClient *RancherClient) *RemoveLabelInputClient {
	return &RemoveLabelInputClient{
		rancherClient: rancherClient,
	}
}

func (c *RemoveLabelInputClient) Create(container *RemoveLabelInput) (*RemoveLabelInput, error) {
	resp := &RemoveLabelInput{}
	err := c.rancherClient.doCreate(REMOVE_LABEL_INPUT_TYPE, container, resp)
	return resp, err
}

func (c *RemoveLabelInputClient) Update(existing *RemoveLabelInput, updates interface{}) (*RemoveLabelInput, error) {
	resp := &RemoveLabelInput{}
	err := c.rancherClient.doUpdate(REMOVE_LABEL_INPUT_TYPE, &existing.Resource, updates, resp)
	return resp, err
}

func (c *RemoveLabelInputClient) List(opts *ListOpts) (*RemoveLabelInputCollection, error) {
	resp := &RemoveLabelInputCollection{}
	err := c.rancherClient.doList(REMOVE_LABEL_INPUT_TYPE, opts, resp)
	return resp, err
}

func (c *RemoveLabelInputClient) ById(id string) (*RemoveLabelInput, error) {
	resp := &RemoveLabelInput{}
	err := c.rancherClient.doById(REMOVE_LABEL_INPUT_TYPE, id, resp)
	return resp, err
}

func (c *RemoveLabelInputClient) Delete(container *RemoveLabelInput) error {
	return c.rancherClient.doResourceDelete(REMOVE_LABEL_INPUT_TYPE, &container.Resource)
}
