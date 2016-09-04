package occi

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

func Provider() terraform.ResourceProvider {
	return &schema.Provider{
	//	Schema: map[string]*schema.Schema{
	//		"site": &schema.Schema{
	//			Type:        schema.TypeString,
	//			Optional:    true,
	//			DefaultFunc: schema.EnvDefaultFunc("OCCI_SITE", nil),
	//			},
		

		ResourcesMap: map[string]*schema.Resource{
			"occi_virtual_machine": resourceVirtualMachine(),
		},

	//	ConfigureFunc: configureProvider,
	}
}

func configureProvider(d *schema.ResourceData) (interface{}, error) {
	config := Config{
		Site: d.Get("site").(string),
	}

	return &config, nil
}
