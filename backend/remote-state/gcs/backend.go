// Package gcs implements remote storage of state on Google Cloud Storage (GCS).
package gcs

import (
	"context"
	"fmt"
	"os"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	"google.golang.org/api/option"
)

// gcsBackend implements "backend".Backend for GCS.
// Input(), Validate() and Configure() are implemented by embedding *schema.Backend.
// State(), DeleteState() and States() are implemented explicitly.
type gcsBackend struct {
	*schema.Backend

	storageClient  *storage.Client
	storageContext context.Context

	bucketName       string
	prefix           string
	defaultStateFile string

	projectID string
	region    string
}

func New() backend.Backend {
	be := &gcsBackend{}
	be.Backend = &schema.Backend{
		ConfigureFunc: be.configure,
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
				Default:     "",
			},

			"project": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Google Cloud Project ID",
				Default:     "",
			},

			"region": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Region / location in which to create the bucket",
				Default:     "",
			},
		},
	}

	return be
}

func (b *gcsBackend) configure(ctx context.Context) error {
	if b.storageClient != nil {
		return nil
	}

	// ctx is a background context with the backend config added.
	// Since no context is passed to remoteClient.Get(), .Lock(), etc. but
	// one is required for calling the GCP API, we're holding on to this
	// context here and re-use it later.
	b.storageContext = ctx

	data := schema.FromContextBackendConfig(b.storageContext)

	b.bucketName = toBucketName(data.Get("bucket").(string))
	b.prefix = strings.TrimLeft(data.Get("prefix").(string), "/")

	b.defaultStateFile = strings.TrimLeft(data.Get("path").(string), "/")

	b.projectID = data.Get("project").(string)
	if id := os.Getenv("GOOGLE_PROJECT"); b.projectID == "" && id != "" {
		b.projectID = id
	}
	b.region = data.Get("region").(string)
	if r := os.Getenv("GOOGLE_REGION"); b.projectID == "" && r != "" {
		b.region = r
	}

	opts := []option.ClientOption{
		option.WithScopes(storage.ScopeReadWrite),
		option.WithUserAgent(terraform.UserAgentString()),
	}
	if credentialsFile := data.Get("credentials").(string); credentialsFile != "" {
		opts = append(opts, option.WithCredentialsFile(credentialsFile))
	} else if credentialsFile := os.Getenv("GOOGLE_CREDENTIALS"); credentialsFile != "" {
		opts = append(opts, option.WithCredentialsFile(credentialsFile))
	}

	client, err := storage.NewClient(b.storageContext, opts...)
	if err != nil {
		return fmt.Errorf("storage.NewClient() failed: %v", err)
	}

	b.storageClient = client

	return b.ensureBucketExists()
}

func (b *gcsBackend) ensureBucketExists() error {
	_, err := b.storageClient.Bucket(b.bucketName).Attrs(b.storageContext)
	if err != storage.ErrBucketNotExist {
		return err
	}

	if b.projectID == "" {
		return fmt.Errorf("bucket %q does not exist; specify the \"project\" option or create the bucket manually using `gsutil mb gs://%s`", b.bucketName, b.bucketName)
	}

	attrs := &storage.BucketAttrs{
		Location: b.region,
	}

	return b.storageClient.Bucket(b.bucketName).Create(b.storageContext, b.projectID, attrs)
}

// toBucketName returns a copy of in that is suitable for use as a bucket name.
// All upper case characters are converted to lower case, other invalid
// characters are replaced by '_'.
func toBucketName(in string) string {
	// Bucket names must contain only lowercase letters, numbers, dashes
	// (-), and underscores (_).
	isValid := func(r rune) bool {
		switch {
		case r >= 'a' && r <= 'z':
			return true
		case r >= '0' && r <= '9':
			return true
		case r == '-' || r == '_':
			return true
		default:
			return false

		}
	}

	out := make([]rune, 0, len(in))
	for _, r := range strings.ToLower(in) {
		if !isValid(r) {
			r = '_'
		}
		out = append(out, r)
	}

	// Bucket names must contain 3 to 63 characters.
	if len(out) > 63 {
		out = out[:63]
	}

	return string(out)
}
