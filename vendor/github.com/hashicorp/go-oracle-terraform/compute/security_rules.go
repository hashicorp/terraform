package compute

const (
	SecurityRuleDescription   = "security rules"
	SecurityRuleContainerPath = "/network/v1/secrule/"
	SecurityRuleResourcePath  = "/network/v1/secrule"
)

type SecurityRuleClient struct {
	ResourceClient
}

// SecurityRules() returns an SecurityRulesClient that can be used to access the
// necessary CRUD functions for Security Rules.
func (c *Client) SecurityRules() *SecurityRuleClient {
	return &SecurityRuleClient{
		ResourceClient: ResourceClient{
			Client:              c,
			ResourceDescription: SecurityRuleDescription,
			ContainerPath:       SecurityRuleContainerPath,
			ResourceRootPath:    SecurityRuleResourcePath,
		},
	}
}

// SecurityRuleInfo contains the exported fields necessary to hold all the information about a
// Security Rule
type SecurityRuleInfo struct {
	// Name of the ACL that contains this rule.
	ACL string `json:"acl"`
	// Description of the Security Rule
	Description string `json:"description"`
	// List of IP address prefix set names to match the packet's destination IP address.
	DstIpAddressPrefixSets []string `json:"dstIpAddressPrefixSets"`
	// Name of virtual NIC set containing the packet's destination virtual NIC.
	DstVnicSet string `json:"dstVnicSet"`
	// Allows the security rule to be disabled.
	Enabled bool `json:"enabledFlag"`
	// Direction of the flow; Can be "egress" or "ingress".
	FlowDirection string `json:"FlowDirection"`
	// The name of the Security Rule
	Name string `json:"name"`
	// List of security protocol names to match the packet's protocol and port.
	SecProtocols []string `json:"secProtocols"`
	// List of multipart names of IP address prefix set to match the packet's source IP address.
	SrcIpAddressPrefixSets []string `json:"srcIpAddressPrefixSets"`
	// Name of virtual NIC set containing the packet's source virtual NIC.
	SrcVnicSet string `json:"srcVnicSet"`
	// Slice of tags associated with the Security Rule
	Tags []string `json:"tags"`
	// Uniform Resource Identifier for the Security Rule
	Uri string `json:"uri"`
}

type CreateSecurityRuleInput struct {
	//Select the name of the access control list (ACL) that you want to add this
	// security rule to. Security rules are applied to vNIC sets by using ACLs.
	// Optional
	ACL string `json:"acl,omitempty"`

	// Description of the Security Rule
	// Optional
	Description string `json:"description"`

	// A list of IP address prefix sets to which you want to permit traffic.
	// Only packets to IP addresses in the specified IP address prefix sets are permitted.
	// When no destination IP address prefix sets are specified, traffic to any
	// IP address is permitted.
	// Optional
	DstIpAddressPrefixSets []string `json:"dstIpAddressPrefixSets"`

	// The vNICset to which you want to permit traffic. Only packets to vNICs in the
	// specified vNICset are permitted. When no destination vNICset is specified, traffic
	// to any vNIC is permitted.
	// Optional
	DstVnicSet string `json:"dstVnicSet,omitempty"`

	// Allows the security rule to be enabled or disabled. This parameter is set to
	// true by default. Specify false to disable the security rule.
	// Optional
	Enabled bool `json:"enabledFlag"`

	// Specify the direction of flow of traffic, which is relative to the instances,
	// for this security rule. Allowed values are ingress or egress.
	// An ingress packet is a packet received by a virtual NIC, for example from
	// another virtual NIC or from the public Internet.
	// An egress packet is a packet sent by a virtual NIC, for example to another
	// virtual NIC or to the public Internet.
	// Required
	FlowDirection string `json:"flowDirection"`

	// The name of the Security Rule
	// Object names can contain only alphanumeric characters, hyphens, underscores, and periods.
	// Object names are case-sensitive. When you specify the object name, ensure that an object
	// of the same type and with the same name doesn't already exist.
	// If such an object already exists, another object of the same type and with the same name won't
	// be created and the existing object won't be updated.
	// Required
	Name string `json:"name"`

	// A list of security protocols for which you want to permit traffic. Only packets that
	// match the specified protocols and ports are permitted. When no security protocols are
	// specified, traffic using any protocol over any port is permitted.
	// Optional
	SecProtocols []string `json:"secProtocols"`

	// A list of IP address prefix sets from which you want to permit traffic. Only packets
	// from IP addresses in the specified IP address prefix sets are permitted. When no source
	// IP address prefix sets are specified, traffic from any IP address is permitted.
	// Optional
	SrcIpAddressPrefixSets []string `json:"srcIpAddressPrefixSets"`

	// The vNICset from which you want to permit traffic. Only packets from vNICs in the
	// specified vNICset are permitted. When no source vNICset is specified, traffic from any
	// vNIC is permitted.
	// Optional
	SrcVnicSet string `json:"srcVnicSet,omitempty"`

	// Strings that you can use to tag the security rule.
	// Optional
	Tags []string `json:"tags"`
}

// Create a new Security Rule from an SecurityRuleClient and an input struct.
// Returns a populated Info struct for the Security Rule, and any errors
func (c *SecurityRuleClient) CreateSecurityRule(input *CreateSecurityRuleInput) (*SecurityRuleInfo, error) {
	input.Name = c.getQualifiedName(input.Name)
	input.ACL = c.getQualifiedName(input.ACL)
	input.SrcVnicSet = c.getQualifiedName(input.SrcVnicSet)
	input.DstVnicSet = c.getQualifiedName(input.DstVnicSet)
	input.SrcIpAddressPrefixSets = c.getQualifiedList(input.SrcIpAddressPrefixSets)
	input.DstIpAddressPrefixSets = c.getQualifiedList(input.DstIpAddressPrefixSets)
	input.SecProtocols = c.getQualifiedList(input.SecProtocols)

	var securityRuleInfo SecurityRuleInfo
	if err := c.createResource(&input, &securityRuleInfo); err != nil {
		return nil, err
	}

	return c.success(&securityRuleInfo)
}

type GetSecurityRuleInput struct {
	// The name of the Security Rule to query for. Case-sensitive
	// Required
	Name string `json:"name"`
}

// Returns a populated SecurityRuleInfo struct from an input struct
func (c *SecurityRuleClient) GetSecurityRule(input *GetSecurityRuleInput) (*SecurityRuleInfo, error) {
	input.Name = c.getQualifiedName(input.Name)

	var securityRuleInfo SecurityRuleInfo
	if err := c.getResource(input.Name, &securityRuleInfo); err != nil {
		return nil, err
	}

	return c.success(&securityRuleInfo)
}

// UpdateSecurityRuleInput describes a secruity rule to update
type UpdateSecurityRuleInput struct {
	//Select the name of the access control list (ACL) that you want to add this
	// security rule to. Security rules are applied to vNIC sets by using ACLs.
	// Optional
	ACL string `json:"acl,omitempty"`

	// Description of the Security Rule
	// Optional
	Description string `json:"description"`

	// A list of IP address prefix sets to which you want to permit traffic.
	// Only packets to IP addresses in the specified IP address prefix sets are permitted.
	// When no destination IP address prefix sets are specified, traffic to any
	// IP address is permitted.
	// Optional
	DstIpAddressPrefixSets []string `json:"dstIpAddressPrefixSets"`

	// The vNICset to which you want to permit traffic. Only packets to vNICs in the
	// specified vNICset are permitted. When no destination vNICset is specified, traffic
	// to any vNIC is permitted.
	// Optional
	DstVnicSet string `json:"dstVnicSet,omitempty"`

	// Allows the security rule to be enabled or disabled. This parameter is set to
	// true by default. Specify false to disable the security rule.
	// Optional
	Enabled bool `json:"enabledFlag"`

	// Specify the direction of flow of traffic, which is relative to the instances,
	// for this security rule. Allowed values are ingress or egress.
	// An ingress packet is a packet received by a virtual NIC, for example from
	// another virtual NIC or from the public Internet.
	// An egress packet is a packet sent by a virtual NIC, for example to another
	// virtual NIC or to the public Internet.
	// Required
	FlowDirection string `json:"flowDirection"`

	// The name of the Security Rule
	// Object names can contain only alphanumeric characters, hyphens, underscores, and periods.
	// Object names are case-sensitive. When you specify the object name, ensure that an object
	// of the same type and with the same name doesn't already exist.
	// If such an object already exists, another object of the same type and with the same name won't
	// be created and the existing object won't be updated.
	// Required
	Name string `json:"name"`

	// A list of security protocols for which you want to permit traffic. Only packets that
	// match the specified protocols and ports are permitted. When no security protocols are
	// specified, traffic using any protocol over any port is permitted.
	// Optional
	SecProtocols []string `json:"secProtocols"`

	// A list of IP address prefix sets from which you want to permit traffic. Only packets
	// from IP addresses in the specified IP address prefix sets are permitted. When no source
	// IP address prefix sets are specified, traffic from any IP address is permitted.
	// Optional
	SrcIpAddressPrefixSets []string `json:"srcIpAddressPrefixSets"`

	// The vNICset from which you want to permit traffic. Only packets from vNICs in the
	// specified vNICset are permitted. When no source vNICset is specified, traffic from any
	// vNIC is permitted.
	// Optional
	SrcVnicSet string `json:"srcVnicSet,omitempty"`

	// Strings that you can use to tag the security rule.
	// Optional
	Tags []string `json:"tags"`
}

// UpdateSecRule modifies the properties of the sec rule with the given name.
func (c *SecurityRuleClient) UpdateSecurityRule(updateInput *UpdateSecurityRuleInput) (*SecurityRuleInfo, error) {
	updateInput.Name = c.getQualifiedName(updateInput.Name)
	updateInput.ACL = c.getQualifiedName(updateInput.ACL)
	updateInput.SrcVnicSet = c.getQualifiedName(updateInput.SrcVnicSet)
	updateInput.DstVnicSet = c.getQualifiedName(updateInput.DstVnicSet)
	updateInput.SrcIpAddressPrefixSets = c.getQualifiedList(updateInput.SrcIpAddressPrefixSets)
	updateInput.DstIpAddressPrefixSets = c.getQualifiedList(updateInput.DstIpAddressPrefixSets)
	updateInput.SecProtocols = c.getQualifiedList(updateInput.SecProtocols)

	var securityRuleInfo SecurityRuleInfo
	if err := c.updateResource(updateInput.Name, updateInput, &securityRuleInfo); err != nil {
		return nil, err
	}

	return c.success(&securityRuleInfo)
}

type DeleteSecurityRuleInput struct {
	// The name of the Security Rule to query for. Case-sensitive
	// Required
	Name string `json:"name"`
}

func (c *SecurityRuleClient) DeleteSecurityRule(input *DeleteSecurityRuleInput) error {
	return c.deleteResource(input.Name)
}

// Unqualifies any qualified fields in the IPNetworkExchangeInfo struct
func (c *SecurityRuleClient) success(info *SecurityRuleInfo) (*SecurityRuleInfo, error) {
	c.unqualify(&info.Name, &info.ACL, &info.SrcVnicSet, &info.DstVnicSet)
	info.SrcIpAddressPrefixSets = c.getUnqualifiedList(info.SrcIpAddressPrefixSets)
	info.DstIpAddressPrefixSets = c.getUnqualifiedList(info.DstIpAddressPrefixSets)
	info.SecProtocols = c.getUnqualifiedList(info.SecProtocols)
	return info, nil
}
