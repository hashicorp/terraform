package test

import (
	"github.com/hashicorp/terraform/helper/schema"
)

// This is a test resource to help reproduce GH-12183. This issue came up
// as a complex mixing of core + helper/schema and while we added core tests
// to cover some of the cases, this test helps top it off with an end-to-end
// test.
func testResourceGH12183() *schema.Resource {
	return &schema.Resource{
		Create: testResourceCreate_gh12183,
		Read:   testResourceRead_gh12183,
		Update: testResourceUpdate_gh12183,
		Delete: testResourceDelete_gh12183,
		Schema: map[string]*schema.Schema{
			"key": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"config": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				MinItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},

						"rules": {
							Type:     schema.TypeSet,
							Computed: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set:      schema.HashString,
						},
					},
				},
			},
		},
	}
}

func testResourceCreate_gh12183(d *schema.ResourceData, meta interface{}) error {
	d.SetId("testId")
	return testResourceRead_gh12183(d, meta)
}

func testResourceRead_gh12183(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func testResourceUpdate_gh12183(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func testResourceDelete_gh12183(d *schema.ResourceData, meta interface{}) error {
	d.SetId("")
	return nil
}
