package aws

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsCloudwatchLogGroup() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsCloudwatchLogGroupRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"creation_time": {
				Type:     schema.TypeInt,
				Computed: true,
			},
		},
	}
}

func dataSourceAwsCloudwatchLogGroupRead(d *schema.ResourceData, meta interface{}) error {
	name := d.Get("name").(string)
	conn := meta.(*AWSClient).cloudwatchlogsconn

	logGroup, err := lookupCloudWatchLogGroup(conn, name)
	if err != nil {
		return err
	}
	if logGroup == nil {
		return fmt.Errorf("No log group named %s found\n", name)
	}

	d.SetId(name)
	d.Set("arn", logGroup.Arn)
	d.Set("creation_time", logGroup.CreationTime)

	return nil
}
