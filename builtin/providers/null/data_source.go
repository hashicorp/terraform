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

func dataSource() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceRead,

		Schema: map[string]*schema.Schema{
			"inputs": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
			},
			"outputs": &schema.Schema{
				Type:     schema.TypeMap,
				Computed: true,
			},
			"random": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"has_computed_default": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
		},
	}
}

func dataSourceRead(d *schema.ResourceData, meta interface{}) error {

	inputs := d.Get("inputs")
	d.Set("outputs", inputs)

	d.Set("random", fmt.Sprintf("%d", rand.Int()))
	if d.Get("has_computed_default") == "" {
		d.Set("has_computed_default", "default")
	}

	d.SetId("static")

	return nil
}
