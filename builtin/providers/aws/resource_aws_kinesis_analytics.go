package aws

import (
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/kinesisanalytics"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func shim(*schema.ResourceData, interface{}) error {
	return errors.New("Not implemented")
}

func resourceAwsKinesisAnalytics() *schema.Resource {
	return &schema.Resource{

		Create: resourceAwsKinesisAnalyticsCreate,
		Read:   shim,
		Update: shim,
		Delete: shim,

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
		},
	}
}

func resourceAwsKinesisAnalyticsCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).kinesisanalyticsconn
	name := d.Get("name").(string)
	createOpts := &kinesisanalytics.CreateApplicationInput{
		ApplicationName: aws.String(name),
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
		Timeout:    5 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	streamRaw, err := stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf(
			"Error waiting for Kinesis Analytics Application (%s) to become active: %s",
			name, err)
	}

	s := streamRaw.(*KinesisAnalyticsState)
	d.SetId(s.arn)
	d.Set("arn", s.arn)

	//todo: actually think about this
	return nil
}

type KinesisAnalyticsState struct {
	arn             string
	createTimestamp int64
	status          string
	code            string
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

	state.arn = aws.StringValue(output.ApplicationDetail.ApplicationARN)
	state.createTimestamp = aws.TimeValue(output.ApplicationDetail.CreateTimestamp).Unix()
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
