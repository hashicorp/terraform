package google

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"code.google.com/p/google-api-go-client/compute/v1"
	// oauth2 "github.com/rasa/oauth2-fork-b3f9a68"
	"github.com/golang/oauth2"

	// oauth2 "github.com/rasa/oauth2-fork-b3f9a68/google"
	"github.com/golang/oauth2/google"
)

const clientScopes string = "https://www.googleapis.com/auth/compute"

// Config is the configuration structure used to instantiate the Google
// provider.
type Config struct {
	AccountFile string
	Project     string
	Region      string

	clientCompute *compute.Service
}

func (c *Config) loadAndValidate() error {
	var account accountFile

	// TODO: validation that it isn't blank
	if c.AccountFile == "" {
		c.AccountFile = os.Getenv("GOOGLE_ACCOUNT_FILE")
	}
	if c.Project == "" {
		c.Project = os.Getenv("GOOGLE_PROJECT")
	}
	if c.Region == "" {
		c.Region = os.Getenv("GOOGLE_REGION")
	}

	var f *oauth2.Options
	var err error

	if c.AccountFile != "" {
		if err := loadJSON(&account, c.AccountFile); err != nil {
			return fmt.Errorf(
				"Error loading account file '%s': %s",
				c.AccountFile,
				err)
		}

		// Get the token for use in our requests
		log.Printf("[INFO] Requesting Google token...")
		log.Printf("[INFO]   -- Email: %s", account.ClientEmail)
		log.Printf("[INFO]   -- Scopes: %s", clientScopes)
		log.Printf("[INFO]   -- Private Key Length: %d", len(account.PrivateKey))

		f, err = oauth2.New(
			oauth2.JWTClient(account.ClientEmail, []byte(account.PrivateKey)),
			oauth2.Scope(clientScopes),
			google.JWTEndpoint())

	} else {
		log.Printf("[INFO] Requesting Google token via GCE Service Role...")
		f, err = oauth2.New(google.ComputeEngineAccount(""))

	}

	if err != nil {
		return fmt.Errorf("Error retrieving auth token: %s", err)
	}

	log.Printf("[INFO] Instantiating GCE client...")
	c.clientCompute, err = compute.New(&http.Client{Transport: f.NewTransport()})
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

func loadJSON(result interface{}, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	return dec.Decode(result)
}
