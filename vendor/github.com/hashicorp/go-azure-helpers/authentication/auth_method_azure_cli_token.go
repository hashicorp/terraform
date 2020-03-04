package authentication

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure/cli"
	"github.com/hashicorp/go-multierror"
)

type azureCliTokenAuth struct {
	profile                      *azureCLIProfile
	servicePrincipalAuthDocsLink string
}

func (a azureCliTokenAuth) build(b Builder) (authMethod, error) {
	auth := azureCliTokenAuth{
		profile: &azureCLIProfile{
			clientId:       b.ClientID,
			environment:    b.Environment,
			subscriptionId: b.SubscriptionID,
			tenantId:       b.TenantID,
		},
		servicePrincipalAuthDocsLink: b.ClientSecretDocsLink,
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

	// Authenticating as a Service Principal doesn't return all of the information we need for authentication purposes
	// as such Service Principal authentication is supported using the specific auth method
	if authenticatedAsAUser := auth.profile.verifyAuthenticatedAsAUser(); !authenticatedAsAUser {
		return nil, fmt.Errorf(`Authenticating using the Azure CLI is only supported as a User (not a Service Principal).

To authenticate to Azure using a Service Principal, you can use the separate 'Authenticate using a Service Principal'
auth method - instructions for which can be found here: %s

Alternatively you can authenticate using the Azure CLI by using a User Account.`, auth.servicePrincipalAuthDocsLink)
	}

	err = auth.profile.populateFields()
	if err != nil {
		return nil, fmt.Errorf("Error retrieving the Profile from the Azure CLI: %s Please re-authenticate using `az login`.", err)
	}

	err = auth.profile.populateClientId()
	if err != nil {
		return nil, fmt.Errorf("Error populating Client ID from the Azure CLI: %+v", err)
	}

	return auth, nil
}

func (a azureCliTokenAuth) isApplicable(b Builder) bool {
	return b.SupportsAzureCliToken
}

func (a azureCliTokenAuth) getAuthorizationToken(sender autorest.Sender, oauth *OAuthConfig, endpoint string) (autorest.Authorizer, error) {
	if oauth.OAuth == nil {
		return nil, fmt.Errorf("Error getting Authorization Token for cli auth: an OAuth token wasn't configured correctly; please file a bug with more details")
	}

	// the Azure CLI appears to cache these, so to maintain compatibility with the interface this method is intentionally not on the pointer
	token, err := obtainAuthorizationToken(endpoint, a.profile.subscriptionId)
	if err != nil {
		return nil, fmt.Errorf("Error obtaining Authorization Token from the Azure CLI: %s", err)
	}

	adalToken, err := token.ToADALToken()
	if err != nil {
		return nil, fmt.Errorf("Error converting Authorization Token to an ADAL Token: %s", err)
	}

	spt, err := adal.NewServicePrincipalTokenFromManualToken(*oauth.OAuth, a.profile.clientId, endpoint, adalToken)
	if err != nil {
		return nil, err
	}

	var refreshFunc adal.TokenRefresh = func(ctx context.Context, resource string) (*adal.Token, error) {
		token, err := obtainAuthorizationToken(resource, a.profile.subscriptionId)
		if err != nil {
			return nil, err
		}

		adalToken, err := token.ToADALToken()
		if err != nil {
			return nil, err
		}

		return &adalToken, nil
	}
	spt.SetCustomRefreshFunc(refreshFunc)

	auth := autorest.NewBearerAuthorizer(spt)
	return auth, nil
}

func (a azureCliTokenAuth) name() string {
	return "Obtaining a token from the Azure CLI"
}

func (a azureCliTokenAuth) populateConfig(c *Config) error {
	c.ClientID = a.profile.clientId
	c.TenantID = a.profile.tenantId
	c.Environment = a.profile.environment
	c.SubscriptionID = a.profile.subscriptionId

	c.GetAuthenticatedObjectID = func(ctx context.Context) (string, error) {
		objectId, err := obtainAuthenticatedObjectID()
		if err != nil {
			return "", err
		}

		return objectId, nil
	}

	return nil
}

func (a azureCliTokenAuth) validate() error {
	var err *multierror.Error

	errorMessageFmt := "A %s was not found in your Azure CLI Credentials.\n\nPlease login to the Azure CLI again via `az login`"

	if a.profile == nil {
		return fmt.Errorf("Azure CLI Profile is nil - this is an internal error and should be reported.")
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

func obtainAuthenticatedObjectID() (string, error) {

	var json struct {
		ObjectId string `json:"objectId"`
	}

	err := jsonUnmarshalAzCmd(&json, "ad", "signed-in-user", "show", "-o=json")
	if err != nil {
		return "", fmt.Errorf("Error parsing json result from the Azure CLI: %v", err)
	}

	return json.ObjectId, nil
}

func obtainAuthorizationToken(endpoint string, subscriptionId string) (*cli.Token, error) {
	var token cli.Token
	err := jsonUnmarshalAzCmd(&token, "account", "get-access-token", "--resource", endpoint, "--subscription", subscriptionId, "-o=json")
	if err != nil {
		return nil, fmt.Errorf("Error parsing json result from the Azure CLI: %v", err)
	}

	return &token, nil
}

func jsonUnmarshalAzCmd(i interface{}, arg ...string) error {
	var stderr bytes.Buffer
	var stdout bytes.Buffer

	cmd := exec.Command("az", arg...)

	cmd.Stderr = &stderr
	cmd.Stdout = &stdout

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("Error launching Azure CLI: %+v", err)
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("Error waiting for the Azure CLI: %+v", err)
	}

	stdOutStr := stdout.String()
	stdErrStr := stderr.String()
	if stdErrStr != "" {
		return fmt.Errorf("Error retrieving running Azure CLI: %s", strings.TrimSpace(stdErrStr))
	}

	if err := json.Unmarshal([]byte(stdOutStr), &i); err != nil {
		return fmt.Errorf("Error unmarshaling the result of Azure CLI: %v", err)
	}

	return nil
}
