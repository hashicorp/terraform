package heroku

import (
	"fmt"
	"log"
	"sync"

	"github.com/bgentry/heroku-go"
	"github.com/hashicorp/terraform/flatmap"
	"github.com/hashicorp/terraform/helper/config"
	"github.com/hashicorp/terraform/helper/diff"
	"github.com/hashicorp/terraform/terraform"
)

// Global lock to prevent parallelism for heroku_addon since
// the Heroku API cannot handle a single application requesting
// multiple addons simultaneously.
var addonLock sync.Mutex

func resource_heroku_addon_create(
	s *terraform.ResourceState,
	d *terraform.ResourceDiff,
	meta interface{}) (*terraform.ResourceState, error) {
	addonLock.Lock()
	defer addonLock.Unlock()

	p := meta.(*ResourceProvider)
	client := p.client

	// Merge the diff into the state so that we have all the attributes
	// properly.
	rs := s.MergeDiff(d)

	app := rs.Attributes["app"]
	plan := rs.Attributes["plan"]
	opts := heroku.AddonCreateOpts{}

	if attr, ok := rs.Attributes["config.#"]; ok && attr == "1" {
		vs := flatmap.Expand(
			rs.Attributes, "config").([]interface{})

		config := make(map[string]string)
		for k, v := range vs[0].(map[string]interface{}) {
			config[k] = v.(string)
		}

		opts.Config = &config
	}

	log.Printf("[DEBUG] Addon create configuration: %#v, %#v, %#v", app, plan, opts)

	a, err := client.AddonCreate(app, plan, &opts)

	if err != nil {
		return s, err
	}

	rs.ID = a.Id
	log.Printf("[INFO] Addon ID: %s", rs.ID)

	addon, err := resource_heroku_addon_retrieve(app, rs.ID, client)
	if err != nil {
		return rs, err
	}

	return resource_heroku_addon_update_state(rs, addon)
}

func resource_heroku_addon_update(
	s *terraform.ResourceState,
	d *terraform.ResourceDiff,
	meta interface{}) (*terraform.ResourceState, error) {
	p := meta.(*ResourceProvider)
	client := p.client
	rs := s.MergeDiff(d)

	app := rs.Attributes["app"]

	if attr, ok := d.Attributes["plan"]; ok {
		ad, err := client.AddonUpdate(
			app, rs.ID,
			attr.New)

		if err != nil {
			return s, err
		}

		// Store the new ID
		rs.ID = ad.Id
	}

	addon, err := resource_heroku_addon_retrieve(app, rs.ID, client)

	if err != nil {
		return rs, err
	}

	return resource_heroku_addon_update_state(rs, addon)
}

func resource_heroku_addon_destroy(
	s *terraform.ResourceState,
	meta interface{}) error {
	p := meta.(*ResourceProvider)
	client := p.client

	log.Printf("[INFO] Deleting Addon: %s", s.ID)

	// Destroy the app
	err := client.AddonDelete(s.Attributes["app"], s.ID)

	if err != nil {
		return fmt.Errorf("Error deleting addon: %s", err)
	}

	return nil
}

func resource_heroku_addon_refresh(
	s *terraform.ResourceState,
	meta interface{}) (*terraform.ResourceState, error) {
	p := meta.(*ResourceProvider)
	client := p.client

	app, err := resource_heroku_addon_retrieve(s.Attributes["app"], s.ID, client)
	if err != nil {
		return nil, err
	}

	return resource_heroku_addon_update_state(s, app)
}

func resource_heroku_addon_diff(
	s *terraform.ResourceState,
	c *terraform.ResourceConfig,
	meta interface{}) (*terraform.ResourceDiff, error) {

	b := &diff.ResourceBuilder{
		Attrs: map[string]diff.AttrType{
			"app":    diff.AttrTypeCreate,
			"plan":   diff.AttrTypeUpdate,
			"config": diff.AttrTypeCreate,
		},

		ComputedAttrs: []string{
			"provider_id",
			"config_vars",
		},
	}

	return b.Diff(s, c)
}

func resource_heroku_addon_update_state(
	s *terraform.ResourceState,
	addon *heroku.Addon) (*terraform.ResourceState, error) {

	s.Attributes["name"] = addon.Name
	s.Attributes["plan"] = addon.Plan.Name
	s.Attributes["provider_id"] = addon.ProviderId

	toFlatten := make(map[string]interface{})

	if len(addon.ConfigVars) > 0 {
		toFlatten["config_vars"] = addon.ConfigVars
	}

	for k, v := range flatmap.Flatten(toFlatten) {
		s.Attributes[k] = v
	}

	s.Dependencies = []terraform.ResourceDependency{
		terraform.ResourceDependency{ID: s.Attributes["app"]},
	}

	return s, nil
}

func resource_heroku_addon_retrieve(app string, id string, client *heroku.Client) (*heroku.Addon, error) {
	addon, err := client.AddonInfo(app, id)

	if err != nil {
		return nil, fmt.Errorf("Error retrieving addon: %s", err)
	}

	return addon, nil
}

func resource_heroku_addon_validation() *config.Validator {
	return &config.Validator{
		Required: []string{
			"app",
			"plan",
		},
		Optional: []string{
			"config.*",
		},
	}
}
