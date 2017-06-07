package digitalocean

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/digitalocean/godo"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceDigitalOceanDroplet() *schema.Resource {
	return &schema.Resource{
		Create: resourceDigitalOceanDropletCreate,
		Read:   resourceDigitalOceanDropletRead,
		Update: resourceDigitalOceanDropletUpdate,
		Delete: resourceDigitalOceanDropletDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"image": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"region": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				StateFunc: func(val interface{}) string {
					// DO API V2 region slug is always lowercase
					return strings.ToLower(val.(string))
				},
			},

			"size": {
				Type:     schema.TypeString,
				Required: true,
				StateFunc: func(val interface{}) string {
					// DO API V2 size slug is always lowercase
					return strings.ToLower(val.(string))
				},
			},

			"disk": {
				Type:     schema.TypeInt,
				Computed: true,
			},

			"vcpus": {
				Type:     schema.TypeInt,
				Computed: true,
			},

			"price_hourly": {
				Type:     schema.TypeFloat,
				Computed: true,
			},

			"price_monthly": {
				Type:     schema.TypeFloat,
				Computed: true,
			},

			"resize_disk": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},

			"status": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"locked": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"backups": {
				Type:     schema.TypeBool,
				Optional: true,
			},

			"ipv6": {
				Type:     schema.TypeBool,
				Optional: true,
			},

			"ipv6_address": {
				Type:     schema.TypeString,
				Computed: true,
				StateFunc: func(val interface{}) string {
					return strings.ToLower(val.(string))
				},
			},

			"ipv6_address_private": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"private_networking": {
				Type:     schema.TypeBool,
				Optional: true,
			},

			"ipv4_address": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"ipv4_address_private": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"ssh_keys": {
				Type:     schema.TypeList,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"tags": {
				Type:     schema.TypeList,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"user_data": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"volume_ids": {
				Type:     schema.TypeList,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
			},
		},
	}
}

func resourceDigitalOceanDropletCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*godo.Client)

	// Build up our creation options
	opts := &godo.DropletCreateRequest{
		Image: godo.DropletCreateImage{
			Slug: d.Get("image").(string),
		},
		Name:   d.Get("name").(string),
		Region: d.Get("region").(string),
		Size:   d.Get("size").(string),
	}

	if attr, ok := d.GetOk("backups"); ok {
		opts.Backups = attr.(bool)
	}

	if attr, ok := d.GetOk("ipv6"); ok {
		opts.IPv6 = attr.(bool)
	}

	if attr, ok := d.GetOk("private_networking"); ok {
		opts.PrivateNetworking = attr.(bool)
	}

	if attr, ok := d.GetOk("user_data"); ok {
		opts.UserData = attr.(string)
	}

	if attr, ok := d.GetOk("volume_ids"); ok {
		for _, id := range attr.([]interface{}) {
			opts.Volumes = append(opts.Volumes, godo.DropletCreateVolume{
				ID: id.(string),
			})
		}
	}

	// Get configured ssh_keys
	sshKeys := d.Get("ssh_keys.#").(int)
	if sshKeys > 0 {
		opts.SSHKeys = make([]godo.DropletCreateSSHKey, 0, sshKeys)
		for i := 0; i < sshKeys; i++ {
			key := fmt.Sprintf("ssh_keys.%d", i)
			sshKeyRef := d.Get(key).(string)

			var sshKey godo.DropletCreateSSHKey
			// sshKeyRef can be either an ID or a fingerprint
			if id, err := strconv.Atoi(sshKeyRef); err == nil {
				sshKey.ID = id
			} else {
				sshKey.Fingerprint = sshKeyRef
			}

			opts.SSHKeys = append(opts.SSHKeys, sshKey)
		}
	}

	log.Printf("[DEBUG] Droplet create configuration: %#v", opts)

	droplet, _, err := client.Droplets.Create(context.Background(), opts)

	if err != nil {
		return fmt.Errorf("Error creating droplet: %s", err)
	}

	// Assign the droplets id
	d.SetId(strconv.Itoa(droplet.ID))

	log.Printf("[INFO] Droplet ID: %s", d.Id())

	_, err = WaitForDropletAttribute(d, "active", []string{"new"}, "status", meta)
	if err != nil {
		return fmt.Errorf(
			"Error waiting for droplet (%s) to become ready: %s", d.Id(), err)
	}

	// droplet needs to be active in order to set tags
	err = setTags(client, d)
	if err != nil {
		return fmt.Errorf("Error setting tags: %s", err)
	}

	return resourceDigitalOceanDropletRead(d, meta)
}

func resourceDigitalOceanDropletRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*godo.Client)

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return fmt.Errorf("invalid droplet id: %v", err)
	}

	// Retrieve the droplet properties for updating the state
	droplet, resp, err := client.Droplets.Get(context.Background(), id)
	if err != nil {
		// check if the droplet no longer exists.
		if resp != nil && resp.StatusCode == 404 {
			log.Printf("[WARN] DigitalOcean Droplet (%s) not found", d.Id())
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Error retrieving droplet: %s", err)
	}

	_, err = strconv.Atoi(d.Get("image").(string))
	if err == nil || droplet.Image.Slug == "" {
		// The image field is provided as an ID (number), or
		// the image bash no slug. In both cases we store it as an ID.
		d.Set("image", droplet.Image.ID)
	} else {
		d.Set("image", droplet.Image.Slug)
	}

	d.Set("name", droplet.Name)
	d.Set("region", droplet.Region.Slug)
	d.Set("size", droplet.Size.Slug)
	d.Set("price_hourly", droplet.Size.PriceHourly)
	d.Set("price_monthly", droplet.Size.PriceMonthly)
	d.Set("disk", droplet.Disk)
	d.Set("vcpus", droplet.Vcpus)
	d.Set("status", droplet.Status)
	d.Set("locked", strconv.FormatBool(droplet.Locked))

	if len(droplet.VolumeIDs) > 0 {
		vlms := make([]interface{}, 0, len(droplet.VolumeIDs))
		for _, vid := range droplet.VolumeIDs {
			vlms = append(vlms, vid)
		}
		d.Set("volume_ids", vlms)
	}

	if publicIPv6 := findIPv6AddrByType(droplet, "public"); publicIPv6 != "" {
		d.Set("ipv6", true)
		d.Set("ipv6_address", strings.ToLower(publicIPv6))
		d.Set("ipv6_address_private", findIPv6AddrByType(droplet, "private"))
	}

	d.Set("ipv4_address", findIPv4AddrByType(droplet, "public"))

	if privateIPv4 := findIPv4AddrByType(droplet, "private"); privateIPv4 != "" {
		d.Set("private_networking", true)
		d.Set("ipv4_address_private", privateIPv4)
	}

	// Initialize the connection info
	d.SetConnInfo(map[string]string{
		"type": "ssh",
		"host": findIPv4AddrByType(droplet, "public"),
	})

	d.Set("tags", droplet.Tags)

	return nil
}

func findIPv6AddrByType(d *godo.Droplet, addrType string) string {
	for _, addr := range d.Networks.V6 {
		if addr.Type == addrType {
			return addr.IPAddress
		}
	}
	return ""
}

func findIPv4AddrByType(d *godo.Droplet, addrType string) string {
	for _, addr := range d.Networks.V4 {
		if addr.Type == addrType {
			return addr.IPAddress
		}
	}
	return ""
}

func resourceDigitalOceanDropletUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*godo.Client)

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return fmt.Errorf("invalid droplet id: %v", err)
	}

	resize_disk := d.Get("resize_disk").(bool)
	if d.HasChange("size") || d.HasChange("resize_disk") && resize_disk {
		newSize := d.Get("size")

		_, _, err = client.DropletActions.PowerOff(context.Background(), id)
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
		action, _, err := client.DropletActions.Resize(context.Background(), id, newSize.(string), resize_disk)
		if err != nil {
			newErr := powerOnAndWait(d, meta)
			if newErr != nil {
				return fmt.Errorf(
					"Error powering on droplet (%s) after failed resize: %s", d.Id(), err)
			}
			return fmt.Errorf(
				"Error resizing droplet (%s): %s", d.Id(), err)
		}

		// Wait for the resize action to complete.
		if err := waitForAction(client, action); err != nil {
			newErr := powerOnAndWait(d, meta)
			if newErr != nil {
				return fmt.Errorf(
					"Error powering on droplet (%s) after waiting for resize to finish: %s", d.Id(), err)
			}
			return fmt.Errorf(
				"Error waiting for resize droplet (%s) to finish: %s", d.Id(), err)
		}

		_, _, err = client.DropletActions.PowerOn(context.Background(), id)

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
		_, _, err = client.DropletActions.Rename(context.Background(), id, newName.(string))

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
		_, _, err = client.DropletActions.EnablePrivateNetworking(context.Background(), id)

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
		_, _, err = client.DropletActions.EnableIPv6(context.Background(), id)

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

	if d.HasChange("tags") {
		err = setTags(client, d)
		if err != nil {
			return fmt.Errorf("Error updating tags: %s", err)
		}
	}

	if d.HasChange("volume_ids") {
		oldIDs, newIDs := d.GetChange("volume_ids")
		newSet := func(ids []interface{}) map[string]struct{} {
			out := make(map[string]struct{}, len(ids))
			for _, id := range ids {
				out[id.(string)] = struct{}{}
			}
			return out
		}
		// leftDiff returns all elements in Left that are not in Right
		leftDiff := func(left, right map[string]struct{}) map[string]struct{} {
			out := make(map[string]struct{})
			for l := range left {
				if _, ok := right[l]; !ok {
					out[l] = struct{}{}
				}
			}
			return out
		}
		oldIDSet := newSet(oldIDs.([]interface{}))
		newIDSet := newSet(newIDs.([]interface{}))
		for volumeID := range leftDiff(newIDSet, oldIDSet) {
			action, _, err := client.StorageActions.Attach(context.Background(), volumeID, id)
			if err != nil {
				return fmt.Errorf("Error attaching volume %q to droplet (%s): %s", volumeID, d.Id(), err)
			}
			// can't fire >1 action at a time, so waiting for each is OK
			if err := waitForAction(client, action); err != nil {
				return fmt.Errorf("Error waiting for volume %q to attach to droplet (%s): %s", volumeID, d.Id(), err)
			}
		}
		for volumeID := range leftDiff(oldIDSet, newIDSet) {
			action, _, err := client.StorageActions.DetachByDropletID(context.Background(), volumeID, id)
			if err != nil {
				return fmt.Errorf("Error detaching volume %q from droplet (%s): %s", volumeID, d.Id(), err)
			}
			// can't fire >1 action at a time, so waiting for each is OK
			if err := waitForAction(client, action); err != nil {
				return fmt.Errorf("Error waiting for volume %q to detach from droplet (%s): %s", volumeID, d.Id(), err)
			}
		}
	}

	return resourceDigitalOceanDropletRead(d, meta)
}

func resourceDigitalOceanDropletDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*godo.Client)

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return fmt.Errorf("invalid droplet id: %v", err)
	}

	_, err = WaitForDropletAttribute(
		d, "false", []string{"", "true"}, "locked", meta)

	if err != nil {
		return fmt.Errorf(
			"Error waiting for droplet to be unlocked for destroy (%s): %s", d.Id(), err)
	}

	log.Printf("[INFO] Deleting droplet: %s", d.Id())

	// Destroy the droplet
	_, err = client.Droplets.Delete(context.Background(), id)

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
		Target:     []string{target},
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
	client := meta.(*godo.Client)
	return func() (interface{}, string, error) {
		id, err := strconv.Atoi(d.Id())
		if err != nil {
			return nil, "", err
		}

		err = resourceDigitalOceanDropletRead(d, meta)
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
			droplet, _, err := client.Droplets.Get(context.Background(), id)
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
	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return fmt.Errorf("invalid droplet id: %v", err)
	}

	client := meta.(*godo.Client)
	_, _, err = client.DropletActions.PowerOn(context.Background(), id)
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
