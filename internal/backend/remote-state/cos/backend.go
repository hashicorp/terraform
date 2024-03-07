// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cos

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"time"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	sts "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/sts/v20180813"
	tag "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/tag/v20180813"
	"github.com/tencentyun/cos-go-sdk-v5"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/backend/backendbase"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Default value from environment variable
const (
	PROVIDER_SECRET_ID                    = "TENCENTCLOUD_SECRET_ID"
	PROVIDER_SECRET_KEY                   = "TENCENTCLOUD_SECRET_KEY"
	PROVIDER_SECURITY_TOKEN               = "TENCENTCLOUD_SECURITY_TOKEN"
	PROVIDER_REGION                       = "TENCENTCLOUD_REGION"
	PROVIDER_ENDPOINT                     = "TENCENTCLOUD_ENDPOINT"
	PROVIDER_DOMAIN                       = "TENCENTCLOUD_DOMAIN"
	PROVIDER_ASSUME_ROLE_ARN              = "TENCENTCLOUD_ASSUME_ROLE_ARN"
	PROVIDER_ASSUME_ROLE_SESSION_NAME     = "TENCENTCLOUD_ASSUME_ROLE_SESSION_NAME"
	PROVIDER_ASSUME_ROLE_SESSION_DURATION = "TENCENTCLOUD_ASSUME_ROLE_SESSION_DURATION"
)

// Backend implements "backend".Backend for tencentCloud cos
type Backend struct {
	backendbase.Base

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
	domain  string
}

// New creates a new backend for TencentCloud cos remote state.
func New() backend.Backend {
	return &Backend{
		Base: backendbase.Base{
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"secret_id": {
						Type:        cty.String,
						Optional:    true,
						Description: "Secret id of Tencent Cloud",
					},
					"secret_key": {
						Type:        cty.String,
						Optional:    true,
						Description: "Secret key of Tencent Cloud",
						//Sensitive:   true,
					},
					"security_token": {
						Type:        cty.String,
						Optional:    true,
						Description: "TencentCloud Security Token of temporary access credentials. It can be sourced from the `TENCENTCLOUD_SECURITY_TOKEN` environment variable. Notice: for supported products, please refer to: [temporary key supported products](https://intl.cloud.tencent.com/document/product/598/10588).",
						//Sensitive:   true,
					},
					"region": {
						Type:        cty.String,
						Required:    true,
						Description: "The region of the COS bucket",
						//InputDefault: "ap-guangzhou",
					},
					"bucket": {
						Type:        cty.String,
						Required:    true,
						Description: "The name of the COS bucket",
					},
					"endpoint": {
						Type:        cty.String,
						Optional:    true,
						Description: "The custom endpoint for the COS API, e.g. http://cos-internal.{Region}.tencentcos.cn. Both HTTP and HTTPS are accepted.",
					},
					"domain": {
						Type:        cty.String,
						Optional:    true,
						Description: "The root domain of the API request. Default is tencentcloudapi.com.",
					},
					"prefix": {
						Type:        cty.String,
						Optional:    true,
						Description: "The directory for saving the state file in bucket",
						/*
							ValidateFunc: func(v interface{}, s string) ([]string, []error) {
								prefix := v.(string)
								if strings.HasPrefix(prefix, "/") || strings.HasPrefix(prefix, "./") {
									return nil, []error{fmt.Errorf("prefix must not start with '/' or './'")}
								}
								return nil, nil
							},
						*/
					},
					"key": {
						Type:        cty.String,
						Optional:    true,
						Description: "The path for saving the state file in bucket",
						/*
							ValidateFunc: func(v interface{}, s string) ([]string, []error) {
								if strings.HasPrefix(v.(string), "/") || strings.HasSuffix(v.(string), "/") {
									return nil, []error{fmt.Errorf("key can not start and end with '/'")}
								}
								return nil, nil
							},
						*/
					},
					"encrypt": {
						Type:        cty.Bool,
						Optional:    true,
						Description: "Whether to enable server side encryption of the state file",
					},
					"acl": {
						Type:        cty.String,
						Optional:    true,
						Description: "Object ACL to be applied to the state file",
						/*
							ValidateFunc: func(v interface{}, s string) ([]string, []error) {
								value := v.(string)
								if value != "private" && value != "public-read" {
									return nil, []error{fmt.Errorf(
										"acl value invalid, expected %s or %s, got %s",
										"private", "public-read", value)}
								}
								return nil, nil
							},
						*/
					},
					"accelerate": {
						Type:        cty.Bool,
						Optional:    true,
						Description: "Whether to enable global Acceleration",
					},
				},
				BlockTypes: map[string]*configschema.NestedBlock{
					"assume_role": {
						Nesting: configschema.NestingSingle,
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"role_arn": {
									Type:     cty.String,
									Required: true,
									//DefaultFunc: schema.EnvDefaultFunc(PROVIDER_ASSUME_ROLE_ARN, nil),
									Description: "The ARN of the role to assume. It can be sourced from the `TENCENTCLOUD_ASSUME_ROLE_ARN`.",
								},
								"session_name": {
									Type:     cty.String,
									Required: true,
									//DefaultFunc: schema.EnvDefaultFunc(PROVIDER_ASSUME_ROLE_SESSION_NAME, nil),
									Description: "The session name to use when making the AssumeRole call. It can be sourced from the `TENCENTCLOUD_ASSUME_ROLE_SESSION_NAME`.",
								},
								"session_duration": {
									Type:     cty.Number,
									Required: true,
									/*
										DefaultFunc: func() (interface{}, error) {
											if v := os.Getenv(PROVIDER_ASSUME_ROLE_SESSION_DURATION); v != "" {
												return strconv.Atoi(v)
											}
											return 7200, nil
										},
									*/
									// NOTE: When adapting this validation rule
									// it'll also need to check whether the value
									// is an integer, since cty can't guarantee
									// a whole number.
									// ValidateFunc: validateIntegerInRange(0, 43200),
									Description: "The duration of the session when making the AssumeRole call. Its value ranges from 0 to 43200(seconds), and default is 7200 seconds. It can be sourced from the `TENCENTCLOUD_ASSUME_ROLE_SESSION_DURATION`.",
								},
								"policy": {
									Type:        cty.String,
									Optional:    true,
									Description: "A more restrictive policy when making the AssumeRole call. Its content must not contains `principal` elements. Notice: more syntax references, please refer to: [policies syntax logic](https://intl.cloud.tencent.com/document/product/598/10603).",
								},
							},
						},
					},
				},
			},
			SDKLikeDefaults: backendbase.SDKLikeDefaults{
				"secret_id": {
					EnvVars: []string{PROVIDER_SECRET_ID},
				},
				"secret_key": {
					EnvVars: []string{PROVIDER_SECRET_KEY},
				},
				"security_token": {
					EnvVars: []string{PROVIDER_SECURITY_TOKEN},
				},
				"region": {
					EnvVars: []string{PROVIDER_REGION},
				},
				"endpoint": {
					EnvVars: []string{PROVIDER_ENDPOINT},
				},
				"domain": {
					EnvVars: []string{PROVIDER_DOMAIN},
				},
				"prefix": {
					Fallback: "",
				},
				"key": {
					Fallback: "terraform.tfstate",
				},
				"encrypt": {
					Fallback: "true",
				},
				"acl": {
					Fallback: "private",
				},
				"accelerate": {
					Fallback: "false",
				},
			},
		},
	}
}

/*
// TODO: Adapt this for cty.Number?
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
*/

// configure init cos client
func (b *Backend) Configure(configVal cty.Value) tfdiags.Diagnostics {
	if b.cosClient != nil {
		return nil
	}

	data := backendbase.NewSDKLikeData(configVal)

	// TODO: Pre-validate configVal with similar rules to what were previously
	// declared inline in the schema.

	b.region = data.String("region")
	b.bucket = data.String("bucket")
	b.prefix = data.String("prefix")
	b.key = data.String("key")
	b.encrypt = data.Bool("encrypt")
	b.acl = data.String("acl")

	var (
		u   *url.URL
		err error
	)
	accelerate := data.Bool("accelerate")
	if accelerate {
		u, err = url.Parse(fmt.Sprintf("https://%s.cos.accelerate.myqcloud.com", b.bucket))
	} else {
		u, err = url.Parse(fmt.Sprintf("https://%s.cos.%s.myqcloud.com", b.bucket, b.region))
	}
	if err != nil {
		return backendbase.ErrorAsDiagnostics(err)
	}

	if v := data.String("domain"); v != "" {
		b.domain = v
		log.Printf("[DEBUG] Backend: set domain for TencentCloud API client. Domain: [%s]", b.domain)
	}
	// set url as endpoint when provided
	// "http://{Bucket}.cos-internal.{Region}.tencentcos.cn"
	if v := data.String("endpoint"); v != "" {
		endpoint := v

		re := regexp.MustCompile(`^(http(s)?)://cos-internal\.([^.]+)\.tencentcos\.cn$`)
		matches := re.FindStringSubmatch(endpoint)
		if len(matches) != 4 {
			return backendbase.ErrorAsDiagnostics(
				fmt.Errorf("Invalid URL: %v must be: %v", endpoint, "http(s)://cos-internal.{Region}.tencentcos.cn"),
			)
		}

		protocol := matches[1]
		region := matches[3]

		// URL after converting
		newUrl := fmt.Sprintf("%s://%s.cos-internal.%s.tencentcos.cn", protocol, b.bucket, region)
		u, err = url.Parse(newUrl)
		log.Printf("[DEBUG] Backend: set COS URL as: [%s]", newUrl)
	}
	if err != nil {
		return backendbase.ErrorAsDiagnostics(err)
	}

	secretId := data.String("secret_id")
	secretKey := data.String("secret_key")
	securityToken := data.String("security_token")

	// init credential by AKSK & TOKEN
	b.credential = common.NewTokenCredential(secretId, secretKey, securityToken)
	// update credential if assume role exist
	err = handleAssumeRole(data, b)
	if err != nil {
		return backendbase.ErrorAsDiagnostics(err)
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
	if err != nil {
		return backendbase.ErrorAsDiagnostics(err)
	}
	return nil
}

func handleAssumeRole(data backendbase.SDKLikeData, b *Backend) error {
	assumeRoleVal := data.GetAttr("assume_role", cty.DynamicPseudoType)
	if !assumeRoleVal.IsNull() {
		assumeRole := backendbase.NewSDKLikeData(assumeRoleVal)
		// TODO: Handle the environment-variable-based defaults for these
		// arguments, since backendbase.SDKLikeDefaults only supports
		// top-level attributes, not nested attributes. (This is the only
		// backend that relies on a nested block in its configuration, so
		// probably not worth complicating the shared helpers just for this
		// one exceptional case.)
		assumeRoleArn := assumeRole.String("role_arn")
		assumeRoleSessionName := assumeRole.String("session_name")
		assumeRolePolicy := assumeRole.String("policy")
		assumeRoleSessionDuration, err := assumeRole.Int64("session_duration")
		if err != nil {
			return fmt.Errorf("invalid session_duration: %w", err)
		}

		// NOTE: The following truncates assumeRoleSessionDuration from
		// 64-bit to 32-bit when running on a 32-bit platform.
		err = b.updateCredentialWithSTS(assumeRoleArn, assumeRoleSessionName, int(assumeRoleSessionDuration), assumeRolePolicy)
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
	// request domain
	cpf.HttpProfile.RootDomain = b.domain

	return cpf
}
