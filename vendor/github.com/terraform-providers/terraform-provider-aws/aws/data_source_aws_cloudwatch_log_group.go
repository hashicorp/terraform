package aws

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
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

	input := &cloudwatchlogs.DescribeLogGroupsInput{
		LogGroupNamePrefix: aws.String(name),
	}

	var logGroup *cloudwatchlogs.LogGroup
	// iterate over the pages of log groups until we find the one we are looking for
	err := conn.DescribeLogGroupsPages(input,
		func(resp *cloudwatchlogs.DescribeLogGroupsOutput, _ bool) bool {
			for _, lg := range resp.LogGroups {
				if aws.StringValue(lg.LogGroupName) == name {
					logGroup = lg
					return false
				}
			}
			return true
		})

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
