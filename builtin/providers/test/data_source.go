package test

import (
	"time"

	"github.com/hashicorp/terraform/helper/schema"
)

func testDataSource() *schema.Resource {
	return &schema.Resource{
		Read: testDataSourceRead,

		Schema: map[string]*schema.Schema{
			"list": &schema.Schema{
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func testDataSourceRead(d *schema.ResourceData, meta interface{}) error {
	d.SetId(time.Now().UTC().String())
	d.Set("list", []interface{}{"one", "two", "three"})

	return nil
}
