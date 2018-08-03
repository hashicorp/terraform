package tfe

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"
)

// Compile-time proof of interface implementation.
var _ OAuthClients = (*oAuthClients)(nil)

// OAuthClients describes all the OAuth client related methods that the
// Terraform Enterprise API supports.
//
// TFE API docs:
// https://www.terraform.io/docs/enterprise/api/oauth-clients.html
type OAuthClients interface {
	// Create a VCS connection between an organization and a VCS provider.
	Create(ctx context.Context, organization string, options OAuthClientCreateOptions) (*OAuthClient, error)
}

// oAuthClients implements OAuthClients.
type oAuthClients struct {
	client *Client
}

// ServiceProviderType represents a VCS type.
type ServiceProviderType string

// List of available VCS types.
const (
	ServiceProviderBitbucket       ServiceProviderType = "bitbucket_hosted"
	ServiceProviderBitbucketServer ServiceProviderType = "bitbucket_server"
	ServiceProviderGithub          ServiceProviderType = "github"
	ServiceProviderGithubEE        ServiceProviderType = "github_enterprise"
	ServiceProviderGitlab          ServiceProviderType = "gitlab_hosted"
	ServiceProviderGitlabCE        ServiceProviderType = "gitlab_community_edition"
	ServiceProviderGitlabEE        ServiceProviderType = "gitlab_enterprise_edition"
)

// OAuthClient represents a connection between an organization and a VCS
// provider.
type OAuthClient struct {
	ID                  string              `jsonapi:"primary,oauth-clients"`
	APIURL              string              `jsonapi:"attr,api-url"`
	CallbackURL         string              `jsonapi:"attr,callback-url"`
	ConnectPath         string              `jsonapi:"attr,connect-path"`
	CreatedAt           time.Time           `jsonapi:"attr,created-at,iso8601"`
	HTTPURL             string              `jsonapi:"attr,http-url"`
	Key                 string              `jsonapi:"attr,key"`
	RSAPublicKey        string              `jsonapi:"attr,rsa-public-key"`
	ServiceProvider     ServiceProviderType `jsonapi:"attr,service-provider"`
	ServiceProviderName string              `jsonapi:"attr,service-provider-display-name"`

	// Relations
	Organization *Organization `jsonapi:"relation,organization"`
	OAuthToken   []*OAuthToken `jsonapi:"relation,oauth-token"`
}

// OAuthClientCreateOptions represents the options for creating an OAuth client.
type OAuthClientCreateOptions struct {
	// For internal use only!
	ID string `jsonapi:"primary,oauth-clients"`

	// The base URL of your VCS provider's API.
	APIURL *string `jsonapi:"attr,api-url"`

	// The homepage of your VCS provider.
	HTTPURL *string `jsonapi:"attr,http-url"`

	// The key you were given by your VCS provider.
	Key *string `jsonapi:"attr,key"`

	// The secret you were given by your VCS provider.
	Secret *string `jsonapi:"attr,secret"`

	// The VCS provider being connected with.
	ServiceProvider *ServiceProviderType `jsonapi:"attr,service-provider"`
}

func (o OAuthClientCreateOptions) valid() error {
	if !validString(o.APIURL) {
		return errors.New("APIURL is required")
	}
	if !validString(o.HTTPURL) {
		return errors.New("HTTPURL is required")
	}
	if !validString(o.Key) {
		return errors.New("Key is required")
	}
	if !validString(o.Secret) {
		return errors.New("Secret is required")
	}
	if o.ServiceProvider == nil {
		return errors.New("ServiceProvider is required")
	}
	return nil
}

// Create a VCS connection between an organization and a VCS provider.
func (s *oAuthClients) Create(ctx context.Context, organization string, options OAuthClientCreateOptions) (*OAuthClient, error) {
	if !validStringID(&organization) {
		return nil, errors.New("Invalid value for organization")
	}
	if err := options.valid(); err != nil {
		return nil, err
	}

	// Make sure we don't send a user provided ID.
	options.ID = ""

	u := fmt.Sprintf("organizations/%s/oauth-clients", url.QueryEscape(organization))
	req, err := s.client.newRequest("POST", u, &options)
	if err != nil {
		return nil, err
	}

	oc := &OAuthClient{}
	err = s.client.do(ctx, req, oc)
	if err != nil {
		return nil, err
	}

	return oc, nil
}
