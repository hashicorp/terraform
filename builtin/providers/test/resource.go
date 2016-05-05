package test

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
)

func testResource() *schema.Resource {
	return &schema.Resource{
		Create: testResourceCreate,
		Read:   testResourceRead,
		Update: testResourceUpdate,
		Delete: testResourceDelete,
		Schema: map[string]*schema.Schema{
			"required": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"optional": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"optional_bool": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"optional_force_new": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"optional_computed_map": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				Computed: true,
			},
			"computed_read_only": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
				ForceNew: true,
			},
			"computed_read_only_force_new": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
				ForceNew: true,
			},
		},
	}
}

func testResourceCreate(d *schema.ResourceData, meta interface{}) error {
	d.SetId("testId")

	// Required must make it through to Create
	if _, ok := d.GetOk("required"); !ok {
		return fmt.Errorf("Missing attribute 'required', but it's required!")
	}
	return testResourceRead(d, meta)
}

func testResourceRead(d *schema.ResourceData, meta interface{}) error {
	d.Set("computed_read_only", "value_from_api")
	d.Set("computed_read_only_force_new", "value_from_api")
	if _, ok := d.GetOk("optional_computed_map"); !ok {
		d.Set("optional_computed_map", map[string]string{})
	}
	return nil
}

func testResourceUpdate(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func testResourceDelete(d *schema.ResourceData, meta interface{}) error {
	return nil
}
