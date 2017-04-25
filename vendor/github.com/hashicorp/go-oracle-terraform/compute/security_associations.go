package compute

// SecurityAssociationsClient is a client for the Security Association functions of the Compute API.
type SecurityAssociationsClient struct {
	ResourceClient
}

// SecurityAssociations obtains a SecurityAssociationsClient which can be used to access to the
// Security Association functions of the Compute API
func (c *Client) SecurityAssociations() *SecurityAssociationsClient {
	return &SecurityAssociationsClient{
		ResourceClient: ResourceClient{
			Client:              c,
			ResourceDescription: "security association",
			ContainerPath:       "/secassociation/",
			ResourceRootPath:    "/secassociation",
		}}
}

// SecurityAssociationInfo describes an existing security association.
type SecurityAssociationInfo struct {
	// The three-part name of the Security Association (/Compute-identity_domain/user/object).
	Name string `json:"name"`
	// The name of the Security List that you want to associate with the instance.
	SecList string `json:"seclist"`
	// vCable of the instance that you want to associate with the security list.
	VCable string `json:"vcable"`
	// Uniform Resource Identifier
	URI string `json:"uri"`
}

// CreateSecurityAssociationInput defines a security association to be created.
type CreateSecurityAssociationInput struct {
	// The three-part name of the Security Association (/Compute-identity_domain/user/object).
	// If you don't specify a name for this object, then the name is generated automatically.
	// Object names can contain only alphanumeric characters, hyphens, underscores, and periods. Object names are case-sensitive.
	// Optional
	Name string `json:"name"`
	// The name of the Security list that you want to associate with the instance.
	// Required
	SecList string `json:"seclist"`
	// The name of the vCable of the instance that you want to associate with the security list.
	// Required
	VCable string `json:"vcable"`
}

// CreateSecurityAssociation creates a security association between the given VCable and security list.
func (c *SecurityAssociationsClient) CreateSecurityAssociation(createInput *CreateSecurityAssociationInput) (*SecurityAssociationInfo, error) {
	if createInput.Name != "" {
		createInput.Name = c.getQualifiedName(createInput.Name)
	}
	createInput.VCable = c.getQualifiedName(createInput.VCable)
	createInput.SecList = c.getQualifiedName(createInput.SecList)

	var assocInfo SecurityAssociationInfo
	if err := c.createResource(&createInput, &assocInfo); err != nil {
		return nil, err
	}

	return c.success(&assocInfo)
}

// GetSecurityAssociationInput describes the security association to get
type GetSecurityAssociationInput struct {
	// The three-part name of the Security Association (/Compute-identity_domain/user/object).
	// Required
	Name string `json:"name"`
}

// GetSecurityAssociation retrieves the security association with the given name.
func (c *SecurityAssociationsClient) GetSecurityAssociation(getInput *GetSecurityAssociationInput) (*SecurityAssociationInfo, error) {
	var assocInfo SecurityAssociationInfo
	if err := c.getResource(getInput.Name, &assocInfo); err != nil {
		return nil, err
	}

	return c.success(&assocInfo)
}

// DeleteSecurityAssociationInput describes the security association to delete
type DeleteSecurityAssociationInput struct {
	// The three-part name of the Security Association (/Compute-identity_domain/user/object).
	// Required
	Name string `json:"name"`
}

// DeleteSecurityAssociation deletes the security association with the given name.
func (c *SecurityAssociationsClient) DeleteSecurityAssociation(deleteInput *DeleteSecurityAssociationInput) error {
	return c.deleteResource(deleteInput.Name)
}

func (c *SecurityAssociationsClient) success(assocInfo *SecurityAssociationInfo) (*SecurityAssociationInfo, error) {
	c.unqualify(&assocInfo.Name, &assocInfo.SecList, &assocInfo.VCable)
	return assocInfo, nil
}
