package chef

import (
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"

	chefc "github.com/go-chef/chef"
)

func resourceChefRole() *schema.Resource {
	return &schema.Resource{
		Create: CreateRole,
		Update: UpdateRole,
		Read:   ReadRole,
		Delete: DeleteRole,

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
			"run_list": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type:      schema.TypeString,
					StateFunc: runListEntryStateFunc,
				},
			},
		},
	}
}

func CreateRole(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*chefc.Client)

	role, err := roleFromResourceData(d)
	if err != nil {
		return err
	}

	_, err = client.Roles.Create(role)
	if err != nil {
		return err
	}

	d.SetId(role.Name)
	return ReadRole(d, meta)
}

func UpdateRole(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*chefc.Client)

	role, err := roleFromResourceData(d)
	if err != nil {
		return err
	}

	_, err = client.Roles.Put(role)
	if err != nil {
		return err
	}

	d.SetId(role.Name)
	return ReadRole(d, meta)
}

func ReadRole(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*chefc.Client)

	name := d.Id()

	role, err := client.Roles.Get(name)
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

	d.Set("name", role.Name)
	d.Set("description", role.Description)

	defaultAttrJson, err := json.Marshal(role.DefaultAttributes)
	if err != nil {
		return err
	}
	d.Set("default_attributes_json", defaultAttrJson)

	overrideAttrJson, err := json.Marshal(role.OverrideAttributes)
	if err != nil {
		return err
	}
	d.Set("override_attributes_json", overrideAttrJson)

	runListI := make([]interface{}, len(role.RunList))
	for i, v := range role.RunList {
		runListI[i] = v
	}
	d.Set("run_list", runListI)

	return nil
}

func DeleteRole(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*chefc.Client)

	name := d.Id()

	// For some reason Roles.Delete is not exposed by the
	// underlying client library, so we have to do this manually.

	path := fmt.Sprintf("roles/%s", name)

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

func roleFromResourceData(d *schema.ResourceData) (*chefc.Role, error) {

	role := &chefc.Role{
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
		ChefType:    "role",
	}

	var err error

	err = json.Unmarshal(
		[]byte(d.Get("default_attributes_json").(string)),
		&role.DefaultAttributes,
	)
	if err != nil {
		return nil, fmt.Errorf("default_attributes_json: %s", err)
	}

	err = json.Unmarshal(
		[]byte(d.Get("override_attributes_json").(string)),
		&role.OverrideAttributes,
	)
	if err != nil {
		return nil, fmt.Errorf("override_attributes_json: %s", err)
	}

	runListI := d.Get("run_list").([]interface{})
	role.RunList = make([]string, len(runListI))
	for i, vI := range runListI {
		role.RunList[i] = vI.(string)
	}

	return role, nil
}
