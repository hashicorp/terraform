package kubernetes

import (
	"github.com/hashicorp/terraform/helper/schema"
)

// Pod resources

func genResourceQuantity() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"unscaled": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true, // required
			},

			"scale": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true, // required
			},
		},
	}
}
