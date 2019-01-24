package test

import (
	"fmt"

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
	return nil
}

func testResourceMapRead(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func testResourceMapUpdate(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func testResourceMapDelete(d *schema.ResourceData, meta interface{}) error {
	d.SetId("")
	return nil
}
