package google

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"runtime"
	"strings"

	"github.com/hashicorp/terraform/helper/logging"
	"github.com/hashicorp/terraform/helper/pathorcontents"
	"github.com/hashicorp/terraform/terraform"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jwt"
	"google.golang.org/api/bigquery/v2"
	"google.golang.org/api/cloudbilling/v1"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/container/v1"
	"google.golang.org/api/dns/v1"
	"google.golang.org/api/iam/v1"
	"google.golang.org/api/pubsub/v1"
	"google.golang.org/api/servicemanagement/v1"
	"google.golang.org/api/sqladmin/v1beta4"
	"google.golang.org/api/storage/v1"
)

// Config is the configuration structure used to instantiate the Google
// provider.
type Config struct {
	Credentials string
	Project     string
	Region      string

	clientBilling         *cloudbilling.Service
	clientCompute         *compute.Service
	clientContainer       *container.Service
	clientDns             *dns.Service
	clientPubsub          *pubsub.Service
	clientResourceManager *cloudresourcemanager.Service
	clientStorage         *storage.Service
	clientSqlAdmin        *sqladmin.Service
	clientIAM             *iam.Service
	clientServiceMan      *servicemanagement.APIService
	clientBigQuery        *bigquery.Service
}

func (c *Config) loadAndValidate() error {
	var account accountFile
	clientScopes := []string{
		"https://www.googleapis.com/auth/compute",
		"https://www.googleapis.com/auth/cloud-platform",
		"https://www.googleapis.com/auth/ndev.clouddns.readwrite",
		"https://www.googleapis.com/auth/devstorage.full_control",
	}

	var client *http.Client

	if c.Credentials != "" {
		contents, _, err := pathorcontents.Read(c.Credentials)
		if err != nil {
			return fmt.Errorf("Error loading credentials: %s", err)
		}

		// Assume account_file is a JSON string
		if err := parseJSON(&account, contents); err != nil {
			return fmt.Errorf("Error parsing credentials '%s': %s", contents, err)
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
		log.Printf("[INFO] Authenticating using DefaultClient")
		err := error(nil)
		client, err = google.DefaultClient(oauth2.NoContext, clientScopes...)
		if err != nil {
			return err
		}
	}

	client.Transport = logging.NewTransport("Google", client.Transport)

	versionString := terraform.VersionString()
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

	log.Printf("[INFO] Instantiating Google SqlAdmin Client...")
	c.clientSqlAdmin, err = sqladmin.New(client)
	if err != nil {
		return err
	}
	c.clientSqlAdmin.UserAgent = userAgent

	log.Printf("[INFO] Instantiating Google Pubsub Client...")
	c.clientPubsub, err = pubsub.New(client)
	if err != nil {
		return err
	}
	c.clientPubsub.UserAgent = userAgent

	log.Printf("[INFO] Instantiating Google Cloud ResourceManager Client...")
	c.clientResourceManager, err = cloudresourcemanager.New(client)
	if err != nil {
		return err
	}
	c.clientResourceManager.UserAgent = userAgent

	log.Printf("[INFO] Instantiating Google Cloud IAM Client...")
	c.clientIAM, err = iam.New(client)
	if err != nil {
		return err
	}
	c.clientIAM.UserAgent = userAgent

	log.Printf("[INFO] Instantiating Google Cloud Service Management Client...")
	c.clientServiceMan, err = servicemanagement.New(client)
	if err != nil {
		return err
	}
	c.clientServiceMan.UserAgent = userAgent

	log.Printf("[INFO] Instantiating Google Cloud Billing Client...")
	c.clientBilling, err = cloudbilling.New(client)
	if err != nil {
		return err
	}
	c.clientBilling.UserAgent = userAgent

	log.Printf("[INFO] Instantiating Google Cloud BigQuery Client...")
	c.clientBigQuery, err = bigquery.New(client)
	if err != nil {
		return err
	}
	c.clientBigQuery.UserAgent = userAgent

	return nil
}

// accountFile represents the structure of the account file JSON file.
type accountFile struct {
	PrivateKeyId string `json:"private_key_id"`
	PrivateKey   string `json:"private_key"`
	ClientEmail  string `json:"client_email"`
	ClientId     string `json:"client_id"`
}

func parseJSON(result interface{}, contents string) error {
	r := strings.NewReader(contents)
	dec := json.NewDecoder(r)

	return dec.Decode(result)
}
