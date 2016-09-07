package nsone

// Team wraps an NS1 /accounts/teams resource
type Team struct {
	Id          string         `json:"id,omitempty"`
	Name        string         `json:"name"`
	Permissions PermissionsMap `json:"permissions"`
}
