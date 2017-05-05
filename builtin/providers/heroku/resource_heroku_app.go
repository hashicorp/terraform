package heroku

import (
	"context"
	"fmt"
	"log"

	"github.com/cyberdelia/heroku-go/v3"
	multierror "github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform/helper/schema"
)

// herokuApplication is a value type used to hold the details of an
// application. We use this for common storage of values needed for the
// heroku.App and heroku.OrganizationApp types
type herokuApplication struct {
	Name             string
	Region           string
	Space            string
	Stack            string
	GitURL           string
	WebURL           string
	OrganizationName string
	Locked           bool
}

// type application is used to store all the details of a heroku app
type application struct {
	Id string // Id of the resource

	App          *herokuApplication // The heroku application
	Client       *heroku.Service    // Client to interact with the heroku API
	Vars         map[string]string  // The vars on the application
	Buildpacks   []string           // The application's buildpack names or URLs
	Organization bool               // is the application organization app
}

// Updates the application to have the latest from remote
func (a *application) Update() error {
	var errs []error
	var err error

	if !a.Organization {
		app, err := a.Client.AppInfo(context.TODO(), a.Id)
		if err != nil {
			errs = append(errs, err)
		} else {
			a.App = &herokuApplication{}
			a.App.Name = app.Name
			a.App.Region = app.Region.Name
			a.App.Stack = app.Stack.Name
			a.App.GitURL = app.GitURL
			a.App.WebURL = app.WebURL
		}
	} else {
		app, err := a.Client.OrganizationAppInfo(context.TODO(), a.Id)
		if err != nil {
			errs = append(errs, err)
		} else {
			// No inheritance between OrganizationApp and App is killing it :/
			a.App = &herokuApplication{}
			a.App.Name = app.Name
			a.App.Region = app.Region.Name
			a.App.Stack = app.Stack.Name
			a.App.GitURL = app.GitURL
			a.App.WebURL = app.WebURL
			if app.Space != nil {
				a.App.Space = app.Space.Name
			}
			if app.Organization != nil {
				a.App.OrganizationName = app.Organization.Name
			} else {
				log.Println("[DEBUG] Something is wrong - didn't get information about organization name, while the app is marked as being so")
			}
			a.App.Locked = app.Locked
		}
	}

	a.Buildpacks, err = retrieveBuildpacks(a.Id, a.Client)
	if err != nil {
		errs = append(errs, err)
	}

	a.Vars, err = retrieveConfigVars(a.Id, a.Client)
	if err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return &multierror.Error{Errors: errs}
	}

	return nil
}

func resourceHerokuApp() *schema.Resource {
	return &schema.Resource{
		Create: switchHerokuAppCreate,
		Read:   resourceHerokuAppRead,
		Update: resourceHerokuAppUpdate,
		Delete: resourceHerokuAppDelete,

		Importer: &schema.ResourceImporter{
			State: resourceHerokuAppImport,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"space": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"region": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"stack": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"buildpacks": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			"config_vars": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeMap,
				},
			},

			"all_config_vars": {
				Type:     schema.TypeMap,
				Computed: true,
			},

			"git_url": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"web_url": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"heroku_hostname": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"organization": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},

						"locked": {
							Type:     schema.TypeBool,
							Optional: true,
						},

						"personal": {
							Type:     schema.TypeBool,
							Optional: true,
						},
					},
				},
			},
		},
	}
}

func isOrganizationApp(d *schema.ResourceData) bool {
	v := d.Get("organization").([]interface{})
	return len(v) > 0 && v[0] != nil
}

func resourceHerokuAppImport(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	client := m.(*heroku.Service)

	app, err := client.AppInfo(context.TODO(), d.Id())
	if err != nil {
		return nil, err
	}

	// Flag organization apps by setting the organization name
	if app.Organization != nil {
		d.Set("organization", []map[string]interface{}{
			{"name": app.Organization.Name},
		})
	}

	return []*schema.ResourceData{d}, nil
}

func switchHerokuAppCreate(d *schema.ResourceData, meta interface{}) error {
	if isOrganizationApp(d) {
		return resourceHerokuOrgAppCreate(d, meta)
	}

	return resourceHerokuAppCreate(d, meta)
}

func resourceHerokuAppCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*heroku.Service)

	// Build up our creation options
	opts := heroku.AppCreateOpts{}

	if v, ok := d.GetOk("name"); ok {
		vs := v.(string)
		log.Printf("[DEBUG] App name: %s", vs)
		opts.Name = &vs
	}
	if v, ok := d.GetOk("region"); ok {
		vs := v.(string)
		log.Printf("[DEBUG] App region: %s", vs)
		opts.Region = &vs
	}
	if v, ok := d.GetOk("stack"); ok {
		vs := v.(string)
		log.Printf("[DEBUG] App stack: %s", vs)
		opts.Stack = &vs
	}

	log.Printf("[DEBUG] Creating Heroku app...")
	a, err := client.AppCreate(context.TODO(), opts)
	if err != nil {
		return err
	}

	d.SetId(a.Name)
	log.Printf("[INFO] App ID: %s", d.Id())

	if err := performAppPostCreateTasks(d, client); err != nil {
		return err
	}

	return resourceHerokuAppRead(d, meta)
}

func resourceHerokuOrgAppCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*heroku.Service)
	// Build up our creation options
	opts := heroku.OrganizationAppCreateOpts{}

	v := d.Get("organization").([]interface{})
	if len(v) > 1 {
		return fmt.Errorf("Error Creating Heroku App: Only 1 Heroku Organization is permitted")
	}
	orgDetails := v[0].(map[string]interface{})

	if v := orgDetails["name"]; v != nil {
		vs := v.(string)
		log.Printf("[DEBUG] Organization name: %s", vs)
		opts.Organization = &vs
	}

	if v := orgDetails["personal"]; v != nil {
		vs := v.(bool)
		log.Printf("[DEBUG] Organization Personal: %t", vs)
		opts.Personal = &vs
	}

	if v := orgDetails["locked"]; v != nil {
		vs := v.(bool)
		log.Printf("[DEBUG] Organization locked: %t", vs)
		opts.Locked = &vs
	}

	if v := d.Get("name"); v != nil {
		vs := v.(string)
		log.Printf("[DEBUG] App name: %s", vs)
		opts.Name = &vs
	}
	if v, ok := d.GetOk("region"); ok {
		vs := v.(string)
		log.Printf("[DEBUG] App region: %s", vs)
		opts.Region = &vs
	}
	if v, ok := d.GetOk("space"); ok {
		vs := v.(string)
		log.Printf("[DEBUG] App space: %s", vs)
		opts.Space = &vs
	}
	if v, ok := d.GetOk("stack"); ok {
		vs := v.(string)
		log.Printf("[DEBUG] App stack: %s", vs)
		opts.Stack = &vs
	}

	log.Printf("[DEBUG] Creating Heroku app...")
	a, err := client.OrganizationAppCreate(context.TODO(), opts)
	if err != nil {
		return err
	}

	d.SetId(a.Name)
	log.Printf("[INFO] App ID: %s", d.Id())

	if err := performAppPostCreateTasks(d, client); err != nil {
		return err
	}

	return resourceHerokuAppRead(d, meta)
}

func resourceHerokuAppRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*heroku.Service)

	configVars := make(map[string]string)
	care := make(map[string]struct{})
	for _, v := range d.Get("config_vars").([]interface{}) {
		for k := range v.(map[string]interface{}) {
			care[k] = struct{}{}
		}
	}

	// Only track buildpacks when set in the configuration.
	_, buildpacksConfigured := d.GetOk("buildpacks")

	organizationApp := isOrganizationApp(d)

	// Only set the config_vars that we have set in the configuration.
	// The "all_config_vars" field has all of them.
	app, err := resourceHerokuAppRetrieve(d.Id(), organizationApp, client)
	if err != nil {
		return err
	}

	for k, v := range app.Vars {
		if _, ok := care[k]; ok {
			configVars[k] = v
		}
	}
	var configVarsValue []map[string]string
	if len(configVars) > 0 {
		configVarsValue = []map[string]string{configVars}
	}

	d.Set("name", app.App.Name)
	d.Set("stack", app.App.Stack)
	d.Set("region", app.App.Region)
	d.Set("git_url", app.App.GitURL)
	d.Set("web_url", app.App.WebURL)
	if buildpacksConfigured {
		d.Set("buildpacks", app.Buildpacks)
	}
	d.Set("config_vars", configVarsValue)
	d.Set("all_config_vars", app.Vars)
	if organizationApp {
		d.Set("space", app.App.Space)

		orgDetails := map[string]interface{}{
			"name":     app.App.OrganizationName,
			"locked":   app.App.Locked,
			"personal": false,
		}
		err := d.Set("organization", []interface{}{orgDetails})
		if err != nil {
			return err
		}
	}

	// We know that the hostname on heroku will be the name+herokuapp.com
	// You need this to do things like create DNS CNAME records
	d.Set("heroku_hostname", fmt.Sprintf("%s.herokuapp.com", app.App.Name))

	return nil
}

func resourceHerokuAppUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*heroku.Service)

	// If name changed, update it
	if d.HasChange("name") {
		v := d.Get("name").(string)
		opts := heroku.AppUpdateOpts{
			Name: &v,
		}

		renamedApp, err := client.AppUpdate(context.TODO(), d.Id(), opts)
		if err != nil {
			return err
		}

		// Store the new ID
		d.SetId(renamedApp.Name)
	}

	// If the config vars changed, then recalculate those
	if d.HasChange("config_vars") {
		o, n := d.GetChange("config_vars")
		if o == nil {
			o = []interface{}{}
		}
		if n == nil {
			n = []interface{}{}
		}

		err := updateConfigVars(
			d.Id(), client, o.([]interface{}), n.([]interface{}))
		if err != nil {
			return err
		}
	}

	if d.HasChange("buildpacks") {
		err := updateBuildpacks(d.Id(), client, d.Get("buildpacks").([]interface{}))
		if err != nil {
			return err
		}
	}

	return resourceHerokuAppRead(d, meta)
}

func resourceHerokuAppDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*heroku.Service)

	log.Printf("[INFO] Deleting App: %s", d.Id())
	_, err := client.AppDelete(context.TODO(), d.Id())
	if err != nil {
		return fmt.Errorf("Error deleting App: %s", err)
	}

	d.SetId("")
	return nil
}

func resourceHerokuAppRetrieve(id string, organization bool, client *heroku.Service) (*application, error) {
	app := application{Id: id, Client: client, Organization: organization}

	err := app.Update()

	if err != nil {
		return nil, fmt.Errorf("Error retrieving app: %s", err)
	}

	return &app, nil
}

func retrieveBuildpacks(id string, client *heroku.Service) ([]string, error) {
	results, err := client.BuildpackInstallationList(context.TODO(), id, nil)

	if err != nil {
		return nil, err
	}

	buildpacks := []string{}
	for _, installation := range results {
		buildpacks = append(buildpacks, installation.Buildpack.Name)
	}

	return buildpacks, nil
}

func retrieveConfigVars(id string, client *heroku.Service) (map[string]string, error) {
	vars, err := client.ConfigVarInfoForApp(context.TODO(), id)

	if err != nil {
		return nil, err
	}

	nonNullVars := map[string]string{}
	for k, v := range vars {
		if v != nil {
			nonNullVars[k] = *v
		}
	}

	return nonNullVars, nil
}

// Updates the config vars for from an expanded configuration.
func updateConfigVars(
	id string,
	client *heroku.Service,
	o []interface{},
	n []interface{}) error {
	vars := make(map[string]*string)

	for _, v := range o {
		if v != nil {
			for k := range v.(map[string]interface{}) {
				vars[k] = nil
			}
		}
	}
	for _, v := range n {
		if v != nil {
			for k, v := range v.(map[string]interface{}) {
				val := v.(string)
				vars[k] = &val
			}
		}
	}

	log.Printf("[INFO] Updating config vars: *%#v", vars)
	if _, err := client.ConfigVarUpdate(context.TODO(), id, vars); err != nil {
		return fmt.Errorf("Error updating config vars: %s", err)
	}

	return nil
}

func updateBuildpacks(id string, client *heroku.Service, v []interface{}) error {
	opts := heroku.BuildpackInstallationUpdateOpts{
		Updates: []struct {
			Buildpack string `json:"buildpack" url:"buildpack,key"`
		}{}}

	for _, buildpack := range v {
		opts.Updates = append(opts.Updates, struct {
			Buildpack string `json:"buildpack" url:"buildpack,key"`
		}{
			Buildpack: buildpack.(string),
		})
	}

	if _, err := client.BuildpackInstallationUpdate(context.TODO(), id, opts); err != nil {
		return fmt.Errorf("Error updating buildpacks: %s", err)
	}

	return nil
}

// performAppPostCreateTasks performs post-create tasks common to both org and non-org apps.
func performAppPostCreateTasks(d *schema.ResourceData, client *heroku.Service) error {
	if v, ok := d.GetOk("config_vars"); ok {
		if err := updateConfigVars(d.Id(), client, nil, v.([]interface{})); err != nil {
			return err
		}
	}

	if v, ok := d.GetOk("buildpacks"); ok {
		if err := updateBuildpacks(d.Id(), client, v.([]interface{})); err != nil {
			return err
		}
	}

	return nil
}
