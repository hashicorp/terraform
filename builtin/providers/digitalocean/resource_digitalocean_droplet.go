package digitalocean

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/pearkes/digitalocean"
)

func resourceDigitalOceanDroplet() *schema.Resource {
	return &schema.Resource{
		Create: resourceDigitalOceanDropletCreate,
		Read:   resourceDigitalOceanDropletRead,
		Update: resourceDigitalOceanDropletUpdate,
		Delete: resourceDigitalOceanDropletDelete,

		Schema: map[string]*schema.Schema{
			"image": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"region": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"size": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"status": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"locked": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"backups": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},

			"ipv6": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},

			"ipv6_address": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"ipv6_address_private": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"private_networking": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},

			"ipv4_address": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"ipv4_address_private": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"ssh_keys": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"user_data": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func resourceDigitalOceanDropletCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*digitalocean.Client)

	// Build up our creation options
	opts := &digitalocean.CreateDroplet{
		Image:  d.Get("image").(string),
		Name:   d.Get("name").(string),
		Region: d.Get("region").(string),
		Size:   d.Get("size").(string),
	}

	if attr, ok := d.GetOk("backups"); ok {
		opts.Backups = attr.(bool)
	}

	if attr, ok := d.GetOk("ipv6"); ok {
		opts.IPV6 = attr.(bool)
	}

	if attr, ok := d.GetOk("private_networking"); ok {
		opts.PrivateNetworking = attr.(bool)
	}

	if attr, ok := d.GetOk("user_data"); ok {
		opts.UserData = attr.(string)
	}

	// Get configured ssh_keys
	ssh_keys := d.Get("ssh_keys.#").(int)
	if ssh_keys > 0 {
		opts.SSHKeys = make([]string, 0, ssh_keys)
		for i := 0; i < ssh_keys; i++ {
			key := fmt.Sprintf("ssh_keys.%d", i)
			opts.SSHKeys = append(opts.SSHKeys, d.Get(key).(string))
		}
	}

	log.Printf("[DEBUG] Droplet create configuration: %#v", opts)

	id, err := client.CreateDroplet(opts)

	if err != nil {
		return fmt.Errorf("Error creating droplet: %s", err)
	}

	// Assign the droplets id
	d.SetId(id)

	log.Printf("[INFO] Droplet ID: %s", d.Id())

	_, err = WaitForDropletAttribute(d, "active", []string{"new"}, "status", meta)
	if err != nil {
		return fmt.Errorf(
			"Error waiting for droplet (%s) to become ready: %s", d.Id(), err)
	}

	return resourceDigitalOceanDropletRead(d, meta)
}

func resourceDigitalOceanDropletRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*digitalocean.Client)

	// Retrieve the droplet properties for updating the state
	droplet, err := client.RetrieveDroplet(d.Id())
	if err != nil {
		// check if the droplet no longer exists.
		if err.Error() == "Error retrieving droplet: API Error: 404 Not Found" {
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Error retrieving droplet: %s", err)
	}

	if droplet.ImageSlug() != "" {
		d.Set("image", droplet.ImageSlug())
	} else {
		d.Set("image", droplet.ImageId())
	}

	d.Set("name", droplet.Name)
	d.Set("region", droplet.RegionSlug())
	d.Set("size", droplet.SizeSlug)
	d.Set("status", droplet.Status)
	d.Set("locked", droplet.IsLocked())

	if droplet.IPV6Address("public") != "" {
		d.Set("ipv6", true)
		d.Set("ipv6_address", droplet.IPV6Address("public"))
		d.Set("ipv6_address_private", droplet.IPV6Address("private"))
	}

	d.Set("ipv4_address", droplet.IPV4Address("public"))

	if droplet.NetworkingType() == "private" {
		d.Set("private_networking", true)
		d.Set("ipv4_address_private", droplet.IPV4Address("private"))
	}

	// Initialize the connection info
	d.SetConnInfo(map[string]string{
		"type": "ssh",
		"host": droplet.IPV4Address("public"),
	})

	return nil
}

func resourceDigitalOceanDropletUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*digitalocean.Client)

	if d.HasChange("size") {
		oldSize, newSize := d.GetChange("size")

		err := client.PowerOff(d.Id())

		if err != nil && !strings.Contains(err.Error(), "Droplet is already powered off") {
			return fmt.Errorf(
				"Error powering off droplet (%s): %s", d.Id(), err)
		}

		// Wait for power off
		_, err = WaitForDropletAttribute(d, "off", []string{"active"}, "status", client)
		if err != nil {
			return fmt.Errorf(
				"Error waiting for droplet (%s) to become powered off: %s", d.Id(), err)
		}

		// Resize the droplet
		err = client.Resize(d.Id(), newSize.(string))
		if err != nil {
			newErr := powerOnAndWait(d, meta)
			if newErr != nil {
				return fmt.Errorf(
					"Error powering on droplet (%s) after failed resize: %s", d.Id(), err)
			}
			return fmt.Errorf(
				"Error resizing droplet (%s): %s", d.Id(), err)
		}

		// Wait for the size to change
		_, err = WaitForDropletAttribute(
			d, newSize.(string), []string{"", oldSize.(string)}, "size", meta)

		if err != nil {
			newErr := powerOnAndWait(d, meta)
			if newErr != nil {
				return fmt.Errorf(
					"Error powering on droplet (%s) after waiting for resize to finish: %s", d.Id(), err)
			}
			return fmt.Errorf(
				"Error waiting for resize droplet (%s) to finish: %s", d.Id(), err)
		}

		err = client.PowerOn(d.Id())

		if err != nil {
			return fmt.Errorf(
				"Error powering on droplet (%s) after resize: %s", d.Id(), err)
		}

		// Wait for power off
		_, err = WaitForDropletAttribute(d, "active", []string{"off"}, "status", meta)
		if err != nil {
			return err
		}
	}

	if d.HasChange("name") {
		oldName, newName := d.GetChange("name")

		// Rename the droplet
		err := client.Rename(d.Id(), newName.(string))

		if err != nil {
			return fmt.Errorf(
				"Error renaming droplet (%s): %s", d.Id(), err)
		}

		// Wait for the name to change
		_, err = WaitForDropletAttribute(
			d, newName.(string), []string{"", oldName.(string)}, "name", meta)

		if err != nil {
			return fmt.Errorf(
				"Error waiting for rename droplet (%s) to finish: %s", d.Id(), err)
		}
	}

	// As there is no way to disable private networking,
	// we only check if it needs to be enabled
	if d.HasChange("private_networking") && d.Get("private_networking").(bool) {
		err := client.EnablePrivateNetworking(d.Id())

		if err != nil {
			return fmt.Errorf(
				"Error enabling private networking for droplet (%s): %s", d.Id(), err)
		}

		// Wait for the private_networking to turn on
		_, err = WaitForDropletAttribute(
			d, "true", []string{"", "false"}, "private_networking", meta)

		return fmt.Errorf(
			"Error waiting for private networking to be enabled on for droplet (%s): %s", d.Id(), err)
	}

	// As there is no way to disable IPv6, we only check if it needs to be enabled
	if d.HasChange("ipv6") && d.Get("ipv6").(bool) {
		err := client.EnableIPV6s(d.Id())

		if err != nil {
			return fmt.Errorf(
				"Error turning on ipv6 for droplet (%s): %s", d.Id(), err)
		}

		// Wait for ipv6 to turn on
		_, err = WaitForDropletAttribute(
			d, "true", []string{"", "false"}, "ipv6", meta)

		if err != nil {
			return fmt.Errorf(
				"Error waiting for ipv6 to be turned on for droplet (%s): %s", d.Id(), err)
		}
	}

	return resourceDigitalOceanDropletRead(d, meta)
}

func resourceDigitalOceanDropletDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*digitalocean.Client)

	_, err := WaitForDropletAttribute(
		d, "false", []string{"", "true"}, "locked", meta)

	if err != nil {
		return fmt.Errorf(
			"Error waiting for droplet to be unlocked for destroy (%s): %s", d.Id(), err)
	}

	log.Printf("[INFO] Deleting droplet: %s", d.Id())

	// Destroy the droplet
	err = client.DestroyDroplet(d.Id())

	// Handle remotely destroyed droplets
	if err != nil && strings.Contains(err.Error(), "404 Not Found") {
		return nil
	}

	if err != nil {
		return fmt.Errorf("Error deleting droplet: %s", err)
	}

	return nil
}

func WaitForDropletAttribute(
	d *schema.ResourceData, target string, pending []string, attribute string, meta interface{}) (interface{}, error) {
	// Wait for the droplet so we can get the networking attributes
	// that show up after a while
	log.Printf(
		"[INFO] Waiting for droplet (%s) to have %s of %s",
		d.Id(), attribute, target)

	stateConf := &resource.StateChangeConf{
		Pending:    pending,
		Target:     target,
		Refresh:    newDropletStateRefreshFunc(d, attribute, meta),
		Timeout:    60 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,

		// This is a hack around DO API strangeness.
		// https://github.com/hashicorp/terraform/issues/481
		//
		NotFoundChecks: 60,
	}

	return stateConf.WaitForState()
}

// TODO This function still needs a little more refactoring to make it
// cleaner and more efficient
func newDropletStateRefreshFunc(
	d *schema.ResourceData, attribute string, meta interface{}) resource.StateRefreshFunc {
	client := meta.(*digitalocean.Client)
	return func() (interface{}, string, error) {
		err := resourceDigitalOceanDropletRead(d, meta)
		if err != nil {
			return nil, "", err
		}

		// If the droplet is locked, continue waiting. We can
		// only perform actions on unlocked droplets, so it's
		// pointless to look at that status
		if d.Get("locked").(string) == "true" {
			log.Println("[DEBUG] Droplet is locked, skipping status check and retrying")
			return nil, "", nil
		}

		// See if we can access our attribute
		if attr, ok := d.GetOk(attribute); ok {
			// Retrieve the droplet properties
			droplet, err := client.RetrieveDroplet(d.Id())
			if err != nil {
				return nil, "", fmt.Errorf("Error retrieving droplet: %s", err)
			}

			return &droplet, attr.(string), nil
		}

		return nil, "", nil
	}
}

// Powers on the droplet and waits for it to be active
func powerOnAndWait(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*digitalocean.Client)
	err := client.PowerOn(d.Id())
	if err != nil {
		return err
	}

	// Wait for power on
	_, err = WaitForDropletAttribute(d, "active", []string{"off"}, "status", client)
	if err != nil {
		return err
	}

	return nil
}
