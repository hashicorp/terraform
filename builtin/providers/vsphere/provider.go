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

			"vsphere_server": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("VSPHERE_SERVER", nil),
				Description: "The vSphere Server name for vSphere API operations.",
			},

			"allow_unverified_ssl": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("VSPHERE_ALLOW_UNVERIFIED_SSL", false),
				Description: "If set, VMware vSphere client will permit unverifiable SSL certificates.",
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
		VSphereServer: d.Get("vsphere_server").(string),
		InsecureFlag:  d.Get("allow_unverified_ssl").(bool),
	}

	return config.Client()
}
