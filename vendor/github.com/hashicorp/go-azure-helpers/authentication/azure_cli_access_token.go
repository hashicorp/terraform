package authentication

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure/cli"
)

type azureCliAccessToken struct {
	ClientID     string
	AccessToken  *adal.Token
	IsCloudShell bool
}

func findValidAccessTokenForTenant(tokens []cli.Token, tenantId string) (*azureCliAccessToken, error) {
	for _, accessToken := range tokens {
		token, err := accessToken.ToADALToken()
		if err != nil {
			return nil, fmt.Errorf("[DEBUG] Error converting access token to token: %+v", err)
		}

		expirationDate, err := cli.ParseExpirationDate(accessToken.ExpiresOn)
		if err != nil {
			return nil, fmt.Errorf("Error parsing expiration date: %q", accessToken.ExpiresOn)
		}

		if expirationDate.UTC().Before(time.Now().UTC()) {
			log.Printf("[DEBUG] Token %q has expired", token.AccessToken)
			continue
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
			IsCloudShell: accessToken.RefreshToken == "",
		}
		return &validAccessToken, nil
	}

	return nil, fmt.Errorf("No Access Token was found for the Tenant ID %q", tenantId)
}
