package heroku

import (
	"fmt"
	"log"

	"github.com/bgentry/heroku-go"
	"github.com/hashicorp/terraform/flatmap"
	"github.com/hashicorp/terraform/helper/config"
	"github.com/hashicorp/terraform/helper/diff"
	"github.com/hashicorp/terraform/helper/multierror"
	"github.com/hashicorp/terraform/terraform"
)

// type application is used to store all the details of a heroku app
type application struct {
	Id string // Id of the resource

	App    *heroku.App       // The heroku application
	Client *heroku.Client    // Client to interact with the heroku API
	Vars   map[string]string // The vars on the application
}

// Updates the application to have the latest from remote
func (a *application) Update() error {
	var errs []error
	var err error

	a.App, err = a.Client.AppInfo(a.Id)
	if err != nil {
		errs = append(errs, err)
	}

	a.Vars, err = retrieve_config_vars(a.Id, a.Client)
	if err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return &multierror.Error{Errors: errs}
	}

	return nil
}

func resource_heroku_app_create(
	s *terraform.ResourceState,
	d *terraform.ResourceDiff,
	meta interface{}) (*terraform.ResourceState, error) {
	p := meta.(*ResourceProvider)
	client := p.client

	// Merge the diff into the state so that we have all the attributes
	// properly.
	rs := s.MergeDiff(d)

	// Build up our creation options
	opts := heroku.AppCreateOpts{}

	if attr := rs.Attributes["name"]; attr != "" {
		opts.Name = &attr
	}

	if attr := rs.Attributes["region"]; attr != "" {
		opts.Region = &attr
	}

	if attr := rs.Attributes["stack"]; attr != "" {
		opts.Stack = &attr
	}

	log.Printf("[DEBUG] App create configuration: %#v", opts)

	a, err := client.AppCreate(&opts)
	if err != nil {
		return s, err
	}

	rs.ID = a.Name
	log.Printf("[INFO] App ID: %s", rs.ID)

	if attr, ok := rs.Attributes["config_vars.#"]; ok && attr == "1" {
		vs := flatmap.Expand(
			rs.Attributes, "config_vars").([]interface{})

		err = update_config_vars(rs.ID, vs, client)
		if err != nil {
			return rs, err
		}
	}

	app, err := resource_heroku_app_retrieve(rs.ID, client)
	if err != nil {
		return rs, err
	}

	return resource_heroku_app_update_state(rs, app)
}

func resource_heroku_app_update(
	s *terraform.ResourceState,
	d *terraform.ResourceDiff,
	meta interface{}) (*terraform.ResourceState, error) {
	p := meta.(*ResourceProvider)
	client := p.client
	rs := s.MergeDiff(d)

	if attr, ok := d.Attributes["name"]; ok {
		opts := heroku.AppUpdateOpts{
			Name: &attr.New,
		}

		renamedApp, err := client.AppUpdate(rs.ID, &opts)

		if err != nil {
			return s, err
		}

		// Store the new ID
		rs.ID = renamedApp.Name
	}

	attr, ok := s.Attributes["config_vars.#"]

	// If the config var block was removed, nuke all config vars
	if ok && attr == "1" {
		vs := flatmap.Expand(
			rs.Attributes, "config_vars").([]interface{})

		err := update_config_vars(rs.ID, vs, client)
		if err != nil {
			return rs, err
		}
	} else if ok && attr == "0" {
		log.Println("[INFO] Config vars removed, removing all vars")

		err := update_config_vars(rs.ID, make([]interface{}, 0), client)

		if err != nil {
			return rs, err
		}
	}

	app, err := resource_heroku_app_retrieve(rs.ID, client)
	if err != nil {
		return rs, err
	}

	return resource_heroku_app_update_state(rs, app)
}

func resource_heroku_app_destroy(
	s *terraform.ResourceState,
	meta interface{}) error {
	p := meta.(*ResourceProvider)
	client := p.client

	log.Printf("[INFO] Deleting App: %s", s.ID)

	// Destroy the app
	err := client.AppDelete(s.ID)

	if err != nil {
		return fmt.Errorf("Error deleting App: %s", err)
	}

	return nil
}

func resource_heroku_app_refresh(
	s *terraform.ResourceState,
	meta interface{}) (*terraform.ResourceState, error) {
	p := meta.(*ResourceProvider)
	client := p.client

	app, err := resource_heroku_app_retrieve(s.ID, client)
	if err != nil {
		return nil, err
	}

	return resource_heroku_app_update_state(s, app)
}

func resource_heroku_app_diff(
	s *terraform.ResourceState,
	c *terraform.ResourceConfig,
	meta interface{}) (*terraform.ResourceDiff, error) {

	b := &diff.ResourceBuilder{
		Attrs: map[string]diff.AttrType{
			"name":        diff.AttrTypeUpdate,
			"region":      diff.AttrTypeUpdate,
			"stack":       diff.AttrTypeCreate,
			"config_vars": diff.AttrTypeUpdate,
		},

		ComputedAttrs: []string{
			"name",
			"region",
			"stack",
			"git_url",
			"web_url",
			"id",
			"config_vars",
		},

		ComputedAttrsUpdate: []string{
			"heroku_hostname",
		},
	}

	return b.Diff(s, c)
}

func resource_heroku_app_update_state(
	s *terraform.ResourceState,
	app *application) (*terraform.ResourceState, error) {

	s.Attributes["name"] = app.App.Name
	s.Attributes["stack"] = app.App.Stack.Name
	s.Attributes["region"] = app.App.Region.Name
	s.Attributes["git_url"] = app.App.GitURL
	s.Attributes["web_url"] = app.App.WebURL

	// We know that the hostname on heroku will be the name+herokuapp.com
	// You need this to do things like create DNS CNAME records
	s.Attributes["heroku_hostname"] = fmt.Sprintf("%s.herokuapp.com", app.App.Name)

	toFlatten := make(map[string]interface{})

	if len(app.Vars) > 0 {
		toFlatten["config_vars"] = []map[string]string{app.Vars}
	}

	for k, v := range flatmap.Flatten(toFlatten) {
		s.Attributes[k] = v
	}

	return s, nil
}

func resource_heroku_app_retrieve(id string, client *heroku.Client) (*application, error) {
	app := application{Id: id, Client: client}

	err := app.Update()

	if err != nil {
		return nil, fmt.Errorf("Error retrieving app: %s", err)
	}

	return &app, nil
}

func resource_heroku_app_validation() *config.Validator {
	return &config.Validator{
		Required: []string{},
		Optional: []string{
			"name",
			"region",
			"stack",
			"config_vars.*",
		},
	}
}

func retrieve_config_vars(id string, client *heroku.Client) (map[string]string, error) {
	vars, err := client.ConfigVarInfo(id)

	if err != nil {
		return nil, err
	}

	return vars, nil
}

// Updates the config vars for from an expanded (prior to assertion)
// []map[string]string config
func update_config_vars(id string, vs []interface{}, client *heroku.Client) error {
	vars := make(map[string]*string)

	for k, v := range vs[0].(map[string]interface{}) {
		val := v.(string)
		vars[k] = &val
	}

	log.Printf("[INFO] Updating config vars: *%#v", vars)

	_, err := client.ConfigVarUpdate(id, vars)

	if err != nil {
		return fmt.Errorf("Error updating config vars: %s", err)
	}

	return nil
}
