package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudtrail"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsCloudTrail() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsCloudTrailCreate,
		Read:   resourceAwsCloudTrailRead,
		Update: resourceAwsCloudTrailUpdate,
		Delete: resourceAwsCloudTrailDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"enable_logging": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"s3_bucket_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"s3_key_prefix": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"cloud_watch_logs_role_arn": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"cloud_watch_logs_group_arn": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"include_global_service_events": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"sns_topic_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func resourceAwsCloudTrailCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cloudtrailconn

	input := cloudtrail.CreateTrailInput{
		Name:         aws.String(d.Get("name").(string)),
		S3BucketName: aws.String(d.Get("s3_bucket_name").(string)),
	}

	if v, ok := d.GetOk("cloud_watch_logs_group_arn"); ok {
		input.CloudWatchLogsLogGroupArn = aws.String(v.(string))
	}
	if v, ok := d.GetOk("cloud_watch_logs_role_arn"); ok {
		input.CloudWatchLogsRoleArn = aws.String(v.(string))
	}
	if v, ok := d.GetOk("include_global_service_events"); ok {
		input.IncludeGlobalServiceEvents = aws.Bool(v.(bool))
	}
	if v, ok := d.GetOk("s3_key_prefix"); ok {
		input.S3KeyPrefix = aws.String(v.(string))
	}
	if v, ok := d.GetOk("sns_topic_name"); ok {
		input.SnsTopicName = aws.String(v.(string))
	}

	t, err := conn.CreateTrail(&input)
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] CloudTrail created: %s", t)

	d.SetId(*t.Name)

	// AWS CloudTrail sets newly-created trails to false.
	if v, ok := d.GetOk("enable_logging"); ok && v.(bool) {
		err := cloudTrailSetLogging(conn, v.(bool), d.Id())
		if err != nil {
			return err
		}
	}

	return resourceAwsCloudTrailRead(d, meta)
}

func resourceAwsCloudTrailRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cloudtrailconn

	name := d.Get("name").(string)
	input := cloudtrail.DescribeTrailsInput{
		TrailNameList: []*string{
			aws.String(name),
		},
	}
	resp, err := conn.DescribeTrails(&input)
	if err != nil {
		return err
	}
	if len(resp.TrailList) == 0 {
		return fmt.Errorf("No CloudTrail found, using name %q", name)
	}

	trail := resp.TrailList[0]
	log.Printf("[DEBUG] CloudTrail received: %s", trail)

	d.Set("name", trail.Name)
	d.Set("s3_bucket_name", trail.S3BucketName)
	d.Set("s3_key_prefix", trail.S3KeyPrefix)
	d.Set("cloud_watch_logs_role_arn", trail.CloudWatchLogsRoleArn)
	d.Set("cloud_watch_logs_group_arn", trail.CloudWatchLogsLogGroupArn)
	d.Set("include_global_service_events", trail.IncludeGlobalServiceEvents)
	d.Set("sns_topic_name", trail.SnsTopicName)

	logstatus, err := cloudTrailGetLoggingStatus(conn, trail.Name)
	if err != nil {
		return err
	}
	d.Set("enable_logging", logstatus)

	return nil
}

func resourceAwsCloudTrailUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cloudtrailconn

	input := cloudtrail.UpdateTrailInput{
		Name: aws.String(d.Get("name").(string)),
	}

	if d.HasChange("s3_bucket_name") {
		input.S3BucketName = aws.String(d.Get("s3_bucket_name").(string))
	}
	if d.HasChange("s3_key_prefix") {
		input.S3KeyPrefix = aws.String(d.Get("s3_key_prefix").(string))
	}
	if d.HasChange("cloud_watch_logs_role_arn") {
		input.CloudWatchLogsRoleArn = aws.String(d.Get("cloud_watch_logs_role_arn").(string))
	}
	if d.HasChange("cloud_watch_logs_group_arn") {
		input.CloudWatchLogsLogGroupArn = aws.String(d.Get("cloud_watch_logs_group_arn").(string))
	}
	if d.HasChange("include_global_service_events") {
		input.IncludeGlobalServiceEvents = aws.Bool(d.Get("include_global_service_events").(bool))
	}
	if d.HasChange("sns_topic_name") {
		input.SnsTopicName = aws.String(d.Get("sns_topic_name").(string))
	}

	log.Printf("[DEBUG] Updating CloudTrail: %s", input)
	t, err := conn.UpdateTrail(&input)
	if err != nil {
		return err
	}

	if d.HasChange("enable_logging") {
		log.Printf("[DEBUG] Updating logging on CloudTrail: %s", input)
		err := cloudTrailSetLogging(conn, d.Get("enable_logging").(bool), *input.Name)
		if err != nil {
			return err
		}
	}

	log.Printf("[DEBUG] CloudTrail updated: %s", t)

	return resourceAwsCloudTrailRead(d, meta)
}

func resourceAwsCloudTrailDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cloudtrailconn
	name := d.Get("name").(string)

	log.Printf("[DEBUG] Deleting CloudTrail: %q", name)
	_, err := conn.DeleteTrail(&cloudtrail.DeleteTrailInput{
		Name: aws.String(name),
	})

	return err
}

func cloudTrailGetLoggingStatus(conn *cloudtrail.CloudTrail, id *string) (bool, error) {
	GetTrailStatusOpts := &cloudtrail.GetTrailStatusInput{
		Name: id,
	}
	resp, err := conn.GetTrailStatus(GetTrailStatusOpts)
	if err != nil {
		return false, fmt.Errorf("Error retrieving logging status of CloudTrail (%s): %s", *id, err)
	}

	return *resp.IsLogging, err
}

func cloudTrailSetLogging(conn *cloudtrail.CloudTrail, enabled bool, id string) error {
	if enabled {
		log.Printf(
			"[DEBUG] Starting logging on CloudTrail (%s)",
			id)
		StartLoggingOpts := &cloudtrail.StartLoggingInput{
			Name: aws.String(id),
		}
		if _, err := conn.StartLogging(StartLoggingOpts); err != nil {
			return fmt.Errorf(
				"Error starting logging on CloudTrail (%s): %s",
				id, err)
		}
	} else {
		log.Printf(
			"[DEBUG] Stopping logging on CloudTrail (%s)",
			id)
		StopLoggingOpts := &cloudtrail.StopLoggingInput{
			Name: aws.String(id),
		}
		if _, err := conn.StopLogging(StopLoggingOpts); err != nil {
			return fmt.Errorf(
				"Error stopping logging on CloudTrail (%s): %s",
				id, err)
		}
	}

	return nil
}
