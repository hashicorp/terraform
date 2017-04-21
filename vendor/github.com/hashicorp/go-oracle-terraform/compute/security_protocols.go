package compute

const (
	SecurityProtocolDescription   = "security protocol"
	SecurityProtocolContainerPath = "/network/v1/secprotocol/"
	SecurityProtocolResourcePath  = "/network/v1/secprotocol"
)

type SecurityProtocolsClient struct {
	ResourceClient
}

// SecurityProtocols() returns an SecurityProtocolsClient that can be used to access the
// necessary CRUD functions for Security Protocols.
func (c *Client) SecurityProtocols() *SecurityProtocolsClient {
	return &SecurityProtocolsClient{
		ResourceClient: ResourceClient{
			Client:              c,
			ResourceDescription: SecurityProtocolDescription,
			ContainerPath:       SecurityProtocolContainerPath,
			ResourceRootPath:    SecurityProtocolResourcePath,
		},
	}
}

// SecurityProtocolInfo contains the exported fields necessary to hold all the information about an
// Security Protocol
type SecurityProtocolInfo struct {
	// List of port numbers or port range strings to match the packet's destination port.
	DstPortSet []string `json:"dstPortSet"`
	// Protocol used in the data portion of the IP datagram.
	IPProtocol string `json:"ipProtocol"`
	// List of port numbers or port range strings to match the packet's source port.
	SrcPortSet []string `json:"srcPortSet"`
	// The name of the Security Protocol
	Name string `json:"name"`
	// Description of the Security Protocol
	Description string `json:"description"`
	// Slice of tags associated with the Security Protocol
	Tags []string `json:"tags"`
	// Uniform Resource Identifier for the Security Protocol
	Uri string `json:"uri"`
}

type CreateSecurityProtocolInput struct {
	// The name of the Security Protocol to create. Object names can only contain alphanumeric,
	// underscore, dash, and period characters. Names are case-sensitive.
	// Required
	Name string `json:"name"`

	// Description of the SecurityProtocol
	// Optional
	Description string `json:"description"`

	// Enter a list of port numbers or port range strings.
	//Traffic is enabled by a security rule when a packet's destination port matches the
	// ports specified here.
	// For TCP, SCTP, and UDP, each port is a destination transport port, between 0 and 65535,
	// inclusive. For ICMP, each port is an ICMP type, between 0 and 255, inclusive.
	// If no destination ports are specified, all destination ports or ICMP types are allowed.
	// Optional
	DstPortSet []string `json:"dstPortSet"`

	// The protocol used in the data portion of the IP datagram.
	// Specify one of the permitted values or enter a number in the range 0–254 to
	// represent the protocol that you want to specify. See Assigned Internet Protocol Numbers.
	// Permitted values are: tcp, udp, icmp, igmp, ipip, rdp, esp, ah, gre, icmpv6, ospf, pim, sctp,
	// mplsip, all.
	// Traffic is enabled by a security rule when the protocol in the packet matches the
	// protocol specified here. If no protocol is specified, all protocols are allowed.
	// Optional
	IPProtocol string `json:"ipProtocol"`

	// Enter a list of port numbers or port range strings.
	// Traffic is enabled by a security rule when a packet's source port matches the
	// ports specified here.
	// For TCP, SCTP, and UDP, each port is a source transport port,
	// between 0 and 65535, inclusive.
	// For ICMP, each port is an ICMP type, between 0 and 255, inclusive.
	// If no source ports are specified, all source ports or ICMP types are allowed.
	// Optional
	SrcPortSet []string `json:"srcPortSet"`

	// String slice of tags to apply to the Security Protocol object
	// Optional
	Tags []string `json:"tags"`
}

// Create a new Security Protocol from an SecurityProtocolsClient and an input struct.
// Returns a populated Info struct for the Security Protocol, and any errors
func (c *SecurityProtocolsClient) CreateSecurityProtocol(input *CreateSecurityProtocolInput) (*SecurityProtocolInfo, error) {
	input.Name = c.getQualifiedName(input.Name)

	var ipInfo SecurityProtocolInfo
	if err := c.createResource(&input, &ipInfo); err != nil {
		return nil, err
	}

	return c.success(&ipInfo)
}

type GetSecurityProtocolInput struct {
	// The name of the Security Protocol to query for. Case-sensitive
	// Required
	Name string `json:"name"`
}

// Returns a populated SecurityProtocolInfo struct from an input struct
func (c *SecurityProtocolsClient) GetSecurityProtocol(input *GetSecurityProtocolInput) (*SecurityProtocolInfo, error) {
	input.Name = c.getQualifiedName(input.Name)

	var ipInfo SecurityProtocolInfo
	if err := c.getResource(input.Name, &ipInfo); err != nil {
		return nil, err
	}

	return c.success(&ipInfo)
}

// UpdateSecurityProtocolInput defines what to update in a security protocol
type UpdateSecurityProtocolInput struct {
	// The name of the Security Protocol to create. Object names can only contain alphanumeric,
	// underscore, dash, and period characters. Names are case-sensitive.
	// Required
	Name string `json:"name"`

	// Description of the SecurityProtocol
	// Optional
	Description string `json:"description"`

	// Enter a list of port numbers or port range strings.
	//Traffic is enabled by a security rule when a packet's destination port matches the
	// ports specified here.
	// For TCP, SCTP, and UDP, each port is a destination transport port, between 0 and 65535,
	// inclusive. For ICMP, each port is an ICMP type, between 0 and 255, inclusive.
	// If no destination ports are specified, all destination ports or ICMP types are allowed.
	DstPortSet []string `json:"dstPortSet"`

	// The protocol used in the data portion of the IP datagram.
	// Specify one of the permitted values or enter a number in the range 0–254 to
	// represent the protocol that you want to specify. See Assigned Internet Protocol Numbers.
	// Permitted values are: tcp, udp, icmp, igmp, ipip, rdp, esp, ah, gre, icmpv6, ospf, pim, sctp,
	// mplsip, all.
	// Traffic is enabled by a security rule when the protocol in the packet matches the
	// protocol specified here. If no protocol is specified, all protocols are allowed.
	IPProtocol string `json:"ipProtocol"`

	// Enter a list of port numbers or port range strings.
	// Traffic is enabled by a security rule when a packet's source port matches the
	// ports specified here.
	// For TCP, SCTP, and UDP, each port is a source transport port,
	// between 0 and 65535, inclusive.
	// For ICMP, each port is an ICMP type, between 0 and 255, inclusive.
	// If no source ports are specified, all source ports or ICMP types are allowed.
	SrcPortSet []string `json:"srcPortSet"`

	// String slice of tags to apply to the Security Protocol object
	// Optional
	Tags []string `json:"tags"`
}

// UpdateSecurityProtocol update the security protocol
func (c *SecurityProtocolsClient) UpdateSecurityProtocol(updateInput *UpdateSecurityProtocolInput) (*SecurityProtocolInfo, error) {
	updateInput.Name = c.getQualifiedName(updateInput.Name)
	var ipInfo SecurityProtocolInfo
	if err := c.updateResource(updateInput.Name, updateInput, &ipInfo); err != nil {
		return nil, err
	}

	return c.success(&ipInfo)
}

type DeleteSecurityProtocolInput struct {
	// The name of the Security Protocol to query for. Case-sensitive
	// Required
	Name string `json:"name"`
}

func (c *SecurityProtocolsClient) DeleteSecurityProtocol(input *DeleteSecurityProtocolInput) error {
	return c.deleteResource(input.Name)
}

// Unqualifies any qualified fields in the SecurityProtocolInfo struct
func (c *SecurityProtocolsClient) success(info *SecurityProtocolInfo) (*SecurityProtocolInfo, error) {
	c.unqualify(&info.Name)
	return info, nil
}
