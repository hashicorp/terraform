package obs

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/huaweicloud/golangsdk/openstack/obs"
)

// New creates a new backend for obs remote state.
func New() backend.Backend {
	s := &schema.Backend{
		Schema: map[string]*schema.Schema{
			"region": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The region of the obs bucket",
				DefaultFunc: schema.EnvDefaultFunc("OS_REGION_NAME", ""),
			},

			"bucket": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the obs bucket",
			},

			"key": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The path to the state file inside the bucket",
				Default:     "terraform.tfstate",
				ValidateFunc: func(v interface{}, s string) ([]string, []error) {
					if strings.HasPrefix(v.(string), "/") || strings.HasSuffix(v.(string), "/") {
						return nil, []error{fmt.Errorf("key can not start and end with '/'")}
					}
					return nil, nil
				},
			},

			"prefix": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The directory for saving the state file in bucket",
				ValidateFunc: func(v interface{}, s string) ([]string, []error) {
					prefix := v.(string)
					if strings.HasPrefix(prefix, "/") || strings.HasPrefix(prefix, "./") {
						return nil, []error{fmt.Errorf("prefix must not start with '/' or './'")}
					}
					return nil, nil
				},
			},

			"endpoint": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "An endpoint for the obs API",
			},

			"access_key_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "HuaweiCloud access key",
				DefaultFunc: schema.EnvDefaultFunc("OS_ACCESS_KEY", ""),
			},

			"secret_key_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "HuaweiCloud secret key",
				DefaultFunc: schema.EnvDefaultFunc("OS_SECRET_KEY", ""),
			},

			"acl": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Canned ACL to be applied to the state file",
				Default:     "private",
			},

			"encrypt": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Whether to enable server side encryption of the state file",
				Default:     false,
			},

			"kms_key_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The KMS Key to use for encrypting the state",
				Default:     "",
			},
		},
	}

	result := &Backend{Backend: s}
	result.Backend.ConfigureFunc = result.configure
	return result
}

// Backend implements backend.Backend for obs
type Backend struct {
	*schema.Backend

	// The fields below are set from configure
	obsClient *obs.ObsClient

	bucketName string
	keyName    string
	prefix     string
	acl        string
	encryption bool
	kmsKeyID   string
}

func (b *Backend) configure(ctx context.Context) error {
	if b.obsClient != nil {
		return nil
	}

	// Grab the resource data
	data := schema.FromContextBackendConfig(ctx)

	b.bucketName = data.Get("bucket").(string)
	b.keyName = data.Get("key").(string)
	b.prefix = data.Get("prefix").(string)
	b.acl = data.Get("acl").(string)
	b.encryption = data.Get("encrypt").(bool)
	b.kmsKeyID = data.Get("kms_key_id").(string)

	accessKey := data.Get("access_key_id").(string)
	secretKey := data.Get("secret_key_id").(string)
	region := data.Get("region").(string)
	endpoint := data.Get("endpoint").(string)

	if accessKey == "" {
		return fmt.Errorf("Error: access_key_id or env OS_ACCESS_KEY must be set")
	}

	if secretKey == "" {
		return fmt.Errorf("Error: secret_key_id or env OS_SECRET_KEY must be set")
	}

	if endpoint == "" {
		endpoint = b.getDefaultOBSEndpoint(region)
	}
	if !strings.HasPrefix(endpoint, "http") {
		endpoint = fmt.Sprintf("https://%s", endpoint)
	}

	obsclient, err := obs.New(accessKey, secretKey, endpoint)
	if err != nil {
		return err
	}
	b.obsClient = obsclient

	return nil
}

func (b *Backend) getDefaultOBSEndpoint(region string) (endpoint string) {
	endpoint = fmt.Sprintf("https://obs.%s.myhuaweicloud.com", region)

	return endpoint
}
