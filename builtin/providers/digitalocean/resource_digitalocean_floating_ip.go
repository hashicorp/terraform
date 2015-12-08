package digitalocean

import (
	"fmt"
	"log"
	"time"

	"github.com/digitalocean/godo"
	"github.com/hashicorp/terraform/helper/resource"
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
				Required: true,
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

	log.Printf("[INFO] Create a FloatingIP In a Region")
	regionOpts := &godo.FloatingIPCreateRequest{
		Region: d.Get("region").(string),
	}

	log.Printf("[DEBUG] FloatingIP Create: %#v", regionOpts)
	floatingIp, _, err := client.FloatingIPs.Create(regionOpts)
	if err != nil {
		return fmt.Errorf("Error creating FloatingIP: %s", err)
	}

	d.SetId(floatingIp.IP)

	if v, ok := d.GetOk("droplet_id"); ok {

		log.Printf("[INFO] Assigning the Floating IP to the Droplet %d", v.(int))
		action, _, err := client.FloatingIPActions.Assign(d.Id(), v.(int))
		if err != nil {
			return fmt.Errorf(
				"Error Assigning FloatingIP (%s) to the droplet: %s", d.Id(), err)
		}

		_, unassignedErr := waitForFloatingIPReady(d, "completed", []string{"new", "in-progress"}, "status", meta, action.ID)
		if unassignedErr != nil {
			return fmt.Errorf(
				"Error waiting for FloatingIP (%s) to be Assigned: %s", d.Id(), unassignedErr)
		}
	}

	return resourceDigitalOceanFloatingIpRead(d, meta)
}

func resourceDigitalOceanFloatingIpRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*godo.Client)

	log.Printf("[INFO] Reading the details of the FloatingIP %s", d.Id())
	floatingIp, _, err := client.FloatingIPs.Get(d.Id())
	if err != nil {
		return fmt.Errorf("Error retrieving FloatingIP: %s", err)
	}

	if floatingIp.Droplet != nil {
		log.Printf("[INFO] A droplet was detected on the FloatingIP so setting the Region based on the Droplet")
		log.Printf("[INFO] The region of the Droplet is %s", floatingIp.Droplet.Region.Slug)
		d.Set("region", floatingIp.Droplet.Region.Slug)
	} else {
		d.Set("region", floatingIp.Region.Slug)
	}

	d.Set("ip_address", floatingIp.IP)

	return nil
}

func resourceDigitalOceanFloatingIpDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*godo.Client)

	if _, ok := d.GetOk("droplet_id"); ok {
		log.Printf("[INFO] Unassigning the Floating IP from the Droplet")
		action, _, err := client.FloatingIPActions.Unassign(d.Id())
		if err != nil {
			return fmt.Errorf(
				"Error Unassigning FloatingIP (%s) from the droplet: %s", d.Id(), err)
		}

		_, unassignedErr := waitForFloatingIPReady(d, "completed", []string{"new", "in-progress"}, "status", meta, action.ID)
		if unassignedErr != nil {
			return fmt.Errorf(
				"Error waiting for FloatingIP (%s) to be unassigned: %s", d.Id(), unassignedErr)
		}
	}

	log.Printf("[INFO] Deleting FloatingIP: %s", d.Id())
	_, err := client.FloatingIPs.Delete(d.Id())
	if err != nil {
		return fmt.Errorf("Error deleting FloatingIP: %s", err)
	}

	d.SetId("")
	return nil
}

func waitForFloatingIPReady(
	d *schema.ResourceData, target string, pending []string, attribute string, meta interface{}, actionId int) (interface{}, error) {
	log.Printf(
		"[INFO] Waiting for FloatingIP (%s) to have %s of %s",
		d.Id(), attribute, target)

	stateConf := &resource.StateChangeConf{
		Pending:    pending,
		Target:     target,
		Refresh:    newFloatingIPStateRefreshFunc(d, attribute, meta, actionId),
		Timeout:    60 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,

		NotFoundChecks: 60,
	}

	return stateConf.WaitForState()
}

func newFloatingIPStateRefreshFunc(
	d *schema.ResourceData, attribute string, meta interface{}, actionId int) resource.StateRefreshFunc {
	client := meta.(*godo.Client)
	return func() (interface{}, string, error) {

		log.Printf("[INFO] Assigning the Floating IP to the Droplet")
		action, _, err := client.FloatingIPActions.Get(d.Id(), actionId)
		if err != nil {
			return nil, "", fmt.Errorf("Error retrieving FloatingIP (%s) ActionId (%d): %s", d.Id(), actionId, err)
		}

		log.Printf("[INFO] The FloatingIP Action Status is %s", action.Status)
		return &action, action.Status, nil
	}
}
