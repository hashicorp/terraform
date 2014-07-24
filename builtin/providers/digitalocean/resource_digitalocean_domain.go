package digitalocean

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/config"
	"github.com/hashicorp/terraform/helper/diff"
	"github.com/hashicorp/terraform/terraform"
	"github.com/pearkes/digitalocean"
)

func resource_digitalocean_domain_create(
	s *terraform.ResourceState,
	d *terraform.ResourceDiff,
	meta interface{}) (*terraform.ResourceState, error) {
	p := meta.(*ResourceProvider)
	client := p.client
	// Merge the diff into the state so that we have all the attributes
	// properly.
	rs := s.MergeDiff(d)

	// Build up our creation options
	opts := digitalocean.CreateDomain{
		Name:      rs.Attributes["name"],
		IPAddress: rs.Attributes["ip_address"],
	}

	log.Printf("[DEBUG] Domain create configuration: %#v", opts)

	name, err := client.CreateDomain(&opts)
	if err != nil {
		return nil, fmt.Errorf("Error creating Domain: %s", err)
	}

	rs.ID = name
	log.Printf("[INFO] Domain Name: %s", name)

	return rs, nil
}

func resource_digitalocean_domain_destroy(
	s *terraform.ResourceState,
	meta interface{}) error {
	p := meta.(*ResourceProvider)
	client := p.client

	log.Printf("[INFO] Deleting Domain: %s", s.ID)

	err := client.DestroyDomain(s.ID)

	if err != nil {
		return fmt.Errorf("Error deleting Domain: %s", err)
	}

	return nil
}

func resource_digitalocean_domain_refresh(
	s *terraform.ResourceState,
	meta interface{}) (*terraform.ResourceState, error) {
	p := meta.(*ResourceProvider)
	client := p.client

	domain, err := client.RetrieveDomain(s.ID)

	if err != nil {
		return s, fmt.Errorf("Error retrieving domain: %s", err)
	}

	s.Attributes["name"] = domain.Name

	return s, nil
}

func resource_digitalocean_domain_diff(
	s *terraform.ResourceState,
	c *terraform.ResourceConfig,
	meta interface{}) (*terraform.ResourceDiff, error) {

	b := &diff.ResourceBuilder{
		Attrs: map[string]diff.AttrType{
			"name":       diff.AttrTypeCreate,
			"ip_address": diff.AttrTypeCreate,
		},

		ComputedAttrs: []string{},
	}

	return b.Diff(s, c)
}

func resource_digitalocean_domain_validation() *config.Validator {
	return &config.Validator{
		Required: []string{
			"name",
			"ip_address",
		},
		Optional: []string{},
	}
}
