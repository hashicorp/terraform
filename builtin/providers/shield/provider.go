package shield

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"serverurl": {
				Required:    true,
				Type:        schema.TypeString,
				DefaultFunc: schema.EnvDefaultFunc("SHIELD_SERVER_URL", nil),
			},
			"username": {
				Required:    true,
				Type:        schema.TypeString,
				DefaultFunc: schema.EnvDefaultFunc("SHIELD_USERNAME", nil),
			},
			"password": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("SHIELD_PASSWORD", nil),
			},
			"insecure": {
				Type:        schema.TypeBool,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("OS_INSECURE", ""),
			},
		},
		ConfigureFunc: providerConfigure,
		ResourcesMap: map[string]*schema.Resource{
			"shield_target":           resourceTarget(),
			"shield_schedule":         resourceSchedule(),
			"shield_retention_policy": resourceRetention(),
			"shield_store":            resourceStore(),
			"shield_job":              resourceJob(),
			//"shield_archive": resourceArchive(),
			//"shield_task": resourceTask(),
		},
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	client := &ShieldClient{
		ServerUrl: d.Get("serverurl").(string),
		Username:  d.Get("username").(string),
		Password:  d.Get("password").(string),
		Insecure:  d.Get("insecure").(bool),
	}

	return client, nil
}
