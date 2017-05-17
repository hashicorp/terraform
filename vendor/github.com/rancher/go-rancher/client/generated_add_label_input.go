package client

const (
	ADD_LABEL_INPUT_TYPE = "addLabelInput"
)

type AddLabelInput struct {
	Resource

	Key string `json:"key,omitempty"`

	Value string `json:"value,omitempty"`
}

type AddLabelInputCollection struct {
	Collection
	Data []AddLabelInput `json:"data,omitempty"`
}

type AddLabelInputClient struct {
	rancherClient *RancherClient
}

type AddLabelInputOperations interface {
	List(opts *ListOpts) (*AddLabelInputCollection, error)
	Create(opts *AddLabelInput) (*AddLabelInput, error)
	Update(existing *AddLabelInput, updates interface{}) (*AddLabelInput, error)
	ById(id string) (*AddLabelInput, error)
	Delete(container *AddLabelInput) error
}

func newAddLabelInputClient(rancherClient *RancherClient) *AddLabelInputClient {
	return &AddLabelInputClient{
		rancherClient: rancherClient,
	}
}

func (c *AddLabelInputClient) Create(container *AddLabelInput) (*AddLabelInput, error) {
	resp := &AddLabelInput{}
	err := c.rancherClient.doCreate(ADD_LABEL_INPUT_TYPE, container, resp)
	return resp, err
}

func (c *AddLabelInputClient) Update(existing *AddLabelInput, updates interface{}) (*AddLabelInput, error) {
	resp := &AddLabelInput{}
	err := c.rancherClient.doUpdate(ADD_LABEL_INPUT_TYPE, &existing.Resource, updates, resp)
	return resp, err
}

func (c *AddLabelInputClient) List(opts *ListOpts) (*AddLabelInputCollection, error) {
	resp := &AddLabelInputCollection{}
	err := c.rancherClient.doList(ADD_LABEL_INPUT_TYPE, opts, resp)
	return resp, err
}

func (c *AddLabelInputClient) ById(id string) (*AddLabelInput, error) {
	resp := &AddLabelInput{}
	err := c.rancherClient.doById(ADD_LABEL_INPUT_TYPE, id, resp)
	return resp, err
}

func (c *AddLabelInputClient) Delete(container *AddLabelInput) error {
	return c.rancherClient.doResourceDelete(ADD_LABEL_INPUT_TYPE, &container.Resource)
}
