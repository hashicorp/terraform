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
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"foo": {
							Type:     schema.TypeString,
							Optional: true,
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
	const k = "resource_as_attr"
	if l, ok := d.Get(k).([]interface{}); !ok {
		return fmt.Errorf("%s should appear as []interface{}, not %T", k, l)
	} else {
		for i, item := range l {
			if _, ok := item.(map[string]interface{}); !ok {
				return fmt.Errorf("%s[%d] should appear as map[string]interface{}, not %T", k, i, item)
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
