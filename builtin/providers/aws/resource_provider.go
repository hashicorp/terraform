package aws

import (
	"fmt"

	"github.com/hashicorp/terraform/terraform"
)

type ResourceProvider struct {
}

func (p *ResourceProvider) Validate(c *terraform.ResourceConfig) ([]string, []error) {
	errs := c.CheckSet([]string{
		"access_key",
		"secret_key",
	})

	return nil, errs
}

func (p *ResourceProvider) Configure(*terraform.ResourceConfig) error {
	return nil
}

func (p *ResourceProvider) Apply(
	s *terraform.ResourceState,
	d *terraform.ResourceDiff) (*terraform.ResourceState, error) {
	result := &terraform.ResourceState{
		ID: "foo",
	}
	result = result.MergeDiff(d)

	return result, nil
}

func (p *ResourceProvider) Diff(
	s *terraform.ResourceState,
	c *terraform.ResourceConfig) (*terraform.ResourceDiff, error) {
	b := diffMap.Get(s.Type)
	if b == nil {
		return nil, fmt.Errorf("Unknown type: %s", s.Type)
	}

	return b.Diff(s, c)
}

func (p *ResourceProvider) Refresh(
	s *terraform.ResourceState) (*terraform.ResourceState, error) {
	return s, nil
}

func (p *ResourceProvider) Resources() []terraform.ResourceType {
	return []terraform.ResourceType{
		terraform.ResourceType{
			Name: "aws_instance",
		},
	}
}
