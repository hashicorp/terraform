package nsone

import (
	"fmt"
	"github.com/bobtfish/go-nsone-api"
	"github.com/hashicorp/terraform/helper/schema"
	"log"
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
			"link": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"meta"}, // FIXME
			},
			"ttl": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
			},
			"nx_ttl": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
			},
			"retry": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
			},
			"expiry": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
			},
			"hostmaster": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"primary": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"meta": metaSchema(),
		},
		Create: ZoneCreate,
		Read:   ZoneRead,
		Update: ZoneUpdate,
		Delete: ZoneDelete,
	}
}

func zoneToResourceData(d *schema.ResourceData, z *nsone.Zone) {
	d.SetId(z.Id)
	d.Set("hostmaster", z.Hostmaster)
	d.Set("ttl", z.Ttl)
	d.Set("nx_ttl", z.Nx_ttl)
	d.Set("retry", z.Retry)
	d.Set("expiry", z.Expiry)
	if z.Meta != nil {
		d.Set("meta", z.Meta)
	}
	if z.Secondary != nil && z.Secondary.Enabled {
		d.Set("primary", z.Secondary.Primary_ip)
	}
	if z.Link != "" {
		d.Set("link", z.Link)
	}
	log.Println(fmt.Sprintf("MOO: ID %i", z.Id))
	log.Println(fmt.Sprintf("MOO: TTL %i", z.Ttl))
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
	if v, ok := d.GetOk("meta"); ok {
		z.Meta = v.(map[string]string)
	}
	if v, ok := d.GetOk("primary"); ok {
		z.MakeSecondary(v.(string))
	}
	if v, ok := d.GetOk("link"); ok {
		z.LinkTo(v.(string))
	}
	err := client.CreateZone(z)
	//    zone := d.Get("zone").(string)
	//    hostmaster := d.Get("hostmaster").(string)
	if err != nil {
		return err
	}
	zoneToResourceData(d, z)
	return nil
}

func ZoneRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*nsone.APIClient)
	z, err := client.GetZone(d.Get("zone").(string))
	//    zone := d.Get("zone").(string)
	//    hostmaster := d.Get("hostmaster").(string)
	if err != nil {
		return err
	}
	zoneToResourceData(d, z)
	log.Println(z)
	log.Println("Return from ZoneRead")
	return nil
}

func ZoneDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*nsone.APIClient)
	err := client.DeleteZone(d.Get("zone").(string))
	d.SetId("")
	return err
}

func ZoneUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*nsone.APIClient)
	z := nsone.NewZone(d.Get("zone").(string))
	z.Id = d.Id()
	if d.HasChange("ttl") {
		z.Ttl = d.Get("ttl").(int)
	}
	if d.HasChange("nx_ttl") {
		z.Ttl = d.Get("nx_ttl").(int)
	}
	if d.HasChange("retry") {
		z.Retry = d.Get("retry").(int)
	}
	if d.HasChange("expiry") {
		z.Expiry = d.Get("expiry").(int)
	}
	if d.HasChange("meta") {
		z.Meta = d.Get("meta").(map[string]string)
	}
	err := client.UpdateZone(z)
	if err != nil {
		return err
	}
	zoneToResourceData(d, z)
	return nil
}
