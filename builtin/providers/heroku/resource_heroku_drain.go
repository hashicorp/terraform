package heroku

import (
	"fmt"
	"log"

	"github.com/bgentry/heroku-go"
	"github.com/hashicorp/terraform/helper/config"
	"github.com/hashicorp/terraform/helper/diff"
	"github.com/hashicorp/terraform/terraform"
)

func resource_heroku_drain_create(
	s *terraform.ResourceState,
	d *terraform.ResourceDiff,
	meta interface{}) (*terraform.ResourceState, error) {
	p := meta.(*ResourceProvider)
	client := p.client

	// Merge the diff into the state so that we have all the attributes
	// properly.
	rs := s.MergeDiff(d)

	app := rs.Attributes["app"]
	url := rs.Attributes["url"]

	log.Printf("[DEBUG] Drain create configuration: %#v, %#v", app, url)

	dr, err := client.LogDrainCreate(app, url)

	if err != nil {
		return s, err
	}

	rs.ID = dr.Id
	rs.Attributes["url"] = dr.URL
	rs.Attributes["token"] = dr.Token

	log.Printf("[INFO] Drain ID: %s", rs.ID)

	return rs, nil
}

func resource_heroku_drain_update(
	s *terraform.ResourceState,
	d *terraform.ResourceDiff,
	meta interface{}) (*terraform.ResourceState, error) {

	panic("Cannot update drain")

	return nil, nil
}

func resource_heroku_drain_destroy(
	s *terraform.ResourceState,
	meta interface{}) error {
	p := meta.(*ResourceProvider)
	client := p.client

	log.Printf("[INFO] Deleting drain: %s", s.ID)

	// Destroy the app
	err := client.LogDrainDelete(s.Attributes["app"], s.ID)

	if err != nil {
		return fmt.Errorf("Error deleting drain: %s", err)
	}

	return nil
}

func resource_heroku_drain_refresh(
	s *terraform.ResourceState,
	meta interface{}) (*terraform.ResourceState, error) {
	p := meta.(*ResourceProvider)
	client := p.client

	drain, err := resource_heroku_drain_retrieve(s.Attributes["app"], s.ID, client)
	if err != nil {
		return nil, err
	}

	s.Attributes["url"] = drain.URL
	s.Attributes["token"] = drain.Token

	return s, nil
}

func resource_heroku_drain_diff(
	s *terraform.ResourceState,
	c *terraform.ResourceConfig,
	meta interface{}) (*terraform.ResourceDiff, error) {

	b := &diff.ResourceBuilder{
		Attrs: map[string]diff.AttrType{
			"url": diff.AttrTypeCreate,
			"app": diff.AttrTypeCreate,
		},

		ComputedAttrs: []string{
			"token",
		},
	}

	return b.Diff(s, c)
}

func resource_heroku_drain_retrieve(app string, id string, client *heroku.Client) (*heroku.LogDrain, error) {
	drain, err := client.LogDrainInfo(app, id)

	if err != nil {
		return nil, fmt.Errorf("Error retrieving drain: %s", err)
	}

	return drain, nil
}

func resource_heroku_drain_validation() *config.Validator {
	return &config.Validator{
		Required: []string{
			"url",
			"app",
		},
		Optional: []string{},
	}
}
