package authentication

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	"github.com/hashicorp/go-azure-helpers/sender"
)

func buildServicePrincipalObjectIDFunc(c *Config) func(ctx context.Context) (string, error) {
	return func(ctx context.Context) (string, error) {
		env, err := DetermineEnvironment(c.Environment)
		if err != nil {
			return "", err
		}

		s := sender.BuildSender("GoAzureHelpers")

		oauthConfig, err := c.BuildOAuthConfig(env.ActiveDirectoryEndpoint)
		if err != nil {
			return "", err
		}

		// Graph Endpoints
		graphEndpoint := env.GraphEndpoint
		graphAuth, err := c.GetAuthorizationToken(s, oauthConfig, env.GraphEndpoint)
		if err != nil {
			return "", err
		}

		client := graphrbac.NewServicePrincipalsClientWithBaseURI(graphEndpoint, c.TenantID)
		client.Authorizer = graphAuth
		client.Sender = s

		filter := fmt.Sprintf("appId eq '%s'", c.ClientID)
		listResult, listErr := client.List(ctx, filter)

		if listErr != nil {
			return "", fmt.Errorf("Error listing Service Principals: %#v", listErr)
		}

		if listResult.Values() == nil || len(listResult.Values()) != 1 || listResult.Values()[0].ObjectID == nil {
			return "", fmt.Errorf("Unexpected Service Principal query result: %#v", listResult.Values())
		}

		return *listResult.Values()[0].ObjectID, nil
	}
}
