package null

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/hashicorp/terraform/helper/schema"
)

func init() {
	rand.Seed(time.Now().Unix())
}

func resource() *schema.Resource {
	return &schema.Resource{
		Create: resourceCreate,
		Read:   resourceRead,
		Delete: resourceDelete,

		Schema: map[string]*schema.Schema{
			"triggers": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func resourceCreate(d *schema.ResourceData, meta interface{}) error {
	d.SetId(fmt.Sprintf("%d", rand.Int()))
	return nil
}

func resourceRead(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceDelete(d *schema.ResourceData, meta interface{}) error {
	d.SetId("")
	return nil
}
