package google

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"

	// TODO(dcunnin): Use version code from version.go
	// "github.com/hashicorp/terraform"
	"github.com/mitchellh/go-homedir"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jwt"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/container/v1"
	"google.golang.org/api/dns/v1"
	"google.golang.org/api/storage/v1"
)

// Config is the configuration structure used to instantiate the Google
// provider.
type Config struct {
	AccountFile string
	Project     string
	Region      string

	clientCompute   *compute.Service
	clientContainer *container.Service
	clientDns       *dns.Service
	clientStorage   *storage.Service
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

	var client *http.Client

	if c.AccountFile != "" {
		var path string
		path, err := homedir.Expand(c.AccountFile)
		if err != nil {
			return fmt.Errorf("Error expanding path %q: %q",
				c.AccountFile, err.Error())
		}

		if err := loadJSON(&account, path); err != nil {
			return fmt.Errorf(
				"Error loading account file '%s': %s",
				c.AccountFile,
				err)
		}

		clientScopes := []string{
			"https://www.googleapis.com/auth/compute",
			"https://www.googleapis.com/auth/cloud-platform",
			"https://www.googleapis.com/auth/ndev.clouddns.readwrite",
			"https://www.googleapis.com/auth/devstorage.full_control",
		}

		// Get the token for use in our requests
		log.Printf("[INFO] Requesting Google token...")
		log.Printf("[INFO]   -- Email: %s", account.ClientEmail)
		log.Printf("[INFO]   -- Scopes: %s", clientScopes)
		log.Printf("[INFO]   -- Private Key Length: %d", len(account.PrivateKey))

		conf := jwt.Config{
			Email:      account.ClientEmail,
			PrivateKey: []byte(account.PrivateKey),
			Scopes:     clientScopes,
			TokenURL:   "https://accounts.google.com/o/oauth2/token",
		}

		// Initiate an http.Client. The following GET request will be
		// authorized and authenticated on the behalf of
		// your service account.
		client = conf.Client(oauth2.NoContext)

	} else {
		log.Printf("[INFO] Requesting Google token via GCE Service Role...")
		client = &http.Client{
			Transport: &oauth2.Transport{
				// Fetch from Google Compute Engine's metadata server to retrieve
				// an access token for the provided account.
				// If no account is specified, "default" is used.
				Source: google.ComputeTokenSource(""),
			},
		}

	}

	// Build UserAgent
	versionString := "0.0.0"
	// TODO(dcunnin): Use Terraform's version code from version.go
	// versionString := main.Version
	// if main.VersionPrerelease != "" {
	// 	versionString = fmt.Sprintf("%s-%s", versionString, main.VersionPrerelease)
	// }
	userAgent := fmt.Sprintf(
		"(%s %s) Terraform/%s", runtime.GOOS, runtime.GOARCH, versionString)

	var err error

	log.Printf("[INFO] Instantiating GCE client...")
	c.clientCompute, err = compute.New(client)
	if err != nil {
		return err
	}
	c.clientCompute.UserAgent = userAgent

	log.Printf("[INFO] Instantiating GKE client...")
	c.clientContainer, err = container.New(client)
	if err != nil {
		return err
	}
	c.clientContainer.UserAgent = userAgent

	log.Printf("[INFO] Instantiating Google Cloud DNS client...")
	c.clientDns, err = dns.New(client)
	if err != nil {
		return err
	}
	c.clientDns.UserAgent = userAgent

	log.Printf("[INFO] Instantiating Google Storage Client...")
	c.clientStorage, err = storage.New(client)
	if err != nil {
		return err
	}
	c.clientStorage.UserAgent = userAgent

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
