package test

import (
	"github.com/hashicorp/terraform/helper/schema"
)

func testResourceRequiredMin() *schema.Resource {
	return &schema.Resource{
		Create: testResourceRequiredMinCreate,
		Read:   testResourceRequiredMinRead,
		Update: testResourceRequiredMinUpdate,
		Delete: testResourceRequiredMinDelete,

		CustomizeDiff: func(d *schema.ResourceDiff, _ interface{}) error {
			if d.HasChange("dependent_list") {
				d.SetNewComputed("computed_list")
			}
			return nil
		},

		Schema: map[string]*schema.Schema{
			"min_items": {
				Type:     schema.TypeList,
				Optional: true,
				MinItems: 2,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"val": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"required_min_items": {
				Type:     schema.TypeList,
				Required: true,
				MinItems: 2,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"val": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
		},
	}
}

func testResourceRequiredMinCreate(d *schema.ResourceData, meta interface{}) error {
	d.SetId("testId")
	return testResourceRequiredMinRead(d, meta)
}

func testResourceRequiredMinRead(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func testResourceRequiredMinUpdate(d *schema.ResourceData, meta interface{}) error {
	return testResourceRequiredMinRead(d, meta)
}

func testResourceRequiredMinDelete(d *schema.ResourceData, meta interface{}) error {
	d.SetId("")
	return nil
}
