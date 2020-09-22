package authentication

import (
	"fmt"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/hashicorp/go-multierror"
)

type servicePrincipalClientSecretMultiTenantAuth struct {
	clientId           string
	clientSecret       string
	subscriptionId     string
	tenantId           string
	auxiliaryTenantIDs []string
}

func (a servicePrincipalClientSecretMultiTenantAuth) build(b Builder) (authMethod, error) {
	method := servicePrincipalClientSecretMultiTenantAuth{
		clientId:           b.ClientID,
		clientSecret:       b.ClientSecret,
		subscriptionId:     b.SubscriptionID,
		tenantId:           b.TenantID,
		auxiliaryTenantIDs: b.AuxiliaryTenantIDs,
	}
	return method, nil
}

func (a servicePrincipalClientSecretMultiTenantAuth) isApplicable(b Builder) bool {
	return b.SupportsClientSecretAuth && b.ClientSecret != "" && b.SupportsAuxiliaryTenants && (len(b.AuxiliaryTenantIDs) > 0)
}

func (a servicePrincipalClientSecretMultiTenantAuth) name() string {
	return "Multi Tenant Service Principal / Client Secret"
}

func (a servicePrincipalClientSecretMultiTenantAuth) getAuthorizationToken(sender autorest.Sender, oauth *OAuthConfig, endpoint string) (autorest.Authorizer, error) {
	if oauth.MultiTenantOauth == nil {
		return nil, fmt.Errorf("Error getting Authorization Token for client cert: an MultiTenantOauth token wasn't configured correctly; please file a bug with more details")
	}

	spt, err := adal.NewMultiTenantServicePrincipalToken(*oauth.MultiTenantOauth, a.clientId, a.clientSecret, endpoint)
	if err != nil {
		return nil, err
	}

	spt.PrimaryToken.SetSender(sender)
	for _, t := range spt.AuxiliaryTokens {
		t.SetSender(sender)
	}

	auth := autorest.NewMultiTenantServicePrincipalTokenAuthorizer(spt)
	return auth, nil
}

func (a servicePrincipalClientSecretMultiTenantAuth) populateConfig(c *Config) error {
	c.AuthenticatedAsAServicePrincipal = true
	c.GetAuthenticatedObjectID = buildServicePrincipalObjectIDFunc(c)
	return nil
}

func (a servicePrincipalClientSecretMultiTenantAuth) validate() error {
	var err *multierror.Error

	fmtErrorMessage := "A %s must be configured when authenticating as a Service Principal using a Multi Tenant Client Secret."

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
	if len(a.auxiliaryTenantIDs) == 0 {
		err = multierror.Append(err, fmt.Errorf(fmtErrorMessage, "Auxiliary Tenant IDs"))
	}
	return err.ErrorOrNil()
}
