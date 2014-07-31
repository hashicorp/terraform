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

	return nil, nil
}

func resource_openstack_compute_update(
	s *terraform.ResourceState,
	d *terraform.ResourceDiff,
	meta interface{}) (*terraform.ResourceState, error) {

	log.Printf("[INFO] update")

	return nil, nil
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

	return nil, nil
}

func resource_openstack_compute_diff(
	s *terraform.ResourceState,
	c *terraform.ResourceConfig,
	meta interface{}) (*terraform.ResourceDiff, error) {

  log.Printf("[INFO] diff")

  b := &diff.ResourceBuilder{
    Attrs: map[string]diff.AttrType{

    },

    ComputedAttrs: []string{

    },

    ComputedAttrsUpdate: []string{

    },
  }

  return b.Diff(s, c)

	return nil, nil
}
