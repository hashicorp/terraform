/*
 * Datadog API for Go
 *
 * Please see the included LICENSE file for licensing information.
 *
 * Copyright 2013 by authors and contributors.
 */

package datadog

type User struct {
	Handle   *string `json:"handle,omitempty"`
	Email    *string `json:"email,omitempty"`
	Name     *string `json:"name,omitempty"`
	Role     *string `json:"role,omitempty"`
	IsAdmin  *bool   `json:"is_admin,omitempty"`
	Verified *bool   `json:"verified,omitempty"`
	Disabled *bool   `json:"disabled,omitempty"`
}

// reqInviteUsers contains email addresses to send invitations to.
type reqInviteUsers struct {
	Emails []string `json:"emails,omitempty"`
}

// InviteUsers takes a slice of email addresses and sends invitations to them.
func (client *Client) InviteUsers(emails []string) error {
	return client.doJsonRequest("POST", "/v1/invite_users",
		reqInviteUsers{Emails: emails}, nil)
}

// CreateUser creates an user account for an email address
func (self *Client) CreateUser(handle, name *string) (*User, error) {
	in := struct {
		Handle *string `json:"handle"`
		Name   *string `json:"name"`
	}{
		Handle: handle,
		Name:   name,
	}

	out := struct {
		*User `json:"user"`
	}{}
	if err := self.doJsonRequest("POST", "/v1/user", in, &out); err != nil {
		return nil, err
	}
	return out.User, nil
}

// internal type to retrieve users from the api
type usersData struct {
	Users []User `json:"users,omitempty"`
}

// GetUsers returns all user, or an error if not found
func (client *Client) GetUsers() (users []User, err error) {
	var udata usersData
	uri := "/v1/user"
	err = client.doJsonRequest("GET", uri, nil, &udata)
	users = udata.Users
	return
}

// internal type to retrieve single user from the api
type userData struct {
	User User `json:"user"`
}

// GetUser returns the user that match a handle, or an error if not found
func (client *Client) GetUser(handle string) (user User, err error) {
	var udata userData
	uri := "/v1/user/" + handle
	err = client.doJsonRequest("GET", uri, nil, &udata)
	user = udata.User
	return
}

// UpdateUser updates a user with the content of `user`,
// and returns an error if the update failed
func (client *Client) UpdateUser(user User) error {
	uri := "/v1/user/" + *user.Handle
	return client.doJsonRequest("PUT", uri, user, nil)
}

// DeleteUser deletes a user and returns an error if deletion failed
func (client *Client) DeleteUser(handle string) error {
	uri := "/v1/user/" + handle
	return client.doJsonRequest("DELETE", uri, nil, nil)
}
