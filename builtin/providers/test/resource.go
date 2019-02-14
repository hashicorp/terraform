package test

import (
	"errors"
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
)

func testResource() *schema.Resource {
	return &schema.Resource{
		Create: testResourceCreate,
		Read:   testResourceRead,
		Update: testResourceUpdate,
		Delete: testResourceDelete,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"required": {
				Type:     schema.TypeString,
				Required: true,
			},
			"optional": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"optional_bool": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"optional_force_new": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"optional_computed_map": {
				Type:     schema.TypeMap,
				Optional: true,
				Computed: true,
			},
			"optional_computed_force_new": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"computed_read_only": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"computed_from_required": {
				Type:     schema.TypeString,
				Computed: true,
				ForceNew: true,
			},
			"computed_read_only_force_new": {
				Type:     schema.TypeString,
				Computed: true,
				ForceNew: true,
			},
			"computed_list": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"set": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Set: schema.HashString,
			},
			"computed_set": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Set: schema.HashString,
			},
			"map": {
				Type:     schema.TypeMap,
				Optional: true,
			},
			"optional_map": {
				Type:     schema.TypeMap,
				Optional: true,
			},
			"required_map": {
				Type:     schema.TypeMap,
				Required: true,
			},
			"map_that_look_like_set": {
				Type:     schema.TypeMap,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"computed_map": {
				Type:     schema.TypeMap,
				Computed: true,
			},
			"list": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"list_of_map": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeMap,
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
				},
			},
			"apply_error": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "return and error during apply",
			},
		},
	}
}

func testResourceCreate(d *schema.ResourceData, meta interface{}) error {
	d.SetId("testId")

	errMsg, _ := d.Get("apply_error").(string)
	if errMsg != "" {
		return errors.New(errMsg)
	}

	// Required must make it through to Create
	if _, ok := d.GetOk("required"); !ok {
		return fmt.Errorf("Missing attribute 'required', but it's required!")
	}
	if _, ok := d.GetOk("required_map"); !ok {
		return fmt.Errorf("Missing attribute 'required_map', but it's required!")
	}

	d.Set("computed_from_required", d.Get("required"))

	return testResourceRead(d, meta)
}

func testResourceRead(d *schema.ResourceData, meta interface{}) error {
	d.Set("computed_read_only", "value_from_api")
	d.Set("computed_read_only_force_new", "value_from_api")
	if _, ok := d.GetOk("optional_computed_map"); !ok {
		d.Set("optional_computed_map", map[string]string{})
	}
	d.Set("computed_map", map[string]string{"key1": "value1"})
	d.Set("computed_list", []string{"listval1", "listval2"})
	d.Set("computed_set", []string{"setval1", "setval2"})
	return nil
}

func testResourceUpdate(d *schema.ResourceData, meta interface{}) error {
	errMsg, _ := d.Get("apply_error").(string)
	if errMsg != "" {
		return errors.New(errMsg)
	}
	return testResourceRead(d, meta)
}

func testResourceDelete(d *schema.ResourceData, meta interface{}) error {
	d.SetId("")
	return nil
}
