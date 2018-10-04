package artifactory

import (
	"encoding/json"
)

// PermissionTarget represents the json returned by Artifactory for a permission target
type PermissionTarget struct {
	Name string `json:"name"`
	URI  string `json:"uri"`
}

// PermissionTargetDetails represents the json returned by Artifactory for permission target details
type PermissionTargetDetails struct {
	Name            string     `json:"name,omitempty"`
	IncludesPattern string     `json:"includesPattern,omitempty"`
	ExcludesPattern string     `json:"excludesPattern,omitempty"`
	Repositories    []string   `json:"repositories,omitempty"`
	Principals      Principals `json:"principals,omitempty"`
}

// Principals represents the json response for principals in Artifactory
type Principals struct {
	Users  map[string][]string `json:"users"`
	Groups map[string][]string `json:"groups"`
}

// GetPermissionTargets returns all permission targets
func (c *Client) GetPermissionTargets() ([]PermissionTarget, error) {
	var res []PermissionTarget
	d, err := c.Get("/api/security/permissions", make(map[string]string))
	if err != nil {
		return res, err
	}
	err = json.Unmarshal(d, &res)

	return res, err
}

// GetPermissionTargetDetails returns the details of the provided permission target
func (c *Client) GetPermissionTargetDetails(key string, q map[string]string) (PermissionTargetDetails, error) {
	var res PermissionTargetDetails
	d, err := c.Get("/api/security/permissions/"+key, q)
	if err != nil {
		return res, err
	}
	err = json.Unmarshal(d, &res)

	return res, err
}

// CreatePermissionTarget creates the named permission target
func (c *Client) CreatePermissionTarget(key string, p PermissionTargetDetails, q map[string]string) error {
	j, err := json.Marshal(p)
	if err != nil {
		return err
	}
	_, err = c.Put("/api/security/permissions/"+key, j, q)
	return err
}
