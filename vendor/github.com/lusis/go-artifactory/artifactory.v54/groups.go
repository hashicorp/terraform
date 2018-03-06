package artifactory

import (
	"encoding/json"
)

// Group represents the json response for a group in Artifactory
type Group struct {
	Name string `json:"name"`
	URI  string `json:"uri"`
}

// GroupDetails represents the json response for a group's details in artifactory
type GroupDetails struct {
	Name            string `json:"name,omitempty"`
	Description     string `json:"description,omitempty"`
	AutoJoin        bool   `json:"autoJoin,omitempty"`
	Admin           bool   `json:"admin,omitempty"`
	Realm           string `json:"realm,omitempty"`
	RealmAttributes string `json:"realmAttributes,omitempty"`
}

// GetGroups gets a list of groups from artifactory
func (c *Client) GetGroups() ([]Group, error) {
	var res []Group
	d, err := c.Get("/api/security/groups", make(map[string]string))
	if err != nil {
		return res, err
	}
	err = json.Unmarshal(d, &res)
	return res, err
}

// GetGroupDetails returns details for a Group
func (c *Client) GetGroupDetails(key string, q map[string]string) (GroupDetails, error) {
	var res GroupDetails
	d, err := c.Get("/api/security/groups/"+key, q)
	if err != nil {
		return res, err
	}
	err = json.Unmarshal(d, &res)
	return res, err
}

// CreateGroup creates a group in artifactory
func (c *Client) CreateGroup(key string, g GroupDetails, q map[string]string) error {
	j, err := json.Marshal(g)
	if err != nil {
		return err
	}
	_, err = c.Put("/api/security/groups/"+key, j, q)
	return err
}
