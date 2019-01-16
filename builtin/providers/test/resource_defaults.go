package test

import (
	"fmt"
	"math/rand"

	"github.com/hashicorp/terraform/helper/schema"
)

func testResourceDefaults() *schema.Resource {
	return &schema.Resource{
		Create: testResourceDefaultsCreate,
		Read:   testResourceDefaultsRead,
		Delete: testResourceDefaultsDelete,
		Update: testResourceDefaultsUpdate,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
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
			"nested": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"string": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "default nested",
						},
						"optional": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
		},
	}
}

func testResourceDefaultsCreate(d *schema.ResourceData, meta interface{}) error {
	d.SetId(fmt.Sprintf("%x", rand.Int63()))
	return testResourceDefaultsRead(d, meta)
}

func testResourceDefaultsUpdate(d *schema.ResourceData, meta interface{}) error {
	return testResourceDefaultsRead(d, meta)
}

func testResourceDefaultsRead(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func testResourceDefaultsDelete(d *schema.ResourceData, meta interface{}) error {
	d.SetId("")
	return nil
}
