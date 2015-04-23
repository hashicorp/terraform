package sdc

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

// Provider returns a schema.Provider for SDC.
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"url": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("SDC_URL", nil),
				Description: "The SDC API url, default is 'https://us-west-1.api.joyentcloud.com'.",
			},
			"account": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("SDC_ACCOUNT", nil),
				Description: "The root account username.",
			},
			"user": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("SDC_USER", nil),
				Description: "The username is you are using RBAC.",
			},
			"key_id": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("SDC_KEY_ID", nil),
				Description: "The fingerprint of an SSH key you have added to your account.",
			},
			"key": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("SDC_KEY", nil),
				Description: "The path to your SSH key. Default is '$HOME/.ssh/id_rsa'.",
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"sdc_machine": resourceSDCMachine(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	config := Config{
		Url:     d.Get("url").(string),
		Account: d.Get("account").(string),
		User:    d.Get("user").(string),
		KeyId:   d.Get("key_id").(string),
		Key:     d.Get("key").(string),
	}

	return config.Client()
}
