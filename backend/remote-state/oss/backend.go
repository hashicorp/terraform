package oss

import (
	"context"
	"fmt"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/helper/schema"
	"os"
	"strings"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/auth/credentials"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/resource"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/utils"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/location"
	"github.com/hashicorp/go-cleanhttp"
	"github.com/hashicorp/terraform/version"
	"log"
	"net/http"
	"strconv"
	"time"
)

// New creates a new backend for OSS remote state.
func New() backend.Backend {
	s := &schema.Backend{
		Schema: map[string]*schema.Schema{
			"access_key": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Alibaba Cloud Access Key ID",
				DefaultFunc: schema.EnvDefaultFunc("ALICLOUD_ACCESS_KEY", os.Getenv("ALICLOUD_ACCESS_KEY_ID")),
			},

			"secret_key": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Alibaba Cloud Access Secret Key",
				DefaultFunc: schema.EnvDefaultFunc("ALICLOUD_SECRET_KEY", os.Getenv("ALICLOUD_ACCESS_KEY_SECRET")),
			},

			"security_token": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Alibaba Cloud Security Token",
				DefaultFunc: schema.EnvDefaultFunc("ALICLOUD_SECURITY_TOKEN", os.Getenv("SECURITY_TOKEN")),
			},

			"region": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The region of the OSS bucket.",
				DefaultFunc: schema.EnvDefaultFunc("ALICLOUD_REGION", os.Getenv("ALICLOUD_DEFAULT_REGION")),
			},

			"endpoint": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A custom endpoint for the OSS API",
				DefaultFunc: schema.EnvDefaultFunc("ALICLOUD_OSS_ENDPOINT", os.Getenv("OSS_ENDPOINT")),
			},

			"bucket": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the OSS bucket",
			},

			"path": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "The path relative to your object storage directory where the state file will be stored.",
			},

			"name": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The name of the state file inside the bucket",
				ValidateFunc: func(v interface{}, s string) ([]string, []error) {
					if strings.HasPrefix(v.(string), "/") || strings.HasSuffix(v.(string), "/") {
						return nil, []error{fmt.Errorf("name can not start and end with '/'")}
					}
					return nil, nil
				},
				Default: "terraform.tfstate",
			},

			"lock": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Whether to lock state access. Defaults to true",
				Default:     true,
			},

			"encrypt": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Whether to enable server side encryption of the state file",
				Default:     false,
			},

			"acl": &schema.Schema{
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

	bucketName           string
	statePath            string
	stateName            string
	serverSideEncryption bool
	acl                  string
	endpoint             string
	lock                 bool
}

func (b *Backend) configure(ctx context.Context) error {
	if b.ossClient != nil {
		return nil
	}

	// Grab the resource data
	d := schema.FromContextBackendConfig(ctx)

	b.bucketName = d.Get("bucket").(string)
	dir := strings.Trim(d.Get("path").(string), "/")
	if strings.HasPrefix(dir, "./") {
		dir = strings.TrimPrefix(dir, "./")

	}

	b.statePath = dir
	b.stateName = d.Get("name").(string)
	b.serverSideEncryption = d.Get("encrypt").(bool)
	b.acl = d.Get("acl").(string)
	b.lock = d.Get("lock").(bool)

	access_key := d.Get("access_key").(string)
	secret_key := d.Get("secret_key").(string)
	security_token := d.Get("security_token").(string)
	region := d.Get("region").(string)
	endpoint := d.Get("endpoint").(string)
	schma := "https"

	if endpoint == "" {
		endpointItem, _ := b.getOSSEndpointByRegion(access_key, secret_key, security_token, region)
		if endpointItem != nil && len(endpointItem.Endpoint) > 0 {
			if len(endpointItem.Protocols.Protocols) > 0 {
				// HTTP or HTTPS
				schma = strings.ToLower(endpointItem.Protocols.Protocols[0])
				for _, p := range endpointItem.Protocols.Protocols {
					if strings.ToLower(p) == "https" {
						schma = strings.ToLower(p)
						break
					}
				}
			}
			endpoint = endpointItem.Endpoint
		} else {
			endpoint = fmt.Sprintf("oss-%s.aliyuncs.com", region)
		}
	}
	if !strings.HasPrefix(endpoint, "http") {
		endpoint = fmt.Sprintf("%s://%s", schma, endpoint)
	}
	log.Printf("[DEBUG] Instantiate OSS client using endpoint: %#v", endpoint)
	var options []oss.ClientOption
	if security_token != "" {
		options = append(options, oss.SecurityToken(security_token))
	}
	options = append(options, oss.UserAgent(fmt.Sprintf("%s/%s", TerraformUA, TerraformVersion)))

	client, err := oss.New(endpoint, access_key, secret_key, options...)
	b.ossClient = client

	return err
}

func (b *Backend) getOSSEndpointByRegion(access_key, secret_key, security_token, region string) (*location.DescribeEndpointResponse, error) {
	args := location.CreateDescribeEndpointRequest()
	args.ServiceCode = "oss"
	args.Id = region
	args.Domain = "location-readonly.aliyuncs.com"

	locationClient, err := location.NewClientWithOptions(region, getSdkConfig(), credentials.NewStsTokenCredential(access_key, secret_key, security_token))
	if err != nil {
		return nil, fmt.Errorf("Unable to initialize the location client: %#v", err)

	}
	locationClient.AppendUserAgent(TerraformUA, TerraformVersion)
	endpointsResponse, err := locationClient.DescribeEndpoint(args)
	if err != nil {
		return nil, fmt.Errorf("Describe oss endpoint using region: %#v got an error: %#v.", region, err)
	}
	return endpointsResponse, nil
}

func getSdkConfig() *sdk.Config {
	// Fix bug "open /usr/local/go/lib/time/zoneinfo.zip: no such file or directory" which happened in windows.
	if data, ok := resource.GetTZData("GMT"); ok {
		utils.TZData = data
		utils.LoadLocationFromTZData = time.LoadLocationFromTZData
	}
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
				return fmt.Errorf("Retry timeout and got an error: %#v.", err)
			} else {
				time.Sleep(time.Duration(catcher.RetryWaitSeconds) * time.Second)
				return a.Run(f)
			}
		}
	}
	return err
}
