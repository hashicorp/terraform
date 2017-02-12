package rabbithole

import (
	"encoding/json"
	"net/http"
	"net/url"
)

//
// GET /api/permissions
//

// Example response:
//
// [{"user":"guest","vhost":"/","configure":".*","write":".*","read":".*"}]

type PermissionInfo struct {
	User  string `json:"user"`
	Vhost string `json:"vhost"`

	// Configuration permissions
	Configure string `json:"configure"`
	// Write permissions
	Write string `json:"write"`
	// Read permissions
	Read string `json:"read"`
}

// Returns permissions for all users and virtual hosts.
func (c *Client) ListPermissions() (rec []PermissionInfo, err error) {
	req, err := newGETRequest(c, "permissions/")
	if err != nil {
		return []PermissionInfo{}, err
	}

	if err = executeAndParseRequest(c, req, &rec); err != nil {
		return []PermissionInfo{}, err
	}

	return rec, nil
}

//
// GET /api/users/{user}/permissions
//

// Returns permissions of a specific user.
func (c *Client) ListPermissionsOf(username string) (rec []PermissionInfo, err error) {
	req, err := newGETRequest(c, "users/"+url.QueryEscape(username)+"/permissions")
	if err != nil {
		return []PermissionInfo{}, err
	}

	if err = executeAndParseRequest(c, req, &rec); err != nil {
		return []PermissionInfo{}, err
	}

	return rec, nil
}

//
// GET /api/permissions/{vhost}/{user}
//

// Returns permissions of user in virtual host.
func (c *Client) GetPermissionsIn(vhost, username string) (rec PermissionInfo, err error) {
	req, err := newGETRequest(c, "permissions/"+url.QueryEscape(vhost)+"/"+url.QueryEscape(username))
	if err != nil {
		return PermissionInfo{}, err
	}

	if err = executeAndParseRequest(c, req, &rec); err != nil {
		return PermissionInfo{}, err
	}

	return rec, nil
}

//
// PUT /api/permissions/{vhost}/{user}
//

type Permissions struct {
	Configure string `json:"configure"`
	Write     string `json:"write"`
	Read      string `json:"read"`
}

// Updates permissions of user in virtual host.
func (c *Client) UpdatePermissionsIn(vhost, username string, permissions Permissions) (res *http.Response, err error) {
	body, err := json.Marshal(permissions)
	if err != nil {
		return nil, err
	}

	req, err := newRequestWithBody(c, "PUT", "permissions/"+url.QueryEscape(vhost)+"/"+url.QueryEscape(username), body)
	if err != nil {
		return nil, err
	}

	res, err = executeRequest(c, req)
	if err != nil {
		return nil, err
	}

	return res, nil
}

//
// DELETE /api/permissions/{vhost}/{user}
//

// Clears (deletes) permissions of user in virtual host.
func (c *Client) ClearPermissionsIn(vhost, username string) (res *http.Response, err error) {
	req, err := newRequestWithBody(c, "DELETE", "permissions/"+url.QueryEscape(vhost)+"/"+url.QueryEscape(username), nil)
	if err != nil {
		return nil, err
	}

	res, err = executeRequest(c, req)
	if err != nil {
		return nil, err
	}

	return res, nil
}
