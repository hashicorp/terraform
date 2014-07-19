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
		Name:              rs.Attributes["name"],
		Region:            rs.Attributes["region"],
		Image:             rs.Attributes["image"],
		Size:              rs.Attributes["size"],
		Backups:           rs.Attributes["backups"],
		IPV6:              rs.Attributes["ipv6"],
		PrivateNetworking: rs.Attributes["private_networking"],
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

	// Wait for the droplet so we can get the networking attributes
	// that show up after a while
	log.Printf(
		"[DEBUG] Waiting for Droplet (%s) to become running",
		id)

	stateConf := &resource.StateChangeConf{
		Pending: []string{"new"},
		Target:  "active",
		Refresh: DropletStateRefreshFunc(client, id),
		Timeout: 10 * time.Minute,
	}

	dropletRaw, err := stateConf.WaitForState()

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

	panic("No update")

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
			"name":               diff.AttrTypeUpdate,
			"backups":            diff.AttrTypeUpdate,
			"ipv6":               diff.AttrTypeUpdate,
			"private_networking": diff.AttrTypeUpdate,
			"region":             diff.AttrTypeCreate,
			"image":              diff.AttrTypeCreate,
			"size":               diff.AttrTypeCreate,
			"ssh_keys":           diff.AttrTypeCreate,
		},

		ComputedAttrs: []string{
			"ipv4_address",
			"ipv6_address",
			"status",
			"locked",
			"private_networking",
			"ipv6",
			"backups",
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

	s.Attributes["size"] = droplet.SizeSlug()
	s.Attributes["private_networking"] = droplet.NetworkingType()
	s.Attributes["locked"] = droplet.IsLocked()
	s.Attributes["status"] = droplet.Status
	s.Attributes["ipv4_address"] = droplet.IPV4Address()
	s.Attributes["ipv6_address"] = droplet.IPV6Address()

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
			"name",
			"region",
			"size",
			"image",
		},
		Optional: []string{
			"ssh_keys.*",
			"backups",
			"ipv6",
			"private_networking",
		},
	}
}

// DropletStateRefreshFunc returns a resource.StateRefreshFunc that is used to watch
// a droplet.
func DropletStateRefreshFunc(client *digitalocean.Client, id string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		droplet, err := resource_digitalocean_droplet_retrieve(id, client)

		// It's not actually "active"
		// until we can see the image slug
		if droplet.ImageSlug() == "" {
			return nil, "", nil
		}

		if err != nil {
			log.Printf("Error on DropletStateRefresh: %s", err)
			return nil, "", err
		}

		return droplet, droplet.Status, nil
	}
}
