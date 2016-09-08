package logentries

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	"github.com/logentries/le_goclient"
)

// Provider returns a terraform.ResourceProvider.
func Provider() terraform.ResourceProvider {

	// The actual provider
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"account_key": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("LOGENTRIES_ACCOUNT_KEY", nil),
				Description: descriptions["account_key"],
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"logentries_log":    resourceLogentriesLog(),
			"logentries_logset": resourceLogentriesLogSet(),
		},

		ConfigureFunc: providerConfigure,
	}
}

var descriptions map[string]string

func init() {
	descriptions = map[string]string{
		"account_key": "The Log Entries account key.",
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	return logentries.NewClient(d.Get("account_key").(string)), nil
}
