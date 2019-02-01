package authentication

import (
	"fmt"
	"strings"

	"github.com/Azure/go-autorest/autorest/azure/cli"
)

func (a *azureCLIProfile) populateSubscriptionID() error {
	subscriptionId, err := a.findDefaultSubscriptionId()
	if err != nil {
		return err
	}

	a.subscriptionId = subscriptionId
	return nil
}

func (a *azureCLIProfile) populateTenantID() error {
	subscription, err := a.findSubscription(a.subscriptionId)
	if err != nil {
		return err
	}

	a.tenantId = subscription.TenantID
	return nil
}

func (a *azureCLIProfile) populateClientId() error {
	// we can now pull out the ClientID and the Access Token to use from the Access Token
	tokensPath, err := cli.AccessTokensPath()
	if err != nil {
		return fmt.Errorf("Error loading the Tokens Path from the Azure CLI: %+v", err)
	}

	tokens, err := cli.LoadTokens(tokensPath)
	if err != nil {
		return fmt.Errorf("No Authorization Tokens were found - please ensure the Azure CLI is installed and then log-in with `az login`.")
	}

	validToken, err := findValidAccessTokenForTenant(tokens, a.tenantId)
	if err != nil {
		return fmt.Errorf("No Authorization Tokens were found - please re-authenticate using `az login`.")
	}

	token := *validToken
	a.clientId = token.ClientID

	return nil
}

func (a *azureCLIProfile) populateEnvironment() error {
	subscription, err := a.findSubscription(a.subscriptionId)
	if err != nil {
		return err
	}

	a.environment = normalizeEnvironmentName(subscription.EnvironmentName)
	return nil
}

func (a azureCLIProfile) findDefaultSubscriptionId() (string, error) {
	for _, subscription := range a.profile.Subscriptions {
		if subscription.IsDefault {
			return subscription.ID, nil
		}
	}

	return "", fmt.Errorf("No Subscription was Marked as Default in the Azure Profile.")
}

func (a azureCLIProfile) findSubscription(subscriptionId string) (*cli.Subscription, error) {
	for _, subscription := range a.profile.Subscriptions {
		if strings.EqualFold(subscription.ID, subscriptionId) {
			return &subscription, nil
		}
	}

	return nil, fmt.Errorf("Subscription %q was not found in your Azure CLI credentials. Please verify it exists in `az account list`.", subscriptionId)
}
