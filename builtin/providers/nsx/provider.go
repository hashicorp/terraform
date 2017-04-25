package nsx

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/mutexkv"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	"os"
)

// Provider is a basic structure that describes a provider: the configuration
// keys it takes, the resources it supports, a callback to configure, etc.
func Provider() terraform.ResourceProvider {
	// The actual provider
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"debug": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"insecure": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"nsx_user": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  os.Getenv("NSX_USER"),
			},
			"nsx_password": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  os.Getenv("NSX_PASSWORD"),
			},
			"nsx_server": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  os.Getenv("NSX_SERVER"),
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"nsx_service":                 resourceService(),
			"nsx_security_tag":            resourceSecurityTag(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	debug := d.Get("debug").(bool)
	insecure := d.Get("insecure").(bool)
	nsx_user := d.Get("nsx_user").(string)

	if nsx_user == "" {
		return nil, fmt.Errorf("nsx_user must be provided")
	}

	nsx_password := d.Get("nsx_password").(string)

	if nsx_password == "" {
		return nil, fmt.Errorf("nsx_password must be provided")
	}

	nsx_server := d.Get("nsx_server").(string)

	if nsx_server == "" {
		return nil, fmt.Errorf("nsx_server must be provided")
	}

	config := Config{
		Debug:       debug,
		Insecure:    insecure,
		NSXUser:     nsx_user,
		NSXPassword: nsx_password,
		NSXServer:   nsx_server,
	}

	return config.Client()
}

// This is a global MutexKV for use within this plugin.
var nsxMutexKV = mutexkv.NewMutexKV()
