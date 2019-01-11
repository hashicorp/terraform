package test

import (
	"github.com/hashicorp/terraform/helper/schema"
)

func testResourceDeprecated() *schema.Resource {
	return &schema.Resource{
		Create: testResourceDeprecatedCreate,
		Read:   testResourceDeprecatedRead,
		Update: testResourceDeprecatedUpdate,
		Delete: testResourceDeprecatedDelete,

		Schema: map[string]*schema.Schema{
			"map_deprecated": {
				Type:       schema.TypeMap,
				Optional:   true,
				Deprecated: "deprecated",
			},
			"map_removed": {
				Type:     schema.TypeMap,
				Optional: true,
				Removed:  "removed",
			},
			"set_block_deprecated": {
				Type:       schema.TypeSet,
				Optional:   true,
				MaxItems:   1,
				Deprecated: "deprecated",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"value": {
							Type:       schema.TypeString,
							Required:   true,
							Deprecated: "deprecated",
						},
						"optional": {
							Type:       schema.TypeString,
							ForceNew:   true,
							Optional:   true,
							Deprecated: "deprecated",
						},
					},
				},
			},
			"set_block_removed": {
				Type:     schema.TypeSet,
				Optional: true,
				MaxItems: 1,
				Removed:  "Removed",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"optional": {
							Type:     schema.TypeString,
							ForceNew: true,
							Optional: true,
							Computed: true,
							Removed:  "removed",
						},
					},
				},
			},
			"list_block_deprecated": {
				Type:       schema.TypeList,
				Optional:   true,
				Deprecated: "deprecated",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"value": {
							Type:       schema.TypeString,
							Required:   true,
							Deprecated: "deprecated",
						},
						"optional": {
							Type:       schema.TypeString,
							ForceNew:   true,
							Optional:   true,
							Deprecated: "deprecated",
						},
					},
				},
			},
			"list_block_removed": {
				Type:     schema.TypeList,
				Optional: true,
				Removed:  "removed",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"optional": {
							Type:     schema.TypeString,
							ForceNew: true,
							Optional: true,
							Removed:  "removed",
						},
					},
				},
			},
		},
	}
}

func testResourceDeprecatedCreate(d *schema.ResourceData, meta interface{}) error {
	d.SetId("testId")
	return nil
}

func testResourceDeprecatedRead(d *schema.ResourceData, meta interface{}) error {

	return nil
}

func testResourceDeprecatedUpdate(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func testResourceDeprecatedDelete(d *schema.ResourceData, meta interface{}) error {
	d.SetId("")
	return nil
}
