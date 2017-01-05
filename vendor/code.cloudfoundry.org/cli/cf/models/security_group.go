package models

// represents just the attributes for an security group
type SecurityGroupFields struct {
	Name     string
	GUID     string
	SpaceURL string `json:"spaces_url,omitempty"`
	Rules    []map[string]interface{}
}

// represents the JSON that we send up to CC when the user creates / updates a record
type SecurityGroupParams struct {
	Name  string                   `json:"name,omitempty"`
	GUID  string                   `json:"guid,omitempty"`
	Rules []map[string]interface{} `json:"rules"`
}

// represents a fully instantiated model returned by the CC (e.g.: with its attributes and the fields for its child objects)
type SecurityGroup struct {
	SecurityGroupFields
	Spaces []Space
}
