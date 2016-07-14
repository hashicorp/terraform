package client

const (
	MACHINE_DRIVER_ERROR_INPUT_TYPE = "machineDriverErrorInput"
)

type MachineDriverErrorInput struct {
	Resource

	ErrorMessage string `json:"errorMessage,omitempty" yaml:"error_message,omitempty"`
}

type MachineDriverErrorInputCollection struct {
	Collection
	Data []MachineDriverErrorInput `json:"data,omitempty"`
}

type MachineDriverErrorInputClient struct {
	rancherClient *RancherClient
}

type MachineDriverErrorInputOperations interface {
	List(opts *ListOpts) (*MachineDriverErrorInputCollection, error)
	Create(opts *MachineDriverErrorInput) (*MachineDriverErrorInput, error)
	Update(existing *MachineDriverErrorInput, updates interface{}) (*MachineDriverErrorInput, error)
	ById(id string) (*MachineDriverErrorInput, error)
	Delete(container *MachineDriverErrorInput) error
}

func newMachineDriverErrorInputClient(rancherClient *RancherClient) *MachineDriverErrorInputClient {
	return &MachineDriverErrorInputClient{
		rancherClient: rancherClient,
	}
}

func (c *MachineDriverErrorInputClient) Create(container *MachineDriverErrorInput) (*MachineDriverErrorInput, error) {
	resp := &MachineDriverErrorInput{}
	err := c.rancherClient.doCreate(MACHINE_DRIVER_ERROR_INPUT_TYPE, container, resp)
	return resp, err
}

func (c *MachineDriverErrorInputClient) Update(existing *MachineDriverErrorInput, updates interface{}) (*MachineDriverErrorInput, error) {
	resp := &MachineDriverErrorInput{}
	err := c.rancherClient.doUpdate(MACHINE_DRIVER_ERROR_INPUT_TYPE, &existing.Resource, updates, resp)
	return resp, err
}

func (c *MachineDriverErrorInputClient) List(opts *ListOpts) (*MachineDriverErrorInputCollection, error) {
	resp := &MachineDriverErrorInputCollection{}
	err := c.rancherClient.doList(MACHINE_DRIVER_ERROR_INPUT_TYPE, opts, resp)
	return resp, err
}

func (c *MachineDriverErrorInputClient) ById(id string) (*MachineDriverErrorInput, error) {
	resp := &MachineDriverErrorInput{}
	err := c.rancherClient.doById(MACHINE_DRIVER_ERROR_INPUT_TYPE, id, resp)
	if apiError, ok := err.(*ApiError); ok {
		if apiError.StatusCode == 404 {
			return nil, nil
		}
	}
	return resp, err
}

func (c *MachineDriverErrorInputClient) Delete(container *MachineDriverErrorInput) error {
	return c.rancherClient.doResourceDelete(MACHINE_DRIVER_ERROR_INPUT_TYPE, &container.Resource)
}
