package aws

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/config"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/goamz/ec2"
)

type ResourceProvider struct {
	Config Config

	ec2conn *ec2.EC2
}

func (p *ResourceProvider) Validate(c *terraform.ResourceConfig) ([]string, []error) {
	return nil, nil
}

func (p *ResourceProvider) Configure(c *terraform.ResourceConfig) error {
	if _, err := config.Decode(&p.Config, c.Config); err != nil {
		return err
	}

	// Get the auth and region. This can fail if keys/regions were not
	// specified and we're attempting to use the environment.
	var errs []error
	log.Println("Building AWS auth structure")
	auth, err := p.Config.AWSAuth()
	if err != nil {
		errs = append(errs, err)
	}

	log.Println("Building AWS region structure")
	region, err := p.Config.AWSRegion()
	if err != nil {
		errs = append(errs, err)
	}

	if len(errs) == 0 {
		log.Println("Initializing EC2 connection")
		p.ec2conn = ec2.New(auth, region)
	}

	if len(errs) > 0 {
		return &terraform.MultiError{Errors: errs}
	}

	return nil
}

func (p *ResourceProvider) Apply(
	s *terraform.ResourceState,
	d *terraform.ResourceDiff) (*terraform.ResourceState, error) {
	result := &terraform.ResourceState{
		ID: "foo",
	}
	result = result.MergeDiff(d)
	result.Attributes["public_dns"] = "foo"
	result.Attributes["public_ip"] = "foo"
	result.Attributes["private_dns"] = "foo"
	result.Attributes["private_ip"] = "foo"

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
	// If there isn't an ID previously, then the thing didn't exist,
	// so there is nothing to refresh.
	if s.ID == "" {
		return s, nil
	}

	f, ok := refreshMap[s.Type]
	if !ok {
		return s, fmt.Errorf("Unknown resource type: %s", s.Type)
	}

	return f(p, s)
}

func (p *ResourceProvider) Resources() []terraform.ResourceType {
	return []terraform.ResourceType{
		terraform.ResourceType{
			Name: "aws_instance",
		},
	}
}
