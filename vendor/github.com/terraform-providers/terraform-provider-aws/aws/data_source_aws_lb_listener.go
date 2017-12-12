package aws

import "github.com/hashicorp/terraform/helper/schema"

func dataSourceAwsLbListener() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsLbListenerRead,

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Required: true,
			},

			"load_balancer_arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"port": {
				Type:     schema.TypeInt,
				Computed: true,
			},

			"protocol": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"ssl_policy": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"certificate_arn": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"default_action": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"target_group_arn": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"type": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func dataSourceAwsLbListenerRead(d *schema.ResourceData, meta interface{}) error {
	d.SetId(d.Get("arn").(string))
	return resourceAwsLbListenerRead(d, meta)
}
