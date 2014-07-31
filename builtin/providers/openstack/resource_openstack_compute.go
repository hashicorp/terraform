package openstack

import (
	"log"

	"github.com/hashicorp/terraform/helper/diff"
	"github.com/hashicorp/terraform/terraform"
)

func resource_openstack_compute_create(
	s *terraform.ResourceState,
	d *terraform.ResourceDiff,
	meta interface{}) (*terraform.ResourceState, error) {

	log.Printf("[INFO] create")

	rs := s.MergeDiff(d)
	rs.Attributes["id"] = "1234"

	return rs, nil
}

func resource_openstack_compute_update(
	s *terraform.ResourceState,
	d *terraform.ResourceDiff,
	meta interface{}) (*terraform.ResourceState, error) {

	log.Printf("[INFO] update")

	return s, nil
}

func resource_openstack_compute_destroy(
	s *terraform.ResourceState,
	meta interface{}) error {

	log.Printf("[INFO] destroy")

	return nil
}

func resource_openstack_compute_refresh(
	s *terraform.ResourceState,
	meta interface{}) (*terraform.ResourceState, error) {

	log.Printf("[INFO] refresh")

	return s, nil
}

func resource_openstack_compute_diff(
	s *terraform.ResourceState,
	c *terraform.ResourceConfig,
	meta interface{}) (*terraform.ResourceDiff, error) {

	log.Printf("[INFO] diff")

	b := &diff.ResourceBuilder{
		Attrs: map[string]diff.AttrType{
			"imageRef":  diff.AttrTypeCreate,
			"flavorRef": diff.AttrTypeCreate,
		},

		ComputedAttrs: []string{
			"id",
			"name",
		},

		ComputedAttrsUpdate: []string{},
	}

	return b.Diff(s, c)
}
