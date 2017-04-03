package ibmcloud

import (
	"fmt"
	"time"

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
			"ibmid_password": {
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
			"softlayer_timeout": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "The timeout (in seconds) to set for any SoftLayer API calls made.",
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{"SL_TIMEOUT", "SOFTLAYER_TIMEOUT"}, 60),
			},
			"softlayer_account_number": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The SoftLayer IMS account number linked with IBM ID.",
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{"SL_ACCOUNT_NUMBER", "SOFTLAYER_ACCOUNT_NUMBER"}, ""),
			},
		},

		DataSourcesMap: map[string]*schema.Resource{
			"ibmcloud_infra_ssh_key": dataSourceIBMCloudInfraSSHKey(),
			"ibmcloud_infra_vlan":    dataSourceIBMCloudInfraVlan(),
		},

		ResourcesMap: map[string]*schema.Resource{
			"ibmcloud_infra_ssh_key":       resourceIBMCloudInfraSSHKey(),
			"ibmcloud_infra_virtual_guest": resourceIBMCloudInfraVirtualGuest(),
			"ibmcloud_infra_vlan":          resourceIBMCloudInfraVlan(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	softlayerTimeout := d.Get("softlayer_timeout").(int)

	region := d.Get("region").(string)
	domain := fmt.Sprintf("%s.bluemix.net", region)
	iamEndpoint := fmt.Sprintf("https://iam.%s", domain)

	config := Config{
		IBMID:                   d.Get("ibmid").(string),
		IBMIDPassword:           d.Get("ibmid_password").(string),
		Region:                  d.Get("region").(string),
		SoftLayerTimeout:        time.Duration(softlayerTimeout) * time.Second,
		SoftLayerAccountNumber:  d.Get("softlayer_account_number").(string),
		IAMEndpoint:             iamEndpoint,
		RetryCount:              10,
		RetryDelay:              20 * time.Millisecond,
		SoftLayerEndpointURL:    SoftlayerRestEndpoint,
		SoftlayerXMLRPCEndpoint: SoftlayerXMLRPCEndpoint,
	}

	return config.ClientSession()
}
