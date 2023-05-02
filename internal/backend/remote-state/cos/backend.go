// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package cos

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/legacy/helper/schema"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	sts "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/sts/v20180813"
	tag "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/tag/v20180813"
	"github.com/tencentyun/cos-go-sdk-v5"
)

// Default value from environment variable
const (
	PROVIDER_SECRET_ID                    = "TENCENTCLOUD_SECRET_ID"
	PROVIDER_SECRET_KEY                   = "TENCENTCLOUD_SECRET_KEY"
	PROVIDER_SECURITY_TOKEN               = "TENCENTCLOUD_SECURITY_TOKEN"
	PROVIDER_REGION                       = "TENCENTCLOUD_REGION"
	PROVIDER_ASSUME_ROLE_ARN              = "TENCENTCLOUD_ASSUME_ROLE_ARN"
	PROVIDER_ASSUME_ROLE_SESSION_NAME     = "TENCENTCLOUD_ASSUME_ROLE_SESSION_NAME"
	PROVIDER_ASSUME_ROLE_SESSION_DURATION = "TENCENTCLOUD_ASSUME_ROLE_SESSION_DURATION"
)

// Backend implements "backend".Backend for tencentCloud cos
type Backend struct {
	*schema.Backend
	credential *common.Credential

	cosContext context.Context
	cosClient  *cos.Client
	tagClient  *tag.Client
	stsClient  *sts.Client

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
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc(PROVIDER_SECRET_ID, nil),
				Description: "Secret id of Tencent Cloud",
			},
			"secret_key": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc(PROVIDER_SECRET_KEY, nil),
				Description: "Secret key of Tencent Cloud",
				Sensitive:   true,
			},
			"security_token": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc(PROVIDER_SECURITY_TOKEN, nil),
				Description: "TencentCloud Security Token of temporary access credentials. It can be sourced from the `TENCENTCLOUD_SECURITY_TOKEN` environment variable. Notice: for supported products, please refer to: [temporary key supported products](https://intl.cloud.tencent.com/document/product/598/10588).",
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
			"accelerate": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Whether to enable global Acceleration",
				Default:     false,
			},
			"assume_role": {
				Type:        schema.TypeSet,
				Optional:    true,
				MaxItems:    1,
				Description: "The `assume_role` block. If provided, terraform will attempt to assume this role using the supplied credentials.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"role_arn": {
							Type:        schema.TypeString,
							Required:    true,
							DefaultFunc: schema.EnvDefaultFunc(PROVIDER_ASSUME_ROLE_ARN, nil),
							Description: "The ARN of the role to assume. It can be sourced from the `TENCENTCLOUD_ASSUME_ROLE_ARN`.",
						},
						"session_name": {
							Type:        schema.TypeString,
							Required:    true,
							DefaultFunc: schema.EnvDefaultFunc(PROVIDER_ASSUME_ROLE_SESSION_NAME, nil),
							Description: "The session name to use when making the AssumeRole call. It can be sourced from the `TENCENTCLOUD_ASSUME_ROLE_SESSION_NAME`.",
						},
						"session_duration": {
							Type:     schema.TypeInt,
							Required: true,
							DefaultFunc: func() (interface{}, error) {
								if v := os.Getenv(PROVIDER_ASSUME_ROLE_SESSION_DURATION); v != "" {
									return strconv.Atoi(v)
								}
								return 7200, nil
							},
							ValidateFunc: validateIntegerInRange(0, 43200),
							Description:  "The duration of the session when making the AssumeRole call. Its value ranges from 0 to 43200(seconds), and default is 7200 seconds. It can be sourced from the `TENCENTCLOUD_ASSUME_ROLE_SESSION_DURATION`.",
						},
						"policy": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "A more restrictive policy when making the AssumeRole call. Its content must not contains `principal` elements. Notice: more syntax references, please refer to: [policies syntax logic](https://intl.cloud.tencent.com/document/product/598/10603).",
						},
					},
				},
			},
		},
	}

	result := &Backend{Backend: s}
	result.Backend.ConfigureFunc = result.configure

	return result
}

func validateIntegerInRange(min, max int64) schema.SchemaValidateFunc {
	return func(v interface{}, k string) (ws []string, errors []error) {
		value := int64(v.(int))
		if value < min {
			errors = append(errors, fmt.Errorf(
				"%q cannot be lower than %d: %d", k, min, value))
		}
		if value > max {
			errors = append(errors, fmt.Errorf(
				"%q cannot be higher than %d: %d", k, max, value))
		}
		return
	}
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

	var (
		u   *url.URL
		err error
	)
	accelerate := data.Get("accelerate").(bool)
	if accelerate {
		u, err = url.Parse(fmt.Sprintf("https://%s.cos.accelerate.myqcloud.com", b.bucket))
	} else {
		u, err = url.Parse(fmt.Sprintf("https://%s.cos.%s.myqcloud.com", b.bucket, b.region))
	}
	if err != nil {
		return err
	}

	secretId := data.Get("secret_id").(string)
	secretKey := data.Get("secret_key").(string)
	securityToken := data.Get("security_token").(string)

	// init credential by AKSK & TOKEN
	b.credential = common.NewTokenCredential(secretId, secretKey, securityToken)
	// update credential if assume role exist
	err = handleAssumeRole(data, b)
	if err != nil {
		return err
	}

	b.cosClient = cos.NewClient(
		&cos.BaseURL{BucketURL: u},
		&http.Client{
			Timeout: 60 * time.Second,
			Transport: &cos.AuthorizationTransport{
				SecretID:     b.credential.SecretId,
				SecretKey:    b.credential.SecretKey,
				SessionToken: b.credential.Token,
			},
		},
	)

	b.tagClient = b.UseTagClient()
	return err
}

func handleAssumeRole(data *schema.ResourceData, b *Backend) error {
	assumeRoleList := data.Get("assume_role").(*schema.Set).List()
	if len(assumeRoleList) == 1 {
		assumeRole := assumeRoleList[0].(map[string]interface{})
		assumeRoleArn := assumeRole["role_arn"].(string)
		assumeRoleSessionName := assumeRole["session_name"].(string)
		assumeRoleSessionDuration := assumeRole["session_duration"].(int)
		assumeRolePolicy := assumeRole["policy"].(string)

		err := b.updateCredentialWithSTS(assumeRoleArn, assumeRoleSessionName, assumeRoleSessionDuration, assumeRolePolicy)
		if err != nil {
			return err
		}
	}
	return nil
}

func (b *Backend) updateCredentialWithSTS(assumeRoleArn, assumeRoleSessionName string, assumeRoleSessionDuration int, assumeRolePolicy string) error {
	// assume role by STS
	request := sts.NewAssumeRoleRequest()
	request.RoleArn = &assumeRoleArn
	request.RoleSessionName = &assumeRoleSessionName
	duration := uint64(assumeRoleSessionDuration)
	request.DurationSeconds = &duration
	if assumeRolePolicy != "" {
		policy := url.QueryEscape(assumeRolePolicy)
		request.Policy = &policy
	}

	response, err := b.UseStsClient().AssumeRole(request)
	if err != nil {
		return err
	}
	// update credentials by result of assume role
	b.credential = common.NewTokenCredential(
		*response.Response.Credentials.TmpSecretId,
		*response.Response.Credentials.TmpSecretKey,
		*response.Response.Credentials.Token,
	)

	return nil
}

// UseStsClient returns sts client for service
func (b *Backend) UseStsClient() *sts.Client {
	if b.stsClient != nil {
		return b.stsClient
	}
	cpf := b.NewClientProfile(300)
	b.stsClient, _ = sts.NewClient(b.credential, b.region, cpf)
	b.stsClient.WithHttpTransport(&LogRoundTripper{})

	return b.stsClient
}

// UseTagClient returns tag client for service
func (b *Backend) UseTagClient() *tag.Client {
	if b.tagClient != nil {
		return b.tagClient
	}
	cpf := b.NewClientProfile(300)
	cpf.Language = "en-US"
	b.tagClient, _ = tag.NewClient(b.credential, b.region, cpf)
	return b.tagClient
}

// NewClientProfile returns a new ClientProfile
func (b *Backend) NewClientProfile(timeout int) *profile.ClientProfile {
	cpf := profile.NewClientProfile()

	// all request use method POST
	cpf.HttpProfile.ReqMethod = "POST"
	// request timeout
	cpf.HttpProfile.ReqTimeout = timeout

	return cpf
}
