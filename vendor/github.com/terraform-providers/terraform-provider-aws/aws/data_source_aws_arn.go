package aws

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsArn() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsArnRead,

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateArn,
			},
			"partition": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"service": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"region": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"account": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"resource": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceAwsArnRead(d *schema.ResourceData, meta interface{}) error {
	v := d.Get("arn").(string)
	arn, err := arn.Parse(v)
	if err != nil {
		return fmt.Errorf("Error parsing '%s': %s", v, err.Error())
	}

	d.SetId(arn.String())
	d.Set("partition", arn.Partition)
	d.Set("service", arn.Service)
	d.Set("region", arn.Region)
	d.Set("account", arn.AccountID)
	d.Set("resource", arn.Resource)

	return nil
}
