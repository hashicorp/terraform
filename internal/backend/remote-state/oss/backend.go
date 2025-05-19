// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package oss

import (
	"context"
	"encoding/json"
	"fmt"
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

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/endpoints"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/auth/credentials"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/responses"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/location"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/sts"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/aliyun/aliyun-tablestore-go-sdk/tablestore"
	"github.com/hashicorp/go-cleanhttp"
	"github.com/jmespath/go-jmespath"
	"github.com/mitchellh/go-homedir"

	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/legacy/helper/schema"
	"github.com/hashicorp/terraform/version"
)

// Deprecated in favor of flattening assume_role_* options
func deprecatedAssumeRoleSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeSet,
		Optional: true,
		MaxItems: 1,
		//Deprecated: "use assume_role_* options instead",
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"role_arn": {
					Type:        schema.TypeString,
					Required:    true,
					Description: "The ARN of a RAM role to assume prior to making API calls.",
					DefaultFunc: schema.EnvDefaultFunc("ALICLOUD_ASSUME_ROLE_ARN", ""),
				},
				"session_name": {
					Type:        schema.TypeString,
					Optional:    true,
					Description: "The session name to use when assuming the role.",
					DefaultFunc: schema.EnvDefaultFunc("ALICLOUD_ASSUME_ROLE_SESSION_NAME", ""),
				},
				"policy": {
					Type:        schema.TypeString,
					Optional:    true,
					Description: "The permissions applied when assuming a role. You cannot use this policy to grant permissions which exceed those of the role that is being assumed.",
				},
				"session_expiration": {
					Type:        schema.TypeInt,
					Optional:    true,
					Description: "The time after which the established session for assuming role expires.",
					ValidateFunc: func(v interface{}, k string) ([]string, []error) {
						min := 900
						max := 3600
						value, ok := v.(int)
						if !ok {
							return nil, []error{fmt.Errorf("expected type of %s to be int", k)}
						}

						if value < min || value > max {
							return nil, []error{fmt.Errorf("expected %s to be in the range (%d - %d), got %d", k, min, max, v)}
						}

						return nil, nil
					},
				},
			},
		},
	}
}

// New creates a new backend for OSS remote state.
func New() backend.Backend {
	s := &schema.Backend{
		Schema: map[string]*schema.Schema{
			"access_key": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Alibaba Cloud Access Key ID",
				DefaultFunc: schema.EnvDefaultFunc("ALICLOUD_ACCESS_KEY", os.Getenv("ALICLOUD_ACCESS_KEY_ID")),
			},

			"secret_key": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Alibaba Cloud Access Secret Key",
				DefaultFunc: schema.EnvDefaultFunc("ALICLOUD_SECRET_KEY", os.Getenv("ALICLOUD_ACCESS_KEY_SECRET")),
			},

			"security_token": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Alibaba Cloud Security Token",
				DefaultFunc: schema.EnvDefaultFunc("ALICLOUD_SECURITY_TOKEN", ""),
			},

			"ecs_role_name": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("ALICLOUD_ECS_ROLE_NAME", os.Getenv("ALICLOUD_ECS_ROLE_NAME")),
				Description: "The RAM Role Name attached on a ECS instance for API operations. You can retrieve this from the 'Access Control' section of the Alibaba Cloud console.",
			},

			"region": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The region of the OSS bucket.",
				DefaultFunc: schema.EnvDefaultFunc("ALICLOUD_REGION", os.Getenv("ALICLOUD_DEFAULT_REGION")),
			},
			"sts_endpoint": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A custom endpoint for the STS API",
				DefaultFunc: schema.EnvDefaultFunc("ALICLOUD_STS_ENDPOINT", ""),
			},
			"tablestore_endpoint": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A custom endpoint for the TableStore API",
				DefaultFunc: schema.EnvDefaultFunc("ALICLOUD_TABLESTORE_ENDPOINT", ""),
			},
			"endpoint": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A custom endpoint for the OSS API",
				DefaultFunc: schema.EnvDefaultFunc("ALICLOUD_OSS_ENDPOINT", os.Getenv("OSS_ENDPOINT")),
			},

			"bucket": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the OSS bucket",
			},

			"prefix": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The directory where state files will be saved inside the bucket",
				Default:     "env:",
				ValidateFunc: func(v interface{}, s string) ([]string, []error) {
					prefix := v.(string)
					if strings.HasPrefix(prefix, "/") || strings.HasPrefix(prefix, "./") {
						return nil, []error{fmt.Errorf("workspace_key_prefix must not start with '/' or './'")}
					}
					return nil, nil
				},
			},

			"key": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The path of the state file inside the bucket",
				ValidateFunc: func(v interface{}, s string) ([]string, []error) {
					if strings.HasPrefix(v.(string), "/") || strings.HasSuffix(v.(string), "/") {
						return nil, []error{fmt.Errorf("key can not start and end with '/'")}
					}
					return nil, nil
				},
				Default: "terraform.tfstate",
			},

			"tablestore_table": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "TableStore table for state locking and consistency",
				Default:     "",
			},

			"encrypt": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Whether to enable server side encryption of the state file",
				Default:     false,
			},

			"acl": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Object ACL to be applied to the state file",
				Default:     "",
				ValidateFunc: func(v interface{}, k string) ([]string, []error) {
					if value := v.(string); value != "" {
						acls := oss.ACLType(value)
						if acls != oss.ACLPrivate && acls != oss.ACLPublicRead && acls != oss.ACLPublicReadWrite {
							return nil, []error{fmt.Errorf(
								"%q must be a valid ACL value , expected %s, %s or %s, got %q",
								k, oss.ACLPrivate, oss.ACLPublicRead, oss.ACLPublicReadWrite, acls)}
						}
					}
					return nil, nil
				},
			},
			"shared_credentials_file": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("ALICLOUD_SHARED_CREDENTIALS_FILE", ""),
				Description: "This is the path to the shared credentials file. If this is not set and a profile is specified, `~/.aliyun/config.json` will be used.",
			},
			"profile": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "This is the Alibaba Cloud profile name as set in the shared credentials file. It can also be sourced from the `ALICLOUD_PROFILE` environment variable.",
				DefaultFunc: schema.EnvDefaultFunc("ALICLOUD_PROFILE", ""),
			},
			"assume_role": deprecatedAssumeRoleSchema(),
			"assume_role_role_arn": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The ARN of a RAM role to assume prior to making API calls.",
				DefaultFunc: schema.EnvDefaultFunc("ALICLOUD_ASSUME_ROLE_ARN", ""),
			},
			"assume_role_session_name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The session name to use when assuming the role.",
				DefaultFunc: schema.EnvDefaultFunc("ALICLOUD_ASSUME_ROLE_SESSION_NAME", ""),
			},
			"assume_role_policy": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The permissions applied when assuming a role. You cannot use this policy to grant permissions which exceed those of the role that is being assumed.",
			},
			"assume_role_session_expiration": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "The time after which the established session for assuming role expires.",
				ValidateFunc: func(v interface{}, k string) ([]string, []error) {
					min := 900
					max := 3600
					value, ok := v.(int)
					if !ok {
						return nil, []error{fmt.Errorf("expected type of %s to be int", k)}
					}

					if value < min || value > max {
						return nil, []error{fmt.Errorf("expected %s to be in the range (%d - %d), got %d", k, min, max, v)}
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

type Backend struct {
	*schema.Backend

	// The fields below are set from configure
	ossClient *oss.Client
	otsClient *tablestore.TableStoreClient

	bucketName           string
	statePrefix          string
	stateKey             string
	serverSideEncryption bool
	acl                  string
	otsEndpoint          string
	otsTable             string
}

func (b *Backend) configure(ctx context.Context) error {
	if b.ossClient != nil {
		return nil
	}

	// Grab the resource data
	d := schema.FromContextBackendConfig(ctx)

	b.bucketName = d.Get("bucket").(string)
	b.statePrefix = strings.TrimPrefix(strings.Trim(d.Get("prefix").(string), "/"), "./")
	b.stateKey = d.Get("key").(string)
	b.serverSideEncryption = d.Get("encrypt").(bool)
	b.acl = d.Get("acl").(string)

	var getBackendConfig = func(str string, key string) string {
		if str == "" {
			value, err := getConfigFromProfile(d, key)
			if err == nil && value != nil {
				str = value.(string)
			}
		}
		return str
	}

	accessKey := getBackendConfig(d.Get("access_key").(string), "access_key_id")
	secretKey := getBackendConfig(d.Get("secret_key").(string), "access_key_secret")
	securityToken := getBackendConfig(d.Get("security_token").(string), "sts_token")
	region := getBackendConfig(d.Get("region").(string), "region_id")

	stsEndpoint := d.Get("sts_endpoint").(string)
	endpoint := d.Get("endpoint").(string)
	schma := "https"

	roleArn := getBackendConfig("", "ram_role_arn")
	sessionName := getBackendConfig("", "ram_session_name")
	var policy string
	var sessionExpiration int
	expiredSeconds, err := getConfigFromProfile(d, "expired_seconds")
	if err == nil && expiredSeconds != nil {
		sessionExpiration = (int)(expiredSeconds.(float64))
	}

	if v, ok := d.GetOk("assume_role_role_arn"); ok && v.(string) != "" {
		roleArn = v.(string)
		if v, ok := d.GetOk("assume_role_session_name"); ok {
			sessionName = v.(string)
		}
		if v, ok := d.GetOk("assume_role_policy"); ok {
			policy = v.(string)
		}
		if v, ok := d.GetOk("assume_role_session_expiration"); ok {
			sessionExpiration = v.(int)
		}
	} else if v, ok := d.GetOk("assume_role"); ok {
		// deprecated assume_role block
		for _, v := range v.(*schema.Set).List() {
			assumeRole := v.(map[string]interface{})
			if assumeRole["role_arn"].(string) != "" {
				roleArn = assumeRole["role_arn"].(string)
			}
			if assumeRole["session_name"].(string) != "" {
				sessionName = assumeRole["session_name"].(string)
			}
			policy = assumeRole["policy"].(string)
			sessionExpiration = assumeRole["session_expiration"].(int)
		}
	}

	if sessionName == "" {
		sessionName = "terraform"
	}
	if sessionExpiration == 0 {
		if v := os.Getenv("ALICLOUD_ASSUME_ROLE_SESSION_EXPIRATION"); v != "" {
			if expiredSeconds, err := strconv.Atoi(v); err == nil {
				sessionExpiration = expiredSeconds
			}
		}
		if sessionExpiration == 0 {
			sessionExpiration = 3600
		}
	}

	if accessKey == "" {
		ecsRoleName := getBackendConfig(d.Get("ecs_role_name").(string), "ram_role_name")
		subAccessKeyId, subAccessKeySecret, subSecurityToken, err := getAuthCredentialByEcsRoleName(ecsRoleName)
		if err != nil {
			return err
		}
		accessKey, secretKey, securityToken = subAccessKeyId, subAccessKeySecret, subSecurityToken
	}

	if roleArn != "" {
		subAccessKeyId, subAccessKeySecret, subSecurityToken, err := getAssumeRoleAK(accessKey, secretKey, securityToken, region, roleArn, sessionName, policy, stsEndpoint, sessionExpiration)
		if err != nil {
			return err
		}
		accessKey, secretKey, securityToken = subAccessKeyId, subAccessKeySecret, subSecurityToken
	}

	if endpoint == "" {
		endpointsResponse, err := b.getOSSEndpointByRegion(accessKey, secretKey, securityToken, region)
		if err != nil {
			log.Printf("[WARN] getting oss endpoint failed and using oss-%s.aliyuncs.com instead. Error: %#v.", region, err)
		} else {
			for _, endpointItem := range endpointsResponse.Endpoints.Endpoint {
				if endpointItem.Type == "openAPI" {
					endpoint = endpointItem.Endpoint
					break
				}
			}
		}
		if endpoint == "" {
			endpoint = fmt.Sprintf("oss-%s.aliyuncs.com", region)
		}
	}
	if !strings.HasPrefix(endpoint, "http") {
		endpoint = fmt.Sprintf("%s://%s", schma, endpoint)
	}
	log.Printf("[DEBUG] Instantiate OSS client using endpoint: %#v", endpoint)
	var options []oss.ClientOption
	if securityToken != "" {
		options = append(options, oss.SecurityToken(securityToken))
	}
	options = append(options, oss.UserAgent(fmt.Sprintf("%s/%s", TerraformUA, TerraformVersion)))

	proxyUrl := getHttpProxyUrl()
	if proxyUrl != nil {
		options = append(options, oss.Proxy(proxyUrl.String()))
	}

	client, err := oss.New(endpoint, accessKey, secretKey, options...)
	b.ossClient = client
	otsEndpoint := d.Get("tablestore_endpoint").(string)
	if otsEndpoint != "" {
		if !strings.HasPrefix(otsEndpoint, "http") {
			otsEndpoint = fmt.Sprintf("%s://%s", schma, otsEndpoint)
		}
		b.otsEndpoint = otsEndpoint
		parts := strings.Split(strings.TrimPrefix(strings.TrimPrefix(otsEndpoint, "https://"), "http://"), ".")
		b.otsClient = tablestore.NewClientWithConfig(otsEndpoint, parts[0], accessKey, secretKey, securityToken, tablestore.NewDefaultTableStoreConfig())
	}
	b.otsTable = d.Get("tablestore_table").(string)

	return err
}

func (b *Backend) getOSSEndpointByRegion(access_key, secret_key, security_token, region string) (*location.DescribeEndpointsResponse, error) {
	args := location.CreateDescribeEndpointsRequest()
	args.ServiceCode = "oss"
	args.Id = region
	args.Domain = "location-readonly.aliyuncs.com"

	locationClient, err := location.NewClientWithOptions(region, getSdkConfig(), credentials.NewStsTokenCredential(access_key, secret_key, security_token))
	if err != nil {
		return nil, fmt.Errorf("unable to initialize the location client: %#v", err)

	}
	locationClient.AppendUserAgent(TerraformUA, TerraformVersion)
	endpointsResponse, err := locationClient.DescribeEndpoints(args)
	if err != nil {
		return nil, fmt.Errorf("describe oss endpoint using region: %#v got an error: %#v", region, err)
	}
	return endpointsResponse, nil
}

func getAssumeRoleAK(accessKey, secretKey, stsToken, region, roleArn, sessionName, policy, stsEndpoint string, sessionExpiration int) (string, string, string, error) {
	request := sts.CreateAssumeRoleRequest()
	request.RoleArn = roleArn
	request.RoleSessionName = sessionName
	request.DurationSeconds = requests.NewInteger(sessionExpiration)
	request.Policy = policy
	request.Scheme = "https"

	var client *sts.Client
	var err error
	if stsToken == "" {
		client, err = sts.NewClientWithAccessKey(region, accessKey, secretKey)
	} else {
		client, err = sts.NewClientWithStsToken(region, accessKey, secretKey, stsToken)
	}
	if err != nil {
		return "", "", "", err
	}
	if stsEndpoint != "" {
		endpoints.AddEndpointMapping(region, "STS", stsEndpoint)
	}
	response, err := client.AssumeRole(request)
	if err != nil {
		return "", "", "", err
	}
	return response.Credentials.AccessKeyId, response.Credentials.AccessKeySecret, response.Credentials.SecurityToken, nil
}

func getSdkConfig() *sdk.Config {
	return sdk.NewConfig().
		WithMaxRetryTime(5).
		WithTimeout(time.Duration(30) * time.Second).
		WithGoRoutinePoolSize(10).
		WithDebug(false).
		WithHttpTransport(getTransport()).
		WithScheme("HTTPS")
}

func getTransport() *http.Transport {
	handshakeTimeout, err := strconv.Atoi(os.Getenv("TLSHandshakeTimeout"))
	if err != nil {
		handshakeTimeout = 120
	}
	transport := cleanhttp.DefaultTransport()
	transport.TLSHandshakeTimeout = time.Duration(handshakeTimeout) * time.Second
	transport.Proxy = http.ProxyFromEnvironment
	return transport
}

type Invoker struct {
	catchers []*Catcher
}

type Catcher struct {
	Reason           string
	RetryCount       int
	RetryWaitSeconds int
}

const TerraformUA = "HashiCorp-Terraform"

var TerraformVersion = strings.TrimSuffix(version.String(), "-dev")
var ClientErrorCatcher = Catcher{"AliyunGoClientFailure", 10, 3}
var ServiceBusyCatcher = Catcher{"ServiceUnavailable", 10, 3}

func NewInvoker() Invoker {
	i := Invoker{}
	i.AddCatcher(ClientErrorCatcher)
	i.AddCatcher(ServiceBusyCatcher)
	return i
}

func (a *Invoker) AddCatcher(catcher Catcher) {
	a.catchers = append(a.catchers, &catcher)
}

func (a *Invoker) Run(f func() error) error {
	err := f()

	if err == nil {
		return nil
	}

	for _, catcher := range a.catchers {
		if strings.Contains(err.Error(), catcher.Reason) {
			catcher.RetryCount--

			if catcher.RetryCount <= 0 {
				return fmt.Errorf("retry timeout and got an error: %#v", err)
			} else {
				time.Sleep(time.Duration(catcher.RetryWaitSeconds) * time.Second)
				return a.Run(f)
			}
		}
	}
	return err
}

var providerConfig map[string]interface{}

func getConfigFromProfile(d *schema.ResourceData, ProfileKey string) (interface{}, error) {

	if providerConfig == nil {
		if v, ok := d.GetOk("profile"); !ok || v.(string) == "" {
			return nil, nil
		}
		current := d.Get("profile").(string)
		// Set CredsFilename, expanding home directory
		profilePath, err := homedir.Expand(d.Get("shared_credentials_file").(string))
		if err != nil {
			return nil, err
		}
		if profilePath == "" {
			profilePath = fmt.Sprintf("%s/.aliyun/config.json", os.Getenv("HOME"))
			if runtime.GOOS == "windows" {
				profilePath = fmt.Sprintf("%s/.aliyun/config.json", os.Getenv("USERPROFILE"))
			}
		}
		providerConfig = make(map[string]interface{})
		_, err = os.Stat(profilePath)
		if !os.IsNotExist(err) {
			data, err := ioutil.ReadFile(profilePath)
			if err != nil {
				return nil, err
			}
			config := map[string]interface{}{}
			err = json.Unmarshal(data, &config)
			if err != nil {
				return nil, err
			}
			for _, v := range config["profiles"].([]interface{}) {
				if current == v.(map[string]interface{})["name"] {
					providerConfig = v.(map[string]interface{})
				}
			}
		}
	}

	mode := ""
	if v, ok := providerConfig["mode"]; ok {
		mode = v.(string)
	} else {
		return v, nil
	}
	switch ProfileKey {
	case "access_key_id", "access_key_secret":
		if mode == "EcsRamRole" {
			return "", nil
		}
	case "ram_role_name":
		if mode != "EcsRamRole" {
			return "", nil
		}
	case "sts_token":
		if mode != "StsToken" {
			return "", nil
		}
	case "ram_role_arn", "ram_session_name":
		if mode != "RamRoleArn" {
			return "", nil
		}
	case "expired_seconds":
		if mode != "RamRoleArn" {
			return float64(0), nil
		}
	}

	return providerConfig[ProfileKey], nil
}

var securityCredURL = "http://100.100.100.200/latest/meta-data/ram/security-credentials/"

// getAuthCredentialByEcsRoleName aims to access meta to get sts credential
// Actually, the job should be done by sdk, but currently not all resources and products support alibaba-cloud-sdk-go,
// and their go sdk does support ecs role name.
// This method is a temporary solution and it should be removed after all go sdk support ecs role name
// The related PR: https://github.com/terraform-providers/terraform-provider-alicloud/pull/731
func getAuthCredentialByEcsRoleName(ecsRoleName string) (accessKey, secretKey, token string, err error) {

	if ecsRoleName == "" {
		return
	}
	requestUrl := securityCredURL + ecsRoleName
	httpRequest, err := http.NewRequest(requests.GET, requestUrl, strings.NewReader(""))
	if err != nil {
		err = fmt.Errorf("build sts requests err: %s", err.Error())
		return
	}
	httpClient := &http.Client{}
	httpResponse, err := httpClient.Do(httpRequest)
	if err != nil {
		err = fmt.Errorf("get Ecs sts token err : %s", err.Error())
		return
	}

	response := responses.NewCommonResponse()
	err = responses.Unmarshal(response, httpResponse, "")
	if err != nil {
		err = fmt.Errorf("unmarshal Ecs sts token response err : %s", err.Error())
		return
	}

	if response.GetHttpStatus() != http.StatusOK {
		err = fmt.Errorf("get Ecs sts token err, httpStatus: %d, message = %s", response.GetHttpStatus(), response.GetHttpContentString())
		return
	}
	var data interface{}
	err = json.Unmarshal(response.GetHttpContentBytes(), &data)
	if err != nil {
		err = fmt.Errorf("refresh Ecs sts token err, json.Unmarshal fail: %s", err.Error())
		return
	}
	code, err := jmespath.Search("Code", data)
	if err != nil {
		err = fmt.Errorf("refresh Ecs sts token err, fail to get Code: %s", err.Error())
		return
	}
	if code.(string) != "Success" {
		err = fmt.Errorf("refresh Ecs sts token err, Code is not Success")
		return
	}
	accessKeyId, err := jmespath.Search("AccessKeyId", data)
	if err != nil {
		err = fmt.Errorf("refresh Ecs sts token err, fail to get AccessKeyId: %s", err.Error())
		return
	}
	accessKeySecret, err := jmespath.Search("AccessKeySecret", data)
	if err != nil {
		err = fmt.Errorf("refresh Ecs sts token err, fail to get AccessKeySecret: %s", err.Error())
		return
	}
	securityToken, err := jmespath.Search("SecurityToken", data)
	if err != nil {
		err = fmt.Errorf("refresh Ecs sts token err, fail to get SecurityToken: %s", err.Error())
		return
	}

	if accessKeyId == nil || accessKeySecret == nil || securityToken == nil {
		err = fmt.Errorf("there is no any available accesskey, secret and security token for Ecs role %s", ecsRoleName)
		return
	}

	return accessKeyId.(string), accessKeySecret.(string), securityToken.(string), nil
}

func getHttpProxyUrl() *url.URL {
	for _, v := range []string{"HTTPS_PROXY", "https_proxy", "HTTP_PROXY", "http_proxy"} {
		value := strings.Trim(os.Getenv(v), " ")
		if value != "" {
			if !regexp.MustCompile(`^http(s)?://`).MatchString(value) {
				value = fmt.Sprintf("https://%s", value)
			}
			proxyUrl, err := url.Parse(value)
			if err == nil {
				return proxyUrl
			}
			break
		}
	}
	return nil
}
