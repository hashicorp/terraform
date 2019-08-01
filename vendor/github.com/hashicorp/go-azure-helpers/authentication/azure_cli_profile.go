package authentication

import (
	"strings"

	"github.com/Azure/go-autorest/autorest/azure/cli"
)

type azureCLIProfile struct {
	profile cli.Profile

	clientId        string
	environment     string
	subscriptionId  string
	tenantId        string
}

func (a *azureCLIProfile) populateFields() error {
	// ensure we know the Subscription ID - since it's needed for everything else
	if a.subscriptionId == "" {
		err := a.populateSubscriptionID()
		if err != nil {
			return err
		}
	}

	if a.tenantId == "" {
		// now we know the subscription ID, find the associated Tenant ID
		err := a.populateTenantID()
		if err != nil {
			return err
		}
	}

	// always pull the environment from the Azure CLI, since the Access Token's associated with it
	return a.populateEnvironment()
}

func (a *azureCLIProfile) verifyAuthenticatedAsAUser() bool {
	for _, subscription := range a.profile.Subscriptions {
		if subscription.User == nil {
			continue
		}

		authenticatedAsAUser := strings.EqualFold(subscription.User.Type, "user")
		if authenticatedAsAUser {
			return true
		}
	}

	return false
}