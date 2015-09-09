package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/hashicorp/terraform/helper/schema"
)

// Number of times to retry if a throttling- or test message exception occurs
const CLOUDWATCH_LOGS_LOG_GROUP_MAX_THROTTLE_RETRIES = 10

// How long to sleep when a throttle-event happens
const CLOUDWATCH_LOGS_LOG_GROUP_THROTTLE_SLEEP = 5 * time.Second

func resourceAwsCloudwatchLogsLogGroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsCloudwatchLogsLogGroupCreate,
		Read:   resourceAwsCloudwatchLogsLogGroupRead,
		Delete: resourceAwsCloudwatchLogsLogGroupDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceAwsCloudwatchLogsLogGroupCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cloudwatchlogsconn

	name := d.Get("name").(string)

	log.Printf("[DEBUG] Make sure that LogGroup exists %s", name)
	log_groups_response, log_groups_err := conn.DescribeLogGroups(&cloudwatchlogs.DescribeLogGroupsInput{
		LogGroupNamePrefix: aws.String(name),
	})
	if log_groups_err != nil {
		if awsErr, ok := log_groups_err.(awserr.Error); ok {
			return fmt.Errorf("[WARN] Error searching for LogGroup %s, message: \"%s\", code: \"%s\"",
				name, awsErr.Message(), awsErr.Code())
		}
		return log_groups_err
	}

	for _, l := range log_groups_response.LogGroups {
		if *l.LogGroupName == name {
			return nil // log group exists, do nothing
		}
	}

	log.Printf("[DEBUG] Creating LogGroup %s", name)
	params := &cloudwatchlogs.CreateLogGroupInput{
		LogGroupName: aws.String(name),
	}
	_, err := conn.CreateLogGroup(params)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			return fmt.Errorf("[WARN] Error creating LogGroup %s, message: \"%s\", code: \"%s\"",
				name, awsErr.Message(), awsErr.Code())
		}
		return err
	}

	return nil
}

func resourceAwsCloudwatchLogsLogGroupRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cloudwatchlogsconn

	name := d.Get("name").(string) // "name" is a required field in the schema

	resp, err := conn.DescribeLogGroups(&cloudwatchlogs.DescribeLogGroupsInput{
		LogGroupNamePrefix: aws.String(name),
	})

	if err != nil {
		return fmt.Errorf("Error looking up log groups with name prefix %s: %#v", name, err)
	}

	for _, LogGroup := range resp.LogGroups {
		if *LogGroup.LogGroupName == name {
			d.SetId(name)
			return nil // OK, matching loggroup
		}
	}

	return fmt.Errorf("Log group with name %s not found!", name)
}

func resourceAwsCloudwatchLogsLogGroupDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cloudwatchlogsconn

	name := d.Get("name").(string)

	_, err := conn.DeleteLogGroup(&cloudwatchlogs.DeleteLogGroupInput{
		LogGroupName: &name,
	})

	if err != nil {
		return fmt.Errorf(
			"Error deleting log group: %s", name)
	}
	d.SetId("")
	return nil
}
