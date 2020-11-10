package test

import (
	"github.com/hashicorp/terraform/helper/schema"
)

func testResourceSignal() *schema.Resource {
	return &schema.Resource{
		Create: testResourceSignalCreate,
		Read:   testResourceSignalRead,
		Update: testResourceSignalUpdate,
		Delete: testResourceSignalDelete,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"optional": {
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func testResourceSignalCreate(d *schema.ResourceData, meta interface{}) error {
	d.SetId("testId")

	return testResourceSignalRead(d, meta)
}

func testResourceSignalRead(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func testResourceSignalUpdate(d *schema.ResourceData, meta interface{}) error {
	return testResourceSignalRead(d, meta)
}

func testResourceSignalDelete(d *schema.ResourceData, meta interface{}) error {
	return nil
}
