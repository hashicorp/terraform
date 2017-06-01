package compute

// ACLsClient is a client for the ACLs functions of the Compute API.
type ACLsClient struct {
	ResourceClient
}

const (
	ACLDescription   = "acl"
	ACLContainerPath = "/network/v1/acl/"
	ACLResourcePath  = "/network/v1/acl"
)

// ACLs obtains a ACLsClient which can be used to access to the
// ACLs functions of the Compute API
func (c *Client) ACLs() *ACLsClient {
	return &ACLsClient{
		ResourceClient: ResourceClient{
			Client:              c,
			ResourceDescription: ACLDescription,
			ContainerPath:       ACLContainerPath,
			ResourceRootPath:    ACLResourcePath,
		}}
}

// ACLInfo describes an existing ACL.
type ACLInfo struct {
	// Description of the ACL
	Description string `json:"description"`
	// Indicates whether the ACL is enabled
	Enabled bool `json:"enabledFlag"`
	// The name of the ACL
	Name string `json:"name"`
	// Tags associated with the ACL
	Tags []string `json:"tags"`
	// Uniform Resource Identifier for the ACL
	URI string `json:"uri"`
}

// CreateACLInput defines a ACL to be created.
type CreateACLInput struct {
	// Description of the ACL
	// Optional
	Description string `json:"description"`

	// Enables or disables the ACL. Set to true by default.
	//Set this to false to disable the ACL.
	// Optional
	Enabled bool `json:"enabledFlag"`

	// The name of the ACL to create. Object names can only contain alphanumeric,
	// underscore, dash, and period characters. Names are case-sensitive.
	// Required
	Name string `json:"name"`

	// Strings that you can use to tag the ACL.
	// Optional
	Tags []string `json:"tags"`
}

// CreateACL creates a new ACL.
func (c *ACLsClient) CreateACL(createInput *CreateACLInput) (*ACLInfo, error) {
	createInput.Name = c.getQualifiedName(createInput.Name)

	var aclInfo ACLInfo
	if err := c.createResource(createInput, &aclInfo); err != nil {
		return nil, err
	}

	return c.success(&aclInfo)
}

// GetACLInput describes the ACL to get
type GetACLInput struct {
	// The name of the ACL to query for
	// Required
	Name string `json:"name"`
}

// GetACL retrieves the ACL with the given name.
func (c *ACLsClient) GetACL(getInput *GetACLInput) (*ACLInfo, error) {
	var aclInfo ACLInfo
	if err := c.getResource(getInput.Name, &aclInfo); err != nil {
		return nil, err
	}

	return c.success(&aclInfo)
}

// UpdateACLInput describes a secruity rule to update
type UpdateACLInput struct {
	// Description of the ACL
	// Optional
	Description string `json:"description"`

	// Enables or disables the ACL. Set to true by default.
	//Set this to false to disable the ACL.
	// Optional
	Enabled bool `json:"enabledFlag"`

	// The name of the ACL to create. Object names can only contain alphanumeric,
	// underscore, dash, and period characters. Names are case-sensitive.
	// Required
	Name string `json:"name"`

	// Strings that you can use to tag the ACL.
	// Optional
	Tags []string `json:"tags"`
}

// UpdateACL modifies the properties of the ACL with the given name.
func (c *ACLsClient) UpdateACL(updateInput *UpdateACLInput) (*ACLInfo, error) {
	updateInput.Name = c.getQualifiedName(updateInput.Name)

	var aclInfo ACLInfo
	if err := c.updateResource(updateInput.Name, updateInput, &aclInfo); err != nil {
		return nil, err
	}

	return c.success(&aclInfo)
}

// DeleteACLInput describes the ACL to delete
type DeleteACLInput struct {
	// The name of the ACL to delete.
	// Required
	Name string `json:"name"`
}

// DeleteACL deletes the ACL with the given name.
func (c *ACLsClient) DeleteACL(deleteInput *DeleteACLInput) error {
	return c.deleteResource(deleteInput.Name)
}

func (c *ACLsClient) success(aclInfo *ACLInfo) (*ACLInfo, error) {
	aclInfo.Name = c.getUnqualifiedName(aclInfo.Name)
	return aclInfo, nil
}
