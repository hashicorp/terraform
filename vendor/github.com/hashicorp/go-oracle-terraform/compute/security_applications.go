package compute

// SecurityApplicationsClient is a client for the Security Application functions of the Compute API.
type SecurityApplicationsClient struct {
	ResourceClient
}

// SecurityApplications obtains a SecurityApplicationsClient which can be used to access to the
// Security Application functions of the Compute API
func (c *Client) SecurityApplications() *SecurityApplicationsClient {
	return &SecurityApplicationsClient{
		ResourceClient: ResourceClient{
			Client:              c,
			ResourceDescription: "security application",
			ContainerPath:       "/secapplication/",
			ResourceRootPath:    "/secapplication",
		}}
}

// SecurityApplicationInfo describes an existing security application.
type SecurityApplicationInfo struct {
	// A description of the security application.
	Description string `json:"description"`
	// The TCP or UDP destination port number. This can be a port range, such as 5900-5999 for TCP.
	DPort string `json:"dport"`
	// The ICMP code.
	ICMPCode SecurityApplicationICMPCode `json:"icmpcode"`
	// The ICMP type.
	ICMPType SecurityApplicationICMPType `json:"icmptype"`
	// The three-part name of the Security Application (/Compute-identity_domain/user/object).
	Name string `json:"name"`
	// The protocol to use.
	Protocol SecurityApplicationProtocol `json:"protocol"`
	// The Uniform Resource Identifier
	URI string `json:"uri"`
}

type SecurityApplicationProtocol string

const (
	All    SecurityApplicationProtocol = "all"
	AH     SecurityApplicationProtocol = "ah"
	ESP    SecurityApplicationProtocol = "esp"
	ICMP   SecurityApplicationProtocol = "icmp"
	ICMPV6 SecurityApplicationProtocol = "icmpv6"
	IGMP   SecurityApplicationProtocol = "igmp"
	IPIP   SecurityApplicationProtocol = "ipip"
	GRE    SecurityApplicationProtocol = "gre"
	MPLSIP SecurityApplicationProtocol = "mplsip"
	OSPF   SecurityApplicationProtocol = "ospf"
	PIM    SecurityApplicationProtocol = "pim"
	RDP    SecurityApplicationProtocol = "rdp"
	SCTP   SecurityApplicationProtocol = "sctp"
	TCP    SecurityApplicationProtocol = "tcp"
	UDP    SecurityApplicationProtocol = "udp"
)

type SecurityApplicationICMPCode string

const (
	Admin    SecurityApplicationICMPCode = "admin"
	Df       SecurityApplicationICMPCode = "df"
	Host     SecurityApplicationICMPCode = "host"
	Network  SecurityApplicationICMPCode = "network"
	Port     SecurityApplicationICMPCode = "port"
	Protocol SecurityApplicationICMPCode = "protocol"
)

type SecurityApplicationICMPType string

const (
	Echo        SecurityApplicationICMPType = "echo"
	Reply       SecurityApplicationICMPType = "reply"
	TTL         SecurityApplicationICMPType = "ttl"
	TraceRoute  SecurityApplicationICMPType = "traceroute"
	Unreachable SecurityApplicationICMPType = "unreachable"
)

func (c *SecurityApplicationsClient) success(result *SecurityApplicationInfo) (*SecurityApplicationInfo, error) {
	c.unqualify(&result.Name)
	return result, nil
}

// CreateSecurityApplicationInput describes the Security Application to create
type CreateSecurityApplicationInput struct {
	// A description of the security application.
	// Optional
	Description string `json:"description"`
	// The TCP or UDP destination port number.
	// You can also specify a port range, such as 5900-5999 for TCP.
	// This parameter isn't relevant to the icmp protocol.
	// Required if the Protocol is TCP or UDP
	DPort string `json:"dport"`
	// The ICMP code. This parameter is relevant only if you specify ICMP as the protocol.
	// If you specify icmp as the protocol and don't specify icmptype or icmpcode, then all ICMP packets are matched.
	// Optional
	ICMPCode SecurityApplicationICMPCode `json:"icmpcode,omitempty"`
	// This parameter is relevant only if you specify ICMP as the protocol.
	// If you specify icmp as the protocol and don't specify icmptype or icmpcode, then all ICMP packets are matched.
	// Optional
	ICMPType SecurityApplicationICMPType `json:"icmptype,omitempty"`
	// The three-part name of the Security Application (/Compute-identity_domain/user/object).
	// Object names can contain only alphanumeric characters, hyphens, underscores, and periods. Object names are case-sensitive.
	// Required
	Name string `json:"name"`
	// The protocol to use.
	// Required
	Protocol SecurityApplicationProtocol `json:"protocol"`
}

// CreateSecurityApplication creates a new security application.
func (c *SecurityApplicationsClient) CreateSecurityApplication(input *CreateSecurityApplicationInput) (*SecurityApplicationInfo, error) {
	input.Name = c.getQualifiedName(input.Name)

	var appInfo SecurityApplicationInfo
	if err := c.createResource(&input, &appInfo); err != nil {
		return nil, err
	}

	return c.success(&appInfo)
}

// GetSecurityApplicationInput describes the Security Application to obtain
type GetSecurityApplicationInput struct {
	// The three-part name of the Security Application (/Compute-identity_domain/user/object).
	// Required
	Name string `json:"name"`
}

// GetSecurityApplication retrieves the security application with the given name.
func (c *SecurityApplicationsClient) GetSecurityApplication(input *GetSecurityApplicationInput) (*SecurityApplicationInfo, error) {
	var appInfo SecurityApplicationInfo
	if err := c.getResource(input.Name, &appInfo); err != nil {
		return nil, err
	}

	return c.success(&appInfo)
}

// DeleteSecurityApplicationInput  describes the Security Application to delete
type DeleteSecurityApplicationInput struct {
	// The three-part name of the Security Application (/Compute-identity_domain/user/object).
	// Required
	Name string `json:"name"`
}

// DeleteSecurityApplication deletes the security application with the given name.
func (c *SecurityApplicationsClient) DeleteSecurityApplication(input *DeleteSecurityApplicationInput) error {
	return c.deleteResource(input.Name)
}
