package authentication

import (
	"fmt"
	"log"
	"strings"

	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure/cli"
)

type azureCliAccessToken struct {
	ClientID     string
	AccessToken  *adal.Token
}

func findValidAccessTokenForTenant(tokens []cli.Token, tenantId string) (*azureCliAccessToken, error) {
	for _, accessToken := range tokens {
		token, err := accessToken.ToADALToken()
		if err != nil {
			return nil, fmt.Errorf("[DEBUG] Error converting access token to token: %+v", err)
		}

		if !strings.Contains(accessToken.Resource, "management") {
			log.Printf("[DEBUG] Resource %q isn't a management domain", accessToken.Resource)
			continue
		}

		if !strings.HasSuffix(accessToken.Authority, tenantId) {
			log.Printf("[DEBUG] Resource %q isn't for the correct Tenant", accessToken.Resource)
			continue
		}

		validAccessToken := azureCliAccessToken{
			ClientID:     accessToken.ClientID,
			AccessToken:  &token,
		}
		return &validAccessToken, nil
	}

	return nil, fmt.Errorf("No Access Token was found for the Tenant ID %q", tenantId)
}
