/*
 * Datadog API for Go
 *
 * Please see the included LICENSE file for licensing information.
 *
 * Copyright 2013 by authors and contributors.
 */

package datadog

type User struct {
	Handle   string `json:"handle,omitempty"`
	Email    string `json:"email,omitempty"`
	Name     string `json:"name,omitempty"`
	Role     string `json:"role,omitempty"`
	IsAdmin  bool   `json:"is_admin,omitempty"`
	Verified bool   `json:"verified,omitempty"`
	Disabled bool   `json:"disabled,omitempty"`
}

// reqInviteUsers contains email addresses to send invitations to.
type reqInviteUsers struct {
	Emails []string `json:"emails,omitempty"`
}

// InviteUsers takes a slice of email addresses and sends invitations to them.
func (self *Client) InviteUsers(emails []string) error {
	return self.doJsonRequest("POST", "/v1/invite_users",
		reqInviteUsers{Emails: emails}, nil)
}

// internal type to retrieve users from the api
type usersData struct {
	Users []User `json:"users"`
}

// GetUsers returns all user, or an error if not found
func (self *Client) GetUsers() (users []User, err error) {
	var udata usersData
	uri := "/v1/user"
	err = self.doJsonRequest("GET", uri, nil, &udata)
	users = udata.Users
	return
}

// internal type to retrieve single user from the api
type userData struct {
	User User `json:"user"`
}

// GetUser returns the user that match a handle, or an error if not found
func (self *Client) GetUser(handle string) (user User, err error) {
	var udata userData
	uri := "/v1/user/" + handle
	err = self.doJsonRequest("GET", uri, nil, &udata)
	user = udata.User
	return
}

// UpdateUser updates a user with the content of `user`,
// and returns an error if the update failed
func (self *Client) UpdateUser(user User) error {
	uri := "/v1/user/" + user.Handle
	return self.doJsonRequest("PUT", uri, user, nil)
}

// DeleteUser deletes a user and returns an error if deletion failed
func (self *Client) DeleteUser(handle string) error {
	uri := "/v1/user/" + handle
	return self.doJsonRequest("DELETE", uri, nil, nil)
}
