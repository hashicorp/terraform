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

			"never_set": {
				Type:     schema.TypeList,
				MaxItems: 1,
				Optional: true,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"sublist": {
							Type:     schema.TypeList,
							MaxItems: 1,
							ForceNew: true,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"bool": {
										Type:     schema.TypeBool,
										ForceNew: true,
										Required: true,
									},
									"string": {
										Type:     schema.TypeString,
										Computed: true,
									},
								},
							},
						},
					},
				},
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

	// "computing" these values should insert empty containers into the
	// never_set block.
	values := make(map[string]interface{})
	values["sublist"] = []interface{}{}
	d.Set("never_set", []interface{}{values})

	return nil
}

func testResourceListUpdate(d *schema.ResourceData, meta interface{}) error {
	block := d.Get("never_set").([]interface{})
	if len(block) > 0 {
		// if profiles contains any values, they should not be nil
		_ = block[0].(map[string]interface{})
	}
	return testResourceListRead(d, meta)
}

func testResourceListDelete(d *schema.ResourceData, meta interface{}) error {
	d.SetId("")
	return nil
}
