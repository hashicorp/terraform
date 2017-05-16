package test

import (
	"github.com/hashicorp/terraform/helper/schema"
)

func providerLabelDataSource() *schema.Resource {
	return &schema.Resource{
		Read: providerLabelDataSourceRead,

		Schema: map[string]*schema.Schema{
			"label": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func providerLabelDataSourceRead(d *schema.ResourceData, meta interface{}) error {
	label := meta.(string)
	d.SetId(label)
	d.Set("label", label)
	return nil
}
