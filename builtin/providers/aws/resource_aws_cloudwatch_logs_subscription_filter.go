package aws

import (
	"bytes"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
)

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

	params := getAwsCloudWatchLogsSubscriptionFilterInput(d)

	log.Printf("[DEBUG] Creating SubscriptionFilter %#v", params)
	_, err := conn.PutSubscriptionFilter(&params)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			return fmt.Errorf("[WARN] Error creating SubscriptionFilter (%s) for LogGroup (%s), message: \"%s\", code: \"%s\"",
				d.Get("name").(string), d.Get("log_group").(string), awsErr.Message(), awsErr.Code())
		}
		return err
	}

	d.SetId(CloudwatchLogsSubscriptionFilterId(d))
	return resourceAwsCloudwatchLogsSubscriptionFilterRead(d, meta)
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

	d.SetId(CloudwatchLogsSubscriptionFilterId(d))
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

	req := &cloudwatchlogs.DescribeSubscriptionFiltersInput{
		LogGroupName: aws.String(d.Get("log_group").(string)), // Required
	}

	if _, ok := d.GetOk("name"); ok {
		req.FilterNamePrefix = aws.String(d.Get("name").(string))
	}

	resp, err := conn.DescribeSubscriptionFilters(req)
	if err != nil {
		return fmt.Errorf("Error reading SubscriptionFilters for log group %s with name prefix %s: %#v", log_group, d.Get("name").(string), err)
	}

	for _, subscriptionFilter := range resp.SubscriptionFilters {
		if *subscriptionFilter.LogGroupName == log_group {
			if name, ok := d.GetOk("name"); ok {
				if *subscriptionFilter.FilterName == name.(string) {
					return nil // OK, matching subscription filter found
				}
			} else {
				return nil // OK, matching subscription filter found - name not given
			}
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

func CloudwatchLogsSubscriptionFilterId(d *schema.ResourceData) string {
	var buf bytes.Buffer

	name := d.Get("name").(string)
	destination := d.Get("destination").(string)
	filter_pattern := d.Get("filter_pattern").(string)
	log_group := d.Get("log_group").(string)

	buf.WriteString(fmt.Sprintf("%s-", name))
	buf.WriteString(fmt.Sprintf("%s-", destination))
	buf.WriteString(fmt.Sprintf("%s-", log_group))
	buf.WriteString(fmt.Sprintf("%s-", filter_pattern))

	return fmt.Sprintf("cwlsf-%d", hashcode.String(buf.String()))
}
