package heroku

import "github.com/hashicorp/terraform/helper/schema"

func resourceHerokuSpace() *schema.Resource {
	return &schema.Resource{
		Create: func(_ *schema.ResourceData, _ interface{}) error { return nil },
		Read:   func(_ *schema.ResourceData, _ interface{}) error { return nil },
		Update: func(_ *schema.ResourceData, _ interface{}) error { return nil },
		Delete: func(_ *schema.ResourceData, _ interface{}) error { return nil },

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"organization": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"region": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}
