package google

import(
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceComputeInstance() *schema.Resource {
	return &schema.Resource{
		Create: resourceComputeInstanceCreate,

		Schema: map[string]*schema.Schema{},
	}
}

func resourceComputeInstanceCreate(d *schema.ResourceData, meta interface{}) error {
	return nil
}
