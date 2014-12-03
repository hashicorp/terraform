package sdc

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"sdc_key_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"sdc_instance": resourceComputeInstance(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	config := &Config{
		SdcKeyName: d.Get("sdc_key_name").(string),
	}

	if err := config.initialize(); err != nil {
		return nil, err
	}

	return config, nil
}
