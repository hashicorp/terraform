package compute

const (
	IPAddressPrefixSetDescription   = "ip address prefix set"
	IPAddressPrefixSetContainerPath = "/network/v1/ipaddressprefixset/"
	IPAddressPrefixSetResourcePath  = "/network/v1/ipaddressprefixset"
)

type IPAddressPrefixSetsClient struct {
	ResourceClient
}

// IPAddressPrefixSets() returns an IPAddressPrefixSetsClient that can be used to access the
// necessary CRUD functions for IP Address Prefix Sets.
func (c *Client) IPAddressPrefixSets() *IPAddressPrefixSetsClient {
	return &IPAddressPrefixSetsClient{
		ResourceClient: ResourceClient{
			Client:              c,
			ResourceDescription: IPAddressPrefixSetDescription,
			ContainerPath:       IPAddressPrefixSetContainerPath,
			ResourceRootPath:    IPAddressPrefixSetResourcePath,
		},
	}
}

// IPAddressPrefixSetInfo contains the exported fields necessary to hold all the information about an
// IP Address Prefix Set
type IPAddressPrefixSetInfo struct {
	// The name of the IP Address Prefix Set
	Name string `json:"name"`
	// Description of the IP Address Prefix Set
	Description string `json:"description"`
	// List of CIDR IPv4 prefixes assigned in the virtual network.
	IPAddressPrefixes []string `json:"ipAddressPrefixes"`
	// Slice of tags associated with the IP Address Prefix Set
	Tags []string `json:"tags"`
	// Uniform Resource Identifier for the IP Address Prefix Set
	Uri string `json:"uri"`
}

type CreateIPAddressPrefixSetInput struct {
	// The name of the IP Address Prefix Set to create. Object names can only contain alphanumeric,
	// underscore, dash, and period characters. Names are case-sensitive.
	// Required
	Name string `json:"name"`

	// Description of the IPAddressPrefixSet
	// Optional
	Description string `json:"description"`

	// List of CIDR IPv4 prefixes assigned in the virtual network.
	// Optional
	IPAddressPrefixes []string `json:"ipAddressPrefixes"`

	// String slice of tags to apply to the IP Address Prefix Set object
	// Optional
	Tags []string `json:"tags"`
}

// Create a new IP Address Prefix Set from an IPAddressPrefixSetsClient and an input struct.
// Returns a populated Info struct for the IP Address Prefix Set, and any errors
func (c *IPAddressPrefixSetsClient) CreateIPAddressPrefixSet(input *CreateIPAddressPrefixSetInput) (*IPAddressPrefixSetInfo, error) {
	input.Name = c.getQualifiedName(input.Name)

	var ipInfo IPAddressPrefixSetInfo
	if err := c.createResource(&input, &ipInfo); err != nil {
		return nil, err
	}

	return c.success(&ipInfo)
}

type GetIPAddressPrefixSetInput struct {
	// The name of the IP Address Prefix Set to query for. Case-sensitive
	// Required
	Name string `json:"name"`
}

// Returns a populated IPAddressPrefixSetInfo struct from an input struct
func (c *IPAddressPrefixSetsClient) GetIPAddressPrefixSet(input *GetIPAddressPrefixSetInput) (*IPAddressPrefixSetInfo, error) {
	input.Name = c.getQualifiedName(input.Name)

	var ipInfo IPAddressPrefixSetInfo
	if err := c.getResource(input.Name, &ipInfo); err != nil {
		return nil, err
	}

	return c.success(&ipInfo)
}

// UpdateIPAddressPrefixSetInput defines what to update in a ip address prefix set
type UpdateIPAddressPrefixSetInput struct {
	// The name of the IP Address Prefix Set to create. Object names can only contain alphanumeric,
	// underscore, dash, and period characters. Names are case-sensitive.
	// Required
	Name string `json:"name"`

	// Description of the IPAddressPrefixSet
	// Optional
	Description string `json:"description"`

	// List of CIDR IPv4 prefixes assigned in the virtual network.
	IPAddressPrefixes []string `json:"ipAddressPrefixes"`

	// String slice of tags to apply to the IP Address Prefix Set object
	// Optional
	Tags []string `json:"tags"`
}

// UpdateIPAddressPrefixSet update the ip address prefix set
func (c *IPAddressPrefixSetsClient) UpdateIPAddressPrefixSet(updateInput *UpdateIPAddressPrefixSetInput) (*IPAddressPrefixSetInfo, error) {
	updateInput.Name = c.getQualifiedName(updateInput.Name)
	var ipInfo IPAddressPrefixSetInfo
	if err := c.updateResource(updateInput.Name, updateInput, &ipInfo); err != nil {
		return nil, err
	}

	return c.success(&ipInfo)
}

type DeleteIPAddressPrefixSetInput struct {
	// The name of the IP Address Prefix Set to query for. Case-sensitive
	// Required
	Name string `json:"name"`
}

func (c *IPAddressPrefixSetsClient) DeleteIPAddressPrefixSet(input *DeleteIPAddressPrefixSetInput) error {
	return c.deleteResource(input.Name)
}

// Unqualifies any qualified fields in the IPAddressPrefixSetInfo struct
func (c *IPAddressPrefixSetsClient) success(info *IPAddressPrefixSetInfo) (*IPAddressPrefixSetInfo, error) {
	c.unqualify(&info.Name)
	return info, nil
}
