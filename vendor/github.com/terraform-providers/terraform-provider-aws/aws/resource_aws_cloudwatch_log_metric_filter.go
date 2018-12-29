package aws

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
)

func resourceAwsCloudWatchLogMetricFilter() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsCloudWatchLogMetricFilterUpdate,
		Read:   resourceAwsCloudWatchLogMetricFilterRead,
		Update: resourceAwsCloudWatchLogMetricFilterUpdate,
		Delete: resourceAwsCloudWatchLogMetricFilterDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateLogMetricFilterName,
			},

			"pattern": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringLenBetween(0, 1024),
				StateFunc: func(v interface{}) string {
					s, ok := v.(string)
					if !ok {
						return ""
					}
					return strings.TrimSpace(s)
				},
			},

			"log_group_name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateLogGroupName,
			},

			"metric_transformation": {
				Type:     schema.TypeList,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateLogMetricFilterTransformationName,
						},
						"namespace": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateLogMetricFilterTransformationName,
						},
						"value": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validation.StringLenBetween(0, 100),
						},
						"default_value": {
							Type:     schema.TypeFloat,
							Optional: true,
						},
					},
				},
			},
		},
	}
}

func resourceAwsCloudWatchLogMetricFilterUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cloudwatchlogsconn

	input := cloudwatchlogs.PutMetricFilterInput{
		FilterName:    aws.String(d.Get("name").(string)),
		FilterPattern: aws.String(strings.TrimSpace(d.Get("pattern").(string))),
		LogGroupName:  aws.String(d.Get("log_group_name").(string)),
	}

	transformations := d.Get("metric_transformation").([]interface{})
	o := transformations[0].(map[string]interface{})
	metricsTransformations, err := expandCloudWachLogMetricTransformations(o)
	if err != nil {
		return err
	}
	input.MetricTransformations = metricsTransformations

	log.Printf("[DEBUG] Creating/Updating CloudWatch Log Metric Filter: %s", input)
	_, err = conn.PutMetricFilter(&input)
	if err != nil {
		return fmt.Errorf("Creating/Updating CloudWatch Log Metric Filter failed: %s", err)
	}

	d.SetId(d.Get("name").(string))

	log.Println("[INFO] CloudWatch Log Metric Filter created/updated")

	return resourceAwsCloudWatchLogMetricFilterRead(d, meta)
}

func resourceAwsCloudWatchLogMetricFilterRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cloudwatchlogsconn

	mf, err := lookupCloudWatchLogMetricFilter(conn, d.Get("name").(string),
		d.Get("log_group_name").(string), nil)
	if err != nil {
		if _, ok := err.(*resource.NotFoundError); ok {
			log.Printf("[WARN] Removing CloudWatch Log Metric Filter as it is gone")
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Failed reading CloudWatch Log Metric Filter: %s", err)
	}

	log.Printf("[DEBUG] Found CloudWatch Log Metric Filter: %s", mf)

	d.Set("name", mf.FilterName)
	d.Set("pattern", mf.FilterPattern)
	d.Set("metric_transformation", flattenCloudWachLogMetricTransformations(mf.MetricTransformations))

	return nil
}

func lookupCloudWatchLogMetricFilter(conn *cloudwatchlogs.CloudWatchLogs,
	name, logGroupName string, nextToken *string) (*cloudwatchlogs.MetricFilter, error) {

	input := cloudwatchlogs.DescribeMetricFiltersInput{
		FilterNamePrefix: aws.String(name),
		LogGroupName:     aws.String(logGroupName),
		NextToken:        nextToken,
	}
	log.Printf("[DEBUG] Reading CloudWatch Log Metric Filter: %s", input)
	resp, err := conn.DescribeMetricFilters(&input)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "ResourceNotFoundException" {
			return nil, &resource.NotFoundError{
				Message: fmt.Sprintf("CloudWatch Log Metric Filter %q / %q not found via"+
					" initial DescribeMetricFilters call", name, logGroupName),
				LastError:   err,
				LastRequest: input,
			}
		}

		return nil, fmt.Errorf("Failed describing CloudWatch Log Metric Filter: %s", err)
	}

	for _, mf := range resp.MetricFilters {
		if *mf.FilterName == name {
			return mf, nil
		}
	}

	if resp.NextToken != nil {
		return lookupCloudWatchLogMetricFilter(conn, name, logGroupName, resp.NextToken)
	}

	return nil, &resource.NotFoundError{
		Message: fmt.Sprintf("CloudWatch Log Metric Filter %q / %q not found "+
			"in given results from DescribeMetricFilters", name, logGroupName),
		LastResponse: resp,
		LastRequest:  input,
	}
}

func resourceAwsCloudWatchLogMetricFilterDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cloudwatchlogsconn

	input := cloudwatchlogs.DeleteMetricFilterInput{
		FilterName:   aws.String(d.Get("name").(string)),
		LogGroupName: aws.String(d.Get("log_group_name").(string)),
	}
	log.Printf("[INFO] Deleting CloudWatch Log Metric Filter: %s", d.Id())
	_, err := conn.DeleteMetricFilter(&input)
	if err != nil {
		return fmt.Errorf("Error deleting CloudWatch Log Metric Filter: %s", err)
	}
	log.Println("[INFO] CloudWatch Log Metric Filter deleted")

	return nil
}
