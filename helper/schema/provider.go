package schema

import (
	"fmt"
	"sort"

	"github.com/hashicorp/terraform/terraform"
)

// Provider represents a Resource provider in Terraform, and properly
// implements all of the ResourceProvider API.
//
// This is a friendlier API than the core Terraform ResourceProvider API,
// and is recommended to be used over that.
type Provider struct {
	Schema       map[string]*Schema
	ResourcesMap map[string]*Resource

	ConfigureFunc ConfigureFunc

	meta interface{}
}

// ConfigureFunc is the function used to configure a Provider.
//
// The interface{} value returned by this function is stored and passed into
// the subsequent resources as the meta parameter.
type ConfigureFunc func(*ResourceData) (interface{}, error)

// Validate validates the provider configuration against the schema.
func (p *Provider) Validate(c *terraform.ResourceConfig) ([]string, []error) {
	return schemaMap(p.Schema).Validate(c)
}

// ValidateResource validates the resource configuration against the
// proper schema.
func (p *Provider) ValidateResource(
	t string, c *terraform.ResourceConfig) ([]string, []error) {
	r, ok := p.ResourcesMap[t]
	if !ok {
		return nil, []error{fmt.Errorf(
			"Provider doesn't support resource: %s", t)}
	}

	return r.Validate(c)
}

// Configure implementation of terraform.ResourceProvider interface.
func (p *Provider) Configure(c *terraform.ResourceConfig) error {
	// No configuration
	if p.ConfigureFunc == nil {
		return nil
	}

	sm := schemaMap(p.Schema)

	// Get a ResourceData for this configuration. To do this, we actually
	// generate an intermediary "diff" although that is never exposed.
	diff, err := sm.Diff(nil, c)
	if err != nil {
		return err
	}

	data, err := sm.Data(nil, diff)
	if err != nil {
		return err
	}

	meta, err := p.ConfigureFunc(data)
	if err != nil {
		return err
	}

	p.meta = meta
	return nil
}

// Apply implementation of terraform.ResourceProvider interface.
func (p *Provider) Apply(
	s *terraform.ResourceState,
	d *terraform.ResourceDiff) (*terraform.ResourceState, error) {
	r, ok := p.ResourcesMap[s.Type]
	if !ok {
		return nil, fmt.Errorf("unknown resource type: %s", s.Type)
	}

	return r.Apply(s, d, p.meta)
}

// Diff implementation of terraform.ResourceProvider interface.
func (p *Provider) Diff(
	s *terraform.ResourceState,
	c *terraform.ResourceConfig) (*terraform.ResourceDiff, error) {
	r, ok := p.ResourcesMap[s.Type]
	if !ok {
		return nil, fmt.Errorf("unknown resource type: %s", s.Type)
	}

	return r.Diff(s, c)
}

// Refresh implementation of terraform.ResourceProvider interface.
func (p *Provider) Refresh(
	s *terraform.ResourceState) (*terraform.ResourceState, error) {
	r, ok := p.ResourcesMap[s.Type]
	if !ok {
		return nil, fmt.Errorf("unknown resource type: %s", s.Type)
	}

	return r.Refresh(s, p.meta)
}

// Resources implementation of terraform.ResourceProvider interface.
func (p *Provider) Resources() []terraform.ResourceType {
	keys := make([]string, 0, len(p.ResourcesMap))
	for k, _ := range p.ResourcesMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	result := make([]terraform.ResourceType, 0, len(keys))
	for _, k := range keys {
		result = append(result, terraform.ResourceType{
			Name: k,
		})
	}

	return result
}
