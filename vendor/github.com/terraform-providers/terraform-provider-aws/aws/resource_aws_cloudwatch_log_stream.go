package aws

import (
	"fmt"
	"log"
	"regexp"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsCloudWatchLogStream() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsCloudWatchLogStreamCreate,
		Read:   resourceAwsCloudWatchLogStreamRead,
		Delete: resourceAwsCloudWatchLogStreamDelete,

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateCloudWatchLogStreamName,
			},

			"log_group_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceAwsCloudWatchLogStreamCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cloudwatchlogsconn

	log.Printf("[DEBUG] Creating CloudWatch Log Stream: %s", d.Get("name").(string))
	_, err := conn.CreateLogStream(&cloudwatchlogs.CreateLogStreamInput{
		LogGroupName:  aws.String(d.Get("log_group_name").(string)),
		LogStreamName: aws.String(d.Get("name").(string)),
	})
	if err != nil {
		return fmt.Errorf("Creating CloudWatch Log Stream failed: %s", err)
	}

	d.SetId(d.Get("name").(string))

	return resourceAwsCloudWatchLogStreamRead(d, meta)
}

func resourceAwsCloudWatchLogStreamRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cloudwatchlogsconn

	group := d.Get("log_group_name").(string)

	ls, exists, err := lookupCloudWatchLogStream(conn, d.Id(), group, nil)
	if err != nil {
		if !isAWSErr(err, cloudwatchlogs.ErrCodeResourceNotFoundException, "") {
			return err
		}

		log.Printf("[DEBUG] container CloudWatch group %q Not Found.", group)
		exists = false
	}

	if !exists {
		log.Printf("[DEBUG] CloudWatch Stream %q Not Found. Removing from state", d.Id())
		d.SetId("")
		return nil
	}

	d.Set("arn", ls.Arn)
	d.Set("name", ls.LogStreamName)

	return nil
}

func resourceAwsCloudWatchLogStreamDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cloudwatchlogsconn

	log.Printf("[INFO] Deleting CloudWatch Log Stream: %s", d.Id())
	params := &cloudwatchlogs.DeleteLogStreamInput{
		LogGroupName:  aws.String(d.Get("log_group_name").(string)),
		LogStreamName: aws.String(d.Id()),
	}
	_, err := conn.DeleteLogStream(params)
	if err != nil {
		return fmt.Errorf("Error deleting CloudWatch Log Stream: %s", err)
	}

	return nil
}

func lookupCloudWatchLogStream(conn *cloudwatchlogs.CloudWatchLogs,
	name string, logStreamName string, nextToken *string) (*cloudwatchlogs.LogStream, bool, error) {
	input := &cloudwatchlogs.DescribeLogStreamsInput{
		LogStreamNamePrefix: aws.String(name),
		LogGroupName:        aws.String(logStreamName),
		NextToken:           nextToken,
	}
	resp, err := conn.DescribeLogStreams(input)
	if err != nil {
		return nil, true, err
	}

	for _, ls := range resp.LogStreams {
		if *ls.LogStreamName == name {
			return ls, true, nil
		}
	}

	if resp.NextToken != nil {
		return lookupCloudWatchLogStream(conn, name, logStreamName, resp.NextToken)
	}

	return nil, false, nil
}

func validateCloudWatchLogStreamName(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if regexp.MustCompile(`:`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"colons not allowed in %q:", k))
	}
	if len(value) < 1 || len(value) > 512 {
		errors = append(errors, fmt.Errorf(
			"%q must be between 1 and 512 characters: %q", k, value))
	}

	return

}
