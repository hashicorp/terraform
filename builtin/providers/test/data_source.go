package test

import (
	"time"

	"github.com/hashicorp/terraform/helper/schema"
)

func testDataSource() *schema.Resource {
	return &schema.Resource{
		Read: testDataSourceRead,

		Schema: map[string]*schema.Schema{
			"list": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"input": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"output": {
				Type:     schema.TypeString,
				Computed: true,
			},
			// this attribute is computed, but never set by the provider
			"nil": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"input_map": {
				Type:     schema.TypeMap,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
			},
			"output_map": {
				Type:     schema.TypeMap,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Computed: true,
			},
		},
	}
}

func testDataSourceRead(d *schema.ResourceData, meta interface{}) error {
	d.SetId(time.Now().UTC().String())
	d.Set("list", []interface{}{"one", "two", "three"})

	if input, hasInput := d.GetOk("input"); hasInput {
		d.Set("output", input)
	} else {
		d.Set("output", "some output")
	}

	if inputMap, hasInput := d.GetOk("input_map"); hasInput {
		d.Set("output_map", inputMap)
	}
	return nil
}
