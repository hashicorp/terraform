package nsone

type Apikey struct {
	Id          string         `json:"id,omitempty"`
	Name        string         `json:"name"`
	Key         string         `json:"key,omitempty"`
	LastAccess  int            `json:"last_access,omitempty"`
	Teams       []string       `json:"teams"`
	Permissions PermissionsMap `json:"permissions"`
}
