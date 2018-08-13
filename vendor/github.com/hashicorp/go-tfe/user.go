package tfe

import (
	"context"
)

// Compile-time proof of interface implementation.
var _ Users = (*users)(nil)

// Users describes all the user related methods that the Terraform
// Enterprise API supports.
//
// TFE API docs: https://www.terraform.io/docs/enterprise/api/user.html
type Users interface {
	// ReadCurrent reads the details of the currently authenticated user.
	ReadCurrent(ctx context.Context) (*User, error)

	// Update attributes of the currently authenticated user.
	Update(ctx context.Context, options UserUpdateOptions) (*User, error)
}

// users implements Users.
type users struct {
	client *Client
}

// User represents a Terraform Enterprise user.
type User struct {
	ID               string     `jsonapi:"primary,users"`
	AvatarURL        string     `jsonapi:"attr,avatar-url"`
	Email            string     `jsonapi:"attr,email"`
	IsServiceAccount bool       `jsonapi:"attr,is-service-account"`
	TwoFactor        *TwoFactor `jsonapi:"attr,two-factor"`
	UnconfirmedEmail string     `jsonapi:"attr,unconfirmed-email"`
	Username         string     `jsonapi:"attr,username"`
	V2Only           bool       `jsonapi:"attr,v2-only"`

	// Relations
	// AuthenticationTokens *AuthenticationTokens `jsonapi:"relation,authentication-tokens"`
}

// TwoFactor represents the organization permissions.
type TwoFactor struct {
	Enabled  bool `json:"enabled"`
	Verified bool `json:"verified"`
}

// ReadCurrent reads the details of the currently authenticated user.
func (s *users) ReadCurrent(ctx context.Context) (*User, error) {
	req, err := s.client.newRequest("GET", "account/details", nil)
	if err != nil {
		return nil, err
	}

	u := &User{}
	err = s.client.do(ctx, req, u)
	if err != nil {
		return nil, err
	}

	return u, nil
}

// UserUpdateOptions represents the options for updating a user.
type UserUpdateOptions struct {
	// For internal use only!
	ID string `jsonapi:"primary,users"`

	// New username.
	Username *string `jsonapi:"attr,username,omitempty"`

	// New email address (must be consumed afterwards to take effect).
	Email *string `jsonapi:"attr,email,omitempty"`
}

// Update attributes of the currently authenticated user.
func (s *users) Update(ctx context.Context, options UserUpdateOptions) (*User, error) {
	// Make sure we don't send a user provided ID.
	options.ID = ""

	req, err := s.client.newRequest("PATCH", "account/update", &options)
	if err != nil {
		return nil, err
	}

	u := &User{}
	err = s.client.do(ctx, req, u)
	if err != nil {
		return nil, err
	}

	return u, nil
}
