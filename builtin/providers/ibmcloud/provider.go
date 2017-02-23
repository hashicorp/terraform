package ibmcloud

import (
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

// Provider returns a terraform.ResourceProvider.
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"ibmid": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The IBM ID.",
				DefaultFunc: func() (interface{}, error) {
					return ValueFromEnv("ibmid"), nil
				},
			},
			"password": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The password for the IBM ID.",
				DefaultFunc: func() (interface{}, error) {
					return ValueFromEnv("password"), nil
				},
			},
			"region": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The Bluemix Region (for example 'ng').",
				DefaultFunc: func() (interface{}, error) {
					return ValueFromEnv("region"), nil
				},
			},
			"timeout": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "The timeout (in seconds) to set for any BlueMix API calls made.",
			},
			"softlayer_username": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The SoftLayer user name.",
				DefaultFunc: func() (interface{}, error) {
					return ValueFromEnv("softlayer_username"), nil
				},
			},
			"softlayer_api_key": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The API key for SoftLayer API operations.",
				DefaultFunc: func() (interface{}, error) {
					return ValueFromEnv("softlayer_api_key"), nil
				},
			},
			"softlayer_endpoint_url": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The endpoint url for the SoftLayer API.",
				DefaultFunc: func() (interface{}, error) {
					return "", nil
				},
			},
			"softlayer_timeout": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "The timeout (in seconds) to set for any SoftLayer API calls made.",
			},
			"softlayer_account_number": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The SoftLayer IMS account number.",
				DefaultFunc: func() (interface{}, error) {
					return ValueFromEnv("softlayer_account_number"), nil
				},
			},
		},

		DataSourcesMap: map[string]*schema.Resource{
			"ibmcloud_infra_ssh_key": dataSourceIBMCloudInfraSSHKey(),
		},

		ResourcesMap: map[string]*schema.Resource{
			"ibmcloud_infra_ssh_key":       resourceIBMCloudInfraSSHKey(),
			"ibmcloud_infra_virtual_guest": resourceIBMCloudInfraVirtualGuest(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	timeout := ""
	softlayerTimeout := ""
	if rawTimeout, ok := d.GetOk("timeout"); ok {
		timeout = strconv.Itoa(rawTimeout.(int))
		envFallback(&timeout, "timeout")
	}
	if rawSoftlayerTimeout, ok := d.GetOk("softlayer_timeout"); ok {
		softlayerTimeout = strconv.Itoa(rawSoftlayerTimeout.(int))
	}

	region := d.Get("region").(string)
	bmDomainName := fmt.Sprintf("%s.bluemix.net", region)
	bluemixAPIEndpoint := fmt.Sprintf("https://login.%s/UAALoginServerWAR", bmDomainName)
	iamEndpoint := fmt.Sprintf("https://iam.%s", bmDomainName)

	config := Config{
		IBMID:                  d.Get("ibmid").(string),
		Password:               d.Get("password").(string),
		Region:                 d.Get("region").(string),
		Timeout:                timeout,
		SoftLayerAPIKey:        d.Get("softlayer_api_key").(string),
		SoftLayerUsername:      d.Get("softlayer_username").(string),
		SoftLayerEndpointURL:   d.Get("softlayer_endpoint_url").(string),
		SoftLayerTimeout:       softlayerTimeout,
		SoftLayerAccountNumber: d.Get("softlayer_account_number").(string),
		Endpoint:               bluemixAPIEndpoint,
		IAMEndpoint:            iamEndpoint,
	}

	return config.ClientSession()

}
