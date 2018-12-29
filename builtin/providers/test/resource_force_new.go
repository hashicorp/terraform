package test

import (
	"github.com/hashicorp/terraform/helper/schema"
)

func testResourceForceNew() *schema.Resource {
	return &schema.Resource{
		Create: testResourceForceNewCreate,
		Read:   testResourceForceNewRead,
		Delete: testResourceForceNewDelete,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"triggers": {
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func testResourceForceNewCreate(d *schema.ResourceData, meta interface{}) error {
	d.SetId("testId")
	return testResourceForceNewRead(d, meta)
}

func testResourceForceNewRead(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func testResourceForceNewDelete(d *schema.ResourceData, meta interface{}) error {
	d.SetId("")
	return nil
}
