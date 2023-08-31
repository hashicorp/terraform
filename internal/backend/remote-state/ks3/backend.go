// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package ks3

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/KscSDK/ksc-sdk-go/ksc"
	"github.com/KscSDK/ksc-sdk-go/ksc/utils"
	"github.com/KscSDK/ksc-sdk-go/service/sts"
	tag "github.com/KscSDK/ksc-sdk-go/service/tagv2"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/legacy/helper/schema"
	"github.com/mitchellh/mapstructure"
	"github.com/wilac-pv/ksyun-ks3-go-sdk/ks3"
)

// Default value from environment variable
const (
	ENV_ACCESS_KEY                   = "KSYUN_ACCESS_KEY"
	ENV_SECRET_KEY                   = "KSYUN_SECRET_KEY"
	ENV_REGION                       = "KSYUN_REGION"
	ENV_ASSUME_ROLE_ARN              = "KSYUN_ASSUME_ROLE_ARN"
	ENV_ASSUME_ROLE_SESSION_NAME     = "KSYUN_ASSUME_ROLE_SESSION_NAME"
	ENV_ASSUME_ROLE_SESSION_DURATION = "KSYUN_ASSUME_ROLE_SESSION_DURATION"
	ENV_KS3_ENDPOINT                 = "KSYUN_KS3_ENDPOINT"
)

// kscBaseCfg base configurations
type kscBaseCfg struct {
	session   *session.Session
	regionCfg *ksc.Config
	url       *utils.UrlInfo
}

// AssumeRoleResponse assume role response
type AssumeRoleResponse struct {
	RequestId        string           `mapstructure:"RequestId"`
	AssumeRoleResult AssumeRoleResult `mapstructure:"AssumeRoleResult"`
}

// AssumeRoleResult credentials result for assume role
type AssumeRoleResult struct {
	Credentials     Credentials `mapstructure:"Credentials"`
	AssumedRoleUser struct {
		Krn           string `mapstructure:"Krn"`
		AssumedRoleId string `mapstructure:"AssumedRoleId"`
	}
}

type Credentials struct {
	SecretAccessKey string `mapstructure:"SecretAccessKey"`
	Expiration      string `mapstructure:"Expiration"`
	AccessKeyId     string `mapstructure:"AccessKeyId"`
	SecurityToken   string `mapstructure:"SecurityToken"`
}

// Backend implements "backend".Backend for KSYUN cos
type Backend struct {
	*schema.Backend
	baseCfg *kscBaseCfg

	ks3Context context.Context
	ks3Client  *ks3.Client
	tagClient  *tag.Tagv2
	stsClient  *sts.Sts

	region             string
	bucket             string
	workspaceKeyPrefix string
	key                string
	encrypt            bool
	acl                string

	// lockDuration indicates the lifetime of lock
	lockDuration string
}

// New creates a new backend for KSYUN cos remote state.
func New() backend.Backend {
	s := &schema.Backend{
		Schema: map[string]*schema.Schema{
			"access_key": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Ksyun access key, It can be sourced from the `" + ENV_ACCESS_KEY + "` environment variable.",
				DefaultFunc: schema.EnvDefaultFunc(ENV_ACCESS_KEY, nil),
			},
			"secret_key": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Ksyun secret key, It can be sourced from the `" + ENV_SECRET_KEY + "` environment variable.",
				DefaultFunc: schema.EnvDefaultFunc(ENV_SECRET_KEY, nil),
			},
			"bucket": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the KS3 bucket",
			},
			"key": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The path to the state file inside the bucket",
				Default:     "terraform.tfstate",
				ValidateFunc: func(v interface{}, s string) ([]string, []error) {
					if strings.HasPrefix(v.(string), "/") || strings.HasSuffix(v.(string), "/") {
						return nil, []error{fmt.Errorf("key can not start and end with '/'")}
					}
					return nil, nil
				},
			},
			// "storage_type": {
			// 	Type:        schema.TypeString,
			// 	Optional:    true,
			// 	Default:     "STANDARD",
			// 	Description: "the state file storage type in Ksyun.",
			// 	ValidateFunc: stringInSlice([]string{
			// 		"STANDARD",
			// 		"STANDARD_IA",
			// 		"ARCHIVE",
			// 	}, false),
			// },
			"region": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Ksyun region of the KS3 Bucket. It can be sourced from the `" + ENV_REGION + "` environment variable.",
				DefaultFunc: schema.EnvDefaultFunc(ENV_REGION, nil),
			},
			"endpoint": {
				Type:        schema.TypeString,
				Required:    true,
				Default:     schema.EnvDefaultFunc(ENV_KS3_ENDPOINT, nil),
				Description: "A custom endpoint for the KS3 API , It can be sourced from the `" + ENV_KS3_ENDPOINT + "` environment variable. the details of relationship of endpoint and region see [Endpoint and Region](https://docs.ksyun.com/documents/6761)",
			},
			"encrypt": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Whether to enable server side encryption of the state file, Default `false`",
			},
			"acl": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Canned ACL to be applied to the state file",
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

			"role_krn": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The krn of role to be assumed, It can be sourced from the `" + ENV_ASSUME_ROLE_ARN + "` environment variable.",
				DefaultFunc: schema.EnvDefaultFunc(ENV_ASSUME_ROLE_ARN, nil),
			},
			"session_name": {
				Type:     schema.TypeString,
				Optional: true,
				ValidateFunc: func(i interface{}, s string) (warns []string, errs []error) {
					if reflect.DeepEqual(i, "") {
						err := fmt.Errorf("session_name is not blank letter")
						errs = append(errs, err)
					}
					return
				},
				Description: "The session name to use when assuming the role. If you ready to assume a role, must set `session_name`.  It can be sourced from the `" + ENV_ASSUME_ROLE_SESSION_NAME + "` environment variable.",
				DefaultFunc: schema.EnvDefaultFunc(ENV_ASSUME_ROLE_SESSION_NAME, nil),
			},

			"assume_role_duration_seconds": {
				Type:         schema.TypeInt,
				Optional:     true,
				ValidateFunc: validateIntegerInRange(900, 86400),
				DefaultFunc: func() (interface{}, error) {
					if v := os.Getenv(ENV_ASSUME_ROLE_SESSION_DURATION); v != "" {
						return strconv.Atoi(v)
					}
					return 3600, nil
				},
				Description: "Seconds to restrict the assume role session duration. Range in [900, 86400], Default 3600.  It can be sourced from the `" + ENV_ASSUME_ROLE_SESSION_DURATION + "` environment variable.",
			},

			"assume_role_policy": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "IAM Policy JSON describing further restricting permissions for the IAM Role being assumed. It will be set the intersection permissions of role and the got policy, if you set.",
			},

			"workspace_key_prefix": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The prefix applied to the non-default state path inside the bucket.",
				ValidateFunc: func(i interface{}, s string) (ret []string, errs []error) {
					val := i.(string)
					if reflect.DeepEqual(val, "..") ||
						strings.Contains(val, " ") ||
						strings.Contains(val, "@") {
						errs = append(errs, fmt.Errorf("the name cannot be '..', and the following characters must not be included: ' ' '@'"))
						return ret, errs
					}
					ret = append(ret, val)
					return ret, errs
				},
			},

			"max_retries": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     3,
				Description: "The maximum number of times an AWS API request is retried on retryable failure.",
			},
			"lock_duration": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "-1",
				ValidateFunc: validateLockDurations,
				Description:  "Sets the lock's duration. Generally speaking, the lock will be destroyed after the Terraform operations. Unfortunately, the lock is leaved because of the Terraform process is terminated. So, `lock_duration` field is provided that canned set it in order to deal with **the dead lock**. **Warning:** if `lock_duration` value is insufficient for your operation, the remote state file may be not your expectation, because of the operation race probably caused. Valid Value: `0`: ignore the existed lock, `-1`: unlimited, `number[mh]`: number, natural number, is time length; `m` is minutes; `h` is hours",
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

func validateLockDurations(i interface{}, k string) (warnings []string, errors []error) {
	v, ok := i.(string)
	if !ok {
		errors = append(errors, fmt.Errorf("expected type of %s to be string", k))
		return
	}
	switch v {
	case "-1":
	case "0":
	default:
		suffix := v[len(v)-1:]
		switch suffix {
		case "h", "m":
			length := v[:len(v)-1]
			_, err := strconv.Atoi(length)
			if err != nil {
				errors = append(errors, fmt.Errorf("lock_duration valid value: number[mh], e.g. 1h, err:%s", err))
				return
			}
		default:
			errors = append(errors, fmt.Errorf("lock_duration valid value: number[mh], e.g. 1h"))
			return
		}
	}
	return
}

func stringInSlice(valid []string, ignoreCase bool) schema.SchemaValidateFunc {
	return func(i interface{}, k string) (warnings []string, errors []error) {
		v, ok := i.(string)
		if !ok {
			errors = append(errors, fmt.Errorf("expected type of %s to be string", k))
			return warnings, errors
		}

		for _, str := range valid {
			if v == str || (ignoreCase && strings.ToLower(v) == strings.ToLower(str)) {
				return warnings, errors
			}
		}

		errors = append(errors, fmt.Errorf("expected %s to be one of %v, got %s", k, valid, v))
		return warnings, errors
	}
}

// configure init cos client
func (b *Backend) configure(ctx context.Context) error {
	if b.ks3Client != nil {
		return nil
	}

	b.ks3Context = ctx
	data := schema.FromContextBackendConfig(b.ks3Context)

	b.region = data.Get("region").(string)
	b.bucket = data.Get("bucket").(string)
	b.workspaceKeyPrefix = data.Get("workspace_key_prefix").(string)
	b.key = data.Get("key").(string)
	b.encrypt = data.Get("encrypt").(bool)
	b.acl = data.Get("acl").(string)
	b.lockDuration = data.Get("lock_duration").(string)

	var (
		err error
	)

	accessKey := data.Get("access_key").(string)
	secretKey := data.Get("secret_key").(string)
	// securityToken := data.Get("security_token").(string)
	endpoint := data.Get("endpoint").(string)
	maxRetries := data.Get("max_retries").(int)

	sess := ksc.NewClient(accessKey, secretKey)
	sess.Config.MaxRetries = &maxRetries

	// register session retryer
	// sess.Config.Retryer = retryer

	b.baseCfg = &kscBaseCfg{
		session: ksc.NewClient(accessKey, secretKey),
		regionCfg: &ksc.Config{
			Region: &b.region,
		},
		url: &utils.UrlInfo{
			UseSSL: false,
			Locate: false,
		},
	}

	// configure tag server max retries
	b.baseCfg.session.Config.MaxRetries = &maxRetries

	// here we go to assume role.
	roleKrn := data.Get("role_krn")
	if !reflect.ValueOf(roleKrn).IsZero() {
		credential, err := assumeRole(data, b)
		if err != nil {
			return err
		}
		accessKey = credential.AccessKeyId
		secretKey = credential.SecretAccessKey
	}

	// configure ks3 server max retries
	ks3RetryOption := func(client *ks3.Client) {
		client.Config.RetryTimes = uint(maxRetries)
	}

	b.ks3Client, err = ks3.New(endpoint, accessKey, secretKey, ks3RetryOption)
	if err != nil {
		return err
	}

	b.tagClient = b.TagClient()
	return err
}

// assumeRole assume role
func assumeRole(data *schema.ResourceData, b *Backend) (*Credentials, error) {
	params := make(map[string]interface{}, 4)
	roleKrn := data.Get("role_krn").(string)
	sessionName := data.Get("session_name").(string)

	if durationSeconds, ok := data.GetOk("assume_role_duration_seconds"); ok {
		params["DurationSeconds"] = durationSeconds
	}
	if externalPolicy, ok := data.GetOk("assume_role_policy"); ok {
		params["Policy"] = externalPolicy
	}

	params["RoleKrn"] = roleKrn
	params["RoleSessionName"] = sessionName

	resp, err := b.StsClient().AssumeRole(&params)
	if err != nil {
		return nil, err
	}
	assumedResult := &AssumeRoleResponse{}
	if err := mapstructureFiller(*resp, assumedResult); err != nil {
		return nil, err
	}

	return &assumedResult.AssumeRoleResult.Credentials, nil
}

// StsClient returns sts client for service
func (b *Backend) StsClient() *sts.Sts {
	if b.stsClient != nil {
		return b.stsClient
	}

	b.stsClient = sts.SdkNew(b.baseCfg.session, b.baseCfg.regionCfg, b.baseCfg.url)

	return b.stsClient
}

// TagClient returns tag client for service
func (b *Backend) TagClient() *tag.Tagv2 {
	if b.tagClient != nil {
		return b.tagClient
	}

	b.tagClient = tag.SdkNew(b.baseCfg.session, b.baseCfg.regionCfg, b.baseCfg.url)
	return b.tagClient
}

func mapstructureFiller(m interface{}, s interface{}) error {
	return mapstructure.Decode(m, s)
}
