package cos

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	tag "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/tag/v20180813"
	"github.com/tencentyun/cos-go-sdk-v5"
)

// Default value from environment variable
const (
	PROVIDER_SECRET_ID  = "TENCENTCLOUD_SECRET_ID"
	PROVIDER_SECRET_KEY = "TENCENTCLOUD_SECRET_KEY"
	PROVIDER_REGION     = "TENCENTCLOUD_REGION"
)

// Backend implements "backend".Backend for tencentCloud cos
type Backend struct {
	*schema.Backend

	cosContext context.Context
	cosClient  *cos.Client
	tagClient  *tag.Client

	region  string
	bucket  string
	prefix  string
	key     string
	encrypt bool
	acl     string
}

// New creates a new backend for TencentCloud cos remote state.
func New() backend.Backend {
	s := &schema.Backend{
		Schema: map[string]*schema.Schema{
			"secret_id": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc(PROVIDER_SECRET_ID, nil),
				Description: "Secret id of Tencent Cloud",
			},
			"secret_key": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc(PROVIDER_SECRET_KEY, nil),
				Description: "Secret key of Tencent Cloud",
				Sensitive:   true,
			},
			"region": {
				Type:         schema.TypeString,
				Required:     true,
				DefaultFunc:  schema.EnvDefaultFunc(PROVIDER_REGION, nil),
				Description:  "The region of the COS bucket",
				InputDefault: "ap-guangzhou",
			},
			"bucket": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the COS bucket",
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
			"key": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The path for saving the state file in bucket",
				Default:     "terraform.tfstate",
				ValidateFunc: func(v interface{}, s string) ([]string, []error) {
					if strings.HasPrefix(v.(string), "/") || strings.HasSuffix(v.(string), "/") {
						return nil, []error{fmt.Errorf("key can not start and end with '/'")}
					}
					return nil, nil
				},
			},
			"encrypt": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Whether to enable server side encryption of the state file",
				Default:     true,
			},
			"acl": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Object ACL to be applied to the state file",
				Default:     "private",
				ValidateFunc: func(v interface{}, s string) ([]string, []error) {
					value := v.(string)
					if value != "private" && value != "public-read" {
						return nil, []error{fmt.Errorf(
							"acl value invalid, expected %s or %s, got %s",
							"private", "public-read", value)}
					}
					return nil, nil
				},
			},
		},
	}

	result := &Backend{Backend: s}
	result.Backend.ConfigureFunc = result.configure

	return result
}

// configure init cos client
func (b *Backend) configure(ctx context.Context) error {
	if b.cosClient != nil {
		return nil
	}

	b.cosContext = ctx
	data := schema.FromContextBackendConfig(b.cosContext)

	b.region = data.Get("region").(string)
	b.bucket = data.Get("bucket").(string)
	b.prefix = data.Get("prefix").(string)
	b.key = data.Get("key").(string)
	b.encrypt = data.Get("encrypt").(bool)
	b.acl = data.Get("acl").(string)

	u, err := url.Parse(fmt.Sprintf("https://%s.cos.%s.myqcloud.com", b.bucket, b.region))
	if err != nil {
		return err
	}

	b.cosClient = cos.NewClient(
		&cos.BaseURL{BucketURL: u},
		&http.Client{
			Timeout: 60 * time.Second,
			Transport: &cos.AuthorizationTransport{
				SecretID:  data.Get("secret_id").(string),
				SecretKey: data.Get("secret_key").(string),
			},
		},
	)

	credential := common.NewCredential(
		data.Get("secret_id").(string),
		data.Get("secret_key").(string),
	)

	cpf := profile.NewClientProfile()
	cpf.HttpProfile.ReqMethod = "POST"
	cpf.HttpProfile.ReqTimeout = 300
	cpf.Language = "en-US"
	b.tagClient, err = tag.NewClient(credential, b.region, cpf)

	return err
}
