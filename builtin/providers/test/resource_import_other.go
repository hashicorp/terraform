package test

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
)

func testResourceImportOther() *schema.Resource {
	return &schema.Resource{
		Create: testResourceImportOtherCreate,
		Read:   testResourceImportOtherRead,
		Delete: testResourceImportOtherDelete,
		Update: testResourceImportOtherUpdate,

		Importer: &schema.ResourceImporter{
			State: testResourceImportOtherImportState,
		},

		Schema: map[string]*schema.Schema{
			"default_string": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "default string",
			},
			"default_bool": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  true,
			},
			"computed": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func testResourceImportOtherImportState(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	var results []*schema.ResourceData

	results = append(results, d)

	{
		other := testResourceDefaults()
		od := other.Data(nil)
		od.SetType("test_resource_defaults")
		od.SetId("import_other_other")
		results = append(results, od)
	}

	return results, nil
}

func testResourceImportOtherCreate(d *schema.ResourceData, meta interface{}) error {
	d.SetId("import_other_main")
	return testResourceImportOtherRead(d, meta)
}

func testResourceImportOtherUpdate(d *schema.ResourceData, meta interface{}) error {
	return testResourceImportOtherRead(d, meta)
}

func testResourceImportOtherRead(d *schema.ResourceData, meta interface{}) error {
	err := d.Set("computed", "hello!")
	if err != nil {
		return fmt.Errorf("failed to set 'computed' attribute: %s", err)
	}
	return nil
}

func testResourceImportOtherDelete(d *schema.ResourceData, meta interface{}) error {
	return nil
}
