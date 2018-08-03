package tfe

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"
)

// Compile-time proof of interface implementation.
var _ OAuthTokens = (*oAuthTokens)(nil)

// OAuthTokens describes all the OAuth token related methods that the
// Terraform Enterprise API supports.
//
// TFE API docs:
// https://www.terraform.io/docs/enterprise/api/oauth-tokens.html
type OAuthTokens interface {
	// List all the OAuth Tokens for a given organization.
	List(ctx context.Context, organization string) ([]*OAuthToken, error)
}

// oAuthTokens implements OAuthTokens.
type oAuthTokens struct {
	client *Client
}

// OAuthToken represents a VCS configuration including the associated
// OAuth token
type OAuthToken struct {
	ID                  string    `jsonapi:"primary,oauth-tokens"`
	UID                 string    `jsonapi:"attr,uid"`
	CreatedAt           time.Time `jsonapi:"attr,created-at,iso8601"`
	HasSSHKey           bool      `jsonapi:"attr,has-ssh-key"`
	ServiceProviderUser string    `jsonapi:"attr,service-provider-user"`

	// Relations
	OAuthClient *OAuthClient `jsonapi:"relation,oauth-client"`
}

// List all the OAuth Tokens for a given organization.
func (s *oAuthTokens) List(ctx context.Context, organization string) ([]*OAuthToken, error) {
	if !validStringID(&organization) {
		return nil, errors.New("Invalid value for organization")
	}

	u := fmt.Sprintf("organizations/%s/oauth-tokens", url.QueryEscape(organization))
	req, err := s.client.newRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	var ots []*OAuthToken
	err = s.client.do(ctx, req, &ots)
	if err != nil {
		return nil, err
	}

	return ots, nil
}
