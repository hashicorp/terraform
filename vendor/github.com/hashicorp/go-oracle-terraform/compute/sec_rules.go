package compute

// SecRulesClient is a client for the Sec Rules functions of the Compute API.
type SecRulesClient struct {
	ResourceClient
}

// SecRules obtains a SecRulesClient which can be used to access to the
// Sec Rules functions of the Compute API
func (c *Client) SecRules() *SecRulesClient {
	return &SecRulesClient{
		ResourceClient: ResourceClient{
			Client:              c,
			ResourceDescription: "security ip list",
			ContainerPath:       "/secrule/",
			ResourceRootPath:    "/secrule",
		}}
}

// SecRuleInfo describes an existing sec rule.
type SecRuleInfo struct {
	// Set this parameter to PERMIT.
	Action string `json:"action"`
	// The name of the security application
	Application string `json:"application"`
	// A description of the sec rule
	Description string `json:"description"`
	// Indicates whether the security rule is enabled
	Disabled bool `json:"disabled"`
	// The name of the destination security list or security IP list.
	DestinationList string `json:"dst_list"`
	// The name of the sec rule
	Name string `json:"name"`
	// The name of the source security list or security IP list.
	SourceList string `json:"src_list"`
	// Uniform Resource Identifier for the sec rule
	URI string `json:"uri"`
}

// CreateSecRuleInput defines a sec rule to be created.
type CreateSecRuleInput struct {
	// Set this parameter to PERMIT.
	// Required
	Action string `json:"action"`

	// The name of the security application for user-defined or predefined security applications.
	// Required
	Application string `json:"application"`

	// Description of the IP Network
	// Optional
	Description string `json:"description"`

	// Indicates whether the sec rule is enabled (set to false) or disabled (true).
	// The default setting is false.
	// Optional
	Disabled bool `json:"disabled"`

	// The name of the destination security list or security IP list.
	//
	// You must use the prefix seclist: or seciplist: to identify the list type.
	//
	// You can specify a security IP list as the destination in a secrule,
	// provided src_list is a security list that has DENY as its outbound policy.
	//
	// You cannot specify any of the security IP lists in the /oracle/public container
	// as a destination in a secrule.
	// Required
	DestinationList string `json:"dst_list"`

	// The name of the Sec Rule to create. Object names can only contain alphanumeric,
	// underscore, dash, and period characters. Names are case-sensitive.
	// Required
	Name string `json:"name"`

	// The name of the source security list or security IP list.
	//
	// You must use the prefix seclist: or seciplist: to identify the list type.
	//
	// Required
	SourceList string `json:"src_list"`
}

// CreateSecRule creates a new sec rule.
func (c *SecRulesClient) CreateSecRule(createInput *CreateSecRuleInput) (*SecRuleInfo, error) {
	createInput.Name = c.getQualifiedName(createInput.Name)
	createInput.SourceList = c.getQualifiedListName(createInput.SourceList)
	createInput.DestinationList = c.getQualifiedListName(createInput.DestinationList)
	createInput.Application = c.getQualifiedName(createInput.Application)

	var ruleInfo SecRuleInfo
	if err := c.createResource(createInput, &ruleInfo); err != nil {
		return nil, err
	}

	return c.success(&ruleInfo)
}

// GetSecRuleInput describes the Sec Rule to get
type GetSecRuleInput struct {
	// The name of the Sec Rule to query for
	// Required
	Name string `json:"name"`
}

// GetSecRule retrieves the sec rule with the given name.
func (c *SecRulesClient) GetSecRule(getInput *GetSecRuleInput) (*SecRuleInfo, error) {
	var ruleInfo SecRuleInfo
	if err := c.getResource(getInput.Name, &ruleInfo); err != nil {
		return nil, err
	}

	return c.success(&ruleInfo)
}

// UpdateSecRuleInput describes a secruity rule to update
type UpdateSecRuleInput struct {
	// Set this parameter to PERMIT.
	// Required
	Action string `json:"action"`

	// The name of the security application for user-defined or predefined security applications.
	// Required
	Application string `json:"application"`

	// Description of the IP Network
	// Optional
	Description string `json:"description"`

	// Indicates whether the sec rule is enabled (set to false) or disabled (true).
	// The default setting is false.
	// Optional
	Disabled bool `json:"disabled"`

	// The name of the destination security list or security IP list.
	//
	// You must use the prefix seclist: or seciplist: to identify the list type.
	//
	// You can specify a security IP list as the destination in a secrule,
	// provided src_list is a security list that has DENY as its outbound policy.
	//
	// You cannot specify any of the security IP lists in the /oracle/public container
	// as a destination in a secrule.
	// Required
	DestinationList string `json:"dst_list"`

	// The name of the Sec Rule to create. Object names can only contain alphanumeric,
	// underscore, dash, and period characters. Names are case-sensitive.
	// Required
	Name string `json:"name"`

	// The name of the source security list or security IP list.
	//
	// You must use the prefix seclist: or seciplist: to identify the list type.
	//
	// Required
	SourceList string `json:"src_list"`
}

// UpdateSecRule modifies the properties of the sec rule with the given name.
func (c *SecRulesClient) UpdateSecRule(updateInput *UpdateSecRuleInput) (*SecRuleInfo, error) {
	updateInput.Name = c.getQualifiedName(updateInput.Name)
	updateInput.SourceList = c.getQualifiedListName(updateInput.SourceList)
	updateInput.DestinationList = c.getQualifiedListName(updateInput.DestinationList)
	updateInput.Application = c.getQualifiedName(updateInput.Application)

	var ruleInfo SecRuleInfo
	if err := c.updateResource(updateInput.Name, updateInput, &ruleInfo); err != nil {
		return nil, err
	}

	return c.success(&ruleInfo)
}

// DeleteSecRuleInput describes the sec rule to delete
type DeleteSecRuleInput struct {
	// The name of the Sec Rule to delete.
	// Required
	Name string `json:"name"`
}

// DeleteSecRule deletes the sec rule with the given name.
func (c *SecRulesClient) DeleteSecRule(deleteInput *DeleteSecRuleInput) error {
	return c.deleteResource(deleteInput.Name)
}

func (c *SecRulesClient) success(ruleInfo *SecRuleInfo) (*SecRuleInfo, error) {
	ruleInfo.Name = c.getUnqualifiedName(ruleInfo.Name)
	ruleInfo.SourceList = c.unqualifyListName(ruleInfo.SourceList)
	ruleInfo.DestinationList = c.unqualifyListName(ruleInfo.DestinationList)
	ruleInfo.Application = c.getUnqualifiedName(ruleInfo.Application)
	return ruleInfo, nil
}
