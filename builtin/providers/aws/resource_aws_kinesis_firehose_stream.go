package aws

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/firehose"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsKinesisFirehoseDeliveryStream() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsKinesisFirehoseDeliveryStreamCreate,
		Read:   resourceAwsKinesisFirehoseDeliveryStreamRead,
		Update: resourceAwsKinesisFirehoseDeliveryStreamUpdate,
		Delete: resourceAwsKinesisFirehoseDeliveryStreamDelete,

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

			"role_arn": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"s3_bucket_arn": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"s3_prefix": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"s3_buffer_size": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  5,
			},

			"s3_buffer_interval": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  300,
			},

			"s3_data_compression": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "UNCOMPRESSED",
			},

			"arn": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
		},
	}
}

func resourceAwsKinesisFirehoseDeliveryStreamCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).firehoseconn

	sn := d.Get("name").(string)
	input := &firehose.CreateDeliveryStreamInput{
		DeliveryStreamName: aws.String(sn),
	}

	s3_config := &firehose.S3DestinationConfiguration{
		BucketARN: aws.String(d.Get("s3_bucket_arn").(string)),
		RoleARN:   aws.String(d.Get("role_arn").(string)),
		BufferingHints: &firehose.BufferingHints{
			IntervalInSeconds: aws.Int64(int64(d.Get("s3_buffer_interval").(int))),
			SizeInMBs:         aws.Int64(int64(d.Get("s3_buffer_size").(int))),
		},
		CompressionFormat: aws.String(d.Get("s3_data_compression").(string)),
	}
	if v, ok := d.GetOk("s3_prefix"); ok {
		s3_config.Prefix = aws.String(v.(string))
	}

	input.S3DestinationConfiguration = s3_config

	_, err := conn.CreateDeliveryStream(input)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			return fmt.Errorf("[WARN] Error creating Kinesis Firehose Delivery Stream: \"%s\", code: \"%s\"", awsErr.Message(), awsErr.Code())
		}
		return err
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"CREATING"},
		Target:     "ACTIVE",
		Refresh:    firehoseStreamStateRefreshFunc(conn, sn),
		Timeout:    5 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	firehoseStream, err := stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf(
			"Error waiting for Kinesis Stream (%s) to become active: %s",
			sn, err)
	}

	s := firehoseStream.(*firehose.DeliveryStreamDescription)
	d.SetId(*s.DeliveryStreamARN)
	d.Set("arn", s.DeliveryStreamARN)

	return resourceAwsKinesisStreamUpdate(d, meta)
}

func resourceAwsKinesisFirehoseDeliveryStreamUpdate(d *schema.ResourceData, meta interface{}) error {
	//	conn := meta.(*AWSClient).firehoseconn

	return resourceAwsKinesisFirehoseDeliveryStreamRead(d, meta)
}

func resourceAwsKinesisFirehoseDeliveryStreamRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).firehoseconn
	sn := d.Get("name").(string)
	describeOpts := &firehose.DeliveryStreamDescription{
		DeliveryStreamName: aws.String(sn),
	}
	resp, err := conn.DescribeDeliveryStream(describeOpts)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == "ResourceNotFoundException" {
				d.SetId("")
				return nil
			}
			return fmt.Errorf("[WARN] Error reading Kinesis Firehose Delivery Stream: \"%s\", code: \"%s\"", awsErr.Message(), awsErr.Code())
		}
		return err
	}

	s := resp.DeliveryStreamDescription
	d.Set("arn", *s.DeliveryStreamARN)

	return nil
}

func resourceAwsKinesisFirehoseDeliveryStreamDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).firehoseconn

	sn := d.Get("name").(string)
	_, err := conn.DeleteDeliveryStream(&firehose.DeleteDeliveryStreamInput{
		DeliveryStreamName: aws.String(sn),
	})

	if err != nil {
		return err
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"DELETING"},
		Target:     "DESTROYED",
		Refresh:    firehoseStreamStateRefreshFunc(conn, sn),
		Timeout:    5 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf(
			"Error waiting for Delivery Stream (%s) to be destroyed: %s",
			sn, err)
	}

	d.SetId("")
	return nil
}

func firehoseStreamStateRefreshFunc(conn *firehose.Firehose, sn string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		describeOpts := &firehose.DescribeDeliveryStreamInput{
			DeliveryStreamName: aws.String(sn),
		}
		resp, err := conn.DescribeDeliveryStream(describeOpts)
		if err != nil {
			if awsErr, ok := err.(awserr.Error); ok {
				if awsErr.Code() == "ResourceNotFoundException" {
					return 42, "DESTROYED", nil
				}
				return nil, awsErr.Code(), err
			}
			return nil, "failed", err
		}

		return resp.DeliveryStreamDescription, *resp.DeliveryStreamDescription.DeliveryStreamStatus, nil
	}
}
