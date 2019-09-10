package authentication

import (
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
)

type authMethod interface {
	build(b Builder) (authMethod, error)

	isApplicable(b Builder) bool

	getAuthorizationToken(oauthConfig *adal.OAuthConfig, endpoint string) (*autorest.BearerAuthorizer, error)

	name() string

	populateConfig(c *Config) error

	validate() error
}
