package test

import (
	"fmt"
	"math/rand"

	"github.com/hashicorp/terraform/helper/schema"
)

func testResourceComputedMap() *schema.Resource {
	return &schema.Resource{
		Create: testResourceComputedMapCreate,
		Read:   testResourceComputedMapRead,
		Delete: testResourceComputedMapDelete,
		Update: testResourceComputedMapUpdate,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"nested_resources": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"limits": {
							Type:     schema.TypeMap,
							Computed: true,
						},
					},
				},
			},

			"computed_map": {
				Type:     schema.TypeMap,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"optional_map": {
				Type:     schema.TypeMap,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func testResourceComputedMapCreate(d *schema.ResourceData, meta interface{}) error {
	d.SetId(fmt.Sprintf("%x", rand.Int63()))
	return testResourceComputedMapRead(d, meta)
}

func testResourceComputedMapRead(d *schema.ResourceData, meta interface{}) error {

	computedMap := map[string]interface{}{
		"computed_key": "value",
	}
	d.Set("computed_map", computedMap)

	resources := []map[string]interface{}{
		{
			"limits": map[string]interface{}{
				"cpu": "100",
			},
		},
	}
	d.Set("nested_resources", resources)

	return nil
}

func testResourceComputedMapUpdate(d *schema.ResourceData, meta interface{}) error {
	return testResourceComputedMapRead(d, meta)
}

func testResourceComputedMapDelete(d *schema.ResourceData, meta interface{}) error {
	d.SetId("")
	return nil
}
