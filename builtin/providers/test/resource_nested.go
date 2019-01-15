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
			"list_block": {
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"sub_list_block": {
							Type:     schema.TypeList,
							Optional: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"bool": {
										Type:     schema.TypeBool,
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
				},
			},
		},
	}
}

func testResourceNestedCreate(d *schema.ResourceData, meta interface{}) error {
	d.SetId(fmt.Sprintf("%x", rand.Int63()))
	return testResourceNestedRead(d, meta)
}

func testResourceNestedUpdate(d *schema.ResourceData, meta interface{}) error {
	return testResourceNestedRead(d, meta)
}

func testResourceNestedRead(d *schema.ResourceData, meta interface{}) error {
	set := []map[string]interface{}{map[string]interface{}{
		"sub_list_block": []map[string]interface{}{map[string]interface{}{
			"bool": false,
			"set":  schema.NewSet(schema.HashString, nil),
		}},
	}}
	d.Set("list_block", set)
	return nil
}

func testResourceNestedDelete(d *schema.ResourceData, meta interface{}) error {
	d.SetId("")
	return nil
}
