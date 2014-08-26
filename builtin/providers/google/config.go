package google

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"code.google.com/p/goauth2/oauth"
	"code.google.com/p/goauth2/oauth/jwt"
	"code.google.com/p/google-api-go-client/compute/v1"
)

const clientScopes string = "https://www.googleapis.com/auth/compute"

// Config is the configuration structure used to instantiate the Google
// provider.
type Config struct {
	AccountFile       string
	ClientSecretsFile string
	Project           string
	Region            string

	clientCompute *compute.Service
}

func (c *Config) loadAndValidate() error {
	var account accountFile
	var secrets clientSecretsFile

	// TODO: validation that it isn't blank
	if c.AccountFile == "" {
		c.AccountFile = os.Getenv("GOOGLE_ACCOUNT_FILE")
	}
	if c.ClientSecretsFile == "" {
		c.ClientSecretsFile = os.Getenv("GOOGLE_CLIENT_FILE")
	}
	if c.Project == "" {
		c.Project = os.Getenv("GOOGLE_PROJECT")
	}
	if c.Region == "" {
		c.Region = os.Getenv("GOOGLE_REGION")
	}

	if err := loadJSON(&account, c.AccountFile); err != nil {
		return fmt.Errorf(
			"Error loading account file '%s': %s",
			c.AccountFile,
			err)
	}

	if err := loadJSON(&secrets, c.ClientSecretsFile); err != nil {
		return fmt.Errorf(
			"Error loading client secrets file '%s': %s",
			c.ClientSecretsFile,
			err)
	}

	// Get the token for use in our requests
	log.Printf("[INFO] Requesting Google token...")
	log.Printf("[INFO]   -- Email: %s", account.ClientEmail)
	log.Printf("[INFO]   -- Scopes: %s", clientScopes)
	log.Printf("[INFO]   -- Private Key Length: %d", len(account.PrivateKey))
	log.Printf("[INFO]   -- Token URL: %s", secrets.Web.TokenURI)
	jwtTok := jwt.NewToken(
		account.ClientEmail,
		clientScopes,
		[]byte(account.PrivateKey))
	jwtTok.ClaimSet.Aud = secrets.Web.TokenURI
	token, err := jwtTok.Assert(new(http.Client))
	if err != nil {
		return fmt.Errorf("Error retrieving auth token: %s", err)
	}

	// Instantiate the transport to communicate to Google
	transport := &oauth.Transport{
		Config: &oauth.Config{
			ClientId: account.ClientId,
			Scope:    clientScopes,
			TokenURL: secrets.Web.TokenURI,
			AuthURL:  secrets.Web.AuthURI,
		},
		Token: token,
	}

	log.Printf("[INFO] Instantiating GCE client...")
	c.clientCompute, err = compute.New(transport.Client())
	if err != nil {
		return err
	}

	return nil
}

// accountFile represents the structure of the account file JSON file.
type accountFile struct {
	PrivateKeyId string `json:"private_key_id"`
	PrivateKey   string `json:"private_key"`
	ClientEmail  string `json:"client_email"`
	ClientId     string `json:"client_id"`
}

// clientSecretsFile represents the structure of the client secrets JSON file.
type clientSecretsFile struct {
	Web struct {
		AuthURI     string `json:"auth_uri"`
		ClientEmail string `json:"client_email"`
		ClientId    string `json:"client_id"`
		TokenURI    string `json:"token_uri"`
	}
}

func loadJSON(result interface{}, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	return dec.Decode(result)
}
