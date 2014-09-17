package cloudflare

import (
	"log"

	"github.com/hashicorp/terraform/helper/config"
	"github.com/hashicorp/terraform/terraform"
	"github.com/pearkes/cloudflare"
)

type ResourceProvider struct {
	Config Config

	client *cloudflare.Client
}

func (p *ResourceProvider) Validate(c *terraform.ResourceConfig) ([]string, []error) {
	v := &config.Validator{
		Required: []string{
			"token",
			"email",
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

	log.Println("[INFO] Initializing CloudFlare client")
	var err error
	p.client, err = p.Config.Client()

	if err != nil {
		return err
	}

	return nil
}

func (p *ResourceProvider) Apply(
	info *terraform.InstanceInfo,
	s *terraform.InstanceState,
	d *terraform.InstanceDiff) (*terraform.InstanceState, error) {
	return resourceMap.Apply(info, s, d, p)
}

func (p *ResourceProvider) Diff(
	info *terraform.InstanceInfo,
	s *terraform.InstanceState,
	c *terraform.ResourceConfig) (*terraform.InstanceDiff, error) {
	return resourceMap.Diff(info, s, c, p)
}

func (p *ResourceProvider) Refresh(
	info *terraform.InstanceInfo,
	s *terraform.InstanceState) (*terraform.InstanceState, error) {
	return resourceMap.Refresh(info, s, p)
}

func (p *ResourceProvider) Resources() []terraform.ResourceType {
	return resourceMap.Resources()
}
