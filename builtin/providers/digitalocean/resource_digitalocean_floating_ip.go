package digitalocean

import (
	"fmt"
	"log"
	"strings"
	"github.com/digitalocean/godo"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceDigitalOceanFloatingIP() *schema.Resource {
	return &schema.Resource{
		Create: resourceDigitalOceanFloatingIPCreate,
		Read:   resourceDigitalOceanFloatingIPRead,
		Update: resourceDigitalOceanFloatingIPUpdate,
		Delete: resourceDigitalOceanFloatingIPDelete,

		Schema: map[string]*schema.Schema{
			"droplet_id": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},

			"region": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceDigitalOceanFloatingIPCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*godo.Client)

	newFloatingIP := godo.FloatingIPCreateRequest{
		DropletID: d.Get("droplet_id").(int),
		Region: d.Get("region").(string),
	}

	log.Printf("[DEBUG] Floating IP create configuration: %#v", newFloatingIP)

	var err	error
	fip, _, err := client.FloatingIPs.Create(&newFloatingIP)
	if err != nil {
		return fmt.Errorf("Failed to create Floating IP: %s", err)
	}

	d.SetId(fip.IP)
	log.Printf("[INFO] Floating IP: %s", d.Id())

	return resourceDigitalOceanFloatingIPRead(d, meta)
}

func resourceDigitalOceanFloatingIPRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*godo.Client)

	ip := d.Id();

	fip, _, err := client.FloatingIPs.Get(ip)
	if err != nil {
		// If the Floating IP is somehow already destroyed, mark as
		// successfully gone
		if strings.Contains(err.Error(), "404 Not Found") {
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Error retrieving Floating IP: %s", err)
	}

	d.Set("region", fip.Region);

	if fip.Droplet != nil {
		d.Set("droplet_id", fip.Droplet.ID);
	}

	return nil
}

func resourceDigitalOceanFloatingIPUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*godo.Client)

	id := d.Id();

	var err error
	if d.HasChange("droplet_id") {
		if droplet_id := d.Get("droplet_id").(int); &droplet_id != nil {
			_, _, err = client.FloatingIPActions.Assign(id, droplet_id);
		} else {
			_, _, err = client.FloatingIPActions.Unassign(id);
		}
	}

	if err != nil {
		return fmt.Errorf("Error updating Floating IP: %s", err)
	}

	return resourceDigitalOceanFloatingIPRead(d, meta)
}

func resourceDigitalOceanFloatingIPDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*godo.Client)

	id := d.Id()

	log.Printf("[INFO] Deleting Floating IP: %s", id);

	_, err := client.FloatingIPs.Delete(id)
	if err != nil {
		// If the Floating IP is somehow already destroyed, mark as
		// successfully gone
		if strings.Contains(err.Error(), "404 Not Found") {
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Error deleting Floating IP: %s", err)
	}

	return nil
}