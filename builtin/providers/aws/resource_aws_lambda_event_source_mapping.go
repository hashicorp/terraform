package aws

import (
//	"bytes"
//	"fmt"
	"log"

//	"github.com/aws/aws-sdk-go/aws"
//	"github.com/aws/aws-sdk-go/aws/awserr"
//	"github.com/aws/aws-sdk-go/service/lambda"
//	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsLambdaEventSourceMapping() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsLambdaEventSourceMappingCreate,
		Read:   resourceAwsLambdaEventSourceMappingRead,
		Update: resourceAwsLambdaEventSourceMappingUpdate,
		Delete: resourceAwsLambdaEventSourceMappingDelete,

		Schema: map[string]*schema.Schema{
			"event_source_arn": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"function_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"starting_position": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"batch_size": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
			},
			"enabled": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},
		},
	}
}

func resourceAwsLambdaEventSourceMappingCreate(d *schema.ResourceData, meta interface{}) error {
//	conn := meta.(*AWSClient).lambdaconn

//	params := getAwsCloudWatchLogsSubscriptionFilterInput(d)
//
 	log.Printf("[DEBUG] Creating EventSourceMapping")
//	_, err := conn.PutSubscriptionFilter(&params)
//	if err != nil {
//		if awsErr, ok := err.(awserr.Error); ok {
//			return fmt.Errorf("[WARN] Error creating SubscriptionFilter (%s) for LogGroup (%s), message: \"%s\", code: \"%s\"",
//				d.Get("name").(string), d.Get("log_group").(string), awsErr.Message(), awsErr.Code())
//		}
//		return err
//	}
//
//	d.SetId(LambdaEventSourceMappingId(d))
//	return resourceAwsLambdaEventSourceMappingRead(d, meta)
	return nil
}

func resourceAwsLambdaEventSourceMappingUpdate(d *schema.ResourceData, meta interface{}) error {
//	conn := meta.(*AWSClient).cloudwatchlogsconn
//
//	params := getAwsCloudWatchLogsSubscriptionFilterInput(d)
//
 	log.Printf("[DEBUG] Updating EventSourceMapping")
//	_, err := conn.PutSubscriptionFilter(&params)
//	if err != nil {
//		if awsErr, ok := err.(awserr.Error); ok {
//			return fmt.Errorf("[WARN] Error updating SubscriptionFilter (%s) for LogGroup (%s), message: \"%s\", code: \"%s\"",
//				d.Get("name").(string), d.Get("log_group").(string), awsErr.Message(), awsErr.Code())
//		}
//		return err
//	}
//
//	d.SetId(LambdaEventSourceMappingId(d))
//	return resourceAwsLambdaEventSourceMappingRead(d, meta)
	return nil
}

//func getAwsCloudWatchLogsSubscriptionFilterInput(d *schema.ResourceData) cloudwatchlogs.PutSubscriptionFilterInput {
//	name := d.Get("name").(string)
//	destination := d.Get("destination").(string)
//	filter_pattern := d.Get("filter_pattern").(string)
//	log_group := d.Get("log_group").(string)
//
//	params := cloudwatchlogs.PutSubscriptionFilterInput{
//		FilterName:     aws.String(name),
//		DestinationArn: aws.String(destination),
//		FilterPattern:  aws.String(filter_pattern),
//		LogGroupName:   aws.String(log_group),
//	}
//
//	if _, ok := d.GetOk("role"); ok {
//		params.RoleArn = aws.String(d.Get("role").(string))
//	}
//
//	return params
//}

func resourceAwsLambdaEventSourceMappingRead(d *schema.ResourceData, meta interface{}) error {
//	conn := meta.(*AWSClient).cloudwatchlogsconn

 	log.Printf("[DEBUG] Reading EventSourceMapping")

//
//	log_group := d.Get("log_group").(string)
//
//	req := &cloudwatchlogs.DescribeSubscriptionFiltersInput{
//		LogGroupName: aws.String(d.Get("log_group").(string)), // Required
//	}
//
//	if _, ok := d.GetOk("name"); ok {
//		req.FilterNamePrefix = aws.String(d.Get("name").(string))
//	}
//
//	resp, err := conn.DescribeSubscriptionFilters(req)
//	if err != nil {
//		return fmt.Errorf("Error reading SubscriptionFilters for log group %s with name prefix %s: %#v", log_group, d.Get("name").(string), err)
//	}
//
//	for _, subscriptionFilter := range resp.SubscriptionFilters {
//		if *subscriptionFilter.LogGroupName == log_group {
//			if name, ok := d.GetOk("name"); ok {
//				if *subscriptionFilter.FilterName == name.(string) {
//					return nil // OK, matching subscription filter found
//				}
//			} else {
//				return nil // OK, matching subscription filter found - name not given
//			}
//		}
//	}
//
//	return fmt.Errorf("Subscription filter for log group %s with name prefix %s not found!", log_group, d.Get("name").(string))
	return nil
}

func resourceAwsLambdaEventSourceMappingDelete(d *schema.ResourceData, meta interface{}) error {
//	conn := meta.(*AWSClient).cloudwatchlogsconn
//
 	log.Printf("[DEBUG] Deleting EventSourceMapping")
//	log_group := d.Get("log_group").(string)
//	name := d.Get("name").(string)
//
//	params := &cloudwatchlogs.DeleteSubscriptionFilterInput{
//		FilterName:   aws.String(name),      // Required
//		LogGroupName: aws.String(log_group), // Required
//	}
//	_, err := conn.DeleteSubscriptionFilter(params)
//
//	if err != nil {
//		return fmt.Errorf(
//			"Error deleting Subscription Filter from log group: %s with name filter name %s", log_group, name)
//	}
//	d.SetId("")
	return nil
}

//func LambdaEventSourceMappingId(d *schema.ResourceData) string {
//	var buf bytes.Buffer
//
//	name := d.Get("name").(string)
//	destination := d.Get("destination").(string)
//	filter_pattern := d.Get("filter_pattern").(string)
//	log_group := d.Get("log_group").(string)
//
//	buf.WriteString(fmt.Sprintf("%s-", name))
//	buf.WriteString(fmt.Sprintf("%s-", destination))
//	buf.WriteString(fmt.Sprintf("%s-", log_group))
//	buf.WriteString(fmt.Sprintf("%s-", filter_pattern))
//
//	return fmt.Sprintf("cwlsf-%d", hashcode.String(buf.String()))
//}
