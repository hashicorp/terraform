package test

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
)

func testResourceAsSingle() *schema.Resource {
	return &schema.Resource{
		Create: testResourceAsSingleCreate,
		Read:   testResourceAsSingleRead,
		Delete: testResourceAsSingleDelete,
		Update: testResourceAsSingleUpdate,

		Schema: map[string]*schema.Schema{
			"list_resource_as_block": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				AsSingle: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"foo": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
			"list_resource_as_attr": {
				Type:       schema.TypeList,
				ConfigMode: schema.SchemaConfigModeAttr,
				Optional:   true,
				MaxItems:   1,
				AsSingle:   true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"foo": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
			"list_primitive": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				AsSingle: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			"set_resource_as_block": {
				Type:     schema.TypeSet,
				Optional: true,
				MaxItems: 1,
				AsSingle: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"foo": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
			"set_resource_as_attr": {
				Type:       schema.TypeSet,
				ConfigMode: schema.SchemaConfigModeAttr,
				Optional:   true,
				MaxItems:   1,
				AsSingle:   true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"foo": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
			"set_primitive": {
				Type:     schema.TypeSet,
				Optional: true,
				MaxItems: 1,
				AsSingle: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func testResourceAsSingleCreate(d *schema.ResourceData, meta interface{}) error {
	d.SetId("placeholder")
	return testResourceAsSingleRead(d, meta)
}

func testResourceAsSingleRead(d *schema.ResourceData, meta interface{}) error {
	for _, k := range []string{"list_resource_as_block", "list_resource_as_attr", "list_primitive"} {
		v := d.Get(k)
		if v == nil {
			continue
		}
		if l, ok := v.([]interface{}); !ok {
			return fmt.Errorf("%s should appear as []interface {}, not %T", k, v)
		} else {
			for i, item := range l {
				switch k {
				case "list_primitive":
					if _, ok := item.(string); item != nil && !ok {
						return fmt.Errorf("%s[%d] should appear as string, not %T", k, i, item)
					}
				default:
					if _, ok := item.(map[string]interface{}); item != nil && !ok {
						return fmt.Errorf("%s[%d] should appear as map[string]interface {}, not %T", k, i, item)
					}
				}
			}
		}
	}
	for _, k := range []string{"set_resource_as_block", "set_resource_as_attr", "set_primitive"} {
		v := d.Get(k)
		if v == nil {
			continue
		}
		if s, ok := v.(*schema.Set); !ok {
			return fmt.Errorf("%s should appear as *schema.Set, not %T", k, v)
		} else {
			for i, item := range s.List() {
				switch k {
				case "set_primitive":
					if _, ok := item.(string); item != nil && !ok {
						return fmt.Errorf("%s[%d] should appear as string, not %T", k, i, item)
					}
				default:
					if _, ok := item.(map[string]interface{}); item != nil && !ok {
						return fmt.Errorf("%s[%d] should appear as map[string]interface {}, not %T", k, i, item)
					}
				}
			}
		}
	}
	return nil
}

func testResourceAsSingleUpdate(d *schema.ResourceData, meta interface{}) error {
	return testResourceAsSingleRead(d, meta)
}

func testResourceAsSingleDelete(d *schema.ResourceData, meta interface{}) error {
	d.SetId("")
	return nil
}
