package heroku

import (
	"fmt"
	"log"

	"github.com/bgentry/heroku-go"
	"github.com/hashicorp/terraform/helper/config"
	"github.com/hashicorp/terraform/helper/diff"
	"github.com/hashicorp/terraform/terraform"
)

func resource_heroku_domain_create(
	s *terraform.ResourceState,
	d *terraform.ResourceDiff,
	meta interface{}) (*terraform.ResourceState, error) {
	p := meta.(*ResourceProvider)
	client := p.client

	// Merge the diff into the state so that we have all the attributes
	// properly.
	rs := s.MergeDiff(d)

	app := rs.Attributes["app"]
	hostname := rs.Attributes["hostname"]

	log.Printf("[DEBUG] Domain create configuration: %#v, %#v", app, hostname)

	do, err := client.DomainCreate(app, hostname)

	if err != nil {
		return s, err
	}

	rs.ID = do.Id
	rs.Attributes["hostname"] = do.Hostname
	rs.Attributes["cname"] = fmt.Sprintf("%s.herokuapp.com", app)

	log.Printf("[INFO] Domain ID: %s", rs.ID)

	return rs, nil
}

func resource_heroku_domain_update(
	s *terraform.ResourceState,
	d *terraform.ResourceDiff,
	meta interface{}) (*terraform.ResourceState, error) {

	panic("Cannot update domain")

	return nil, nil
}

func resource_heroku_domain_destroy(
	s *terraform.ResourceState,
	meta interface{}) error {
	p := meta.(*ResourceProvider)
	client := p.client

	log.Printf("[INFO] Deleting Domain: %s", s.ID)

	// Destroy the app
	err := client.DomainDelete(s.Attributes["app"], s.ID)

	if err != nil {
		return fmt.Errorf("Error deleting domain: %s", err)
	}

	return nil
}

func resource_heroku_domain_refresh(
	s *terraform.ResourceState,
	meta interface{}) (*terraform.ResourceState, error) {
	p := meta.(*ResourceProvider)
	client := p.client

	domain, err := resource_heroku_domain_retrieve(s.Attributes["app"], s.ID, client)
	if err != nil {
		return nil, err
	}

	s.Attributes["hostname"] = domain.Hostname
	s.Attributes["cname"] = fmt.Sprintf("%s.herokuapp.com", s.Attributes["app"])

	return s, nil
}

func resource_heroku_domain_diff(
	s *terraform.ResourceState,
	c *terraform.ResourceConfig,
	meta interface{}) (*terraform.ResourceDiff, error) {

	b := &diff.ResourceBuilder{
		Attrs: map[string]diff.AttrType{
			"hostname": diff.AttrTypeCreate,
			"app":      diff.AttrTypeCreate,
		},

		ComputedAttrs: []string{
			"cname",
		},
	}

	return b.Diff(s, c)
}

func resource_heroku_domain_retrieve(app string, id string, client *heroku.Client) (*heroku.Domain, error) {
	domain, err := client.DomainInfo(app, id)

	if err != nil {
		return nil, fmt.Errorf("Error retrieving domain: %s", err)
	}

	return domain, nil
}

func resource_heroku_domain_validation() *config.Validator {
	return &config.Validator{
		Required: []string{
			"hostname",
			"app",
		},
		Optional: []string{},
	}
}
