// Package gcs implements remote storage of state on Google Cloud Storage (GCS).
package gcs

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/helper/logging"
	"github.com/hashicorp/terraform/helper/pathorcontents"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/httpclient"

	"cloud.google.com/go/storage"
	"golang.org/x/oauth2"
	googleoauth "golang.org/x/oauth2/google"
	"google.golang.org/api/option"
)

// Backend implements "backend".Backend for GCS.
// Input(), Validate() and Configure() are implemented by embedding *schema.Backend.
// State(), DeleteState() and States() are implemented explicitly.
type Backend struct {
	*schema.Backend

	tokenSource    oauth2.TokenSource
	storageClient  *storage.Client
	storageContext context.Context

	bucketName       string
	prefix           string
	defaultStateFile string
	credentials      string
	accessToken      string

	encryptionKey []byte
}

func New() backend.Backend {
	b := &Backend{}
	b.Backend = &schema.Backend{
		ConfigureFunc: b.configure,
		Schema: map[string]*schema.Schema{
			"bucket": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the Google Cloud Storage bucket",
			},

			"path": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Path of the default state file",
				Deprecated:  "Use the \"prefix\" option instead",
			},

			"prefix": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The directory where state files will be saved inside the bucket",
			},

			"credentials": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Google Cloud JSON Account Key",
			},

			"access_token": {
				Type:          schema.TypeString,
				Optional:      true,
				Description:   "A temporary Google Cloud OAuth 2.0 access token obtained from the Google Authorization server.",
				ConflictsWith: []string{"credentials"},
			},

			"encryption_key": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A 32 byte base64 encoded 'customer supplied encryption key' used to encrypt all state.",
				Default:     "",
			},

			"project": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Google Cloud Project ID",
				Default:     "",
				Removed:     "Please remove this attribute. It is not used since the backend no longer creates the bucket if it does not yet exist.",
			},

			"region": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Region / location in which to create the bucket",
				Default:     "",
				Removed:     "Please remove this attribute. It is not used since the backend no longer creates the bucket if it does not yet exist.",
			},
		},
	}

	return b
}

func (b *Backend) configure(ctx context.Context) error {
	if b.storageClient != nil {
		return nil
	}

	// ctx is a background context with the backend config added.
	// Since no context is passed to remoteClient.Get(), .Lock(), etc. but
	// one is required for calling the GCP API, we're holding on to this
	// context here and re-use it later.
	b.storageContext = ctx

	data := schema.FromContextBackendConfig(b.storageContext)

	b.bucketName = data.Get("bucket").(string)
	b.prefix = strings.TrimLeft(data.Get("prefix").(string), "/")
	if b.prefix != "" && !strings.HasSuffix(b.prefix, "/") {
		b.prefix = b.prefix + "/"
	}

	b.defaultStateFile = strings.TrimLeft(data.Get("path").(string), "/")

	var opts []option.ClientOption

	if creds := data.Get("credentials"); creds != nil {
		b.credentials = creds.(string)
	}
	if b.credentials == "" {
		b.credentials = os.Getenv("GOOGLE_CREDENTIALS")
	}

	if token := data.Get("access_token"); token != nil {
		b.accessToken = token.(string)
	}
	if b.accessToken == "" {
		b.accessToken = os.Getenv("GOOGLE_OAUTH_ACCESS_TOKEN")
	}

	tokenSource, err := b.getTokenSource([]string{storage.ScopeReadWrite})
	if err != nil {
		return err
	}
	b.tokenSource = tokenSource

	oauthClient := oauth2.NewClient(context.Background(), tokenSource)
	oauthClient.Transport = logging.NewTransport("Google", oauthClient.Transport)
	// Each individual request should return within 30s - timeouts will be retried.
	// This is a timeout for, e.g. a single GET request of an operation - not a
	// timeout for the maximum amount of time a logical request can take.
	oauthClient.Timeout, _ = time.ParseDuration("30s")

	opts = append(opts, option.WithHTTPClient(oauthClient))
	opts = append(opts, option.WithUserAgent(httpclient.UserAgentString()))
	client, err := storage.NewClient(b.storageContext, opts...)
	if err != nil {
		return fmt.Errorf("storage.NewClient() failed: %v", err)
	}

	b.storageClient = client

	key := data.Get("encryption_key").(string)
	if key == "" {
		key = os.Getenv("GOOGLE_ENCRYPTION_KEY")
	}

	if key != "" {
		kc, _, err := pathorcontents.Read(key)
		if err != nil {
			return fmt.Errorf("Error loading encryption key: %s", err)
		}

		// The GCS client expects a customer supplied encryption key to be
		// passed in as a 32 byte long byte slice. The byte slice is base64
		// encoded before being passed to the API. We take a base64 encoded key
		// to remain consistent with the GCS docs.
		// https://cloud.google.com/storage/docs/encryption#customer-supplied
		// https://github.com/GoogleCloudPlatform/google-cloud-go/blob/def681/storage/storage.go#L1181
		k, err := base64.StdEncoding.DecodeString(kc)
		if err != nil {
			return fmt.Errorf("Error decoding encryption key: %s", err)
		}
		b.encryptionKey = k
	}

	return nil
}

func (b *Backend) getTokenSource(clientScopes []string) (oauth2.TokenSource, error) {
	if b.accessToken != "" {
		contents, _, err := pathorcontents.Read(b.accessToken)
		if err != nil {
			return nil, fmt.Errorf("Error loading access token: %s", err)
		}

		log.Printf("[INFO] Authenticating using configured Google JSON 'access_token'...")
		log.Printf("[INFO]   -- Scopes: %s", clientScopes)
		token := &oauth2.Token{AccessToken: contents}
		return oauth2.StaticTokenSource(token), nil
	}

	if b.credentials != "" {
		contents, _, err := pathorcontents.Read(b.credentials)
		if err != nil {
			return nil, fmt.Errorf("Error loading credentials: %s", err)
		}

		creds, err := googleoauth.CredentialsFromJSON(context.Background(), []byte(contents), clientScopes...)
		if err != nil {
			return nil, fmt.Errorf("Unable to parse credentials from '%s': %s", contents, err)
		}

		log.Printf("[INFO] Authenticating using configured Google JSON 'credentials'...")
		log.Printf("[INFO]   -- Scopes: %s", clientScopes)
		return creds.TokenSource, nil
	}

	log.Printf("[INFO] Authenticating using DefaultClient...")
	log.Printf("[INFO]   -- Scopes: %s", clientScopes)
	return googleoauth.DefaultTokenSource(context.Background(), clientScopes...)
}

// accountFile represents the structure of the account file JSON file.
type accountFile struct {
	PrivateKeyId string `json:"private_key_id"`
	PrivateKey   string `json:"private_key"`
	ClientEmail  string `json:"client_email"`
	ClientId     string `json:"client_id"`
}
