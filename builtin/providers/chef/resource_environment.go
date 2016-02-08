package chef

import (
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"

	chefc "github.com/go-chef/chef"
)

func resourceChefEnvironment() *schema.Resource {
	return &schema.Resource{
		Create: CreateEnvironment,
		Update: UpdateEnvironment,
		Read:   ReadEnvironment,
		Delete: DeleteEnvironment,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "Managed by Terraform",
			},
			"default_attributes_json": &schema.Schema{
				Type:      schema.TypeString,
				Optional:  true,
				Default:   "{}",
				StateFunc: jsonStateFunc,
			},
			"override_attributes_json": &schema.Schema{
				Type:      schema.TypeString,
				Optional:  true,
				Default:   "{}",
				StateFunc: jsonStateFunc,
			},
			"cookbook_constraints": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func CreateEnvironment(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*chefc.Client)

	env, err := environmentFromResourceData(d)
	if err != nil {
		return err
	}

	_, err = client.Environments.Create(env)
	if err != nil {
		return err
	}

	d.SetId(env.Name)
	return ReadEnvironment(d, meta)
}

func UpdateEnvironment(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*chefc.Client)

	env, err := environmentFromResourceData(d)
	if err != nil {
		return err
	}

	_, err = client.Environments.Put(env)
	if err != nil {
		return err
	}

	d.SetId(env.Name)
	return ReadEnvironment(d, meta)
}

func ReadEnvironment(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*chefc.Client)

	name := d.Id()

	env, err := client.Environments.Get(name)
	if err != nil {
		if errRes, ok := err.(*chefc.ErrorResponse); ok {
			if errRes.Response.StatusCode == 404 {
				d.SetId("")
				return nil
			}
		} else {
			return err
		}
	}

	d.Set("name", env.Name)
	d.Set("description", env.Description)

	defaultAttrJson, err := json.Marshal(env.DefaultAttributes)
	if err != nil {
		return err
	}
	d.Set("default_attributes_json", defaultAttrJson)

	overrideAttrJson, err := json.Marshal(env.OverrideAttributes)
	if err != nil {
		return err
	}
	d.Set("override_attributes_json", overrideAttrJson)

	cookbookVersionsI := map[string]interface{}{}
	for k, v := range env.CookbookVersions {
		cookbookVersionsI[k] = v
	}
	d.Set("cookbook_constraints", cookbookVersionsI)

	return nil
}

func DeleteEnvironment(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*chefc.Client)

	name := d.Id()

	// For some reason Environments.Delete is not exposed by the
	// underlying client library, so we have to do this manually.

	path := fmt.Sprintf("environments/%s", name)

	httpReq, err := client.NewRequest("DELETE", path, nil)
	if err != nil {
		return err
	}

	_, err = client.Do(httpReq, nil)
	if err == nil {
		d.SetId("")
	}

	return err
}

func environmentFromResourceData(d *schema.ResourceData) (*chefc.Environment, error) {

	env := &chefc.Environment{
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
		ChefType:    "environment",
	}

	var err error

	err = json.Unmarshal(
		[]byte(d.Get("default_attributes_json").(string)),
		&env.DefaultAttributes,
	)
	if err != nil {
		return nil, fmt.Errorf("default_attributes_json: %s", err)
	}

	err = json.Unmarshal(
		[]byte(d.Get("override_attributes_json").(string)),
		&env.OverrideAttributes,
	)
	if err != nil {
		return nil, fmt.Errorf("override_attributes_json: %s", err)
	}

	env.CookbookVersions = make(map[string]string)
	for k, vI := range d.Get("cookbook_constraints").(map[string]interface{}) {
		env.CookbookVersions[k] = vI.(string)
	}

	return env, nil
}
