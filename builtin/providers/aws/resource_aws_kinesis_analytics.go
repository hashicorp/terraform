package aws

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/kinesisanalytics"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"strconv"
)

func shim(*schema.ResourceData, interface{}) error {
	return errors.New("func UPDATE Not implemented in resource_aws_kinesis_analytics.go")
}

func resourceAwsKinesisAnalytics() *schema.Resource {
	return &schema.Resource{

		Create: resourceAwsKinesisAnalyticsCreate,
		Read:   resourceAwsKinesisAnalyticsRead,
		Update: shim,
		Delete: resourceAwskinesisAnalyticsDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"application_description": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"application_code": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"arn": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"version_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"create_timestamp": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsKinesisAnalyticsCreate(d *schema.ResourceData, meta interface{}) error {

	conn := meta.(*AWSClient).kinesisanalyticsconn

	name := d.Get("name").(string)
	appDesc := d.Get("application_description").(string)
	appCode := d.Get("application_code").(string)

	createOpts := &kinesisanalytics.CreateApplicationInput{
		ApplicationName:        aws.String(name),
		ApplicationDescription: aws.String(appDesc),
		ApplicationCode:        aws.String(appCode),
	}

	_, err := conn.CreateApplication(createOpts)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			return fmt.Errorf("[WARN] Error creating Kinesis Analytics Application: \"%s\", code: \"%s\"", awsErr.Message(), awsErr.Code())
		}
		return err
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"CREATING"},
		Target:     []string{"READY"},
		Refresh:    applicationStateRefreshFunc(conn, name),
		Timeout:    3 * time.Minute,
		Delay:      5 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	state, err := stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf(
			"Error waiting for Kinesis Analytics Application (%s) to become active: %s",
			name, err)
	}

	s := state.(*KinesisAnalyticsState)
	d.SetId(s.arn)
	d.Set("arn", s.arn)
	d.Set("create_timestamp", strconv.FormatInt(s.createTimestamp, 10))
	d.Set("application_description", s.description)
	d.Set("application_code", s.code)

	return nil
}

func resourceAwsKinesisAnalyticsRead(d *schema.ResourceData, meta interface{}) error {

	conn := meta.(*AWSClient).kinesisanalyticsconn
	name := d.Get("name").(string)

	_, err := readKinesisAnalyticsState(conn, name)

	if err != nil {
		return err
	}

	return nil
}

func resourceAwskinesisAnalyticsDelete(d *schema.ResourceData, meta interface{}) error {

	conn := meta.(*AWSClient).kinesisanalyticsconn

	name := d.Get("name").(string)
	cerealizedTime := d.Get("create_timestamp").(string)
	ct, _ := strconv.ParseInt(cerealizedTime, 10, 64)

	createTime := time.Unix(ct, 0)

	input := &kinesisanalytics.DeleteApplicationInput{
		ApplicationName: aws.String(name),
		CreateTimestamp: aws.Time(createTime),
	}

	log.Printf("[DEBUG] Deleting Kinesis Analytics Application: %s", d.Id())

	_, err := conn.DeleteApplication(input)

	if err != nil {
		return fmt.Errorf("Error deleting Kinesis Analytics Application: %s", err)
	}

	return nil
}

type KinesisAnalyticsState struct {
	description     string
	code            string
	arn             string
	createTimestamp int64
	status          string
}

func readKinesisAnalyticsState(conn *kinesisanalytics.KinesisAnalytics, name string) (*KinesisAnalyticsState, error) {

	describeOpts := &kinesisanalytics.DescribeApplicationInput{
		ApplicationName: aws.String(name),
	}

	state := &KinesisAnalyticsState{}

	output, err := conn.DescribeApplication(describeOpts)

	if err != nil {
		return nil, err
	}

	state.description = aws.StringValue(output.ApplicationDetail.ApplicationDescription)

	state.code = aws.StringValue(output.ApplicationDetail.ApplicationCode)

	state.arn = aws.StringValue(output.ApplicationDetail.ApplicationARN)

	state.createTimestamp = aws.Time(*output.ApplicationDetail.CreateTimestamp).Unix()

	state.status = aws.StringValue(output.ApplicationDetail.ApplicationStatus)

	return state, nil
}

func applicationStateRefreshFunc(conn *kinesisanalytics.KinesisAnalytics, name string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		state, err := readKinesisAnalyticsState(conn, name)
		if err != nil {
			if awsErr, ok := err.(awserr.Error); ok {
				if awsErr.Code() == "ResourceNotFoundException" {
					return 42, "DESTROYED", nil
				}
				return nil, awsErr.Code(), err
			}
			return nil, "failed", err
		}

		return state, state.status, nil
	}
}
