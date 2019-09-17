package test

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
)

func testResourceConfigMode() *schema.Resource {
	return &schema.Resource{
		Create: testResourceConfigModeCreate,
		Read:   testResourceConfigModeRead,
		Delete: testResourceConfigModeDelete,
		Update: testResourceConfigModeUpdate,

		Schema: map[string]*schema.Schema{
			"resource_as_attr": {
				Type:       schema.TypeList,
				ConfigMode: schema.SchemaConfigModeAttr,
				Optional:   true,
				Computed:   true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"foo": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
			"nested_set": {
				Type:       schema.TypeSet,
				Optional:   true,
				ConfigMode: schema.SchemaConfigModeAttr,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"value": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"set": {
							Type:     schema.TypeSet,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},
		},
	}
}

func testResourceConfigModeCreate(d *schema.ResourceData, meta interface{}) error {
	d.SetId("placeholder")
	return testResourceConfigModeRead(d, meta)
}

func testResourceConfigModeRead(d *schema.ResourceData, meta interface{}) error {
	if l, ok := d.Get("resource_as_attr").([]interface{}); !ok {
		return fmt.Errorf("resource_as_attr should appear as []interface{}, not %T", l)
	} else {
		for i, item := range l {
			if _, ok := item.(map[string]interface{}); !ok {
				return fmt.Errorf("resource_as_attr[%d] should appear as map[string]interface{}, not %T", i, item)
			}
		}
	}
	return nil
}

func testResourceConfigModeUpdate(d *schema.ResourceData, meta interface{}) error {
	return testResourceConfigModeRead(d, meta)
}

func testResourceConfigModeDelete(d *schema.ResourceData, meta interface{}) error {
	d.SetId("")
	return nil
}
