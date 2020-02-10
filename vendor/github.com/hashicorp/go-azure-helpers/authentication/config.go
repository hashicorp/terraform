package authentication

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
)

// Config is the configuration structure used to instantiate a
// new Azure management client.
type Config struct {
	ClientID           string
	SubscriptionID     string
	TenantID           string
	AuxiliaryTenantIDs []string
	Environment        string

	GetAuthenticatedObjectID         func(context.Context) (string, error)
	AuthenticatedAsAServicePrincipal bool

	// A Custom Resource Manager Endpoint
	// at this time this should only be applicable for Azure Stack.
	CustomResourceManagerEndpoint string

	authMethod authMethod
}

type OAuthConfig struct {
	OAuth            *adal.OAuthConfig
	MultiTenantOauth *adal.MultiTenantOAuthConfig
}

// GetAuthorizationToken returns an authorization token for the authentication method defined in the Config
func (c Config) GetOAuthConfig(activeDirectoryEndpoint string) (*adal.OAuthConfig, error) {
	log.Printf("Getting OAuth config for endpoint %s with  tenant %s", activeDirectoryEndpoint, c.TenantID)

	// fix for ADFS environments, if the login endpoint ends in `/adfs` it's an adfs environment
	// the login endpoint ends up residing in `ActiveDirectoryEndpoint`
	oAuthTenant := c.TenantID
	if strings.HasSuffix(strings.ToLower(activeDirectoryEndpoint), "/adfs") {
		log.Printf("[DEBUG] ADFS environment detected - overriding Tenant ID to `adfs`!")
		oAuthTenant = "adfs"
	}

	oauth, err := adal.NewOAuthConfig(activeDirectoryEndpoint, oAuthTenant)
	if err != nil {
		return nil, err
	}

	// OAuthConfigForTenant returns a pointer, which can be nil.
	if oauth == nil {
		return nil, fmt.Errorf("Unable to configure OAuthConfig for tenant %s", c.TenantID)
	}

	return oauth, nil
}

// GetMultiTenantOAuthConfig returns a multi-tenant authorization token for the authentication method defined in the Config
func (c Config) GetMultiTenantOAuthConfig(activeDirectoryEndpoint string) (*adal.MultiTenantOAuthConfig, error) {
	log.Printf("Getting multi OAuth config for endpoint %s with  tenant %s (aux tenants: %v)", activeDirectoryEndpoint, c.TenantID, c.AuxiliaryTenantIDs)
	oauth, err := adal.NewMultiTenantOAuthConfig(activeDirectoryEndpoint, c.TenantID, c.AuxiliaryTenantIDs, adal.OAuthOptions{})
	if err != nil {
		return nil, err
	}

	// OAuthConfigForTenant returns a pointer, which can be nil.
	if oauth == nil {
		return nil, fmt.Errorf("Unable to configure OAuthConfig for tenant %s (auxiliary tenants %v)", c.TenantID, c.AuxiliaryTenantIDs)
	}

	return &oauth, nil
}

// BuildOAuthConfig builds the authorization configuration for the specified Active Directory Endpoint
func (c Config) BuildOAuthConfig(activeDirectoryEndpoint string) (*OAuthConfig, error) {
	multiAuth := OAuthConfig{}
	var err error

	multiAuth.OAuth, err = c.GetOAuthConfig(activeDirectoryEndpoint)
	if err != nil {
		return nil, err
	}

	if len(c.AuxiliaryTenantIDs) > 0 {
		multiAuth.MultiTenantOauth, err = c.GetMultiTenantOAuthConfig(activeDirectoryEndpoint)
		if err != nil {
			return nil, err
		}
	}

	return &multiAuth, nil
}

// BearerAuthorizerCallback returns a BearerAuthorizer valid only for the Primary Tenant
// this signs a request using the AccessToken returned from the primary Resource Manager authorizer
func (c Config) BearerAuthorizerCallback(sender autorest.Sender, oauthConfig *OAuthConfig) *autorest.BearerAuthorizerCallback {
	return autorest.NewBearerAuthorizerCallback(sender, func(tenantID, resource string) (*autorest.BearerAuthorizer, error) {
		// a BearerAuthorizer is only valid for the primary tenant
		newAuthConfig := &OAuthConfig{
			OAuth: oauthConfig.OAuth,
		}

		storageSpt, err := c.GetAuthorizationToken(sender, newAuthConfig, resource)
		if err != nil {
			return nil, err
		}

		cast, ok := storageSpt.(*autorest.BearerAuthorizer)
		if !ok {
			return nil, fmt.Errorf("Error converting %+v to a BearerAuthorizer", storageSpt)
		}

		return cast, nil
	})
}

// GetAuthorizationToken returns an authorization token for the authentication method defined in the Config
func (c Config) GetAuthorizationToken(sender autorest.Sender, oauth *OAuthConfig, endpoint string) (autorest.Authorizer, error) {
	return c.authMethod.getAuthorizationToken(sender, oauth, endpoint)
}
