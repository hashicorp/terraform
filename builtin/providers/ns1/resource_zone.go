package ns1

import (
	"log"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"

	ns1 "gopkg.in/ns1/ns1-go.v2/rest"
	"gopkg.in/ns1/ns1-go.v2/rest/model/dns"
)

func zoneResource() *schema.Resource {
	return &schema.Resource{
		Create: resourceNS1ZoneCreate,
		Read:   resourceNS1ZoneRead,
		Update: resourceNS1ZoneUpdate,
		Delete: resourceNS1ZoneDelete,
		Exists: resourceNS1ZoneExists,
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
				ConflictsWith: []string{"ttl", "nx_ttl", "refresh", "retry", "expiry"},
			},
			"ttl": &schema.Schema{
				Type:          schema.TypeInt,
				Optional:      true,
				Computed:      true,
				ConflictsWith: []string{"link"},
			},
			"refresh": &schema.Schema{
				Type:          schema.TypeInt,
				Optional:      true,
				Computed:      true,
				ConflictsWith: []string{"link"},
			},
			"retry": &schema.Schema{
				Type:          schema.TypeInt,
				Optional:      true,
				Computed:      true,
				ConflictsWith: []string{"link"},
			},
			"expiry": &schema.Schema{
				Type:          schema.TypeInt,
				Optional:      true,
				Computed:      true,
				ConflictsWith: []string{"link"},
			},
			"nx_ttl": &schema.Schema{
				Type:          schema.TypeInt,
				Optional:      true,
				Computed:      true,
				ConflictsWith: []string{"link"},
			},
			"hostmaster": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"dns_servers": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"primary": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func resourceNS1ZoneCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ns1.Client)

	z := buildNS1ZoneStruct(d)

	log.Printf("[INFO] Creating NS1 zone: %s \n", z.Zone)

	if _, err := client.Zones.Create(z); err != nil {
		return err
	}

	d.SetId(z.Zone)
	return resourceNS1ZoneRead(d, meta)
}

func resourceNS1ZoneRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ns1.Client)

	log.Printf("[INFO] Reading NS1 zone: %s \n", d.Id())

	z, _, err := client.Zones.Get(d.Get("zone").(string))
	if err != nil {
		return err
	}

	d.Set("ttl", z.TTL)
	d.Set("nx_ttl", z.NxTTL)
	d.Set("refresh", z.Refresh)
	d.Set("retry", z.Retry)
	d.Set("expiry", z.Expiry)
	d.Set("hostmaster", z.Hostmaster)
	d.Set("dns_servers", strings.Join(z.DNSServers[:], ","))

	if z.Secondary != nil && z.Secondary.Enabled {
		d.Set("primary", z.Secondary.PrimaryIP)
	}
	if z.Link != nil && *z.Link != "" {
		d.Set("link", *z.Link)
	}

	return nil
}

func resourceNS1ZoneUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ns1.Client)

	z := buildNS1ZoneStruct(d)

	log.Printf("[INFO] Updating NS1 zone: %s \n", d.Id())

	if _, err := client.Zones.Update(z); err != nil {
		return err
	}

	return nil
}

func resourceNS1ZoneDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ns1.Client)

	log.Printf("[INFO] Deleting NS1 zone: %s \n", d.Id())

	if _, err := client.Zones.Delete(d.Get("zone").(string)); err != nil {
		return err
	}

	d.SetId("")

	return nil
}

func resourceNS1ZoneExists(d *schema.ResourceData, meta interface{}) (b bool, err error) {
	client := meta.(*ns1.Client)

	_, _, err = client.Zones.Get(d.Get("zone").(string))
	if err == ns1.ErrZoneMissing {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return true, nil
}

func buildNS1ZoneStruct(d *schema.ResourceData) *dns.Zone {
	z := dns.NewZone(d.Get("zone").(string))

	if v, ok := d.GetOk("ttl"); ok {
		z.TTL = v.(int)
	}
	if v, ok := d.GetOk("nx_ttl"); ok {
		z.NxTTL = v.(int)
	}
	if v, ok := d.GetOk("refresh"); ok {
		z.Refresh = v.(int)
	}
	if v, ok := d.GetOk("retry"); ok {
		z.Retry = v.(int)
	}
	if v, ok := d.GetOk("expiry"); ok {
		z.Expiry = v.(int)
	}
	if v, ok := d.GetOk("primary"); ok {
		z.MakeSecondary(v.(string))
	}
	if v, ok := d.GetOk("link"); ok {
		z.LinkTo(v.(string))
	}

	return z
}
