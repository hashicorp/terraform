package artifactory

import (
	"encoding/json"
)

type PermissionTarget struct {
	Name string `json:"name"`
	Uri  string `json:"uri"`
}

type PermissionTargetDetails struct {
	Name            string     `json:"name,omitempty"`
	IncludesPattern string     `json:"includesPattern,omitempty"`
	ExcludesPattern string     `json:"excludesPattern,omitempty"`
	Repositories    []string   `json:"repositories,omitempty"`
	Principals      Principals `json:"principals,omitempty"`
}

type Principals struct {
	Users  map[string][]string `json:"users"`
	Groups map[string][]string `json:"groups"`
}

func (c *ArtifactoryClient) GetPermissionTargets() ([]PermissionTarget, error) {
	var res []PermissionTarget
	d, e := c.Get("/api/security/permissions", make(map[string]string))
	if e != nil {
		return res, e
	} else {
		err := json.Unmarshal(d, &res)
		if err != nil {
			return res, err
		} else {
			return res, e
		}
	}
}

func (c *ArtifactoryClient) GetPermissionTargetDetails(u string) (PermissionTargetDetails, error) {
	var res PermissionTargetDetails
	d, e := c.Get("/api/security/permissions/"+u, make(map[string]string))
	if e != nil {
		return res, e
	} else {
		err := json.Unmarshal(d, &res)
		if err != nil {
			return res, err
		} else {
			return res, e
		}
	}
}
