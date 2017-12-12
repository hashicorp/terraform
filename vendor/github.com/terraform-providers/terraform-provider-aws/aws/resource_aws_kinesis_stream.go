package aws

import (
	"fmt"
	"log"
	"time"

	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
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

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(5 * time.Minute),
			Update: schema.DefaultTimeout(120 * time.Minute),
			Delete: schema.DefaultTimeout(120 * time.Minute),
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

			"encryption_type": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "NONE",
				ValidateFunc: validation.StringInSlice([]string{"NONE", "KMS"}, true),
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					if strings.ToLower(old) == strings.ToLower(new) {
						return true
					}
					return false
				},
			},

			"kms_key_id": {
				Type:     schema.TypeString,
				Optional: true,
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

	err := resource.Retry(5*time.Minute, func() *resource.RetryError {
		_, err := conn.CreateStream(createOpts)
		if isAWSErr(err, "LimitExceededException", "simultaneously be in CREATING or DELETING") {
			return resource.RetryableError(err)
		}
		// AWS (un)helpfully raises LimitExceededException
		// rather than ThrottlingException here
		if isAWSErr(err, "LimitExceededException", "Rate exceeded for stream") {
			return resource.RetryableError(err)
		}
		return resource.NonRetryableError(err)
	})

	if err != nil {
		return fmt.Errorf("Unable to create stream: %s", err)
	}

	// No error, wait for ACTIVE state
	stateConf := &resource.StateChangeConf{
		Pending:    []string{"CREATING"},
		Target:     []string{"ACTIVE"},
		Refresh:    streamStateRefreshFunc(conn, sn),
		Timeout:    d.Timeout(schema.TimeoutCreate),
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

	if err := updateKinesisShardCount(conn, d); err != nil {
		return err
	}
	if err := setKinesisRetentionPeriod(conn, d); err != nil {
		return err
	}
	if err := updateKinesisShardLevelMetrics(conn, d); err != nil {
		return err
	}

	if err := updateKinesisStreamEncryption(conn, d); err != nil {
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

	d.Set("encryption_type", state.encryptionType)
	d.Set("kms_key_id", state.keyId)

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
		Timeout:    d.Timeout(schema.TimeoutDelete),
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

	if err := waitForKinesisToBeActive(conn, d.Timeout(schema.TimeoutUpdate), sn); err != nil {
		return err
	}

	return nil
}

func updateKinesisShardCount(conn *kinesis.Kinesis, d *schema.ResourceData) error {
	sn := d.Get("name").(string)

	oraw, nraw := d.GetChange("shard_count")
	o := oraw.(int)
	n := nraw.(int)

	if n == o {
		log.Printf("[DEBUG] Kinesis Stream (%q) Shard Count Not Changed", sn)
		return nil
	}

	log.Printf("[DEBUG] Change %s Stream ShardCount to %d", sn, n)
	_, err := conn.UpdateShardCount(&kinesis.UpdateShardCountInput{
		StreamName:       aws.String(sn),
		TargetShardCount: aws.Int64(int64(n)),
		ScalingType:      aws.String("UNIFORM_SCALING"),
	})
	if err != nil {
		return err
	}

	if err := waitForKinesisToBeActive(conn, d.Timeout(schema.TimeoutUpdate), sn); err != nil {
		return err
	}

	return nil
}

func updateKinesisStreamEncryption(conn *kinesis.Kinesis, d *schema.ResourceData) error {
	sn := d.Get("name").(string)

	// If this is not a new resource AND there is no change to encryption_type or kms_key_id
	// return nil
	if !d.IsNewResource() && (!d.HasChange("encryption_type") || !d.HasChange("kms_key_id")) {
		return nil
	}

	oldType, newType := d.GetChange("encryption_type")
	if oldType.(string) != "" && oldType.(string) != "NONE" {
		// This means that we have an old encryption type - i.e. Encryption is enabled and we want to change it
		// The quirk about this API is that, when we are disabling the StreamEncryption
		// We need to pass in that old KMS Key Id that was being used for Encryption and
		// We also need to pass in the type of Encryption we were using - i.e. KMS as that
		// Is the only supported Encryption method right now
		// If we don't get this and pass in the actual EncryptionType we want to move to i.e. NONE
		// We get the following error
		//
		//        InvalidArgumentException: Encryption type cannot be NONE.
		oldKey, _ := d.GetChange("kms_key_id")
		oldType, _ := d.GetChange("encryption_type")

		log.Printf("[INFO] Stopping Stream Encryption for %s", sn)
		params := &kinesis.StopStreamEncryptionInput{
			StreamName:     aws.String(sn),
			EncryptionType: aws.String(oldType.(string)),
			KeyId:          aws.String(oldKey.(string)),
		}

		_, err := conn.StopStreamEncryption(params)
		if err != nil {
			return err
		}
	}

	if newType.(string) != "NONE" {
		if _, ok := d.GetOk("kms_key_id"); !ok {
			return fmt.Errorf("KMS Key Id required when setting encryption_type is not set as NONE")
		}

		log.Printf("[INFO] Starting Stream Encryption for %s", sn)
		params := &kinesis.StartStreamEncryptionInput{
			StreamName:     aws.String(sn),
			EncryptionType: aws.String(newType.(string)),
			KeyId:          aws.String(d.Get("kms_key_id").(string)),
		}

		_, err := conn.StartStreamEncryption(params)
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
			"Error waiting for Stream (%s) to be ACTIVE: %s",
			sn, err)
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
		if err := waitForKinesisToBeActive(conn, d.Timeout(schema.TimeoutUpdate), sn); err != nil {
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
		if err := waitForKinesisToBeActive(conn, d.Timeout(schema.TimeoutUpdate), sn); err != nil {
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
	encryptionType    string
	keyId             string
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
		state.encryptionType = aws.StringValue(page.StreamDescription.EncryptionType)
		state.keyId = aws.StringValue(page.StreamDescription.KeyId)
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

func waitForKinesisToBeActive(conn *kinesis.Kinesis, timeout time.Duration, sn string) error {
	stateConf := &resource.StateChangeConf{
		Pending:    []string{"UPDATING"},
		Target:     []string{"ACTIVE"},
		Refresh:    streamStateRefreshFunc(conn, sn),
		Timeout:    timeout,
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
