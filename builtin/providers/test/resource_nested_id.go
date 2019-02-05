package test

import (
	"github.com/hashicorp/terraform/helper/schema"
)

func testResourceNestedId() *schema.Resource {
	return &schema.Resource{
		Create: testResourceNestedIdCreate,
		Read:   testResourceNestedIdRead,
		Update: testResourceNestedIdUpdate,
		Delete: testResourceNestedIdDelete,

		Schema: map[string]*schema.Schema{
			"list_block": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func testResourceNestedIdCreate(d *schema.ResourceData, meta interface{}) error {
	d.SetId("testId")
	return nil
}

func testResourceNestedIdRead(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func testResourceNestedIdUpdate(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func testResourceNestedIdDelete(d *schema.ResourceData, meta interface{}) error {
	d.SetId("")
	return nil
}
