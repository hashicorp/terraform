package consul

import (
	"github.com/hashicorp/terraform/helper/config"
	"github.com/hashicorp/terraform/terraform"
)

func resource_consul_keys_validation() *config.Validator {
	return &config.Validator{
		Optional: []string{
			"datacenter",
			"*.key",
			"*.value",
			"*.default",
			"*.delete",
		},
	}
}

func resource_consul_keys_create(
	s *terraform.ResourceState,
	d *terraform.ResourceDiff,
	meta interface{}) (*terraform.ResourceState, error) {
	// Merge the diff into the state so that we have all the attributes
	// properly.
	rs := s.MergeDiff(d)
	return rs, nil
}

func resource_consul_keys_destroy(
	s *terraform.ResourceState,
	meta interface{}) error {
	return nil
}

func resource_consul_keys_update(
	s *terraform.ResourceState,
	d *terraform.ResourceDiff,
	meta interface{}) (*terraform.ResourceState, error) {
	// Merge the diff into the state so that we have all the attributes
	// properly.
	rs := s.MergeDiff(d)
	return rs, nil
}

func resource_consul_keys_diff(
	s *terraform.ResourceState,
	c *terraform.ResourceConfig,
	meta interface{}) (*terraform.ResourceDiff, error) {
	return nil, nil
}

func resource_consul_keys_refresh(
	s *terraform.ResourceState,
	meta interface{}) (*terraform.ResourceState, error) {
	return s, nil
}
