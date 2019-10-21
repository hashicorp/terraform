package test

import (
	"github.com/hashicorp/terraform/helper/schema"
)

func testResourceImportRemoved() *schema.Resource {
	return &schema.Resource{
		Create: testResourceImportRemovedCreate,
		Read:   testResourceImportRemovedRead,
		Delete: testResourceImportRemovedDelete,
		Update: testResourceImportRemovedUpdate,

		Importer: &schema.ResourceImporter{
			State: testResourceImportRemovedImportState,
		},

		Schema: map[string]*schema.Schema{
			"removed": {
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
				Removed:  "do not use",
			},
		},
	}
}

func testResourceImportRemovedImportState(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	var results []*schema.ResourceData

	results = append(results, d)

	{
		other := testResourceDefaults()
		od := other.Data(nil)
		od.SetType("test_resource_import_removed")
		od.SetId("foo")
		results = append(results, od)
	}

	return results, nil
}

func testResourceImportRemovedCreate(d *schema.ResourceData, meta interface{}) error {
	d.SetId("foo")
	return testResourceImportRemovedRead(d, meta)
}

func testResourceImportRemovedUpdate(d *schema.ResourceData, meta interface{}) error {
	return testResourceImportRemovedRead(d, meta)
}

func testResourceImportRemovedRead(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func testResourceImportRemovedDelete(d *schema.ResourceData, meta interface{}) error {
	return nil
}
