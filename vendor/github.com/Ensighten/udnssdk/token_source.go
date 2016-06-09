package udnssdk

import (
	"fmt"

	"github.com/Ensighten/udnssdk/passwordcredentials"
	"golang.org/x/oauth2"
)

func NewConfig(username, password, BaseURL string) *passwordcredentials.Config {
	c := passwordcredentials.Config{}
	c.Username = username
	c.Password = password
	c.Endpoint = Endpoint(BaseURL)
	return &c
}

func Endpoint(BaseURL string) oauth2.Endpoint {
	return oauth2.Endpoint{
		TokenURL: TokenURL(BaseURL),
	}
}

func TokenURL(BaseURL string) string {
	return fmt.Sprintf("%s/%s/authorization/token", BaseURL, apiVersion)
}
