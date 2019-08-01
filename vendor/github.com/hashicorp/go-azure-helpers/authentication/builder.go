package authentication

import (
	"fmt"
	"log"
)

// Builder supports all of the possible Authentication values and feature toggles
// required to build a working Config for Authentication purposes.
type Builder struct {
	// Core
	ClientID       string
	SubscriptionID string
	TenantID       string
	Environment    string

	// The custom Resource Manager Endpoint which should be used
	// only applicable for Azure Stack at this time.
	CustomResourceManagerEndpoint string

	// Azure CLI Tokens Auth
	SupportsAzureCliToken bool

	// Managed Service Identity Auth
	SupportsManagedServiceIdentity bool
	MsiEndpoint                    string

	// Service Principal (Client Cert) Auth
	SupportsClientCertAuth bool
	ClientCertPath         string
	ClientCertPassword     string

	// Service Principal (Client Secret) Auth
	SupportsClientSecretAuth bool
	ClientSecret             string
	ClientSecretDocsLink     string
}

// Build takes the configuration from the Builder and builds up a validated Config
// for authenticating with Azure
func (b Builder) Build() (*Config, error) {
	config := Config{
		ClientID:                      b.ClientID,
		SubscriptionID:                b.SubscriptionID,
		TenantID:                      b.TenantID,
		Environment:                   b.Environment,
		CustomResourceManagerEndpoint: b.CustomResourceManagerEndpoint,
	}

	// NOTE: the ordering here is important
	// since the Azure CLI Parsing should always be the last thing checked
	supportedAuthenticationMethods := []authMethod{
		servicePrincipalClientCertificateAuth{},
		servicePrincipalClientSecretAuth{},
		managedServiceIdentityAuth{},
		azureCliTokenAuth{},
	}

	for _, method := range supportedAuthenticationMethods {
		name := method.name()
		log.Printf("Testing if %s is applicable for Authentication..", name)
		if method.isApplicable(b) {
			log.Printf("Using %s for Authentication", name)
			auth, err := method.build(b)
			if err != nil {
				return nil, err
			}

			// populate authentication specific fields on the Config
			// (e.g. is service principal, fields parsed from the azure cli)
			err = auth.populateConfig(&config)
			if err != nil {
				return nil, err
			}

			config.authMethod = auth
			return config.validate()
		}
	}

	return nil, fmt.Errorf("No supported authentication methods were found!")
}
