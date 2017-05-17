package compute

const (
	IPNetworkDescription   = "ip network"
	IPNetworkContainerPath = "/network/v1/ipnetwork/"
	IPNetworkResourcePath  = "/network/v1/ipnetwork"
)

type IPNetworksClient struct {
	ResourceClient
}

// IPNetworks() returns an IPNetworksClient that can be used to access the
// necessary CRUD functions for IP Networks.
func (c *Client) IPNetworks() *IPNetworksClient {
	return &IPNetworksClient{
		ResourceClient: ResourceClient{
			Client:              c,
			ResourceDescription: IPNetworkDescription,
			ContainerPath:       IPNetworkContainerPath,
			ResourceRootPath:    IPNetworkResourcePath,
		},
	}
}

// IPNetworkInfo contains the exported fields necessary to hold all the information about an
// IP Network
type IPNetworkInfo struct {
	// The name of the IP Network
	Name string `json:"name"`
	// The CIDR IPv4 prefix associated with the IP Network
	IPAddressPrefix string `json:"ipAddressPrefix"`
	// Name of the IP Network Exchange associated with the IP Network
	IPNetworkExchange string `json:"ipNetworkExchange,omitempty"`
	// Description of the IP Network
	Description string `json:"description"`
	// Whether public internet access was enabled using NAPT for VNICs without any public IP reservation
	PublicNaptEnabled bool `json:"publicNaptEnabledFlag"`
	// Slice of tags associated with the IP Network
	Tags []string `json:"tags"`
	// Uniform Resource Identifier for the IP Network
	Uri string `json:"uri"`
}

type CreateIPNetworkInput struct {
	// The name of the IP Network to create. Object names can only contain alphanumeric,
	// underscore, dash, and period characters. Names are case-sensitive.
	// Required
	Name string `json:"name"`

	// Specify the size of the IP Subnet. It is a range of IPv4 addresses assigned in the virtual
	// network, in CIDR address prefix format.
	//	While specifying the IP address prefix take care of the following points:
	//
	//* These IP addresses aren't part of the common pool of Oracle-provided IP addresses used by the shared network.
	//
	//* There's no conflict with the range of IP addresses used in another IP network, the IP addresses used your on-premises network, or with the range of private IP addresses used in the shared network. If IP networks with overlapping IP subnets are linked to an IP exchange, packets going to and from those IP networks are dropped.
	//
	//* The upper limit of the CIDR block size for an IP network is /16.
	//
	//Note: The first IP address of any IP network is reserved for the default gateway, the DHCP server, and the DNS server of that IP network.
	// Required
	IPAddressPrefix string `json:"ipAddressPrefix"`

	//Specify the IP network exchange to which the IP network belongs.
	//You can add an IP network to only one IP network exchange, but an IP network exchange
	//can include multiple IP networks. An IP network exchange enables access between IP networks
	//that have non-overlapping addresses, so that instances on these networks can exchange packets
	//with each other without NAT.
	// Optional
	IPNetworkExchange string `json:"ipNetworkExchange,omitempty"`

	// Description of the IPNetwork
	// Optional
	Description string `json:"description"`

	// Enable public internet access using NAPT for VNICs without any public IP reservation
	// Optional
	PublicNaptEnabled bool `json:"publicNaptEnabledFlag"`

	// String slice of tags to apply to the IP Network object
	// Optional
	Tags []string `json:"tags"`
}

// Create a new IP Network from an IPNetworksClient and an input struct.
// Returns a populated Info struct for the IP Network, and any errors
func (c *IPNetworksClient) CreateIPNetwork(input *CreateIPNetworkInput) (*IPNetworkInfo, error) {
	input.Name = c.getQualifiedName(input.Name)
	input.IPNetworkExchange = c.getQualifiedName(input.IPNetworkExchange)

	var ipInfo IPNetworkInfo
	if err := c.createResource(&input, &ipInfo); err != nil {
		return nil, err
	}

	return c.success(&ipInfo)
}

type GetIPNetworkInput struct {
	// The name of the IP Network to query for. Case-sensitive
	// Required
	Name string `json:"name"`
}

// Returns a populated IPNetworkInfo struct from an input struct
func (c *IPNetworksClient) GetIPNetwork(input *GetIPNetworkInput) (*IPNetworkInfo, error) {
	input.Name = c.getQualifiedName(input.Name)

	var ipInfo IPNetworkInfo
	if err := c.getResource(input.Name, &ipInfo); err != nil {
		return nil, err
	}

	return c.success(&ipInfo)
}

type UpdateIPNetworkInput struct {
	// The name of the IP Network to update. Object names can only contain alphanumeric,
	// underscore, dash, and period characters. Names are case-sensitive.
	// Required
	Name string `json:"name"`

	// Specify the size of the IP Subnet. It is a range of IPv4 addresses assigned in the virtual
	// network, in CIDR address prefix format.
	//	While specifying the IP address prefix take care of the following points:
	//
	//* These IP addresses aren't part of the common pool of Oracle-provided IP addresses used by the shared network.
	//
	//* There's no conflict with the range of IP addresses used in another IP network, the IP addresses used your on-premises network, or with the range of private IP addresses used in the shared network. If IP networks with overlapping IP subnets are linked to an IP exchange, packets going to and from those IP networks are dropped.
	//
	//* The upper limit of the CIDR block size for an IP network is /16.
	//
	//Note: The first IP address of any IP network is reserved for the default gateway, the DHCP server, and the DNS server of that IP network.
	// Required
	IPAddressPrefix string `json:"ipAddressPrefix"`

	//Specify the IP network exchange to which the IP network belongs.
	//You can add an IP network to only one IP network exchange, but an IP network exchange
	//can include multiple IP networks. An IP network exchange enables access between IP networks
	//that have non-overlapping addresses, so that instances on these networks can exchange packets
	//with each other without NAT.
	// Optional
	IPNetworkExchange string `json:"ipNetworkExchange,omitempty"`

	// Description of the IPNetwork
	// Optional
	Description string `json:"description"`

	// Enable public internet access using NAPT for VNICs without any public IP reservation
	// Optional
	PublicNaptEnabled bool `json:"publicNaptEnabledFlag"`

	// String slice of tags to apply to the IP Network object
	// Optional
	Tags []string `json:"tags"`
}

func (c *IPNetworksClient) UpdateIPNetwork(input *UpdateIPNetworkInput) (*IPNetworkInfo, error) {
	input.Name = c.getQualifiedName(input.Name)
	input.IPNetworkExchange = c.getQualifiedName(input.IPNetworkExchange)

	var ipInfo IPNetworkInfo
	if err := c.updateResource(input.Name, &input, &ipInfo); err != nil {
		return nil, err
	}

	return c.success(&ipInfo)
}

type DeleteIPNetworkInput struct {
	// The name of the IP Network to query for. Case-sensitive
	// Required
	Name string `json:"name"`
}

func (c *IPNetworksClient) DeleteIPNetwork(input *DeleteIPNetworkInput) error {
	return c.deleteResource(input.Name)
}

// Unqualifies any qualified fields in the IPNetworkInfo struct
func (c *IPNetworksClient) success(info *IPNetworkInfo) (*IPNetworkInfo, error) {
	c.unqualify(&info.Name)
	c.unqualify(&info.IPNetworkExchange)
	return info, nil
}
