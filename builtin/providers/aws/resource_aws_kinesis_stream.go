package aws

import (
	"fmt"
	"log"
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
		Update: resourceAwsKinesisStreamUpdate,
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
			"tags": tagsSchema(),
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

	s := streamRaw.(kinesisStreamState)
	d.SetId(s.arn)
	d.Set("arn", s.arn)
	d.Set("shard_count", s.shardCount)

	return resourceAwsKinesisStreamUpdate(d, meta)
}

func resourceAwsKinesisStreamUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).kinesisconn

	d.Partial(true)
	if err := setTagsKinesis(conn, d); err != nil {
		return err
	}

	d.SetPartial("tags")
	d.Partial(false)

	return resourceAwsKinesisStreamRead(d, meta)
}

func resourceAwsKinesisStreamRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).kinesisconn
	sn := d.Get("name").(string)

	state, err := readKinesisStreamState(conn, sn)
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
	d.Set("arn", state.arn)
	d.Set("shard_count", state.shardCount)

	// set tags
	describeTagsOpts := &kinesis.ListTagsForStreamInput{
		StreamName: aws.String(sn),
	}
	tagsResp, err := conn.ListTagsForStream(describeTagsOpts)
	if err != nil {
		log.Printf("[DEBUG] Error retrieving tags for Stream: %s. %s", sn, err)
	} else {
		d.Set("tags", tagsToMapKinesis(tagsResp.Tags))
	}

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

type kinesisStreamState struct {
	arn        string
	status     string
	shardCount int
}

func readKinesisStreamState(conn *kinesis.Kinesis, sn string) (kinesisStreamState, error) {
	describeOpts := &kinesis.DescribeStreamInput{
		StreamName: aws.String(sn),
	}

	var state kinesisStreamState
	err := conn.DescribeStreamPages(describeOpts, func(page *kinesis.DescribeStreamOutput, last bool) (shouldContinue bool) {
		state.arn = aws.StringValue(page.StreamDescription.StreamARN)
		state.status = aws.StringValue(page.StreamDescription.StreamStatus)
		state.shardCount += len(page.StreamDescription.Shards)
		return !last
	})
	return state, err
}

func streamStateRefreshFunc(conn *kinesis.Kinesis, sn string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		state, err := readKinesisStreamState(conn, sn)
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
