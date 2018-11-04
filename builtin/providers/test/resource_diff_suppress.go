package test

import (
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
)

func testResourceDiffSuppress() *schema.Resource {
	return &schema.Resource{
		Create: testResourceDiffSuppressCreate,
		Read:   testResourceDiffSuppressRead,
		Update: testResourceDiffSuppressUpdate,
		Delete: testResourceDiffSuppressDelete,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"val_to_upper": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				StateFunc: func(val interface{}) string {
					return strings.ToUpper(val.(string))
				},
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					return strings.ToUpper(old) == strings.ToUpper(new)
				},
			},
			"optional": {
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func testResourceDiffSuppressCreate(d *schema.ResourceData, meta interface{}) error {
	d.SetId("testId")

	return testResourceRead(d, meta)
}

func testResourceDiffSuppressRead(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func testResourceDiffSuppressUpdate(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func testResourceDiffSuppressDelete(d *schema.ResourceData, meta interface{}) error {
	d.SetId("")
	return nil
}
