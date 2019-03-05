package test

import (
	"github.com/hashicorp/terraform/helper/schema"
)

func testResourceList() *schema.Resource {
	return &schema.Resource{
		Create: testResourceListCreate,
		Read:   testResourceListRead,
		Update: testResourceListUpdate,
		Delete: testResourceListDelete,

		Schema: map[string]*schema.Schema{
			"list_block": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"string": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"int": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"force_new": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						"sublist": {
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"sublist_block": {
							Type:     schema.TypeList,
							Optional: true,
							Computed: true,
							ForceNew: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"string": {
										Type:     schema.TypeString,
										Required: true,
									},
									"int": {
										Type:     schema.TypeInt,
										Required: true,
									},
								},
							},
						},
					},
				},
			},
			"dependent_list": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"val": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"computed_list": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func testResourceListCreate(d *schema.ResourceData, meta interface{}) error {
	d.SetId("testId")
	return testResourceListRead(d, meta)
}

func testResourceListRead(d *schema.ResourceData, meta interface{}) error {
	fixedIps := d.Get("dependent_list")

	// all_fixed_ips should be set as computed with a CustomizeDiff func, but
	// we're trying to emulate legacy provider behavior, and updating a
	// computed field was a common case.
	ips := []interface{}{}
	if fixedIps != nil {
		for _, v := range fixedIps.([]interface{}) {
			m := v.(map[string]interface{})
			ips = append(ips, m["val"])
		}
	}
	if err := d.Set("computed_list", ips); err != nil {
		return err
	}

	return nil
}

func testResourceListUpdate(d *schema.ResourceData, meta interface{}) error {
	return testResourceListRead(d, meta)
}

func testResourceListDelete(d *schema.ResourceData, meta interface{}) error {
	d.SetId("")
	return nil
}
