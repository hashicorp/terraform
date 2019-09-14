package authentication

import (
	"fmt"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/hashicorp/go-multierror"
)

type servicePrincipalClientSecretAuth struct {
	clientId       string
	clientSecret   string
	subscriptionId string
	tenantId       string
}

func (a servicePrincipalClientSecretAuth) build(b Builder) (authMethod, error) {
	method := servicePrincipalClientSecretAuth{
		clientId:       b.ClientID,
		clientSecret:   b.ClientSecret,
		subscriptionId: b.SubscriptionID,
		tenantId:       b.TenantID,
	}
	return method, nil
}

func (a servicePrincipalClientSecretAuth) isApplicable(b Builder) bool {
	return b.SupportsClientSecretAuth && b.ClientSecret != ""
}

func (a servicePrincipalClientSecretAuth) name() string {
	return "Service Principal / Client Secret"
}

func (a servicePrincipalClientSecretAuth) getAuthorizationToken(oauthConfig *adal.OAuthConfig, endpoint string) (*autorest.BearerAuthorizer, error) {
	spt, err := adal.NewServicePrincipalToken(*oauthConfig, a.clientId, a.clientSecret, endpoint)
	if err != nil {
		return nil, err
	}

	auth := autorest.NewBearerAuthorizer(spt)
	return auth, nil
}

func (a servicePrincipalClientSecretAuth) populateConfig(c *Config) error {
	c.AuthenticatedAsAServicePrincipal = true
	return nil
}

func (a servicePrincipalClientSecretAuth) validate() error {
	var err *multierror.Error

	fmtErrorMessage := "A %s must be configured when authenticating as a Service Principal using a Client Secret."

	if a.subscriptionId == "" {
		err = multierror.Append(err, fmt.Errorf(fmtErrorMessage, "Subscription ID"))
	}
	if a.clientId == "" {
		err = multierror.Append(err, fmt.Errorf(fmtErrorMessage, "Client ID"))
	}
	if a.clientSecret == "" {
		err = multierror.Append(err, fmt.Errorf(fmtErrorMessage, "Client Secret"))
	}
	if a.tenantId == "" {
		err = multierror.Append(err, fmt.Errorf(fmtErrorMessage, "Tenant ID"))
	}

	return err.ErrorOrNil()
}
