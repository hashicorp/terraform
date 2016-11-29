package scaleway

import (
	"github.com/hashicorp/terraform/helper/mutexkv"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

// Provider returns a terraform.ResourceProvider.
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"access_key": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("SCALEWAY_ACCESS_KEY", nil),
				Description: "The API key for Scaleway API operations.",
			},
			"organization": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("SCALEWAY_ORGANIZATION", nil),
				Description: "The Organization ID for Scaleway API operations.",
			},
			"region": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("SCALEWAY_REGION", "par1"),
				Description: "The Scaleway API region to use.",
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"scaleway_server":              resourceScalewayServer(),
			"scaleway_ip":                  resourceScalewayIP(),
			"scaleway_security_group":      resourceScalewaySecurityGroup(),
			"scaleway_security_group_rule": resourceScalewaySecurityGroupRule(),
			"scaleway_volume":              resourceScalewayVolume(),
			"scaleway_volume_attachment":   resourceScalewayVolumeAttachment(),
		},

		DataSourcesMap: map[string]*schema.Resource{
			"scaleway_bootscript": dataSourceScalewayBootscript(),
			"scaleway_image":      dataSourceScalewayImage(),
		},

		ConfigureFunc: providerConfigure,
	}
}

var scalewayMutexKV = mutexkv.NewMutexKV()

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	config := Config{
		Organization: d.Get("organization").(string),
		APIKey:       d.Get("access_key").(string),
		Region:       d.Get("region").(string),
	}

	return config.Client()
}
