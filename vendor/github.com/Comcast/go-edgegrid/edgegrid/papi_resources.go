package edgegrid

// Groups is a representation of the Akamai PAPI
// Groups response available at:
// http://apibase.com/papi/v0/groups/
type Groups struct {
	Groups struct {
		Items []GroupSummary `json:"items"`
	} `json:"groups"`
}

// GroupSummary is a representation of the Akamai PAPI
// group summary associated with each group returned by
// the groups response at:
// http://apibase.com/papi/v0/groups/
type GroupSummary struct {
	GroupID       string   `json:"groupId"`
	Name          string   `json:"groupName"`
	ContractIDs   []string `json:"contractIds"`
	ParentGroupID string   `json:"parentGroupId"`
}

// Products is a representation of the Akamai PAPI
// products response available at:
// http://apibase.com/papi/v0/products?contractId=someId
type Products struct {
	Products struct {
		Items []ProductSummary `json:"items"`
	} `json:"products"`
}

// ProductSummary is a representation of the Akamai PAPI
// product summary associated with each product returned
// by the products response at:
// http://apibase.com/papi/v0/products?contractId=someId
type ProductSummary struct {
	ProductID string `json:"productId"`
	Name      string `json:"productName"`
}

// CpCodes is a representation of the Akamai PAPI
// CP codes associated with the CP codes response available at:
// http://apibase.com/papi/v0/cpcodes/?contractId=contractId&groupId=groupId
type CpCodes struct {
	CpCodes struct {
		Items []CpCodeSummary `json:"items"`
	} `json:"cpcodes"`
}

// CpCodeSummary is a representation of the Akamai PAPI
// CP code summary associated with each CP code returned
// by the CP codes response at:
// http://apibase.com/papi/v0/cpcodes/?contractId=contractId&groupId=groupId
type CpCodeSummary struct {
	CPCodeID    string   `json:"cpcodeId"`
	Name        string   `json:"cpcodeName"`
	ProductIDs  []string `json:"productIds"`
	CreatedDate string   `json:"createdDate"`
}

// Hostnames is a representation of the Akamai PAPI
// hostnames response available at:
// http://apibase.com/papi/v0/edgehostnames?contractId=contractId&groupId=groupId
type Hostnames struct {
	Hostnames struct {
		Items []HostnameSummary `json:"items"`
	} `json:"edgehostnames"`
}

// HostnameSummary is a representation of the Akamai PAPI
// hostname summary associated with each hostname returned
// by the hostnames response at:
// http://apibase.com/papi/v0/edgehostnames?contractId=contractId&groupId=groupId
type HostnameSummary struct {
	EdgeHostnameID     string `json:"edgeHostnameId"`
	DomainPrefix       string `json:"domainPrefix"`
	DomainSuffix       string `json:"domainSuffix"`
	IPVersionBehavior  string `json:"ipVersionBehavior"`
	Secure             bool   `json:"secure"`
	EdgeHostnameDomain string `json:"edgehostnameDomain"`
}

// PapiProperties is a representation of the Akamai PAPI
// properties response available at:
// http://apibase.com/papi/v0/properties/?contractId=contractId&groupId=groupId
type PapiProperties struct {
	Properties struct {
		Items []PapiPropertySummary `json:"items"`
	} `json:"properties"`
}

// PapiPropertySummary is a representation of the Akamai PAPI
// property summary associated with each property returned by
// the properties response at:
// http://apibase.com/papi/v0/properties/?contractId=contractId&groupId=groupId
type PapiPropertySummary struct {
	AccountID         string `json:"accountId"`
	ContractID        string `json:"contractId"`
	GroupID           string `json:"groupId"`
	PropertyID        string `json:"propertyId"`
	Name              string `json:"propertyName"`
	LatestVersion     int    `json:"latestVersion"`
	StagingVersion    int    `json:"stagingVersion"`
	ProductionVersion int    `json:"productionVersion"`
	Note              string `json:"note"`
}

// PapiPropertyVersions is a representation of the Akamai PAPI
// property versions response available at:
// http://apibase.com/papi/v0/properties/propId/versions?contractId=contractId&groupId=groupId
type PapiPropertyVersions struct {
	Versions struct {
		Items []PapiPropertyVersionSummary `json:"items"`
	} `json:"versions"`
}

// PapiPropertyVersionSummary is a representation of the Akamai PAPI
// property version summary associated with each property version at:
// http://apibase.com/papi/v0/properties/propId/versions?contractId=contractId&groupId=groupId
type PapiPropertyVersionSummary struct {
	PropertyVersion  int    `json:"propertyVersion"`
	UpdatedByUser    string `json:"updatedByUser"`
	UpdatedDate      string `json:"updatedDate"`
	ProductionStatus string `json:"productionStatus"`
	StagingStatus    string `json:"stagingStatus"`
	Etag             string `json:"etag"`
	ProductID        string `json:"productId"`
	Note             string `json:"note"`
}

// PapiPropertyRules is a representation of the Akamai PAPI
// property rules response at:
// http://apibase.com/papi/v0/properties/propId/versions/version/rules/?contractId=contractId&groupId=groupId
type PapiPropertyRules struct {
	Rules PapiPropertyRuleSummary `json:"rules"`
}

// PapiPropertyRuleSummary is a representation of the Akamai PAPI
// rule summary associated with each property rule at:
// http://apibase.com/papi/v0/properties/propId/versions/version/rules/?contractId=contractId&groupId=groupId
type PapiPropertyRuleSummary struct {
	Name      string                     `json:"name"`
	UUID      string                     `json:"uuid"`
	Behaviors []PapiPropertyRuleBehavior `json:"behaviors"`
}

// PapiPropertyRuleBehavior is a representation of the Akamai PAPI
// property rule behavior associated with each property rule.
type PapiPropertyRuleBehavior struct {
	Name string `json:"name"`
}

// PapiActivations is a representation of the Akamai PAPI
// activations response at:
// http://apibase.com/papi/v0/properties/propId/activations?contractId=contractId&groupId=groupId
type PapiActivations struct {
	Activations struct {
		Items []PapiActivation `json:"items"`
	} `json:"activations"`
}

// PapiActivation is a representation of each Akamai PAPI
// activation available at:
// http://apibase.com/papi/v0/properties/propId/activations?contractId=contractId&groupId=groupId
type PapiActivation struct {
	ActivationID    string `json:"activationId"`
	PropertyName    string `json:"propertyName"`
	PropertyID      string `json:"propertyId"`
	PropertyVersion int    `json:"propertyVersion"`
	Network         string `json:"network"`
	ActivationType  string `json:"activationType"`
	Status          string `json:"status"`
	SubmitDate      string `json:"submitDate"`
	UpdateDate      string `json:"updateDate"`
	Note            string `json:"note"`
}
