package compute

// SSHKeysClient is a client for the SSH key functions of the Compute API.
type SSHKeysClient struct {
	ResourceClient
}

// SSHKeys obtains an SSHKeysClient which can be used to access to the
// SSH key functions of the Compute API
func (c *Client) SSHKeys() *SSHKeysClient {
	return &SSHKeysClient{
		ResourceClient: ResourceClient{
			Client:              c,
			ResourceDescription: "SSH key",
			ContainerPath:       "/sshkey/",
			ResourceRootPath:    "/sshkey",
		}}
}

// SSHKeyInfo describes an existing SSH key.
type SSHKey struct {
	// Indicates whether the key is enabled (true) or disabled.
	Enabled bool `json:"enabled"`
	// The SSH public key value.
	Key string `json:"key"`
	// The three-part name of the SSH Key (/Compute-identity_domain/user/object).
	Name string `json:"name"`
	// Unique Resource Identifier
	URI string `json:"uri"`
}

// CreateSSHKeyInput defines an SSH key to be created.
type CreateSSHKeyInput struct {
	// The three-part name of the SSH Key (/Compute-identity_domain/user/object).
	// Object names can contain only alphanumeric characters, hyphens, underscores, and periods. Object names are case-sensitive.
	// Required
	Name string `json:"name"`
	// The SSH public key value.
	// Required
	Key string `json:"key"`
	// Indicates whether the key must be enabled (default) or disabled. Note that disabled keys cannot be associated with instances.
	// To explicitly enable the key, specify true. To disable the key, specify false.
	// Optional
	Enabled bool `json:"enabled"`
}

// CreateSSHKey creates a new SSH key with the given name, key and enabled flag.
func (c *SSHKeysClient) CreateSSHKey(createInput *CreateSSHKeyInput) (*SSHKey, error) {
	var keyInfo SSHKey
	// We have to update after create to get the full ssh key into opc
	updateSSHKeyInput := UpdateSSHKeyInput{
		Name:    createInput.Name,
		Key:     createInput.Key,
		Enabled: createInput.Enabled,
	}

	createInput.Name = c.getQualifiedName(createInput.Name)
	if err := c.createResource(&createInput, &keyInfo); err != nil {
		return nil, err
	}

	_, err := c.UpdateSSHKey(&updateSSHKeyInput)
	if err != nil {
		return nil, err
	}

	return c.success(&keyInfo)
}

// GetSSHKeyInput describes the ssh key to get
type GetSSHKeyInput struct {
	// The three-part name of the SSH Key (/Compute-identity_domain/user/object).
	Name string `json:name`
}

// GetSSHKey retrieves the SSH key with the given name.
func (c *SSHKeysClient) GetSSHKey(getInput *GetSSHKeyInput) (*SSHKey, error) {
	var keyInfo SSHKey
	if err := c.getResource(getInput.Name, &keyInfo); err != nil {
		return nil, err
	}

	return c.success(&keyInfo)
}

// UpdateSSHKeyInput defines an SSH key to be updated
type UpdateSSHKeyInput struct {
	// The three-part name of the object (/Compute-identity_domain/user/object).
	Name string `json:"name"`
	// The SSH public key value.
	// Required
	Key string `json:"key"`
	// Indicates whether the key must be enabled (default) or disabled. Note that disabled keys cannot be associated with instances.
	// To explicitly enable the key, specify true. To disable the key, specify false.
	// Optional
	// TODO/NOTE: isn't this required?
	Enabled bool `json:"enabled"`
}

// UpdateSSHKey updates the key and enabled flag of the SSH key with the given name.
func (c *SSHKeysClient) UpdateSSHKey(updateInput *UpdateSSHKeyInput) (*SSHKey, error) {
	var keyInfo SSHKey
	updateInput.Name = c.getQualifiedName(updateInput.Name)
	if err := c.updateResource(updateInput.Name, updateInput, &keyInfo); err != nil {
		return nil, err
	}
	return c.success(&keyInfo)
}

// DeleteKeyInput describes the ssh key to delete
type DeleteSSHKeyInput struct {
	// The three-part name of the SSH Key (/Compute-identity_domain/user/object).
	Name string `json:name`
}

// DeleteSSHKey deletes the SSH key with the given name.
func (c *SSHKeysClient) DeleteSSHKey(deleteInput *DeleteSSHKeyInput) error {
	return c.deleteResource(deleteInput.Name)
}

func (c *SSHKeysClient) success(keyInfo *SSHKey) (*SSHKey, error) {
	c.unqualify(&keyInfo.Name)
	return keyInfo, nil
}
