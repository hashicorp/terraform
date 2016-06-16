package cloudflare

// Organization represents a multi-user organization.
type Organization struct {
	ID          string   `json:"id,omitempty"`
	Name        string   `json:"name,omitempty"`
	Status      string   `json:"status,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
	Roles       []string `json:"roles,omitempty"`
}

// OrganizationResponse represents the response from the Organization endpoint.
type OrganizationResponse struct {
	Response
	Result     []Organization `json:"result"`
	ResultInfo ResultInfo     `json:"result_info"`
}
