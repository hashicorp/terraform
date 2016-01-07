package google

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"google.golang.org/api/dns/v1"
	"google.golang.org/api/googleapi"
)

func resourceDnsManagedZone() *schema.Resource {
	return &schema.Resource{
		Create: resourceDnsManagedZoneCreate,
		Read:   resourceDnsManagedZoneRead,
		Delete: resourceDnsManagedZoneDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"dns_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"name_servers": &schema.Schema{
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			// Google Cloud DNS ManagedZone resources do not have a SelfLink attribute.
		},
	}
}

func resourceDnsManagedZoneCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	// Build the parameter
	zone := &dns.ManagedZone{
		Name:    d.Get("name").(string),
		DnsName: d.Get("dns_name").(string),
	}
	// Optional things
	if v, ok := d.GetOk("description"); ok {
		zone.Description = v.(string)
	}
	if v, ok := d.GetOk("dns_name"); ok {
		zone.DnsName = v.(string)
	}

	log.Printf("[DEBUG] DNS ManagedZone create request: %#v", zone)
	zone, err := config.clientDns.ManagedZones.Create(config.Project, zone).Do()
	if err != nil {
		return fmt.Errorf("Error creating DNS ManagedZone: %s", err)
	}

	d.SetId(zone.Name)

	return resourceDnsManagedZoneRead(d, meta)
}

func resourceDnsManagedZoneRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	zone, err := config.clientDns.ManagedZones.Get(
		config.Project, d.Id()).Do()
	if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 {
			log.Printf("[WARN] Removing DNS Managed Zone %q because it's gone", d.Get("name").(string))
			// The resource doesn't exist anymore
			d.SetId("")

			return nil
		}

		return fmt.Errorf("Error reading DNS ManagedZone: %#v", err)
	}

	d.Set("name_servers", zone.NameServers)

	return nil
}

func resourceDnsManagedZoneDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	err := config.clientDns.ManagedZones.Delete(config.Project, d.Id()).Do()
	if err != nil {
		return fmt.Errorf("Error deleting DNS ManagedZone: %s", err)
	}

	d.SetId("")
	return nil
}
