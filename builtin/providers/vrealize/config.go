package vrealize

import (
	"fmt"
	"log"
	"net/url"
	"time"

	"github.com/dghubble/sling"
	"github.com/sky-mah96/govrealize"
	"golang.org/x/oauth2"
)

// Config represents
type Config struct {
	User     string
	Password string
	Tenant   string
	Server   string
}

// apiToken represents
type apiToken struct {
	Expires time.Time `json:"expires,omitempty"`
	ID      string    `json:"id,omitempty"`
	Tenant  string    `json:"tenant,omitempty"`
}

// apiTokenRequest represents
type apiTokenRequest struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Tenant   string `json:"tenant,omitempty"`
}

// TokenSource represents
type TokenSource struct {
	AccessToken string
	Expiry      time.Time
}

// Token represents
func (t *TokenSource) Token() (*oauth2.Token, error) {
	token := &oauth2.Token{
		AccessToken: t.AccessToken,
		Expiry:      t.Expiry,
	}
	return token, nil
}

// Client returns a new client for accessing VMWare vRealize.
func (c *Config) Client() (*govrealize.Client, error) {
	u, err := url.Parse("https://" + c.Server + "/")

	if err != nil {
		return nil, fmt.Errorf("Error parse url: %s", err)
	}

	vrealizeBase := sling.New().Base(u.String()).Client(nil)

	path := "identity/api/tokens"

	body := &apiTokenRequest{
		Username: c.User,
		Password: c.Password,
		Tenant:   c.Tenant,
	}

	token := new(apiToken)

	_, err = vrealizeBase.New().Set("Accept", "application/json").Post(path).BodyJSON(body).ReceiveSuccess(token)

	if err != nil {
		log.Fatal(err)
	}

	tokenSource := &TokenSource{
		AccessToken: token.ID,
		Expiry:      token.Expires,
	}

	oauthClient := oauth2.NewClient(oauth2.NoContext, tokenSource)

	client := govrealize.NewClient(oauthClient)
	client.BaseURL = u
	client.Tenant = c.Tenant
	client.Username = c.User
	client.Password = c.Password

	if err != nil {
		return nil, fmt.Errorf("Error setting up client: %s", err)
	}

	log.Printf("[INFO] VMWare vRealize Client configured for URL: %s", u)

	return client, nil
}
