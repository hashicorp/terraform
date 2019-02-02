package test

import (
	"fmt"
	"math/rand"

	"github.com/hashicorp/terraform/helper/schema"
)

func testResourceListSet() *schema.Resource {
	return &schema.Resource{
		Create: testResourceListSetCreate,
		Read:   testResourceListSetRead,
		Delete: testResourceListSetDelete,
		Update: testResourceListSetUpdate,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"list": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"set": {
							Type:     schema.TypeSet,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"elem": {
										Type:     schema.TypeString,
										Optional: true,
										DiffSuppressFunc: func(_, o, n string, _ *schema.ResourceData) bool {
											return o == n
										},
									},
								},
							},
							Set: func(v interface{}) int {
								raw := v.(map[string]interface{})
								if el, ok := raw["elem"]; ok {
									return schema.HashString(el)
								}
								return 42
							},
						},
					},
				},
			},
		},
	}
}

func testResourceListSetCreate(d *schema.ResourceData, meta interface{}) error {
	d.SetId(fmt.Sprintf("%x", rand.Int63()))
	return testResourceListSetRead(d, meta)
}

func testResourceListSetUpdate(d *schema.ResourceData, meta interface{}) error {
	return testResourceListSetRead(d, meta)
}

func testResourceListSetRead(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func testResourceListSetDelete(d *schema.ResourceData, meta interface{}) error {
	d.SetId("")
	return nil
}
