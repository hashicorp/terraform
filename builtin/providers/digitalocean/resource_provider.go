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
	s *terraform.ResourceState,
	d *terraform.ResourceDiff) (*terraform.ResourceState, error) {
	if _, ok := p.p.ResourcesMap[s.Type]; ok {
		return p.p.Apply(s, d)
	}

	return resourceMap.Apply(s, d, p)
}

func (p *ResourceProvider) Diff(
	s *terraform.ResourceState,
	c *terraform.ResourceConfig) (*terraform.ResourceDiff, error) {
	if _, ok := p.p.ResourcesMap[s.Type]; ok {
		return p.p.Diff(s, c)
	}

	return resourceMap.Diff(s, c, p)
}

func (p *ResourceProvider) Refresh(
	s *terraform.ResourceState) (*terraform.ResourceState, error) {
	if _, ok := p.p.ResourcesMap[s.Type]; ok {
		return p.p.Refresh(s)
	}

	return resourceMap.Refresh(s, p)
}

func (p *ResourceProvider) Resources() []terraform.ResourceType {
	result := resourceMap.Resources()
	result = append(result, Provider().Resources()...)
	return result
}
