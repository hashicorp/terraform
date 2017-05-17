package client

const (
	MACHINE_DRIVER_UPDATE_INPUT_TYPE = "machineDriverUpdateInput"
)

type MachineDriverUpdateInput struct {
	Resource

	Md5checksum string `json:"md5checksum,omitempty" yaml:"md5checksum,omitempty"`

	Name string `json:"name,omitempty" yaml:"name,omitempty"`

	Uri string `json:"uri,omitempty" yaml:"uri,omitempty"`
}

type MachineDriverUpdateInputCollection struct {
	Collection
	Data []MachineDriverUpdateInput `json:"data,omitempty"`
}

type MachineDriverUpdateInputClient struct {
	rancherClient *RancherClient
}

type MachineDriverUpdateInputOperations interface {
	List(opts *ListOpts) (*MachineDriverUpdateInputCollection, error)
	Create(opts *MachineDriverUpdateInput) (*MachineDriverUpdateInput, error)
	Update(existing *MachineDriverUpdateInput, updates interface{}) (*MachineDriverUpdateInput, error)
	ById(id string) (*MachineDriverUpdateInput, error)
	Delete(container *MachineDriverUpdateInput) error
}

func newMachineDriverUpdateInputClient(rancherClient *RancherClient) *MachineDriverUpdateInputClient {
	return &MachineDriverUpdateInputClient{
		rancherClient: rancherClient,
	}
}

func (c *MachineDriverUpdateInputClient) Create(container *MachineDriverUpdateInput) (*MachineDriverUpdateInput, error) {
	resp := &MachineDriverUpdateInput{}
	err := c.rancherClient.doCreate(MACHINE_DRIVER_UPDATE_INPUT_TYPE, container, resp)
	return resp, err
}

func (c *MachineDriverUpdateInputClient) Update(existing *MachineDriverUpdateInput, updates interface{}) (*MachineDriverUpdateInput, error) {
	resp := &MachineDriverUpdateInput{}
	err := c.rancherClient.doUpdate(MACHINE_DRIVER_UPDATE_INPUT_TYPE, &existing.Resource, updates, resp)
	return resp, err
}

func (c *MachineDriverUpdateInputClient) List(opts *ListOpts) (*MachineDriverUpdateInputCollection, error) {
	resp := &MachineDriverUpdateInputCollection{}
	err := c.rancherClient.doList(MACHINE_DRIVER_UPDATE_INPUT_TYPE, opts, resp)
	return resp, err
}

func (c *MachineDriverUpdateInputClient) ById(id string) (*MachineDriverUpdateInput, error) {
	resp := &MachineDriverUpdateInput{}
	err := c.rancherClient.doById(MACHINE_DRIVER_UPDATE_INPUT_TYPE, id, resp)
	if apiError, ok := err.(*ApiError); ok {
		if apiError.StatusCode == 404 {
			return nil, nil
		}
	}
	return resp, err
}

func (c *MachineDriverUpdateInputClient) Delete(container *MachineDriverUpdateInput) error {
	return c.rancherClient.doResourceDelete(MACHINE_DRIVER_UPDATE_INPUT_TYPE, &container.Resource)
}
