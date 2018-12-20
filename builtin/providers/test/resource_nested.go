package test

import (
	"fmt"
	"math/rand"

	"github.com/hashicorp/terraform/helper/schema"
)

func testResourceNested() *schema.Resource {
	return &schema.Resource{
		Create: testResourceNestedCreate,
		Read:   testResourceNestedRead,
		Delete: testResourceNestedDelete,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"optional": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},
			"nested": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"string": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						"optional": {
							Type:     schema.TypeBool,
							Optional: true,
							ForceNew: true,
						},
						"nested_again": {
							Type:     schema.TypeSet,
							Optional: true,
							ForceNew: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"string": {
										Type:     schema.TypeString,
										Optional: true,
										ForceNew: true,
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

func testResourceNestedCreate(d *schema.ResourceData, meta interface{}) error {
	d.SetId(fmt.Sprintf("%x", rand.Int63()))
	return testResourceNestedRead(d, meta)
}

func testResourceNestedRead(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func testResourceNestedDelete(d *schema.ResourceData, meta interface{}) error {
	d.SetId("")
	return nil
}
