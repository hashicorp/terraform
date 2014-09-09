package digitalocean

import (
	"github.com/hashicorp/terraform/helper/schema"
)

// Provider returns a schema.Provider for DigitalOcean.
//
// NOTE: schema.Provider became available long after the DO provider
// was started, so resources may not be converted to this new structure
// yet. This is a WIP. To assist with the migration, make sure any resources
// you migrate are acceptance tested, then perform the migration.
func Provider() *schema.Provider {
	// TODO: Move the configuration to this

	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"token": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"digitalocean_domain": resourceDomain(),
			"digitalocean_record": resourceRecord(),
		},
	}
}
