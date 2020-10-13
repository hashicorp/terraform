package test

import (
	"fmt"

	"github.com/hashicorp/terraform/configs/hcl2shim"
	"github.com/hashicorp/terraform/helper/schema"
)

func testResourceMap() *schema.Resource {
	return &schema.Resource{
		Create: testResourceMapCreate,
		Read:   testResourceMapRead,
		Update: testResourceMapUpdate,
		Delete: testResourceMapDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"map_of_three": {
				Type:     schema.TypeMap,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				ValidateFunc: func(v interface{}, _ string) ([]string, []error) {
					errs := []error{}
					for k, v := range v.(map[string]interface{}) {
						if v == hcl2shim.UnknownVariableValue {
							errs = append(errs, fmt.Errorf("unknown value in ValidateFunc: %q=%q", k, v))
						}
					}
					return nil, errs
				},
			},
			"map_values": {
				Type:     schema.TypeMap,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"computed_map": {
				Type:     schema.TypeMap,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func testResourceMapCreate(d *schema.ResourceData, meta interface{}) error {
	// make sure all elements are passed to the map
	m := d.Get("map_of_three").(map[string]interface{})
	if len(m) != 3 {
		return fmt.Errorf("expected 3 map values, got %#v\n", m)
	}

	d.SetId("testId")
	return testResourceMapRead(d, meta)
}

func testResourceMapRead(d *schema.ResourceData, meta interface{}) error {
	var computedMap map[string]interface{}
	if v, ok := d.GetOk("map_values"); ok {
		computedMap = v.(map[string]interface{})
	}
	d.Set("computed_map", computedMap)
	return nil
}

func testResourceMapUpdate(d *schema.ResourceData, meta interface{}) error {
	return testResourceMapRead(d, meta)
}

func testResourceMapDelete(d *schema.ResourceData, meta interface{}) error {
	d.SetId("")
	return nil
}
