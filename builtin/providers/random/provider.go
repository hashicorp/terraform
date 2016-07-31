package random

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

// Provider returns a terraform.ResourceProvider.
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{},

		ResourcesMap: map[string]*schema.Resource{
			"random_id":      resourceId(),
			"random_shuffle": resourceShuffle(),
		},
	}
}

// stubRead is a do-nothing Read implementation used for our resources,
// which don't actually need to do anything on read.
func stubRead(d *schema.ResourceData, meta interface{}) error {
	return nil
}

// stubDelete is a do-nothing Dete implementation used for our resources,
// which don't actually need to do anything unusual on delete.
func stubDelete(d *schema.ResourceData, meta interface{}) error {
	d.SetId("")
	return nil
}
