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
		},
	}
}

func testResourceListCreate(d *schema.ResourceData, meta interface{}) error {
	d.SetId("testId")
	return nil
}

func testResourceListRead(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func testResourceListUpdate(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func testResourceListDelete(d *schema.ResourceData, meta interface{}) error {
	d.SetId("")
	return nil
}
