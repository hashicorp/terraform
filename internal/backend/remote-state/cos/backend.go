// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cos

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/legacy/helper/schema"
	"github.com/mitchellh/go-homedir"
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
	PROVIDER_ENDPOINT                     = "TENCENTCLOUD_ENDPOINT"
	PROVIDER_DOMAIN                       = "TENCENTCLOUD_DOMAIN"
	PROVIDER_ASSUME_ROLE_ARN              = "TENCENTCLOUD_ASSUME_ROLE_ARN"
	PROVIDER_ASSUME_ROLE_SESSION_NAME     = "TENCENTCLOUD_ASSUME_ROLE_SESSION_NAME"
	PROVIDER_ASSUME_ROLE_SESSION_DURATION = "TENCENTCLOUD_ASSUME_ROLE_SESSION_DURATION"
	PROVIDER_ASSUME_ROLE_EXTERNAL_ID      = "TENCENTCLOUD_ASSUME_ROLE_EXTERNAL_ID"
	PROVIDER_SHARED_CREDENTIALS_DIR       = "TENCENTCLOUD_SHARED_CREDENTIALS_DIR"
	PROVIDER_PROFILE                      = "TENCENTCLOUD_PROFILE"
	PROVIDER_CAM_ROLE_NAME                = "TENCENTCLOUD_CAM_ROLE_NAME"
)

const (
	DEFAULT_PROFILE = "default"
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
	domain  string
}

type CAMResponse struct {
	TmpSecretId  string `json:"TmpSecretId"`
	TmpSecretKey string `json:"TmpSecretKey"`
	ExpiredTime  int64  `json:"ExpiredTime"`
	Expiration   string `json:"Expiration"`
	Token        string `json:"Token"`
	Code         string `json:"Code"`
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
			"endpoint": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The custom endpoint for the COS API, e.g. http://cos-internal.{Region}.tencentcos.cn. Both HTTP and HTTPS are accepted.",
				DefaultFunc: schema.EnvDefaultFunc(PROVIDER_ENDPOINT, nil),
			},
			"domain": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc(PROVIDER_DOMAIN, nil),
				Description: "The root domain of the API request. Default is tencentcloudapi.com.",
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
						"external_id": {
							Type:        schema.TypeString,
							Optional:    true,
							DefaultFunc: schema.EnvDefaultFunc(PROVIDER_ASSUME_ROLE_EXTERNAL_ID, nil),
							Description: "External role ID, which can be obtained by clicking the role name in the CAM console. It can contain 2-128 letters, digits, and symbols (=,.@:/-). Regex: [\\w+=,.@:/-]*. It can be sourced from the `TENCENTCLOUD_ASSUME_ROLE_EXTERNAL_ID`.",
						},
					},
				},
			},
			"shared_credentials_dir": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc(PROVIDER_SHARED_CREDENTIALS_DIR, nil),
				Description: "The directory of the shared credentials. It can also be sourced from the `TENCENTCLOUD_SHARED_CREDENTIALS_DIR` environment variable. If not set this defaults to ~/.tccli.",
			},
			"profile": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc(PROVIDER_PROFILE, nil),
				Description: "The profile name as set in the shared credentials. It can also be sourced from the `TENCENTCLOUD_PROFILE` environment variable. If not set, the default profile created with `tccli configure` will be used.",
			},
			"cam_role_name": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc(PROVIDER_CAM_ROLE_NAME, nil),
				Description: "The name of the CVM instance CAM role. It can be sourced from the `TENCENTCLOUD_CAM_ROLE_NAME` environment variable.",
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

	if v, ok := data.GetOk("domain"); ok {
		b.domain = v.(string)
		log.Printf("[DEBUG] Backend: set domain for TencentCloud API client. Domain: [%s]", b.domain)
	}
	// set url as endpoint when provided
	// "http://{Bucket}.cos-internal.{Region}.tencentcos.cn"
	if v, ok := data.GetOk("endpoint"); ok {
		endpoint := v.(string)

		re := regexp.MustCompile(`^(http(s)?)://cos-internal\.([^.]+)\.tencentcos\.cn$`)
		matches := re.FindStringSubmatch(endpoint)
		if len(matches) != 4 {
			return fmt.Errorf("Invalid URL: %v must be: %v", endpoint, "http(s)://cos-internal.{Region}.tencentcos.cn")
		}

		protocol := matches[1]
		region := matches[3]

		// URL after converting
		newUrl := fmt.Sprintf("%s://%s.cos-internal.%s.tencentcos.cn", protocol, b.bucket, region)
		u, err = url.Parse(newUrl)
		log.Printf("[DEBUG] Backend: set COS URL as: [%s]", newUrl)
	}
	if err != nil {
		return err
	}

	var getProviderConfig = func(key string) string {
		var str string
		value, err := getConfigFromProfile(data, key)
		if err == nil && value != nil {
			str = value.(string)
		}

		return str
	}

	var (
		secretId      string
		secretKey     string
		securityToken string
	)

	// get auth from tf/env
	if v, ok := data.GetOk("secret_id"); ok {
		secretId = v.(string)
	}

	if v, ok := data.GetOk("secret_key"); ok {
		secretKey = v.(string)
	}

	if v, ok := data.GetOk("security_token"); ok {
		securityToken = v.(string)
	}

	// get auth from tccli
	if secretId == "" && secretKey == "" && securityToken == "" {
		secretId = getProviderConfig("secretId")
		secretKey = getProviderConfig("secretKey")
		securityToken = getProviderConfig("token")
	}

	// get auth from CAM role name
	if v, ok := data.GetOk("cam_role_name"); ok {
		camRoleName := v.(string)
		if camRoleName != "" {
			camResp, err := getAuthFromCAM(camRoleName)
			if err != nil {
				return err
			}

			secretId = camResp.TmpSecretId
			secretKey = camResp.TmpSecretKey
			securityToken = camResp.Token
		}
	}

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
	var (
		assumeRoleArn             string
		assumeRoleSessionName     string
		assumeRoleSessionDuration int
		assumeRolePolicy          string
		assumeRoleExternalId      string
	)

	// get assume role from credential
	if providerConfig["role-arn"] != nil {
		assumeRoleArn = providerConfig["role-arn"].(string)
	}

	if providerConfig["role-session-name"] != nil {
		assumeRoleSessionName = providerConfig["role-session-name"].(string)
	}

	if assumeRoleArn != "" && assumeRoleSessionName != "" {
		assumeRoleSessionDuration = 7200
	}

	// get assume role from env
	envRoleArn := os.Getenv(PROVIDER_ASSUME_ROLE_ARN)
	envSessionName := os.Getenv(PROVIDER_ASSUME_ROLE_SESSION_NAME)
	if envRoleArn != "" && envSessionName != "" {
		assumeRoleArn = envRoleArn
		assumeRoleSessionName = envSessionName
		if envSessionDuration := os.Getenv(PROVIDER_ASSUME_ROLE_SESSION_DURATION); envSessionDuration != "" {
			var err error
			assumeRoleSessionDuration, err = strconv.Atoi(envSessionDuration)
			if err != nil {
				return err
			}
		}

		if assumeRoleSessionDuration == 0 {
			assumeRoleSessionDuration = 7200
		}

		assumeRoleExternalId = os.Getenv(PROVIDER_ASSUME_ROLE_EXTERNAL_ID)
	}

	// get assume role from tf
	assumeRoleList := data.Get("assume_role").(*schema.Set).List()
	if len(assumeRoleList) == 1 {
		assumeRole := assumeRoleList[0].(map[string]interface{})
		assumeRoleArn = assumeRole["role_arn"].(string)
		assumeRoleSessionName = assumeRole["session_name"].(string)
		assumeRoleSessionDuration = assumeRole["session_duration"].(int)
		assumeRolePolicy = assumeRole["policy"].(string)
		assumeRoleExternalId = assumeRole["external_id"].(string)
	}

	if assumeRoleArn != "" && assumeRoleSessionName != "" {
		err := b.updateCredentialWithSTS(assumeRoleArn, assumeRoleSessionName, assumeRoleSessionDuration, assumeRolePolicy, assumeRoleExternalId)
		if err != nil {
			return err
		}
	}

	return nil
}

func (b *Backend) updateCredentialWithSTS(assumeRoleArn, assumeRoleSessionName string, assumeRoleSessionDuration int, assumeRolePolicy string, assumeRoleExternalId string) error {
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

	if assumeRoleExternalId != "" {
		request.ExternalId = &assumeRoleExternalId
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

func getAuthFromCAM(roleName string) (camResp *CAMResponse, err error) {
	url := fmt.Sprintf("http://metadata.tencentyun.com/latest/meta-data/cam/security-credentials/%s", roleName)
	log.Printf("[CRITAL] Request CAM security credentials url: %s\n", url)
	// maximum waiting time
	client := &http.Client{Timeout: 2 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[CRITAL] Request CAM security credentials resp err: %s", err.Error())
		return
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[CRITAL] Request CAM security credentials body read err: %s", err.Error())
		return
	}

	err = json.Unmarshal(body, &camResp)
	if err != nil {
		log.Printf("[CRITAL] Request CAM security credentials resp json err: %s", err.Error())
		return
	}

	return
}

var providerConfig map[string]interface{}

func getConfigFromProfile(d *schema.ResourceData, ProfileKey string) (interface{}, error) {
	if providerConfig == nil {
		var (
			profile              string
			sharedCredentialsDir string
			credentialPath       string
			configurePath        string
		)

		if v, ok := d.GetOk("profile"); ok {
			profile = v.(string)
		} else {
			profile = DEFAULT_PROFILE
		}

		if v, ok := d.GetOk("shared_credentials_dir"); ok {
			sharedCredentialsDir = v.(string)
		}

		tmpSharedCredentialsDir, err := homedir.Expand(sharedCredentialsDir)
		if err != nil {
			return nil, err
		}

		if tmpSharedCredentialsDir == "" {
			credentialPath = fmt.Sprintf("%s/.tccli/%s.credential", os.Getenv("HOME"), profile)
			configurePath = fmt.Sprintf("%s/.tccli/%s.configure", os.Getenv("HOME"), profile)
			if runtime.GOOS == "windows" {
				credentialPath = fmt.Sprintf("%s/.tccli/%s.credential", os.Getenv("USERPROFILE"), profile)
				configurePath = fmt.Sprintf("%s/.tccli/%s.configure", os.Getenv("USERPROFILE"), profile)
			}
		} else {
			credentialPath = fmt.Sprintf("%s/%s.credential", tmpSharedCredentialsDir, profile)
			configurePath = fmt.Sprintf("%s/%s.configure", tmpSharedCredentialsDir, profile)
		}

		providerConfig = make(map[string]interface{})
		_, err = os.Stat(credentialPath)
		if !os.IsNotExist(err) {
			data, err := ioutil.ReadFile(credentialPath)
			if err != nil {
				return nil, err
			}

			config := map[string]interface{}{}
			err = json.Unmarshal(data, &config)
			if err != nil {
				return nil, err
			}

			for k, v := range config {
				if strValue, ok := v.(string); ok {
					providerConfig[k] = strings.TrimSpace(strValue)
				}
			}
		}

		_, err = os.Stat(configurePath)
		if !os.IsNotExist(err) {
			data, err := ioutil.ReadFile(configurePath)
			if err != nil {
				return nil, err
			}

			config := map[string]interface{}{}
			err = json.Unmarshal(data, &config)
			if err != nil {
				return nil, err
			}

		outerLoop:
			for k, v := range config {
				if k == "_sys_param" {
					tmpMap := v.(map[string]interface{})
					for tmpK, tmpV := range tmpMap {
						if tmpK == "region" {
							providerConfig[tmpK] = strings.TrimSpace(tmpV.(string))
							break outerLoop
						}
					}
				}
			}
		}
	}

	return providerConfig[ProfileKey], nil
}
