package google

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"google.golang.org/api/dns/v1"
)

func resourceDnsManagedZone() *schema.Resource {
	return &schema.Resource{
		Create: resourceDnsManagedZoneCreate,
		Read:   resourceDnsManagedZoneRead,
		Delete: resourceDnsManagedZoneDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Schema: map[string]*schema.Schema{
			"dns_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  "Managed by Terraform",
			},

			"name_servers": &schema.Schema{
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			// Google Cloud DNS ManagedZone resources do not have a SelfLink attribute.

			"project": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func resourceDnsManagedZoneCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

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
	zone, err = config.clientDns.ManagedZones.Create(project, zone).Do()
	if err != nil {
		return fmt.Errorf("Error creating DNS ManagedZone: %s", err)
	}

	d.SetId(zone.Name)

	return resourceDnsManagedZoneRead(d, meta)
}

func resourceDnsManagedZoneRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	zone, err := config.clientDns.ManagedZones.Get(
		project, d.Id()).Do()
	if err != nil {
		return handleNotFoundError(err, d, fmt.Sprintf("DNS Managed Zone %q", d.Get("name").(string)))
	}

	d.Set("name_servers", zone.NameServers)
	d.Set("name", zone.Name)
	d.Set("dns_name", zone.DnsName)
	d.Set("description", zone.Description)

	return nil
}

func resourceDnsManagedZoneDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	err = config.clientDns.ManagedZones.Delete(project, d.Id()).Do()
	if err != nil {
		return fmt.Errorf("Error deleting DNS ManagedZone: %s", err)
	}

	d.SetId("")
	return nil
}
