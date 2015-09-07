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
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

// Number of times to retry if a throttling- or test message exception occurs
const CLOUDWATCH_LOGS_SUBSCRIPTION_FILTER_MAX_THROTTLE_RETRIES = 10

// How long to sleep when a throttle-event happens
const CLOUDWATCH_LOGS_SUBSCRIPTION_FILTER_THROTTLE_SLEEP = 5 * time.Second

func resourceAwsCloudwatchLogsSubscriptionFilter() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsCloudwatchLogsSubscriptionFilterCreate,
		Read:   resourceAwsCloudwatchLogsSubscriptionFilterRead,
		Update: resourceAwsCloudwatchLogsSubscriptionFilterUpdate,
		Delete: resourceAwsCloudwatchLogsSubscriptionFilterDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"destination": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"filter_pattern": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"log_group": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"role": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
		},
	}
}

func resourceAwsCloudwatchLogsSubscriptionFilterCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cloudwatchlogsconn
	kinesis_conn := meta.(*AWSClient).kinesisconn

	name := d.Get("name").(string)

	log_group := d.Get("log_group").(string)
	destination_arn_sliced := strings.Split(d.Get("destination").(string), "/")
	destination_name := destination_arn_sliced[len(destination_arn_sliced)-1]

	createLogGroupIfNeeded(conn, log_group)
	waitForKinesisStreamToActivate(kinesis_conn, destination_name)

	params := getAwsCloudWatchLogsSubscriptionFilterInput(d)

	log.Printf("[DEBUG] Creating SubscriptionFilter %#v", params)

	attemptCount := 1
	for attemptCount <= CLOUDWATCH_LOGS_SUBSCRIPTION_FILTER_MAX_THROTTLE_RETRIES {
		_, err := conn.PutSubscriptionFilter(&params)
		if err != nil {
			if awsErr, ok := err.(awserr.Error); ok {
				if awsErr.Code() == "InvalidParameterException" {
					log.Printf("[DEBUG] Attempt %d/%d: Sleeping for a bit to throttle back put request", attemptCount, CLOUDWATCH_LOGS_SUBSCRIPTION_FILTER_MAX_THROTTLE_RETRIES)
					time.Sleep(CLOUDWATCH_LOGS_SUBSCRIPTION_FILTER_THROTTLE_SLEEP)
					attemptCount += 1
				} else {
					// Some other non-retryable exception occurred
					return fmt.Errorf("[WARN] Error creating SubscriptionFilter (%s) for LogGroup (%s) to destination (%s), message: \"%s\", code: \"%s\"",
						name, log_group, destination_name, awsErr.Message(), awsErr.Code())
				}
			} else {
				// Non-AWS exception occurred, give up
				return fmt.Errorf("Error creating Cloudwatch logs subscription filter: %s", name, err)
			}
		} else {
			d.SetId(cloudwatchLogsSubscriptionFilterId(d.Get("log_group").(string)))
			return resourceAwsCloudwatchLogsSubscriptionFilterRead(d, meta)
		}
	}

	// Too many throttling events occurred, give up
	return fmt.Errorf("Unable to create Cloudwatch logs subscription filter '%s' after %d attempts", name, attemptCount)
}

func resourceAwsCloudwatchLogsSubscriptionFilterUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cloudwatchlogsconn

	params := getAwsCloudWatchLogsSubscriptionFilterInput(d)

	log.Printf("[DEBUG] Update SubscriptionFilter %#v", params)
	_, err := conn.PutSubscriptionFilter(&params)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			return fmt.Errorf("[WARN] Error updating SubscriptionFilter (%s) for LogGroup (%s), message: \"%s\", code: \"%s\"",
				d.Get("name").(string), d.Get("log_group").(string), awsErr.Message(), awsErr.Code())
		}
		return err
	}

	d.SetId(cloudwatchLogsSubscriptionFilterId(d.Get("log_group").(string)))
	return resourceAwsCloudwatchLogsSubscriptionFilterRead(d, meta)
}

func getAwsCloudWatchLogsSubscriptionFilterInput(d *schema.ResourceData) cloudwatchlogs.PutSubscriptionFilterInput {
	name := d.Get("name").(string)
	destination := d.Get("destination").(string)
	filter_pattern := d.Get("filter_pattern").(string)
	log_group := d.Get("log_group").(string)

	params := cloudwatchlogs.PutSubscriptionFilterInput{
		FilterName:     aws.String(name),
		DestinationArn: aws.String(destination),
		FilterPattern:  aws.String(filter_pattern),
		LogGroupName:   aws.String(log_group),
	}

	if _, ok := d.GetOk("role"); ok {
		params.RoleArn = aws.String(d.Get("role").(string))
	}

	return params
}

func resourceAwsCloudwatchLogsSubscriptionFilterRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cloudwatchlogsconn

	log_group := d.Get("log_group").(string)
	name := d.Get("name").(string) // "name" is a required field in the schema

	req := &cloudwatchlogs.DescribeSubscriptionFiltersInput{
		LogGroupName:     aws.String(log_group),
		FilterNamePrefix: aws.String(name),
	}

	resp, err := conn.DescribeSubscriptionFilters(req)
	if err != nil {
		return fmt.Errorf("Error reading SubscriptionFilters for log group %s with name prefix %s: %#v", log_group, d.Get("name").(string), err)
	}

	for _, subscriptionFilter := range resp.SubscriptionFilters {
		if *subscriptionFilter.LogGroupName == log_group {
			d.SetId(cloudwatchLogsSubscriptionFilterId(log_group))
			return nil // OK, matching subscription filter found
		}
	}

	return fmt.Errorf("Subscription filter for log group %s with name prefix %s not found!", log_group, d.Get("name").(string))
}

func resourceAwsCloudwatchLogsSubscriptionFilterDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cloudwatchlogsconn

	log_group := d.Get("log_group").(string)
	name := d.Get("name").(string)

	params := &cloudwatchlogs.DeleteSubscriptionFilterInput{
		FilterName:   aws.String(name),      // Required
		LogGroupName: aws.String(log_group), // Required
	}
	_, err := conn.DeleteSubscriptionFilter(params)

	if err != nil {
		return fmt.Errorf(
			"Error deleting Subscription Filter from log group: %s with name filter name %s", log_group, name)
	}
	d.SetId("")
	return nil
}

func waitForKinesisStreamToActivate(conn *kinesis.Kinesis, stream_name string) error {
	// If destination is Kinesis stream, then it must be ACTIVE before creating SubscriptionFilter
	wait := resource.StateChangeConf{
		Pending:    []string{"CREATING", "UPDATING", "DELETING"},
		Target:     "ACTIVE",
		Timeout:    5 * time.Minute,
		MinTimeout: 1 * time.Second,
		Refresh: func() (interface{}, string, error) {
			log.Printf("[DEBUG] Checking if Kinesis stream %s is ACTIVE", stream_name)
			resp, err := conn.DescribeStream(&kinesis.DescribeStreamInput{
				StreamName: aws.String(stream_name),
			})
			if err != nil {
				return resp, "FAILED", err
			}
			stream_status := *resp.StreamDescription.StreamStatus
			log.Printf("[DEBUG] Kinesis stream %s is %s checking for ACTIVE", stream_name, stream_status)
			return resp, stream_status, nil
		},
	}

	_, err := wait.WaitForState()
	if err != nil {
		return err
	}

	return nil
}

func createLogGroupIfNeeded(conn *cloudwatchlogs.CloudWatchLogs, log_group_name string) error {

	log.Printf("[DEBUG] Make sure that LogGroup exists %s", log_group_name)
	log_group_search_params := &cloudwatchlogs.DescribeLogGroupsInput{
		LogGroupNamePrefix: aws.String(log_group_name),
	}
	log_groups_response, log_groups_err := conn.DescribeLogGroups(log_group_search_params)
	if log_groups_err != nil {
		if awsErr, ok := log_groups_err.(awserr.Error); ok {
			return fmt.Errorf("[WARN] Error searching for LogGroup %s, message: \"%s\", code: \"%s\"",
				log_group_name, awsErr.Message(), awsErr.Code())
		}
		return log_groups_err
	}

	var log_group_exists bool = false
	for _, l := range log_groups_response.LogGroups {
		if *l.LogGroupName == log_group_name {
			log_group_exists = true
			break
		}
	}

	if log_group_exists == false {
		log.Printf("[DEBUG] Creating LogGroup %s", log_group_name)
		params := &cloudwatchlogs.CreateLogGroupInput{
			LogGroupName: aws.String(log_group_name),
		}
		_, err := conn.CreateLogGroup(params)
		if err != nil {
			if awsErr, ok := err.(awserr.Error); ok {
				return fmt.Errorf("[WARN] Error creating LogGroup %s, message: \"%s\", code: \"%s\"",
					log_group_name, awsErr.Message(), awsErr.Code())
			}
			return err
		}
	}

	return nil
}

func cloudwatchLogsSubscriptionFilterId(log_group string) string {
	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf("%s-", log_group)) // only one filter allowed per log_group at the moment

	return fmt.Sprintf("cwlsf-%d", hashcode.String(buf.String()))
}
