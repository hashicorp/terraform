package authentication

import (
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
)

// Config is the configuration structure used to instantiate a
// new Azure management client.
type Config struct {
	ClientID                         string
	SubscriptionID                   string
	TenantID                         string
	Environment                      string
	AuthenticatedAsAServicePrincipal bool

	// A Custom Resource Manager Endpoint
	// at this time this should only be applicable for Azure Stack.
	CustomResourceManagerEndpoint string

	authMethod authMethod
}

// GetAuthorizationToken returns an authorization token for the authentication method defined in the Config
func (c Config) GetAuthorizationToken(oauthConfig *adal.OAuthConfig, endpoint string) (*autorest.BearerAuthorizer, error) {
	return c.authMethod.getAuthorizationToken(oauthConfig, endpoint)
}

func (c Config) validate() (*Config, error) {
	err := c.authMethod.validate()
	if err != nil {
		return nil, err
	}

	return &c, nil
}
