package aws

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
)

func resourceAwsCloudWatchLogGroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsCloudWatchLogGroupCreate,
		Read:   resourceAwsCloudWatchLogGroupRead,
		Update: resourceAwsCloudWatchLogGroupUpdate,
		Delete: resourceAwsCloudWatchLogGroupDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateLogGroupName,
			},

			"retention_in_days": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  0,
			},

			"arn": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsCloudWatchLogGroupCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cloudwatchlogsconn

	log.Printf("[DEBUG] Creating CloudWatch Log Group: %s", d.Get("name").(string))
	_, err := conn.CreateLogGroup(&cloudwatchlogs.CreateLogGroupInput{
		LogGroupName: aws.String(d.Get("name").(string)),
	})
	if err != nil {
		return fmt.Errorf("Creating CloudWatch Log Group failed: %s", err)
	}

	d.SetId(d.Get("name").(string))

	log.Println("[INFO] CloudWatch Log Group created")

	return resourceAwsCloudWatchLogGroupUpdate(d, meta)
}

func resourceAwsCloudWatchLogGroupRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cloudwatchlogsconn
	log.Printf("[DEBUG] Reading CloudWatch Log Group: %q", d.Get("name").(string))
	lg, err := lookupCloudWatchLogGroup(conn, d.Get("name").(string), nil)
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Found Log Group: %#v", *lg)

	d.Set("arn", *lg.Arn)
	d.Set("name", *lg.LogGroupName)

	if lg.RetentionInDays != nil {
		d.Set("retention_in_days", *lg.RetentionInDays)
	}

	return nil
}

func lookupCloudWatchLogGroup(conn *cloudwatchlogs.CloudWatchLogs,
	name string, nextToken *string) (*cloudwatchlogs.LogGroup, error) {
	input := &cloudwatchlogs.DescribeLogGroupsInput{
		LogGroupNamePrefix: aws.String(name),
		NextToken:          nextToken,
	}
	resp, err := conn.DescribeLogGroups(input)
	if err != nil {
		return nil, err
	}

	for _, lg := range resp.LogGroups {
		if *lg.LogGroupName == name {
			return lg, nil
		}
	}

	if resp.NextToken != nil {
		return lookupCloudWatchLogGroup(conn, name, resp.NextToken)
	}

	return nil, fmt.Errorf("CloudWatch Log Group %q not found", name)
}

func resourceAwsCloudWatchLogGroupUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cloudwatchlogsconn

	name := d.Get("name").(string)
	log.Printf("[DEBUG] Updating CloudWatch Log Group: %q", name)

	if d.HasChange("retention_in_days") {
		var err error

		if v, ok := d.GetOk("retention_in_days"); ok {
			input := cloudwatchlogs.PutRetentionPolicyInput{
				LogGroupName:    aws.String(name),
				RetentionInDays: aws.Int64(int64(v.(int))),
			}
			log.Printf("[DEBUG] Setting retention for CloudWatch Log Group: %q: %s", name, input)
			_, err = conn.PutRetentionPolicy(&input)
		} else {
			log.Printf("[DEBUG] Deleting retention for CloudWatch Log Group: %q", name)
			_, err = conn.DeleteRetentionPolicy(&cloudwatchlogs.DeleteRetentionPolicyInput{
				LogGroupName: aws.String(name),
			})
		}

		if err != nil {
			return err
		}
	}

	return resourceAwsCloudWatchLogGroupRead(d, meta)
}

func resourceAwsCloudWatchLogGroupDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cloudwatchlogsconn
	log.Printf("[INFO] Deleting CloudWatch Log Group: %s", d.Id())
	_, err := conn.DeleteLogGroup(&cloudwatchlogs.DeleteLogGroupInput{
		LogGroupName: aws.String(d.Get("name").(string)),
	})
	if err != nil {
		return fmt.Errorf("Error deleting CloudWatch Log Group: %s", err)
	}
	log.Println("[INFO] CloudWatch Log Group deleted")

	d.SetId("")

	return nil
}
