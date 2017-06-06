//
// Copyright 2015, Sander van Harmelen
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package gitlab

import (
	"errors"
	"fmt"
	"time"
)

// UsersService handles communication with the user related methods of
// the GitLab API.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/users.md
type UsersService struct {
	client *Client
}

// User represents a GitLab user.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/users.md
type User struct {
	ID               int             `json:"id"`
	Username         string          `json:"username"`
	Email            string          `json:"email"`
	Name             string          `json:"name"`
	State            string          `json:"state"`
	CreatedAt        *time.Time      `json:"created_at"`
	Bio              string          `json:"bio"`
	Skype            string          `json:"skype"`
	Linkedin         string          `json:"linkedin"`
	Twitter          string          `json:"twitter"`
	WebsiteURL       string          `json:"website_url"`
	ExternUID        string          `json:"extern_uid"`
	Provider         string          `json:"provider"`
	ThemeID          int             `json:"theme_id"`
	ColorSchemeID    int             `json:"color_scheme_id"`
	IsAdmin          bool            `json:"is_admin"`
	AvatarURL        string          `json:"avatar_url"`
	CanCreateGroup   bool            `json:"can_create_group"`
	CanCreateProject bool            `json:"can_create_project"`
	ProjectsLimit    int             `json:"projects_limit"`
	CurrentSignInAt  *time.Time      `json:"current_sign_in_at"`
	LastSignInAt     *time.Time      `json:"last_sign_in_at"`
	TwoFactorEnabled bool            `json:"two_factor_enabled"`
	Identities       []*UserIdentity `json:"identities"`
}

// UserIdentity represents a user identity
type UserIdentity struct {
	Provider  string `json:"provider"`
	ExternUID string `json:"extern_uid"`
}

// ListUsersOptions represents the available ListUsers() options.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/users.md#list-users
type ListUsersOptions struct {
	ListOptions
	Active   *bool   `url:"active,omitempty" json:"active,omitempty"`
	Search   *string `url:"search,omitempty" json:"search,omitempty"`
	Username *string `url:"username,omitempty" json:"username,omitempty"`
}

// ListUsers gets a list of users.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/users.md#list-users
func (s *UsersService) ListUsers(opt *ListUsersOptions, options ...OptionFunc) ([]*User, *Response, error) {
	req, err := s.client.NewRequest("GET", "users", opt, options)
	if err != nil {
		return nil, nil, err
	}

	var usr []*User
	resp, err := s.client.Do(req, &usr)
	if err != nil {
		return nil, resp, err
	}

	return usr, resp, err
}

// GetUser gets a single user.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/users.md#single-user
func (s *UsersService) GetUser(user int, options ...OptionFunc) (*User, *Response, error) {
	u := fmt.Sprintf("users/%d", user)

	req, err := s.client.NewRequest("GET", u, nil, options)
	if err != nil {
		return nil, nil, err
	}

	usr := new(User)
	resp, err := s.client.Do(req, usr)
	if err != nil {
		return nil, resp, err
	}

	return usr, resp, err
}

// CreateUserOptions represents the available CreateUser() options.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/users.md#user-creation
type CreateUserOptions struct {
	Email          *string `url:"email,omitempty" json:"email,omitempty"`
	Password       *string `url:"password,omitempty" json:"password,omitempty"`
	Username       *string `url:"username,omitempty" json:"username,omitempty"`
	Name           *string `url:"name,omitempty" json:"name,omitempty"`
	Skype          *string `url:"skype,omitempty" json:"skype,omitempty"`
	Linkedin       *string `url:"linkedin,omitempty" json:"linkedin,omitempty"`
	Twitter        *string `url:"twitter,omitempty" json:"twitter,omitempty"`
	WebsiteURL     *string `url:"website_url,omitempty" json:"website_url,omitempty"`
	ProjectsLimit  *int    `url:"projects_limit,omitempty" json:"projects_limit,omitempty"`
	ExternUID      *string `url:"extern_uid,omitempty" json:"extern_uid,omitempty"`
	Provider       *string `url:"provider,omitempty" json:"provider,omitempty"`
	Bio            *string `url:"bio,omitempty" json:"bio,omitempty"`
	Admin          *bool   `url:"admin,omitempty" json:"admin,omitempty"`
	CanCreateGroup *bool   `url:"can_create_group,omitempty" json:"can_create_group,omitempty"`
	Confirm        *bool   `url:"confirm,omitempty" json:"confirm,omitempty"`
}

// CreateUser creates a new user. Note only administrators can create new users.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/users.md#user-creation
func (s *UsersService) CreateUser(opt *CreateUserOptions, options ...OptionFunc) (*User, *Response, error) {
	req, err := s.client.NewRequest("POST", "users", opt, options)
	if err != nil {
		return nil, nil, err
	}

	usr := new(User)
	resp, err := s.client.Do(req, usr)
	if err != nil {
		return nil, resp, err
	}

	return usr, resp, err
}

// ModifyUserOptions represents the available ModifyUser() options.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/users.md#user-modification
type ModifyUserOptions struct {
	Email          *string `url:"email,omitempty" json:"email,omitempty"`
	Password       *string `url:"password,omitempty" json:"password,omitempty"`
	Username       *string `url:"username,omitempty" json:"username,omitempty"`
	Name           *string `url:"name,omitempty" json:"name,omitempty"`
	Skype          *string `url:"skype,omitempty" json:"skype,omitempty"`
	Linkedin       *string `url:"linkedin,omitempty" json:"linkedin,omitempty"`
	Twitter        *string `url:"twitter,omitempty" json:"twitter,omitempty"`
	WebsiteURL     *string `url:"website_url,omitempty" json:"website_url,omitempty"`
	ProjectsLimit  *int    `url:"projects_limit,omitempty" json:"projects_limit,omitempty"`
	ExternUID      *string `url:"extern_uid,omitempty" json:"extern_uid,omitempty"`
	Provider       *string `url:"provider,omitempty" json:"provider,omitempty"`
	Bio            *string `url:"bio,omitempty" json:"bio,omitempty"`
	Admin          *bool   `url:"admin,omitempty" json:"admin,omitempty"`
	CanCreateGroup *bool   `url:"can_create_group,omitempty" json:"can_create_group,omitempty"`
}

// ModifyUser modifies an existing user. Only administrators can change attributes
// of a user.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/users.md#user-modification
func (s *UsersService) ModifyUser(user int, opt *ModifyUserOptions, options ...OptionFunc) (*User, *Response, error) {
	u := fmt.Sprintf("users/%d", user)

	req, err := s.client.NewRequest("PUT", u, opt, options)
	if err != nil {
		return nil, nil, err
	}

	usr := new(User)
	resp, err := s.client.Do(req, usr)
	if err != nil {
		return nil, resp, err
	}

	return usr, resp, err
}

// DeleteUser deletes a user. Available only for administrators. This is an
// idempotent function, calling this function for a non-existent user id still
// returns a status code 200 OK. The JSON response differs if the user was
// actually deleted or not. In the former the user is returned and in the
// latter not.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/users.md#user-deletion
func (s *UsersService) DeleteUser(user int, options ...OptionFunc) (*Response, error) {
	u := fmt.Sprintf("users/%d", user)

	req, err := s.client.NewRequest("DELETE", u, nil, options)
	if err != nil {
		return nil, err
	}

	return s.client.Do(req, nil)
}

// CurrentUser gets currently authenticated user.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/users.md#current-user
func (s *UsersService) CurrentUser(options ...OptionFunc) (*User, *Response, error) {
	req, err := s.client.NewRequest("GET", "user", nil, options)
	if err != nil {
		return nil, nil, err
	}

	usr := new(User)
	resp, err := s.client.Do(req, usr)
	if err != nil {
		return nil, resp, err
	}

	return usr, resp, err
}

// SSHKey represents a SSH key.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/users.md#list-ssh-keys
type SSHKey struct {
	ID        int        `json:"id"`
	Title     string     `json:"title"`
	Key       string     `json:"key"`
	CreatedAt *time.Time `json:"created_at"`
}

// ListSSHKeys gets a list of currently authenticated user's SSH keys.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/users.md#list-ssh-keys
func (s *UsersService) ListSSHKeys(options ...OptionFunc) ([]*SSHKey, *Response, error) {
	req, err := s.client.NewRequest("GET", "user/keys", nil, options)
	if err != nil {
		return nil, nil, err
	}

	var k []*SSHKey
	resp, err := s.client.Do(req, &k)
	if err != nil {
		return nil, resp, err
	}

	return k, resp, err
}

// ListSSHKeysForUser gets a list of a specified user's SSH keys. Available
// only for admin
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/users.md#list-ssh-keys-for-user
func (s *UsersService) ListSSHKeysForUser(user int, options ...OptionFunc) ([]*SSHKey, *Response, error) {
	u := fmt.Sprintf("users/%d/keys", user)

	req, err := s.client.NewRequest("GET", u, nil, options)
	if err != nil {
		return nil, nil, err
	}

	var k []*SSHKey
	resp, err := s.client.Do(req, &k)
	if err != nil {
		return nil, resp, err
	}

	return k, resp, err
}

// GetSSHKey gets a single key.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/users.md#single-ssh-key
func (s *UsersService) GetSSHKey(kid int, options ...OptionFunc) (*SSHKey, *Response, error) {
	u := fmt.Sprintf("user/keys/%d", kid)

	req, err := s.client.NewRequest("GET", u, nil, options)
	if err != nil {
		return nil, nil, err
	}

	k := new(SSHKey)
	resp, err := s.client.Do(req, k)
	if err != nil {
		return nil, resp, err
	}

	return k, resp, err
}

// AddSSHKeyOptions represents the available AddSSHKey() options.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/projects.md#add-ssh-key
type AddSSHKeyOptions struct {
	Title *string `url:"title,omitempty" json:"title,omitempty"`
	Key   *string `url:"key,omitempty" json:"key,omitempty"`
}

// AddSSHKey creates a new key owned by the currently authenticated user.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/users.md#add-ssh-key
func (s *UsersService) AddSSHKey(opt *AddSSHKeyOptions, options ...OptionFunc) (*SSHKey, *Response, error) {
	req, err := s.client.NewRequest("POST", "user/keys", opt, options)
	if err != nil {
		return nil, nil, err
	}

	k := new(SSHKey)
	resp, err := s.client.Do(req, k)
	if err != nil {
		return nil, resp, err
	}

	return k, resp, err
}

// AddSSHKeyForUser creates new key owned by specified user. Available only for
// admin.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/users.md#add-ssh-key-for-user
func (s *UsersService) AddSSHKeyForUser(user int, opt *AddSSHKeyOptions, options ...OptionFunc) (*SSHKey, *Response, error) {
	u := fmt.Sprintf("users/%d/keys", user)

	req, err := s.client.NewRequest("POST", u, opt, options)
	if err != nil {
		return nil, nil, err
	}

	k := new(SSHKey)
	resp, err := s.client.Do(req, k)
	if err != nil {
		return nil, resp, err
	}

	return k, resp, err
}

// DeleteSSHKey deletes key owned by currently authenticated user. This is an
// idempotent function and calling it on a key that is already deleted or not
// available results in 200 OK.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/users.md#delete-ssh-key-for-current-owner
func (s *UsersService) DeleteSSHKey(kid int, options ...OptionFunc) (*Response, error) {
	u := fmt.Sprintf("user/keys/%d", kid)

	req, err := s.client.NewRequest("DELETE", u, nil, options)
	if err != nil {
		return nil, err
	}

	return s.client.Do(req, nil)
}

// DeleteSSHKeyForUser deletes key owned by a specified user. Available only
// for admin.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/users.md#delete-ssh-key-for-given-user
func (s *UsersService) DeleteSSHKeyForUser(user int, kid int, options ...OptionFunc) (*Response, error) {
	u := fmt.Sprintf("users/%d/keys/%d", user, kid)

	req, err := s.client.NewRequest("DELETE", u, nil, options)
	if err != nil {
		return nil, err
	}

	return s.client.Do(req, nil)
}

// BlockUser blocks the specified user. Available only for admin.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/users.md#block-user
func (s *UsersService) BlockUser(user int, options ...OptionFunc) error {
	u := fmt.Sprintf("users/%d/block", user)

	req, err := s.client.NewRequest("PUT", u, nil, options)
	if err != nil {
		return err
	}

	resp, err := s.client.Do(req, nil)
	if err != nil {
		return err
	}

	switch resp.StatusCode {
	case 200:
		return nil
	case 403:
		return errors.New("Cannot block a user that is already blocked by LDAP synchronization")
	case 404:
		return errors.New("User does not exist")
	default:
		return fmt.Errorf("Received unexpected result code: %d", resp.StatusCode)
	}
}

// UnblockUser unblocks the specified user. Available only for admin.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/users.md#unblock-user
func (s *UsersService) UnblockUser(user int, options ...OptionFunc) error {
	u := fmt.Sprintf("users/%d/unblock", user)

	req, err := s.client.NewRequest("PUT", u, nil, options)
	if err != nil {
		return err
	}

	resp, err := s.client.Do(req, nil)
	if err != nil {
		return err
	}

	switch resp.StatusCode {
	case 200:
		return nil
	case 403:
		return errors.New("Cannot unblock a user that is blocked by LDAP synchronization")
	case 404:
		return errors.New("User does not exist")
	default:
		return fmt.Errorf("Received unexpected result code: %d", resp.StatusCode)
	}
}

// Email represents an Email.
//
// GitLab API docs: https://doc.gitlab.com/ce/api/users.html#list-emails
type Email struct {
	ID    int    `json:"id"`
	Email string `json:"email"`
}

// ListEmails gets a list of currently authenticated user's Emails.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/users.md#list-emails
func (s *UsersService) ListEmails(options ...OptionFunc) ([]*Email, *Response, error) {
	req, err := s.client.NewRequest("GET", "user/emails", nil, options)
	if err != nil {
		return nil, nil, err
	}

	var e []*Email
	resp, err := s.client.Do(req, &e)
	if err != nil {
		return nil, resp, err
	}

	return e, resp, err
}

// ListEmailsForUser gets a list of a specified user's Emails. Available
// only for admin
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/users.md#list-emails-for-user
func (s *UsersService) ListEmailsForUser(uid int, options ...OptionFunc) ([]*Email, *Response, error) {
	u := fmt.Sprintf("users/%d/emails", uid)

	req, err := s.client.NewRequest("GET", u, nil, options)
	if err != nil {
		return nil, nil, err
	}

	var e []*Email
	resp, err := s.client.Do(req, &e)
	if err != nil {
		return nil, resp, err
	}

	return e, resp, err
}

// GetEmail gets a single email.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/users.md#single-email
func (s *UsersService) GetEmail(eid int, options ...OptionFunc) (*Email, *Response, error) {
	u := fmt.Sprintf("user/emails/%d", eid)

	req, err := s.client.NewRequest("GET", u, nil, options)
	if err != nil {
		return nil, nil, err
	}

	e := new(Email)
	resp, err := s.client.Do(req, e)
	if err != nil {
		return nil, resp, err
	}

	return e, resp, err
}

// AddEmailOptions represents the available AddEmail() options.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/projects.md#add-email
type AddEmailOptions struct {
	Email *string `url:"email,omitempty" json:"email,omitempty"`
}

// AddEmail creates a new email owned by the currently authenticated user.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/users.md#add-email
func (s *UsersService) AddEmail(opt *AddEmailOptions, options ...OptionFunc) (*Email, *Response, error) {
	req, err := s.client.NewRequest("POST", "user/emails", opt, options)
	if err != nil {
		return nil, nil, err
	}

	e := new(Email)
	resp, err := s.client.Do(req, e)
	if err != nil {
		return nil, resp, err
	}

	return e, resp, err
}

// AddEmailForUser creates new email owned by specified user. Available only for
// admin.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/users.md#add-email-for-user
func (s *UsersService) AddEmailForUser(uid int, opt *AddEmailOptions, options ...OptionFunc) (*Email, *Response, error) {
	u := fmt.Sprintf("users/%d/emails", uid)

	req, err := s.client.NewRequest("POST", u, opt, options)
	if err != nil {
		return nil, nil, err
	}

	e := new(Email)
	resp, err := s.client.Do(req, e)
	if err != nil {
		return nil, resp, err
	}

	return e, resp, err
}

// DeleteEmail deletes email owned by currently authenticated user. This is an
// idempotent function and calling it on a key that is already deleted or not
// available results in 200 OK.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/users.md#delete-email-for-current-owner
func (s *UsersService) DeleteEmail(eid int, options ...OptionFunc) (*Response, error) {
	u := fmt.Sprintf("user/emails/%d", eid)

	req, err := s.client.NewRequest("DELETE", u, nil, options)
	if err != nil {
		return nil, err
	}

	return s.client.Do(req, nil)
}

// DeleteEmailForUser deletes email owned by a specified user. Available only
// for admin.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/users.md#delete-email-for-given-user
func (s *UsersService) DeleteEmailForUser(uid int, eid int, options ...OptionFunc) (*Response, error) {
	u := fmt.Sprintf("users/%d/emails/%d", uid, eid)

	req, err := s.client.NewRequest("DELETE", u, nil, options)
	if err != nil {
		return nil, err
	}

	return s.client.Do(req, nil)
}
