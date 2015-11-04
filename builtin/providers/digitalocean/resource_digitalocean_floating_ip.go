package digitalocean

import (
	"fmt"
	"log"

	"github.com/digitalocean/godo"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceDigitalOceanFloatingIp() *schema.Resource {
	return &schema.Resource{
		Create: resourceDigitalOceanFloatingIpCreate,
		Read:   resourceDigitalOceanFloatingIpRead,
		Delete: resourceDigitalOceanFloatingIpDelete,

		Schema: map[string]*schema.Schema{
			"ip_address": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"region": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"droplet_id": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func resourceDigitalOceanFloatingIpCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*godo.Client)

	// Build up our creation options
	opts := &godo.FloatingIPCreateRequest{}

	if v, ok := d.GetOk("droplet_id"); ok {
		log.Printf("[INFO] Found a droplet_id to try and attach to the FloatingIP")
		opts.DropletID = v.(int)
	} else if d.Get("region").(string) != "" {
		opts.Region = d.Get("region").(string)
	} else {
		return fmt.Errorf("You must specify either a Droplet ID or a Region for a FloatingIP")
	}

	log.Printf("[DEBUG] FloatingIP Create: %#v", opts)
	floatingIp, _, err := client.FloatingIPs.Create(opts)
	if err != nil {
		return fmt.Errorf("Error creating FloatingIP: %s", err)
	}

	d.SetId(floatingIp.IP)
	log.Printf("[INFO] Floating IP: %s", floatingIp.IP)

	return resourceDigitalOceanFloatingIpRead(d, meta)
}

func resourceDigitalOceanFloatingIpRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*godo.Client)

	floatingIp, _, err := client.FloatingIPs.Get(d.Id())
	if err != nil {
		return fmt.Errorf("Error retrieving FloatingIP: %s", err)
	}

	d.Set("region", floatingIp.Region)
	d.Set("ip_address", floatingIp.IP)

	return nil
}

func resourceDigitalOceanFloatingIpDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*godo.Client)

	log.Printf("[INFO] Deleting FloatingIP: %s", d.Id())
	_, err := client.FloatingIPs.Delete(d.Id())
	if err != nil {
		return fmt.Errorf("Error deleting FloatingIP: %s", err)
	}

	d.SetId("")
	return nil
}
