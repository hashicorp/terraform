package brocadevtm

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
			"vtm_user": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  os.Getenv("VTM_USER"),
			},
			"vtm_password": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  os.Getenv("VTM_PASSWORD"),
			},
			"vtm_server": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  os.Getenv("VTM_SERVER"),
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"brocadevtm_monitor": resourceMonitor(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	debug := d.Get("debug").(bool)
	insecure := d.Get("insecure").(bool)

	vtmUser := d.Get("vtm_user").(string)
	if vtmUser == "" {
		return nil, fmt.Errorf("vtm_user must be provided")
	}

	vtmPassword := d.Get("vtm_password").(string)

	if vtmPassword == "" {
		return nil, fmt.Errorf("vtm_password must be provided")
	}

	vtmServer := d.Get("vtm_server").(string)

	if vtmServer == "" {
		return nil, fmt.Errorf("vtm_server must be provided")
	}

	config := Config{
		Debug:       debug,
		Insecure:    insecure,
		VTMUser:     vtmUser,
		VTMPassword: vtmPassword,
		VTMServer:   vtmServer,
	}

	return config.Client()
}

// This is a global MutexKV for use within this plugin.
var vtmMutexKV = mutexkv.NewMutexKV()
