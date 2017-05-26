package heroku

import (
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceHerokuFormation() *schema.Resource {
	return &schema.Resource{
		Read:   resourceHerokuFormationRead,
		Update: resourceHerokuFormationUpdate,

		Schema: map[string]*schema.Schema{
			"app": {
				Type:     schema.TypeString,
				Required: true,
			},
			"formation": {
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func resourceHerokuFormationRead(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceHerokuFormationUpdate(d *schema.ResourceData, meta interface{}) error {
	return nil
}
