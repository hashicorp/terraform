package compute

// SecurityIPListsClient is a client for the Security IP List functions of the Compute API.
type SecurityIPListsClient struct {
	ResourceClient
}

// SecurityIPLists obtains a SecurityIPListsClient which can be used to access to the
// Security IP List functions of the Compute API
func (c *Client) SecurityIPLists() *SecurityIPListsClient {
	return &SecurityIPListsClient{
		ResourceClient: ResourceClient{
			Client:              c,
			ResourceDescription: "security ip list",
			ContainerPath:       "/seciplist/",
			ResourceRootPath:    "/seciplist",
		}}
}

// SecurityIPListInfo describes an existing security IP list.
type SecurityIPListInfo struct {
	// A description of the security IP list.
	Description string `json:"description"`
	// The three-part name of the object (/Compute-identity_domain/user/object).
	Name string `json:"name"`
	// A comma-separated list of the subnets (in CIDR format) or IPv4 addresses for which you want to create this security IP list.
	SecIPEntries []string `json:"secipentries"`
	// Uniform Resource Identifier
	URI string `json:"uri"`
}

// CreateSecurityIPListInput defines a security IP list to be created.
type CreateSecurityIPListInput struct {
	// A description of the security IP list.
	// Optional
	Description string `json:"description"`
	// The three-part name of the object (/Compute-identity_domain/user/object).
	// Object names can contain only alphanumeric characters, hyphens, underscores, and periods. Object names are case-sensitive.
	// Required
	Name string `json:"name"`
	// A comma-separated list of the subnets (in CIDR format) or IPv4 addresses for which you want to create this security IP list.
	// Required
	SecIPEntries []string `json:"secipentries"`
}

// CreateSecurityIPList creates a security IP list with the given name and entries.
func (c *SecurityIPListsClient) CreateSecurityIPList(createInput *CreateSecurityIPListInput) (*SecurityIPListInfo, error) {
	createInput.Name = c.getQualifiedName(createInput.Name)
	var listInfo SecurityIPListInfo
	if err := c.createResource(createInput, &listInfo); err != nil {
		return nil, err
	}

	return c.success(&listInfo)
}

// GetSecurityIPListInput describes the Security IP List to obtain
type GetSecurityIPListInput struct {
	// The three-part name of the object (/Compute-identity_domain/user/object).
	// Required
	Name string `json:"name"`
}

// GetSecurityIPList gets the security IP list with the given name.
func (c *SecurityIPListsClient) GetSecurityIPList(getInput *GetSecurityIPListInput) (*SecurityIPListInfo, error) {
	var listInfo SecurityIPListInfo
	if err := c.getResource(getInput.Name, &listInfo); err != nil {
		return nil, err
	}

	return c.success(&listInfo)
}

// UpdateSecurityIPListInput describes the security ip list to update
type UpdateSecurityIPListInput struct {
	// A description of the security IP list.
	// Optional
	Description string `json:"description"`
	// The three-part name of the object (/Compute-identity_domain/user/object).
	// Required
	Name string `json:"name"`
	// A comma-separated list of the subnets (in CIDR format) or IPv4 addresses for which you want to create this security IP list.
	// Required
	SecIPEntries []string `json:"secipentries"`
}

// UpdateSecurityIPList modifies the entries in the security IP list with the given name.
func (c *SecurityIPListsClient) UpdateSecurityIPList(updateInput *UpdateSecurityIPListInput) (*SecurityIPListInfo, error) {
	updateInput.Name = c.getQualifiedName(updateInput.Name)
	var listInfo SecurityIPListInfo
	if err := c.updateResource(updateInput.Name, updateInput, &listInfo); err != nil {
		return nil, err
	}

	return c.success(&listInfo)
}

// DeleteSecurityIPListInput describes the security ip list to delete.
type DeleteSecurityIPListInput struct {
	// The three-part name of the object (/Compute-identity_domain/user/object).
	// Required
	Name string `json:"name"`
}

// DeleteSecurityIPList deletes the security IP list with the given name.
func (c *SecurityIPListsClient) DeleteSecurityIPList(deleteInput *DeleteSecurityIPListInput) error {
	return c.deleteResource(deleteInput.Name)
}

func (c *SecurityIPListsClient) success(listInfo *SecurityIPListInfo) (*SecurityIPListInfo, error) {
	c.unqualify(&listInfo.Name)
	return listInfo, nil
}
