package compute

const (
	IPNetworkExchangeDescription   = "ip network exchange"
	IPNetworkExchangeContainerPath = "/network/v1/ipnetworkexchange/"
	IPNetworkExchangeResourcePath  = "/network/v1/ipnetworkexchange"
)

type IPNetworkExchangesClient struct {
	ResourceClient
}

// IPNetworkExchanges() returns an IPNetworkExchangesClient that can be used to access the
// necessary CRUD functions for IP Network Exchanges.
func (c *Client) IPNetworkExchanges() *IPNetworkExchangesClient {
	return &IPNetworkExchangesClient{
		ResourceClient: ResourceClient{
			Client:              c,
			ResourceDescription: IPNetworkExchangeDescription,
			ContainerPath:       IPNetworkExchangeContainerPath,
			ResourceRootPath:    IPNetworkExchangeResourcePath,
		},
	}
}

// IPNetworkExchangeInfo contains the exported fields necessary to hold all the information about an
// IP Network Exchange
type IPNetworkExchangeInfo struct {
	// The name of the IP Network Exchange
	Name string `json:"name"`
	// Description of the IP Network Exchange
	Description string `json:"description"`
	// Slice of tags associated with the IP Network Exchange
	Tags []string `json:"tags"`
	// Uniform Resource Identifier for the IP Network Exchange
	Uri string `json:"uri"`
}

type CreateIPNetworkExchangeInput struct {
	// The name of the IP Network Exchange to create. Object names can only contain alphanumeric,
	// underscore, dash, and period characters. Names are case-sensitive.
	// Required
	Name string `json:"name"`

	// Description of the IPNetworkExchange
	// Optional
	Description string `json:"description"`

	// String slice of tags to apply to the IP Network Exchange object
	// Optional
	Tags []string `json:"tags"`
}

// Create a new IP Network Exchange from an IPNetworkExchangesClient and an input struct.
// Returns a populated Info struct for the IP Network Exchange, and any errors
func (c *IPNetworkExchangesClient) CreateIPNetworkExchange(input *CreateIPNetworkExchangeInput) (*IPNetworkExchangeInfo, error) {
	input.Name = c.getQualifiedName(input.Name)

	var ipInfo IPNetworkExchangeInfo
	if err := c.createResource(&input, &ipInfo); err != nil {
		return nil, err
	}

	return c.success(&ipInfo)
}

type GetIPNetworkExchangeInput struct {
	// The name of the IP Network Exchange to query for. Case-sensitive
	// Required
	Name string `json:"name"`
}

// Returns a populated IPNetworkExchangeInfo struct from an input struct
func (c *IPNetworkExchangesClient) GetIPNetworkExchange(input *GetIPNetworkExchangeInput) (*IPNetworkExchangeInfo, error) {
	input.Name = c.getQualifiedName(input.Name)

	var ipInfo IPNetworkExchangeInfo
	if err := c.getResource(input.Name, &ipInfo); err != nil {
		return nil, err
	}

	return c.success(&ipInfo)
}

type DeleteIPNetworkExchangeInput struct {
	// The name of the IP Network Exchange to query for. Case-sensitive
	// Required
	Name string `json:"name"`
}

func (c *IPNetworkExchangesClient) DeleteIPNetworkExchange(input *DeleteIPNetworkExchangeInput) error {
	return c.deleteResource(input.Name)
}

// Unqualifies any qualified fields in the IPNetworkExchangeInfo struct
func (c *IPNetworkExchangesClient) success(info *IPNetworkExchangeInfo) (*IPNetworkExchangeInfo, error) {
	c.unqualify(&info.Name)
	return info, nil
}
