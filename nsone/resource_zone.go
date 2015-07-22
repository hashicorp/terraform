package nsone

import (
	"github.com/hashicorp/terraform/helper/schema"
)

func zoneResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"zone": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"ttl": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
			"nx_ttl": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
			"retry": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
			"expiry": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
			"hostmaster": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
		},
		Create: ZoneCreate,
		Read:   ZoneRead,
		Update: ZoneUpdate,
		Delete: ZoneDelete,
	}
}

func ZoneCreate(d *schema.ResourceData, meta interface{}) error {
	//    zone := d.Get("zone").(string)
	//    hostmaster := d.Get("hostmaster").(string)
	id := "foo"
	d.SetId(id)
	return nil
}

func ZoneRead(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func ZoneDelete(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func ZoneUpdate(d *schema.ResourceData, meta interface{}) error {
	return nil
}
