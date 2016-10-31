package ns1

import (
	"strings"

	"github.com/hashicorp/terraform/helper/schema"

	nsone "gopkg.in/ns1/ns1-go.v2/rest"
	"gopkg.in/ns1/ns1-go.v2/rest/model/dns"
)

func zoneResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			// Required
			"zone": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			// Optional
			"ttl": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
			},
			// SOA attributes per https://tools.ietf.org/html/rfc1035).
			"refresh": &schema.Schema{
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
			// SOA MINUMUM overloaded as NX TTL per https://tools.ietf.org/html/rfc2308
			"nx_ttl": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
			},
			// TODO: test
			"link": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			// TODO: test
			"primary": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			// Computed
			"id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"dns_servers": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"hostmaster": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
		Create: ZoneCreate,
		Read:   ZoneRead,
		Update: ZoneUpdate,
		Delete: ZoneDelete,
	}
}

func zoneToResourceData(d *schema.ResourceData, z *dns.Zone) {
	d.SetId(z.ID)
	d.Set("hostmaster", z.Hostmaster)
	d.Set("ttl", z.TTL)
	d.Set("nx_ttl", z.NxTTL)
	d.Set("refresh", z.Refresh)
	d.Set("retry", z.Retry)
	d.Set("expiry", z.Expiry)
	d.Set("dns_servers", strings.Join(z.DNSServers[:], ","))
	if z.Secondary != nil && z.Secondary.Enabled {
		d.Set("primary", z.Secondary.PrimaryIP)
	}
	if z.Link != nil && *z.Link != "" {
		d.Set("link", *z.Link)
	}
}

func resourceToZoneData(z *dns.Zone, d *schema.ResourceData) {
	z.ID = d.Id()
	if v, ok := d.GetOk("hostmaster"); ok {
		z.Hostmaster = v.(string)
	}
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
}

// ZoneCreate creates the given zone in ns1
func ZoneCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*nsone.Client)
	z := dns.NewZone(d.Get("zone").(string))
	resourceToZoneData(z, d)
	if _, err := client.Zones.Create(z); err != nil {
		return err
	}
	zoneToResourceData(d, z)
	return nil
}

// ZoneRead reads the given zone data from ns1
func ZoneRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*nsone.Client)
	z, _, err := client.Zones.Get(d.Get("zone").(string))
	if err != nil {
		return err
	}
	zoneToResourceData(d, z)
	return nil
}

// ZoneDelete deteles the given zone from ns1
func ZoneDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*nsone.Client)
	_, err := client.Zones.Delete(d.Get("zone").(string))
	d.SetId("")
	return err
}

// ZoneUpdate updates the zone with given params in ns1
func ZoneUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*nsone.Client)
	z := dns.NewZone(d.Get("zone").(string))
	resourceToZoneData(z, d)
	if _, err := client.Zones.Update(z); err != nil {
		return err
	}
	zoneToResourceData(d, z)
	return nil
}
