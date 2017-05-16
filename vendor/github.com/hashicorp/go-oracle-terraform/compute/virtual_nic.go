package compute

type VirtNICsClient struct {
	ResourceClient
}

func (c *Client) VirtNICs() *VirtNICsClient {
	return &VirtNICsClient{
		ResourceClient: ResourceClient{
			Client:              c,
			ResourceDescription: "Virtual NIC",
			ContainerPath:       "/network/v1/vnic/",
			ResourceRootPath:    "/network/v1/vnic",
		},
	}
}

type VirtualNIC struct {
	// Description of the object.
	Description string `json:"description"`
	// MAC address of this VNIC.
	MACAddress string `json:"macAddress"`
	// The three-part name (/Compute-identity_domain/user/object) of the Virtual NIC.
	Name string `json:"name"`
	// Tags associated with the object.
	Tags []string `json:"tags"`
	// True if the VNIC is of type "transit".
	TransitFlag bool `json:"transitFlag"`
	// Uniform Resource Identifier
	Uri string `json:"uri"`
}

// Can only GET a virtual NIC, not update, create, or delete
type GetVirtualNICInput struct {
	// The three-part name (/Compute-identity_domain/user/object) of the Virtual NIC.
	// Required
	Name string `json:"name"`
}

func (c *VirtNICsClient) GetVirtualNIC(input *GetVirtualNICInput) (*VirtualNIC, error) {
	var virtNIC VirtualNIC
	input.Name = c.getQualifiedName(input.Name)
	if err := c.getResource(input.Name, &virtNIC); err != nil {
		return nil, err
	}
	return c.success(&virtNIC)
}

func (c *VirtNICsClient) success(info *VirtualNIC) (*VirtualNIC, error) {
	c.unqualify(&info.Name)
	return info, nil
}
