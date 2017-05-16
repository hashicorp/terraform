package compute

type VirtNICSetsClient struct {
	ResourceClient
}

func (c *Client) VirtNICSets() *VirtNICSetsClient {
	return &VirtNICSetsClient{
		ResourceClient: ResourceClient{
			Client:              c,
			ResourceDescription: "Virtual NIC Set",
			ContainerPath:       "/network/v1/vnicset/",
			ResourceRootPath:    "/network/v1/vnicset",
		},
	}
}

// Describes an existing virtual nic set
type VirtualNICSet struct {
	// List of ACLs applied to the VNICs in the set.
	AppliedACLs []string `json:"appliedAcls"`
	// Description of the VNIC Set.
	Description string `json:"description"`
	// Name of the VNIC set.
	Name string `json:"name"`
	// The three-part name (/Compute-identity_domain/user/object) of the virtual NIC set.
	Tags []string `json:"tags"`
	// Uniform Resource Identifier
	Uri string `json:"uri"`
	// List of VNICs associated with this VNIC set.
	VirtualNICs []string `json:"vnics"`
}

type CreateVirtualNICSetInput struct {
	// List of ACLs applied to the VNICs in the set.
	// Optional
	AppliedACLs []string `json:"appliedAcls"`
	// Description of the object.
	// Optional
	Description string `json:"description"`
	// The three-part name (/Compute-identity_domain/user/object) of the virtual NIC set.
	// Object names can contain only alphanumeric, underscore (_), dash (-), and period (.) characters. Object names are case-sensitive.
	// Required
	Name string `json:"name"`
	// Tags associated with this VNIC set.
	// Optional
	Tags []string `json:"tags"`
	// List of VNICs associated with this VNIC set.
	// Optional
	VirtualNICs []string `json:"vnics"`
}

func (c *VirtNICSetsClient) CreateVirtualNICSet(input *CreateVirtualNICSetInput) (*VirtualNICSet, error) {
	input.Name = c.getQualifiedName(input.Name)
	input.AppliedACLs = c.getQualifiedAcls(input.AppliedACLs)
	qualifiedNics := c.getQualifiedList(input.VirtualNICs)
	if len(qualifiedNics) != 0 {
		input.VirtualNICs = qualifiedNics
	}

	var virtNicSet VirtualNICSet
	if err := c.createResource(input, &virtNicSet); err != nil {
		return nil, err
	}

	return c.success(&virtNicSet)
}

type GetVirtualNICSetInput struct {
	// The three-part name (/Compute-identity_domain/user/object) of the virtual NIC set.
	// Required
	Name string `json:"name"`
}

func (c *VirtNICSetsClient) GetVirtualNICSet(input *GetVirtualNICSetInput) (*VirtualNICSet, error) {
	var virtNicSet VirtualNICSet
	// Qualify Name
	input.Name = c.getQualifiedName(input.Name)
	if err := c.getResource(input.Name, &virtNicSet); err != nil {
		return nil, err
	}

	return c.success(&virtNicSet)
}

type UpdateVirtualNICSetInput struct {
	// List of ACLs applied to the VNICs in the set.
	// Optional
	AppliedACLs []string `json:"appliedAcls"`
	// Description of the object.
	// Optional
	Description string `json:"description"`
	// The three-part name (/Compute-identity_domain/user/object) of the virtual NIC set.
	// Object names can contain only alphanumeric, underscore (_), dash (-), and period (.) characters. Object names are case-sensitive.
	// Required
	Name string `json:"name"`
	// Tags associated with this VNIC set.
	// Optional
	Tags []string `json:"tags"`
	// List of VNICs associated with this VNIC set.
	// Optional
	VirtualNICs []string `json:"vnics"`
}

func (c *VirtNICSetsClient) UpdateVirtualNICSet(input *UpdateVirtualNICSetInput) (*VirtualNICSet, error) {
	input.Name = c.getQualifiedName(input.Name)
	input.AppliedACLs = c.getQualifiedAcls(input.AppliedACLs)
	// Qualify VirtualNICs
	qualifiedVNICs := c.getQualifiedList(input.VirtualNICs)
	if len(qualifiedVNICs) != 0 {
		input.VirtualNICs = qualifiedVNICs
	}

	var virtNICSet VirtualNICSet
	if err := c.updateResource(input.Name, input, &virtNICSet); err != nil {
		return nil, err
	}

	return c.success(&virtNICSet)
}

type DeleteVirtualNICSetInput struct {
	// The name of the virtual NIC set.
	// Required
	Name string `json:"name"`
}

func (c *VirtNICSetsClient) DeleteVirtualNICSet(input *DeleteVirtualNICSetInput) error {
	input.Name = c.getQualifiedName(input.Name)
	return c.deleteResource(input.Name)
}

func (c *VirtNICSetsClient) getQualifiedAcls(acls []string) []string {
	qualifiedAcls := []string{}
	for _, acl := range acls {
		qualifiedAcls = append(qualifiedAcls, c.getQualifiedName(acl))
	}
	return qualifiedAcls
}

func (c *VirtNICSetsClient) unqualifyAcls(acls []string) []string {
	unqualifiedAcls := []string{}
	for _, acl := range acls {
		unqualifiedAcls = append(unqualifiedAcls, c.getUnqualifiedName(acl))
	}
	return unqualifiedAcls
}

func (c *VirtNICSetsClient) success(info *VirtualNICSet) (*VirtualNICSet, error) {
	c.unqualify(&info.Name)
	info.AppliedACLs = c.unqualifyAcls(info.AppliedACLs)
	info.VirtualNICs = c.getUnqualifiedList(info.VirtualNICs)
	return info, nil
}
