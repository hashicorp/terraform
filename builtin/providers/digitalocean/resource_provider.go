package digitalocean

import (
	"log"

	"github.com/hashicorp/terraform/helper/config"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	"github.com/pearkes/digitalocean"
)

type ResourceProvider struct {
	Config Config

	client *digitalocean.Client

	// This is the schema.Provider. Eventually this will replace much
	// of this structure. For now it is an element of it for compatiblity.
	p *schema.Provider
}

func (p *ResourceProvider) Input(
	input terraform.UIInput,
	c *terraform.ResourceConfig) (*terraform.ResourceConfig, error) {
	return Provider().Input(input, c)
}

func (p *ResourceProvider) Validate(c *terraform.ResourceConfig) ([]string, []error) {
	prov := Provider()
	return prov.Validate(c)
}

func (p *ResourceProvider) ValidateResource(
	t string, c *terraform.ResourceConfig) ([]string, []error) {
	prov := Provider()
	if _, ok := prov.ResourcesMap[t]; ok {
		return prov.ValidateResource(t, c)
	}

	return resourceMap.Validate(t, c)
}

func (p *ResourceProvider) Configure(c *terraform.ResourceConfig) error {
	if _, err := config.Decode(&p.Config, c.Config); err != nil {
		return err
	}

	log.Println("[INFO] Initializing DigitalOcean client")
	var err error
	p.client, err = p.Config.Client()

	if err != nil {
		return err
	}

	// Create the provider, set the meta
	p.p = Provider()
	p.p.SetMeta(p)

	return nil
}

func (p *ResourceProvider) Apply(
	info *terraform.InstanceInfo,
	s *terraform.InstanceState,
	d *terraform.InstanceDiff) (*terraform.InstanceState, error) {
	if _, ok := p.p.ResourcesMap[info.Type]; ok {
		return p.p.Apply(info, s, d)
	}

	return resourceMap.Apply(info, s, d, p)
}

func (p *ResourceProvider) Diff(
	info *terraform.InstanceInfo,
	s *terraform.InstanceState,
	c *terraform.ResourceConfig) (*terraform.InstanceDiff, error) {
	if _, ok := p.p.ResourcesMap[info.Type]; ok {
		return p.p.Diff(info, s, c)
	}

	return resourceMap.Diff(info, s, c, p)
}

func (p *ResourceProvider) Refresh(
	info *terraform.InstanceInfo,
	s *terraform.InstanceState) (*terraform.InstanceState, error) {
	if _, ok := p.p.ResourcesMap[info.Type]; ok {
		return p.p.Refresh(info, s)
	}

	return resourceMap.Refresh(info, s, p)
}

func (p *ResourceProvider) Resources() []terraform.ResourceType {
	result := resourceMap.Resources()
	result = append(result, Provider().Resources()...)
	return result
}
