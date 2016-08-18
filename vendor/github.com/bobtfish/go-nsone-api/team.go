package nsone

type Team struct {
	Id          string         `json:"id,omitempty"`
	Name        string         `json:"name"`
	Permissions PermissionsMap `json:"permissions"`
}
