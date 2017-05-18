package ovh

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

// Provider returns a schema.Provider for OVH.
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"endpoint": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("OVH_ENDPOINT", nil),
				Description: descriptions["endpoint"],
			},
			"application_key": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("OVH_APPLICATION_KEY", ""),
				Description: descriptions["application_key"],
			},
			"application_secret": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("OVH_APPLICATION_SECRET", ""),
				Description: descriptions["application_secret"],
			},
			"consumer_key": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("OVH_CONSUMER_KEY", ""),
				Description: descriptions["consumer_key"],
			},
		},

		DataSourcesMap: map[string]*schema.Resource{
			"ovh_publiccloud_region":  dataSourcePublicCloudRegion(),
			"ovh_publiccloud_regions": dataSourcePublicCloudRegions(),
		},

		ResourcesMap: map[string]*schema.Resource{
			"ovh_publiccloud_private_network":        resourcePublicCloudPrivateNetwork(),
			"ovh_publiccloud_private_network_subnet": resourcePublicCloudPrivateNetworkSubnet(),
			"ovh_publiccloud_user":                   resourcePublicCloudUser(),
			"ovh_vrack_publiccloud_attachment":       resourceVRackPublicCloudAttachment(),
		},

		ConfigureFunc: configureProvider,
	}
}

var descriptions map[string]string

func init() {
	descriptions = map[string]string{
		"endpoint": "The OVH API endpoint to target (ex: \"ovh-eu\").",

		"application_key": "The OVH API Application Key.",

		"application_secret": "The OVH API Application Secret.",
		"consumer_key":       "The OVH API Consumer key.",
	}
}

func configureProvider(d *schema.ResourceData) (interface{}, error) {
	config := Config{
		Endpoint:          d.Get("endpoint").(string),
		ApplicationKey:    d.Get("application_key").(string),
		ApplicationSecret: d.Get("application_secret").(string),
		ConsumerKey:       d.Get("consumer_key").(string),
	}

	if err := config.loadAndValidate(); err != nil {
		return nil, err
	}

	return &config, nil
}
