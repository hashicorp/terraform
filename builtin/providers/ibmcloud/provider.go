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
				DefaultFunc: schema.EnvDefaultFunc("IBMID", ""),
			},
			"password": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The password for the IBM ID.",
				DefaultFunc: schema.EnvDefaultFunc("IBMID_PASSWORD", ""),
			},
			"region": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The Bluemix Region (for example 'ng').",
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{"BM_REGION", "BLUEMIX_REGION"}, "ng"),
			},
			"timeout": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "The timeout (in seconds) to set for any Bluemix API calls made.",
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{"BM_TIMEOUT", "BLUEMIX_TIMEOUT"}, 60),
			},
			"softlayer_username": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The SoftLayer user name.",
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{"SL_USERNAME", "SOFTLAYER_USERNAME"}, ""),
			},
			"softlayer_api_key": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The API key for SoftLayer API operations.",
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{"SL_API_KEY", "SOFTLAYER_API_KEY"}, ""),
			},
			"softlayer_endpoint_url": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The endpoint url for the SoftLayer API.",
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{"SL_ENDPOINT_URL", "SOFTLAYER_ENDPOINT_URL"},
					"https://api.softlayer.com/rest/v3"),
			},
			"softlayer_timeout": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "The timeout (in seconds) to set for any SoftLayer API calls made.",
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{"SL_TIMEOUT", "SOFTLAYER_TIMEOUT"}, 60),
			},
			"softlayer_account_number": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The SoftLayer IMS account number.",
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{"SL_ACCOUNT_NUMBER", "SOFTLAYER_ACCOUNT_NUMBER"}, ""),
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
	timeout := strconv.Itoa(d.Get("timeout").(int))
	softlayerTimeout := strconv.Itoa(d.Get("softlayer_timeout").(int))

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
