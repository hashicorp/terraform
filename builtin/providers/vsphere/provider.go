package vsphere

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

// Provider returns a terraform.ResourceProvider.
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"user": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("VSPHERE_USER", nil),
				Description: "The user name for vSphere API operations.",
			},

			"password": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("VSPHERE_PASSWORD", nil),
				Description: "The user password for vSphere API operations.",
			},

			"vcenter_server": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("VSPHERE_VCENTER", nil),
				Description: "The vCenter Server name for vSphere API operations.",
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"vsphere_virtual_machine": resourceVSphereVirtualMachine(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	config := Config{
		User:          d.Get("user").(string),
		Password:      d.Get("password").(string),
		VCenterServer: d.Get("vcenter_server").(string),
	}

	return config.Client()
}
