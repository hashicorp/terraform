package test

import (
	"fmt"
	"math/rand"

	"github.com/hashicorp/terraform/helper/schema"
)

func testResourceNestedSet() *schema.Resource {
	return &schema.Resource{
		Create: testResourceNestedSetCreate,
		Read:   testResourceNestedSetRead,
		Delete: testResourceNestedSetDelete,
		Update: testResourceNestedSetUpdate,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"optional": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"force_new": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"type_list": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"value": {
							Type:     schema.TypeString,
							ForceNew: true,
							Optional: true,
						},
					},
				},
			},
			"single": {
				Type:     schema.TypeSet,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"value": {
							Type:     schema.TypeString,
							ForceNew: true,
							Required: true,
						},

						"optional": {
							Type:     schema.TypeString,
							ForceNew: true,
							Optional: true,
							Computed: true,
						},
					},
				},
			},
			"multi": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"set": {
							Type:     schema.TypeSet,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"required": {
										Type:     schema.TypeString,
										Required: true,
										ForceNew: true,
									},
									"optional_int": {
										Type:     schema.TypeInt,
										Optional: true,
									},
									"bool": {
										Type:     schema.TypeBool,
										Optional: true,
									},
								},
							},
						},

						"optional": {
							Type: schema.TypeString,
							// commenting this causes it to get missed during apply
							//ForceNew: true,
							Optional: true,
						},
						"bool": {
							Type:     schema.TypeBool,
							Optional: true,
						},
					},
				},
			},
			"with_list": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"required": {
							Type:     schema.TypeString,
							Required: true,
						},
						"list": {
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"list_block": {
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"unused": {
										Type:     schema.TypeString,
										Optional: true,
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

func testResourceNestedSetCreate(d *schema.ResourceData, meta interface{}) error {
	id := fmt.Sprintf("%x", rand.Int63())
	d.SetId(id)

	// replicate some awkward handling of a computed value in a set
	set := d.Get("single").(*schema.Set)
	l := set.List()
	if len(l) == 1 {
		if s, ok := l[0].(map[string]interface{}); ok {
			if v, _ := s["optional"].(string); v == "" {
				s["optional"] = id
			}
		}
	}

	d.Set("single", set)

	return testResourceNestedSetRead(d, meta)
}

func testResourceNestedSetRead(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func testResourceNestedSetDelete(d *schema.ResourceData, meta interface{}) error {
	d.SetId("")
	return nil
}

func testResourceNestedSetUpdate(d *schema.ResourceData, meta interface{}) error {
	return nil
}
