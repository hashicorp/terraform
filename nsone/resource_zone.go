package nsone

import (
	"github.com/bobtfish/go-nsone-api"
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
	client := meta.(*nsone.APIClient)
	z := nsone.NewZone(d.Get("zone").(string))
	z.Hostmaster = d.Get("hostmaster").(string)
	if v, ok := d.GetOk("ttl"); ok {
		z.Ttl = v.(int)
	}
	if v, ok := d.GetOk("nx_ttl"); ok {
		z.Nx_ttl = v.(int)
	}
	if v, ok := d.GetOk("retry"); ok {
		z.Retry = v.(int)
	}
	if v, ok := d.GetOk("expiry"); ok {
		z.Expiry = v.(int)
	}
	err := client.CreateZone(z)
	//    zone := d.Get("zone").(string)
	//    hostmaster := d.Get("hostmaster").(string)
	if err != nil {
		return err
	}
	d.SetId(z.Id)
	d.Set("hostmaster", z.Hostmaster)
	d.Set("ttl", z.Ttl)
	d.Set("nx_ttl", z.Nx_ttl)
	d.Set("retry", z.Retry)
	d.Set("expiry", z.Expiry)
	return nil
}

func ZoneRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*nsone.APIClient)
	z := nsone.NewZone(d.Get("zone").(string))
	err := client.GetZone(z)
	//    zone := d.Get("zone").(string)
	//    hostmaster := d.Get("hostmaster").(string)
	if err != nil {
		return err
	}
	d.SetId(z.Id)
	d.Set("hostmaster", z.Hostmaster)
	d.Set("ttl", z.Ttl)
	d.Set("nx_ttl", z.Nx_ttl)
	d.Set("retry", z.Retry)
	d.Set("expiry", z.Expiry)
	return nil
}

func ZoneDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*nsone.APIClient)
	z := nsone.NewZone(d.Get("zone").(string))
	err := client.DeleteZone(z)
	d.SetId("")
	return err
}

func ZoneUpdate(d *schema.ResourceData, meta interface{}) error {
	panic("Update not implemented")
	return nil
}
