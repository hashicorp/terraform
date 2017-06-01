package aws

import (
	"bytes"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsCloudwatchLogSubscriptionFilter() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsCloudwatchLogSubscriptionFilterCreate,
		Read:   resourceAwsCloudwatchLogSubscriptionFilterRead,
		Update: resourceAwsCloudwatchLogSubscriptionFilterUpdate,
		Delete: resourceAwsCloudwatchLogSubscriptionFilterDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"destination_arn": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"filter_pattern": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: false,
			},
			"log_group_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"role_arn": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
		},
	}
}

func resourceAwsCloudwatchLogSubscriptionFilterCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cloudwatchlogsconn
	params := getAwsCloudWatchLogsSubscriptionFilterInput(d)
	log.Printf("[DEBUG] Creating SubscriptionFilter %#v", params)

	return resource.Retry(5*time.Minute, func() *resource.RetryError {
		_, err := conn.PutSubscriptionFilter(&params)

		if err == nil {
			d.SetId(cloudwatchLogsSubscriptionFilterId(d.Get("log_group_name").(string)))
			log.Printf("[DEBUG] Cloudwatch logs subscription %q created", d.Id())
		}

		awsErr, ok := err.(awserr.Error)
		if !ok {
			return resource.RetryableError(err)
		}

		if awsErr.Code() == "InvalidParameterException" {
			log.Printf("[DEBUG] Caught message: %q, code: %q: Retrying", awsErr.Message(), awsErr.Code())
			if strings.Contains(awsErr.Message(), "Could not deliver test message to specified") {
				return resource.RetryableError(err)
			}
			if strings.Contains(awsErr.Message(), "Could not execute the lambda function") {
				return resource.RetryableError(err)
			}
			resource.NonRetryableError(err)
		}

		return resource.NonRetryableError(err)
	})
}

func resourceAwsCloudwatchLogSubscriptionFilterUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cloudwatchlogsconn

	params := getAwsCloudWatchLogsSubscriptionFilterInput(d)

	log.Printf("[DEBUG] Update SubscriptionFilter %#v", params)
	_, err := conn.PutSubscriptionFilter(&params)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			return fmt.Errorf("[WARN] Error updating SubscriptionFilter (%s) for LogGroup (%s), message: \"%s\", code: \"%s\"",
				d.Get("name").(string), d.Get("log_group_name").(string), awsErr.Message(), awsErr.Code())
		}
		return err
	}

	d.SetId(cloudwatchLogsSubscriptionFilterId(d.Get("log_group_name").(string)))
	return resourceAwsCloudwatchLogSubscriptionFilterRead(d, meta)
}

func getAwsCloudWatchLogsSubscriptionFilterInput(d *schema.ResourceData) cloudwatchlogs.PutSubscriptionFilterInput {
	name := d.Get("name").(string)
	destination_arn := d.Get("destination_arn").(string)
	filter_pattern := d.Get("filter_pattern").(string)
	log_group_name := d.Get("log_group_name").(string)

	params := cloudwatchlogs.PutSubscriptionFilterInput{
		FilterName:     aws.String(name),
		DestinationArn: aws.String(destination_arn),
		FilterPattern:  aws.String(filter_pattern),
		LogGroupName:   aws.String(log_group_name),
	}

	if _, ok := d.GetOk("role_arn"); ok {
		params.RoleArn = aws.String(d.Get("role_arn").(string))
	}

	return params
}

func resourceAwsCloudwatchLogSubscriptionFilterRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cloudwatchlogsconn

	log_group_name := d.Get("log_group_name").(string)
	name := d.Get("name").(string) // "name" is a required field in the schema

	req := &cloudwatchlogs.DescribeSubscriptionFiltersInput{
		LogGroupName:     aws.String(log_group_name),
		FilterNamePrefix: aws.String(name),
	}

	resp, err := conn.DescribeSubscriptionFilters(req)
	if err != nil {
		return fmt.Errorf("Error reading SubscriptionFilters for log group %s with name prefix %s: %#v", log_group_name, d.Get("name").(string), err)
	}

	for _, subscriptionFilter := range resp.SubscriptionFilters {
		if *subscriptionFilter.LogGroupName == log_group_name {
			d.SetId(cloudwatchLogsSubscriptionFilterId(log_group_name))
			return nil // OK, matching subscription filter found
		}
	}

	log.Printf("[DEBUG] Subscription Filter%q Not Found", name)
	d.SetId("")
	return nil
}

func resourceAwsCloudwatchLogSubscriptionFilterDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cloudwatchlogsconn
	log.Printf("[INFO] Deleting CloudWatch Log Group Subscription: %s", d.Id())
	log_group_name := d.Get("log_group_name").(string)
	name := d.Get("name").(string)

	params := &cloudwatchlogs.DeleteSubscriptionFilterInput{
		FilterName:   aws.String(name),           // Required
		LogGroupName: aws.String(log_group_name), // Required
	}
	_, err := conn.DeleteSubscriptionFilter(params)
	if err != nil {
		return fmt.Errorf(
			"Error deleting Subscription Filter from log group: %s with name filter name %s", log_group_name, name)
	}
	d.SetId("")
	return nil
}

func cloudwatchLogsSubscriptionFilterId(log_group_name string) string {
	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf("%s-", log_group_name)) // only one filter allowed per log_group_name at the moment

	return fmt.Sprintf("cwlsf-%d", hashcode.String(buf.String()))
}
