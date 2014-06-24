package aws

import (
	"github.com/hashicorp/terraform/terraform"
)

// RefreshFunc is a function that performs a refresh of a specific type
// of resource.
type RefreshFunc func(
	*ResourceProvider,
	*terraform.ResourceState) (*terraform.ResourceState, error)

// refreshMap keeps track of all the resources that this provider
// can refresh.
var refreshMap map[string]RefreshFunc

func init() {
	refreshMap = map[string]RefreshFunc{
		"aws_instance": refresh_aws_instance,
	}
}

func refresh_aws_instance(
	p *ResourceProvider,
	s *terraform.ResourceState) (*terraform.ResourceState, error) {
	if s.ID != "" {
		panic("OH MY WOW")
	}

	return s, nil
}
