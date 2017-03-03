package udnssdk

import (
	"fmt"

	"golang.org/x/oauth2"

	oauthPassword "github.com/Ensighten/udnssdk/password"
)

// NewConfig creates a new *password.config for UltraDNS OAuth2
func NewConfig(username, password, BaseURL string) *oauthPassword.Config {
	c := oauthPassword.Config{}
	c.Username = username
	c.Password = password
	c.Endpoint = Endpoint(BaseURL)
	return &c
}

// Endpoint returns an oauth2.Endpoint for UltraDNS
func Endpoint(BaseURL string) oauth2.Endpoint {
	return oauth2.Endpoint{
		TokenURL: TokenURL(BaseURL),
	}
}

// TokenURL returns an OAuth2 TokenURL for UltraDNS
func TokenURL(BaseURL string) string {
	return fmt.Sprintf("%s/%s/authorization/token", BaseURL, apiVersion)
}
