package compute

const (
	IPAddressAssociationDescription   = "ip address association"
	IPAddressAssociationContainerPath = "/network/v1/ipassociation/"
	IPAddressAssociationResourcePath  = "/network/v1/ipassociation"
)

type IPAddressAssociationsClient struct {
	ResourceClient
}

// IPAddressAssociations() returns an IPAddressAssociationsClient that can be used to access the
// necessary CRUD functions for IP Address Associations.
func (c *Client) IPAddressAssociations() *IPAddressAssociationsClient {
	return &IPAddressAssociationsClient{
		ResourceClient: ResourceClient{
			Client:              c,
			ResourceDescription: IPAddressAssociationDescription,
			ContainerPath:       IPAddressAssociationContainerPath,
			ResourceRootPath:    IPAddressAssociationResourcePath,
		},
	}
}

// IPAddressAssociationInfo contains the exported fields necessary to hold all the information about an
// IP Address Association
type IPAddressAssociationInfo struct {
	// The name of the NAT IP address reservation.
	IPAddressReservation string `json:"ipAddressReservation"`
	// Name of the virtual NIC associated with this NAT IP reservation.
	Vnic string `json:"vnic"`
	// The name of the IP Address Association
	Name string `json:"name"`
	// Description of the IP Address Association
	Description string `json:"description"`
	// Slice of tags associated with the IP Address Association
	Tags []string `json:"tags"`
	// Uniform Resource Identifier for the IP Address Association
	Uri string `json:"uri"`
}

type CreateIPAddressAssociationInput struct {
	// The name of the IP Address Association to create. Object names can only contain alphanumeric,
	// underscore, dash, and period characters. Names are case-sensitive.
	// Required
	Name string `json:"name"`

	// The name of the NAT IP address reservation.
	// Optional
	IPAddressReservation string `json:"ipAddressReservation,omitempty"`

	// Name of the virtual NIC associated with this NAT IP reservation.
	// Optional
	Vnic string `json:"vnic,omitempty"`

	// Description of the IPAddressAssociation
	// Optional
	Description string `json:"description"`

	// String slice of tags to apply to the IP Address Association object
	// Optional
	Tags []string `json:"tags"`
}

// Create a new IP Address Association from an IPAddressAssociationsClient and an input struct.
// Returns a populated Info struct for the IP Address Association, and any errors
func (c *IPAddressAssociationsClient) CreateIPAddressAssociation(input *CreateIPAddressAssociationInput) (*IPAddressAssociationInfo, error) {
	input.Name = c.getQualifiedName(input.Name)
	input.IPAddressReservation = c.getQualifiedName(input.IPAddressReservation)
	input.Vnic = c.getQualifiedName(input.Vnic)

	var ipInfo IPAddressAssociationInfo
	if err := c.createResource(&input, &ipInfo); err != nil {
		return nil, err
	}

	return c.success(&ipInfo)
}

type GetIPAddressAssociationInput struct {
	// The name of the IP Address Association to query for. Case-sensitive
	// Required
	Name string `json:"name"`
}

// Returns a populated IPAddressAssociationInfo struct from an input struct
func (c *IPAddressAssociationsClient) GetIPAddressAssociation(input *GetIPAddressAssociationInput) (*IPAddressAssociationInfo, error) {
	input.Name = c.getQualifiedName(input.Name)

	var ipInfo IPAddressAssociationInfo
	if err := c.getResource(input.Name, &ipInfo); err != nil {
		return nil, err
	}

	return c.success(&ipInfo)
}

// UpdateIPAddressAssociationInput defines what to update in a ip address association
type UpdateIPAddressAssociationInput struct {
	// The name of the IP Address Association to create. Object names can only contain alphanumeric,
	// underscore, dash, and period characters. Names are case-sensitive.
	// Required
	Name string `json:"name"`

	// The name of the NAT IP address reservation.
	// Optional
	IPAddressReservation string `json:"ipAddressReservation,omitempty"`

	// Name of the virtual NIC associated with this NAT IP reservation.
	// Optional
	Vnic string `json:"vnic,omitempty"`

	// Description of the IPAddressAssociation
	// Optional
	Description string `json:"description"`

	// String slice of tags to apply to the IP Address Association object
	// Optional
	Tags []string `json:"tags"`
}

// UpdateIPAddressAssociation update the ip address association
func (c *IPAddressAssociationsClient) UpdateIPAddressAssociation(updateInput *UpdateIPAddressAssociationInput) (*IPAddressAssociationInfo, error) {
	updateInput.Name = c.getQualifiedName(updateInput.Name)
	updateInput.IPAddressReservation = c.getQualifiedName(updateInput.IPAddressReservation)
	updateInput.Vnic = c.getQualifiedName(updateInput.Vnic)
	var ipInfo IPAddressAssociationInfo
	if err := c.updateResource(updateInput.Name, updateInput, &ipInfo); err != nil {
		return nil, err
	}

	return c.success(&ipInfo)
}

type DeleteIPAddressAssociationInput struct {
	// The name of the IP Address Association to query for. Case-sensitive
	// Required
	Name string `json:"name"`
}

func (c *IPAddressAssociationsClient) DeleteIPAddressAssociation(input *DeleteIPAddressAssociationInput) error {
	return c.deleteResource(input.Name)
}

// Unqualifies any qualified fields in the IPAddressAssociationInfo struct
func (c *IPAddressAssociationsClient) success(info *IPAddressAssociationInfo) (*IPAddressAssociationInfo, error) {
	c.unqualify(&info.Name)
	c.unqualify(&info.Vnic)
	c.unqualify(&info.IPAddressReservation)
	return info, nil
}
