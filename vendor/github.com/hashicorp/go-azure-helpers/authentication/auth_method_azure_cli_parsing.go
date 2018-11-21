package authentication

import (
	"fmt"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure/cli"
	"github.com/hashicorp/go-multierror"
)

type azureCliParsingAuth struct {
	profile *azureCLIProfile
}

func (a azureCliParsingAuth) build(b Builder) (authMethod, error) {
	auth := azureCliParsingAuth{
		profile: &azureCLIProfile{
			clientId:       b.ClientID,
			environment:    b.Environment,
			subscriptionId: b.SubscriptionID,
			tenantId:       b.TenantID,
		},
	}
	profilePath, err := cli.ProfilePath()
	if err != nil {
		return nil, fmt.Errorf("Error loading the Profile Path from the Azure CLI: %+v", err)
	}

	profile, err := cli.LoadProfile(profilePath)
	if err != nil {
		return nil, fmt.Errorf("Azure CLI Authorization Profile was not found. Please ensure the Azure CLI is installed and then log-in with `az login`.")
	}

	auth.profile.profile = profile

	err = auth.profile.populateFields()
	if err != nil {
		return nil, err
	}

	err = auth.profile.populateClientIdAndAccessToken()
	if err != nil {
		return nil, fmt.Errorf("Error populating Access Tokens from the Azure CLI: %+v", err)
	}

	return auth, nil
}

func (a azureCliParsingAuth) isApplicable(b Builder) bool {
	return b.SupportsAzureCliCloudShellParsing
}

func (a azureCliParsingAuth) getAuthorizationToken(oauthConfig *adal.OAuthConfig, endpoint string) (*autorest.BearerAuthorizer, error) {
	if a.profile.usingCloudShell {
		// load the refreshed tokens from the CloudShell Azure CLI credentials
		err := a.profile.populateClientIdAndAccessToken()
		if err != nil {
			return nil, fmt.Errorf("Error loading the refreshed CloudShell tokens: %+v", err)
		}
	}

	spt, err := adal.NewServicePrincipalTokenFromManualToken(*oauthConfig, a.profile.clientId, endpoint, *a.profile.accessToken)
	if err != nil {
		return nil, err
	}

	err = spt.Refresh()

	if err != nil {
		return nil, fmt.Errorf("Error refreshing Service Principal Token: %+v", err)
	}

	auth := autorest.NewBearerAuthorizer(spt)
	return auth, nil
}

func (a azureCliParsingAuth) name() string {
	return "Parsing credentials from the Azure CLI"
}

func (a azureCliParsingAuth) populateConfig(c *Config) error {
	c.ClientID = a.profile.clientId
	c.Environment = a.profile.environment
	c.SubscriptionID = a.profile.subscriptionId
	c.TenantID = a.profile.tenantId
	return nil
}

func (a azureCliParsingAuth) validate() error {
	var err *multierror.Error

	errorMessageFmt := "A %s was not found in your Azure CLI Credentials.\n\nPlease login to the Azure CLI again via `az login`"

	if a.profile == nil {
		return fmt.Errorf("Azure CLI Profile is nil - this is an internal error and should be reported.")
	}

	if a.profile.accessToken == nil {
		err = multierror.Append(err, fmt.Errorf(errorMessageFmt, "Access Token"))
	}

	if a.profile.clientId == "" {
		err = multierror.Append(err, fmt.Errorf(errorMessageFmt, "Client ID"))
	}

	if a.profile.subscriptionId == "" {
		err = multierror.Append(err, fmt.Errorf(errorMessageFmt, "Subscription ID"))
	}

	if a.profile.tenantId == "" {
		err = multierror.Append(err, fmt.Errorf(errorMessageFmt, "Tenant ID"))
	}

	return err.ErrorOrNil()
}
