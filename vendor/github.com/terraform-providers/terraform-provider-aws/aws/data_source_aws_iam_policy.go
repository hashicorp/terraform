package aws

import (
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsIAMPolicy() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsIAMPolicyRead,

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Required: true,
			},
			"name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"policy": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"path": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"description": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceAwsIAMPolicyRead(d *schema.ResourceData, meta interface{}) error {
	d.SetId(d.Get("arn").(string))
	return resourceAwsIamPolicyRead(d, meta)
}
