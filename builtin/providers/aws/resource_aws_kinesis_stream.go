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

			"retention_period": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  24,
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(int)
					if value < 24 || value > 168 {
						errors = append(errors, fmt.Errorf(
							"%q must be between 24 and 168 hours", k))
					}
					return
				},
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
		Target:     []string{"ACTIVE"},
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

	if err := setKinesisRetentionPeriod(conn, d); err != nil {
		return err
	}

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
	d.Set("retention_period", state.retentionPeriod)

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
		Target:     []string{"DESTROYED"},
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

func setKinesisRetentionPeriod(conn *kinesis.Kinesis, d *schema.ResourceData) error {
	sn := d.Get("name").(string)

	oraw, nraw := d.GetChange("retention_period")
	o := oraw.(int)
	n := nraw.(int)

	if n == 0 {
		log.Printf("[DEBUG] Kinesis Stream (%q) Retention Period Not Changed", sn)
		return nil
	}

	if n > o {
		log.Printf("[DEBUG] Increasing %s Stream Retention Period to %d", sn, n)
		_, err := conn.IncreaseStreamRetentionPeriod(&kinesis.IncreaseStreamRetentionPeriodInput{
			StreamName:           aws.String(sn),
			RetentionPeriodHours: aws.Int64(int64(n)),
		})
		if err != nil {
			return err
		}

	} else {
		log.Printf("[DEBUG] Decreasing %s Stream Retention Period to %d", sn, n)
		_, err := conn.DecreaseStreamRetentionPeriod(&kinesis.DecreaseStreamRetentionPeriodInput{
			StreamName:           aws.String(sn),
			RetentionPeriodHours: aws.Int64(int64(n)),
		})
		if err != nil {
			return err
		}
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"UPDATING"},
		Target:     []string{"ACTIVE"},
		Refresh:    streamStateRefreshFunc(conn, sn),
		Timeout:    5 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err := stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf(
			"Error waiting for Kinesis Stream (%s) to become active: %s",
			sn, err)
	}

	return nil
}

type kinesisStreamState struct {
	arn             string
	status          string
	shardCount      int
	retentionPeriod int64
}

func readKinesisStreamState(conn *kinesis.Kinesis, sn string) (kinesisStreamState, error) {
	describeOpts := &kinesis.DescribeStreamInput{
		StreamName: aws.String(sn),
	}

	var state kinesisStreamState
	err := conn.DescribeStreamPages(describeOpts, func(page *kinesis.DescribeStreamOutput, last bool) (shouldContinue bool) {
		state.arn = aws.StringValue(page.StreamDescription.StreamARN)
		state.status = aws.StringValue(page.StreamDescription.StreamStatus)
		state.shardCount += len(openShards(page.StreamDescription.Shards))
		state.retentionPeriod = aws.Int64Value(page.StreamDescription.RetentionPeriodHours)
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

// See http://docs.aws.amazon.com/kinesis/latest/dev/kinesis-using-sdk-java-resharding-merge.html
func openShards(shards []*kinesis.Shard) []*kinesis.Shard {
	var open []*kinesis.Shard
	for _, s := range shards {
		if s.SequenceNumberRange.EndingSequenceNumber == nil {
			open = append(open, s)
		}
	}

	return open
}
