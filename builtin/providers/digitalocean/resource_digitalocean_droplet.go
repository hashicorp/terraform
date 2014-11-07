package digitalocean

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/terraform/flatmap"
	"github.com/hashicorp/terraform/helper/config"
	"github.com/hashicorp/terraform/helper/diff"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/pearkes/digitalocean"
)

func resource_digitalocean_droplet_create(
	s *terraform.InstanceState,
	d *terraform.InstanceDiff,
	meta interface{}) (*terraform.InstanceState, error) {
	p := meta.(*ResourceProvider)
	client := p.client

	// Merge the diff into the state so that we have all the attributes
	// properly.
	rs := s.MergeDiff(d)

	// Build up our creation options
	opts := digitalocean.CreateDroplet{
		Backups:           rs.Attributes["backups"],
		Image:             rs.Attributes["image"],
		IPV6:              rs.Attributes["ipv6"],
		Name:              rs.Attributes["name"],
		PrivateNetworking: rs.Attributes["private_networking"],
		Region:            rs.Attributes["region"],
		Size:              rs.Attributes["size"],
		UserData:          rs.Attributes["user_data"],
	}

	// Only expand ssh_keys if we have them
	if _, ok := rs.Attributes["ssh_keys.#"]; ok {
		v := flatmap.Expand(rs.Attributes, "ssh_keys").([]interface{})
		if len(v) > 0 {
			vs := make([]string, 0, len(v))

			// here we special case the * expanded lists. For example:
			//
			//	 ssh_keys = ["${digitalocean_key.foo.*.id}"]
			//
			if len(v) == 1 && strings.Contains(v[0].(string), ",") {
				vs = strings.Split(v[0].(string), ",")
			}

			for _, v := range v {
				vs = append(vs, v.(string))
			}

			opts.SSHKeys = vs
		}
	}

	log.Printf("[DEBUG] Droplet create configuration: %#v", opts)

	id, err := client.CreateDroplet(&opts)

	if err != nil {
		return nil, fmt.Errorf("Error creating Droplet: %s", err)
	}

	// Assign the droplets id
	rs.ID = id

	log.Printf("[INFO] Droplet ID: %s", id)

	dropletRaw, err := WaitForDropletAttribute(id, "active", []string{"new"}, "status", client)

	if err != nil {
		return rs, fmt.Errorf(
			"Error waiting for droplet (%s) to become ready: %s",
			id, err)
	}

	droplet := dropletRaw.(*digitalocean.Droplet)

	// Initialize the connection info
	rs.Ephemeral.ConnInfo["type"] = "ssh"
	rs.Ephemeral.ConnInfo["host"] = droplet.IPV4Address("public")

	return resource_digitalocean_droplet_update_state(rs, droplet)
}

func resource_digitalocean_droplet_update(
	s *terraform.InstanceState,
	d *terraform.InstanceDiff,
	meta interface{}) (*terraform.InstanceState, error) {
	p := meta.(*ResourceProvider)
	client := p.client
	rs := s.MergeDiff(d)

	var err error

	if attr, ok := d.Attributes["size"]; ok {
		err = client.PowerOff(rs.ID)

		if err != nil && !strings.Contains(err.Error(), "Droplet is already powered off") {
			return s, err
		}

		// Wait for power off
		_, err = WaitForDropletAttribute(
			rs.ID, "off", []string{"active"}, "status", client)

		err = client.Resize(rs.ID, attr.New)

		if err != nil {
			newErr := power_on_and_wait(rs.ID, client)
			if newErr != nil {
				return rs, newErr
			}
			return rs, err
		}

		// Wait for the size to change
		_, err = WaitForDropletAttribute(
			rs.ID, attr.New, []string{"", attr.Old}, "size", client)

		if err != nil {
			newErr := power_on_and_wait(rs.ID, client)
			if newErr != nil {
				return rs, newErr
			}
			return s, err
		}

		err = client.PowerOn(rs.ID)

		if err != nil {
			return s, err
		}

		// Wait for power off
		_, err = WaitForDropletAttribute(
			rs.ID, "active", []string{"off"}, "status", client)

		if err != nil {
			return s, err
		}
	}

	if attr, ok := d.Attributes["name"]; ok {
		err = client.Rename(rs.ID, attr.New)

		if err != nil {
			return s, err
		}

		// Wait for the name to change
		_, err = WaitForDropletAttribute(
			rs.ID, attr.New, []string{"", attr.Old}, "name", client)
	}

	if attr, ok := d.Attributes["private_networking"]; ok {
		err = client.Rename(rs.ID, attr.New)

		if err != nil {
			return s, err
		}

		// Wait for the private_networking to turn on/off
		_, err = WaitForDropletAttribute(
			rs.ID, attr.New, []string{"", attr.Old}, "private_networking", client)
	}

	if attr, ok := d.Attributes["ipv6"]; ok {
		err = client.Rename(rs.ID, attr.New)

		if err != nil {
			return s, err
		}

		// Wait for ipv6 to turn on/off
		_, err = WaitForDropletAttribute(
			rs.ID, attr.New, []string{"", attr.Old}, "ipv6", client)
	}

	droplet, err := resource_digitalocean_droplet_retrieve(rs.ID, client)

	if err != nil {
		return s, err
	}

	return resource_digitalocean_droplet_update_state(rs, droplet)
}

func resource_digitalocean_droplet_destroy(
	s *terraform.InstanceState,
	meta interface{}) error {
	p := meta.(*ResourceProvider)
	client := p.client

	log.Printf("[INFO] Deleting Droplet: %s", s.ID)

	// Destroy the droplet
	err := client.DestroyDroplet(s.ID)

	// Handle remotely destroyed droplets
	if err != nil && strings.Contains(err.Error(), "404 Not Found") {
		return nil
	}

	if err != nil {
		return fmt.Errorf("Error deleting Droplet: %s", err)
	}

	return nil
}

func resource_digitalocean_droplet_refresh(
	s *terraform.InstanceState,
	meta interface{}) (*terraform.InstanceState, error) {
	p := meta.(*ResourceProvider)
	client := p.client

	droplet, err := resource_digitalocean_droplet_retrieve(s.ID, client)

	// Handle remotely destroyed droplets
	if err != nil && strings.Contains(err.Error(), "404 Not Found") {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return resource_digitalocean_droplet_update_state(s, droplet)
}

func resource_digitalocean_droplet_diff(
	s *terraform.InstanceState,
	c *terraform.ResourceConfig,
	meta interface{}) (*terraform.InstanceDiff, error) {

	b := &diff.ResourceBuilder{
		Attrs: map[string]diff.AttrType{
			"backups":            diff.AttrTypeUpdate,
			"image":              diff.AttrTypeCreate,
			"ipv6":               diff.AttrTypeUpdate,
			"name":               diff.AttrTypeUpdate,
			"private_networking": diff.AttrTypeUpdate,
			"region":             diff.AttrTypeCreate,
			"size":               diff.AttrTypeUpdate,
			"ssh_keys":           diff.AttrTypeCreate,
			"user_data":          diff.AttrTypeCreate,
		},

		ComputedAttrs: []string{
			"backups",
			"ipv4_address",
			"ipv4_address_private",
			"ipv6",
			"ipv6_address",
			"ipv6_address_private",
			"locked",
			"private_networking",
			"status",
		},
	}

	return b.Diff(s, c)
}

func resource_digitalocean_droplet_update_state(
	s *terraform.InstanceState,
	droplet *digitalocean.Droplet) (*terraform.InstanceState, error) {

	s.Attributes["name"] = droplet.Name
	s.Attributes["region"] = droplet.RegionSlug()

	if droplet.ImageSlug() == "" && droplet.ImageId() != "" {
		s.Attributes["image"] = droplet.ImageId()
	} else {
		s.Attributes["image"] = droplet.ImageSlug()
	}

	if droplet.IPV6Address("public") != "" {
		s.Attributes["ipv6"] = "true"
		s.Attributes["ipv6_address"] = droplet.IPV6Address("public")
		s.Attributes["ipv6_address_private"] = droplet.IPV6Address("private")
	}

	s.Attributes["ipv4_address"] = droplet.IPV4Address("public")
	s.Attributes["locked"] = droplet.IsLocked()

	if droplet.NetworkingType() == "private" {
		s.Attributes["private_networking"] = "true"
		s.Attributes["ipv4_address_private"] = droplet.IPV4Address("private")
	}

	s.Attributes["size"] = droplet.SizeSlug()
	s.Attributes["status"] = droplet.Status

	return s, nil
}

// retrieves an ELB by its ID
func resource_digitalocean_droplet_retrieve(id string, client *digitalocean.Client) (*digitalocean.Droplet, error) {
	// Retrieve the ELB properties for updating the state
	droplet, err := client.RetrieveDroplet(id)

	if err != nil {
		return nil, fmt.Errorf("Error retrieving droplet: %s", err)
	}

	return &droplet, nil
}

func resource_digitalocean_droplet_validation() *config.Validator {
	return &config.Validator{
		Required: []string{
			"image",
			"name",
			"region",
			"size",
		},
		Optional: []string{
			"backups",
			"user_data",
			"ipv6",
			"private_networking",
			"ssh_keys.*",
		},
	}
}

func WaitForDropletAttribute(id string, target string, pending []string, attribute string, client *digitalocean.Client) (interface{}, error) {
	// Wait for the droplet so we can get the networking attributes
	// that show up after a while
	log.Printf(
		"[INFO] Waiting for Droplet (%s) to have %s of %s",
		id, attribute, target)

	stateConf := &resource.StateChangeConf{
		Pending:    pending,
		Target:     target,
		Refresh:    new_droplet_state_refresh_func(id, attribute, client),
		Timeout:    10 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	return stateConf.WaitForState()
}

func new_droplet_state_refresh_func(id string, attribute string, client *digitalocean.Client) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		// Retrieve the ELB properties for updating the state
		droplet, err := client.RetrieveDroplet(id)

		if err != nil {
			log.Printf("Error on retrieving droplet when waiting: %s", err)
			return nil, "", err
		}

		// If the droplet is locked, continue waiting. We can
		// only perform actions on unlocked droplets, so it's
		// pointless to look at that status
		if droplet.IsLocked() == "true" {
			log.Println("[DEBUG] Droplet is locked, skipping status check and retrying")
			return nil, "", nil
		}

		// Use our mapping to get back a map of the
		// droplet properties
		resourceMap, err := resource_digitalocean_droplet_update_state(
			&terraform.InstanceState{Attributes: map[string]string{}}, &droplet)

		if err != nil {
			log.Printf("Error creating map from droplet: %s", err)
			return nil, "", err
		}

		// See if we can access our attribute
		if attr, ok := resourceMap.Attributes[attribute]; ok {
			return &droplet, attr, nil
		}

		return nil, "", nil
	}
}

// Powers on the droplet and waits for it to be active
func power_on_and_wait(id string, client *digitalocean.Client) error {
	err := client.PowerOn(id)

	if err != nil {
		return err
	}

	// Wait for power on
	_, err = WaitForDropletAttribute(
		id, "active", []string{"off"}, "status", client)

	if err != nil {
		return err
	}

	return nil
}
