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
	s *terraform.ResourceState,
	d *terraform.ResourceDiff,
	meta interface{}) (*terraform.ResourceState, error) {
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

	return resource_digitalocean_droplet_update_state(rs, droplet)
}

func resource_digitalocean_droplet_update(
	s *terraform.ResourceState,
	d *terraform.ResourceDiff,
	meta interface{}) (*terraform.ResourceState, error) {
	// p := meta.(*ResourceProvider)
	// client := p.client
	// rs := s.MergeDiff(d)

	// var err error

	// if _, ok := d.Attributes["size"]; ok {

	// }

	return nil, nil
}

func resource_digitalocean_droplet_destroy(
	s *terraform.ResourceState,
	meta interface{}) error {
	p := meta.(*ResourceProvider)
	client := p.client

	log.Printf("[INFO] Deleting Droplet: %s", s.ID)

	// Destroy the droplet
	err := client.DestroyDroplet(s.ID)

	if err != nil {
		return fmt.Errorf("Error deleting Droplet: %s", err)
	}

	return nil
}

func resource_digitalocean_droplet_refresh(
	s *terraform.ResourceState,
	meta interface{}) (*terraform.ResourceState, error) {
	p := meta.(*ResourceProvider)
	client := p.client

	droplet, err := resource_digitalocean_droplet_retrieve(s.ID, client)
	if err != nil {
		return nil, err
	}

	return resource_digitalocean_droplet_update_state(s, droplet)
}

func resource_digitalocean_droplet_diff(
	s *terraform.ResourceState,
	c *terraform.ResourceConfig,
	meta interface{}) (*terraform.ResourceDiff, error) {

	b := &diff.ResourceBuilder{
		Attrs: map[string]diff.AttrType{
			"backups":            diff.AttrTypeUpdate,
			"image":              diff.AttrTypeCreate,
			"ipv6":               diff.AttrTypeUpdate,
			"name":               diff.AttrTypeUpdate,
			"private_networking": diff.AttrTypeUpdate,
			"region":             diff.AttrTypeCreate,
			"size":               diff.AttrTypeCreate,
			"ssh_keys":           diff.AttrTypeCreate,
		},

		ComputedAttrs: []string{
			"backups",
			"ipv4_address",
			"ipv6",
			"ipv6_address",
			"locked",
			"private_networking",
			"status",
		},
	}

	return b.Diff(s, c)
}

func resource_digitalocean_droplet_update_state(
	s *terraform.ResourceState,
	droplet *digitalocean.Droplet) (*terraform.ResourceState, error) {

	s.Attributes["name"] = droplet.Name
	s.Attributes["region"] = droplet.RegionSlug()

	if droplet.ImageSlug() == "" && droplet.ImageId() != "" {
		s.Attributes["image"] = droplet.ImageId()
	} else {
		s.Attributes["image"] = droplet.ImageSlug()
	}

	s.Attributes["ipv4_address"] = droplet.IPV4Address()
	s.Attributes["ipv6_address"] = droplet.IPV6Address()
	s.Attributes["locked"] = droplet.IsLocked()
	s.Attributes["private_networking"] = droplet.NetworkingType()
	s.Attributes["size"] = droplet.SizeSlug()
	s.Attributes["status"] = droplet.Status

	return s, nil
}

// retrieves an ELB by it's ID
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
		"[DEBUG] Waiting for Droplet (%s) to have %s of %s",
		id, attribute, target)

	stateConf := &resource.StateChangeConf{
		Pending: pending,
		Target:  target,
		Refresh: new_droplet_state_refresh_func(id, attribute, client),
		Timeout: 10 * time.Minute,
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

		// Use our mapping to get back a map of the
		// droplet properties
		resourceMap, err := resource_digitalocean_droplet_update_state(
			&terraform.ResourceState{Attributes: map[string]string{}}, &droplet)

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
