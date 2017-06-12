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
		Importer: &schema.ResourceImporter{
			State: resourceAwsKinesisStreamImport,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"shard_count": {
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},

			"retention_period": {
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

			"shard_level_metrics": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"arn": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"tags": tagsSchema(),
		},
	}
}

func resourceAwsKinesisStreamImport(
	d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	d.Set("name", d.Id())
	return []*schema.ResourceData{d}, nil
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

	s := streamRaw.(*kinesisStreamState)
	d.SetId(s.arn)
	d.Set("arn", s.arn)
	d.Set("shard_count", len(s.openShards))

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
	if err := updateKinesisShardLevelMetrics(conn, d); err != nil {
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
	d.SetId(state.arn)
	d.Set("arn", state.arn)
	d.Set("shard_count", len(state.openShards))
	d.Set("retention_period", state.retentionPeriod)

	if len(state.shardLevelMetrics) > 0 {
		d.Set("shard_level_metrics", state.shardLevelMetrics)
	}

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

	if err := waitForKinesisToBeActive(conn, sn); err != nil {
		return err
	}

	return nil
}

func updateKinesisShardLevelMetrics(conn *kinesis.Kinesis, d *schema.ResourceData) error {
	sn := d.Get("name").(string)

	o, n := d.GetChange("shard_level_metrics")
	if o == nil {
		o = new(schema.Set)
	}
	if n == nil {
		n = new(schema.Set)
	}

	os := o.(*schema.Set)
	ns := n.(*schema.Set)

	disableMetrics := os.Difference(ns)
	if disableMetrics.Len() != 0 {
		metrics := disableMetrics.List()
		log.Printf("[DEBUG] Disabling shard level metrics %v for stream %s", metrics, sn)

		props := &kinesis.DisableEnhancedMonitoringInput{
			StreamName:        aws.String(sn),
			ShardLevelMetrics: expandStringList(metrics),
		}

		_, err := conn.DisableEnhancedMonitoring(props)
		if err != nil {
			return fmt.Errorf("Failure to disable shard level metrics for stream %s: %s", sn, err)
		}
		if err := waitForKinesisToBeActive(conn, sn); err != nil {
			return err
		}
	}

	enabledMetrics := ns.Difference(os)
	if enabledMetrics.Len() != 0 {
		metrics := enabledMetrics.List()
		log.Printf("[DEBUG] Enabling shard level metrics %v for stream %s", metrics, sn)

		props := &kinesis.EnableEnhancedMonitoringInput{
			StreamName:        aws.String(sn),
			ShardLevelMetrics: expandStringList(metrics),
		}

		_, err := conn.EnableEnhancedMonitoring(props)
		if err != nil {
			return fmt.Errorf("Failure to enable shard level metrics for stream %s: %s", sn, err)
		}
		if err := waitForKinesisToBeActive(conn, sn); err != nil {
			return err
		}
	}

	return nil
}

type kinesisStreamState struct {
	arn               string
	creationTimestamp int64
	status            string
	retentionPeriod   int64
	openShards        []string
	closedShards      []string
	shardLevelMetrics []string
}

func readKinesisStreamState(conn *kinesis.Kinesis, sn string) (*kinesisStreamState, error) {
	describeOpts := &kinesis.DescribeStreamInput{
		StreamName: aws.String(sn),
	}

	state := &kinesisStreamState{}
	err := conn.DescribeStreamPages(describeOpts, func(page *kinesis.DescribeStreamOutput, last bool) (shouldContinue bool) {
		state.arn = aws.StringValue(page.StreamDescription.StreamARN)
		state.creationTimestamp = aws.TimeValue(page.StreamDescription.StreamCreationTimestamp).Unix()
		state.status = aws.StringValue(page.StreamDescription.StreamStatus)
		state.retentionPeriod = aws.Int64Value(page.StreamDescription.RetentionPeriodHours)
		state.openShards = append(state.openShards, flattenShards(openShards(page.StreamDescription.Shards))...)
		state.closedShards = append(state.closedShards, flattenShards(closedShards(page.StreamDescription.Shards))...)
		state.shardLevelMetrics = flattenKinesisShardLevelMetrics(page.StreamDescription.EnhancedMonitoring)
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

func waitForKinesisToBeActive(conn *kinesis.Kinesis, sn string) error {
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

func openShards(shards []*kinesis.Shard) []*kinesis.Shard {
	return filterShards(shards, true)
}

func closedShards(shards []*kinesis.Shard) []*kinesis.Shard {
	return filterShards(shards, false)
}

// See http://docs.aws.amazon.com/kinesis/latest/dev/kinesis-using-sdk-java-resharding-merge.html
func filterShards(shards []*kinesis.Shard, open bool) []*kinesis.Shard {
	res := make([]*kinesis.Shard, 0, len(shards))
	for _, s := range shards {
		if open && s.SequenceNumberRange.EndingSequenceNumber == nil {
			res = append(res, s)
		} else if !open && s.SequenceNumberRange.EndingSequenceNumber != nil {
			res = append(res, s)
		}
	}
	return res
}

func flattenShards(shards []*kinesis.Shard) []string {
	res := make([]string, len(shards))
	for i, s := range shards {
		res[i] = aws.StringValue(s.ShardId)
	}
	return res
}
