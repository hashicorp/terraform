package aws

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsKinesisStream() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsKinesisStreamCreate,
		Read:   resourceAwsKinesisStreamRead,
		Delete: resourceAwsKinesisStreamDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"shard_count": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},

			"arn": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
		},
	}
}

func resourceAwsKinesisStreamCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).kinesisconn
	sn := d.Get("name").(string)
	createOpts := &kinesis.CreateStreamInput{
		ShardCount: aws.Int64(int64(d.Get("shard_count").(int))),
		StreamName: aws.String(sn),
	}

	_, err := conn.CreateStream(createOpts)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			return fmt.Errorf("[WARN] Error creating Kinesis Stream: \"%s\", code: \"%s\"", awsErr.Message(), awsErr.Code())
		}
		return err
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"CREATING"},
		Target:     "ACTIVE",
		Refresh:    streamStateRefreshFunc(conn, sn),
		Timeout:    5 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	streamRaw, err := stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf(
			"Error waiting for Kinesis Stream (%s) to become active: %s",
			sn, err)
	}

	s := streamRaw.(*kinesis.StreamDescription)
	d.SetId(*s.StreamARN)
	d.Set("arn", s.StreamARN)

	return nil
}

func resourceAwsKinesisStreamRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).kinesisconn
	describeOpts := &kinesis.DescribeStreamInput{
		StreamName: aws.String(d.Get("name").(string)),
		Limit:      aws.Int64(1),
	}
	resp, err := conn.DescribeStream(describeOpts)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == "ResourceNotFoundException" {
				d.SetId("")
				return nil
			}
			return fmt.Errorf("[WARN] Error reading Kinesis Stream: \"%s\", code: \"%s\"", awsErr.Message(), awsErr.Code())
		}
		return err
	}

	s := resp.StreamDescription
	d.Set("arn", *s.StreamARN)
	d.Set("shard_count", len(s.Shards))

	return nil
}

func resourceAwsKinesisStreamDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).kinesisconn
	sn := d.Get("name").(string)
	_, err := conn.DeleteStream(&kinesis.DeleteStreamInput{
		StreamName: aws.String(sn),
	})

	if err != nil {
		return err
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"DELETING"},
		Target:     "DESTROYED",
		Refresh:    streamStateRefreshFunc(conn, sn),
		Timeout:    5 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf(
			"Error waiting for Stream (%s) to be destroyed: %s",
			sn, err)
	}

	d.SetId("")
	return nil
}

func streamStateRefreshFunc(conn *kinesis.Kinesis, sn string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		describeOpts := &kinesis.DescribeStreamInput{
			StreamName: aws.String(sn),
			Limit:      aws.Int64(1),
		}
		resp, err := conn.DescribeStream(describeOpts)
		if err != nil {
			if awsErr, ok := err.(awserr.Error); ok {
				if awsErr.Code() == "ResourceNotFoundException" {
					return 42, "DESTROYED", nil
				}
				return nil, awsErr.Code(), err
			}
			return nil, "failed", err
		}

		return resp.StreamDescription, *resp.StreamDescription.StreamStatus, nil
	}
}
