package openstack

import (
	"log"

	"github.com/hashicorp/terraform/helper/config"
	"github.com/hashicorp/terraform/terraform"
)

type ResourceProvider struct {
	Config Config

	client *OpenstackClient
}

func (p *ResourceProvider) Validate(c *terraform.ResourceConfig) ([]string, []error) {
	v := &config.Validator{
		Required: []string{
			"url",
			"user",
			"password",
			"tenantName",
			"computeEndpoint",
		},
	}

	return v.Validate(c)
}

func (p *ResourceProvider) ValidateResource(
	t string, c *terraform.ResourceConfig) ([]string, []error) {
	return resourceMap.Validate(t, c)
}

func (p *ResourceProvider) Configure(c *terraform.ResourceConfig) error {
	if _, err := config.Decode(&p.Config, c.Config); err != nil {
		return err
	}
	log.Println("[INFO] Initializing OpenStack client")

	var err error
	p.client, err = p.Config.Client()

	if err != nil {
		return err
	}

	log.Println("[INFO] OpenStack client connected")

	return nil
}

func (p *ResourceProvider) Apply(
	s *terraform.ResourceState,
	d *terraform.ResourceDiff) (*terraform.ResourceState, error) {
	return resourceMap.Apply(s, d, p)
}

func (p *ResourceProvider) Diff(
	s *terraform.ResourceState,
	c *terraform.ResourceConfig) (*terraform.ResourceDiff, error) {
	return resourceMap.Diff(s, c, p)
}

func (p *ResourceProvider) Refresh(
	s *terraform.ResourceState) (*terraform.ResourceState, error) {
	return resourceMap.Refresh(s, p)
}

func (p *ResourceProvider) Resources() []terraform.ResourceType {
	return resourceMap.Resources()
}
