package aws

import (
	"fmt"
	"log"
	"strings"
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

		SchemaVersion: 1,
		MigrateState:  resourceAwsKinesisFirehoseMigrateState,
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(string)
					if len(value) > 64 {
						errors = append(errors, fmt.Errorf(
							"%q cannot be longer than 64 characters", k))
					}
					return
				},
			},

			"destination": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				StateFunc: func(v interface{}) string {
					value := v.(string)
					return strings.ToLower(value)
				},
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(string)
					if value != "s3" && value != "redshift" && value != "elasticsearch" {
						errors = append(errors, fmt.Errorf(
							"%q must be one of 's3', 'redshift', 'elasticsearch'", k))
					}
					return
				},
			},

			// elements removed in v0.7.0
			"role_arn": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Removed:  "role_arn has been removed. Use a s3_configuration block instead. See https://terraform.io/docs/providers/aws/r/kinesis_firehose_delivery_stream.html",
			},

			"s3_bucket_arn": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Removed:  "s3_bucket_arn has been removed. Use a s3_configuration block instead. See https://terraform.io/docs/providers/aws/r/kinesis_firehose_delivery_stream.html",
			},

			"s3_prefix": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Removed:  "s3_prefix has been removed. Use a s3_configuration block instead. See https://terraform.io/docs/providers/aws/r/kinesis_firehose_delivery_stream.html",
			},

			"s3_buffer_size": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Removed:  "s3_buffer_size has been removed. Use a s3_configuration block instead. See https://terraform.io/docs/providers/aws/r/kinesis_firehose_delivery_stream.html",
			},

			"s3_buffer_interval": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Removed:  "s3_buffer_interval has been removed. Use a s3_configuration block instead. See https://terraform.io/docs/providers/aws/r/kinesis_firehose_delivery_stream.html",
			},

			"s3_data_compression": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Removed:  "s3_data_compression has been removed. Use a s3_configuration block instead. See https://terraform.io/docs/providers/aws/r/kinesis_firehose_delivery_stream.html",
			},

			"s3_configuration": &schema.Schema{
				Type:     schema.TypeList,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"bucket_arn": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"buffer_size": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							Default:  5,
						},

						"buffer_interval": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							Default:  300,
						},

						"compression_format": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Default:  "UNCOMPRESSED",
						},

						"kms_key_arn": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},

						"role_arn": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"prefix": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},

						"log_enabled": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},

						"log_group_name": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},

						"log_stream_name": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},

			"redshift_configuration": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"cluster_jdbcurl": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"username": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"password": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"role_arn": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"copy_options": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},

						"data_table_columns": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},

						"data_table_name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},

			"elasticsearch_configuration": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"buffering_interval": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							Default:  300,
							ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
								value := v.(int)
								if value < 60 || value > 900 {
									errors = append(errors, fmt.Errorf(
										"%q must be in the range from 60 to 900 seconds.", k))
								}
								return
							},
						},

						"buffering_size": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							Default:  5,
							ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
								value := v.(int)
								if value < 1 || value > 100 {
									errors = append(errors, fmt.Errorf(
										"%q must be in the range from 1 to 100 MB.", k))
								}
								return
							},
						},

						"domain_arn": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"index_name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"index_rotation_period": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Default:  "OneDay",
							ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
								value := v.(string)
								if value != "NoRotation" && value != "OneHour" && value != "OneDay" && value != "OneWeek" && value != "OneMonth" {
									errors = append(errors, fmt.Errorf(
										"%q must be one of 'NoRotation', 'OneHour', 'OneDay', 'OneWeek', 'OneMonth'", k))
								}
								return
							},
						},

						"retry_duration": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							Default:  300,
							ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
								value := v.(int)
								if value < 0 || value > 7200 {
									errors = append(errors, fmt.Errorf(
										"%q must be in the range from 0 to 7200 seconds.", k))
								}
								return
							},
						},

						"role_arn": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"s3_backup_mode": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Default:  "FailedDocumentsOnly",
							ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
								value := v.(string)
								if value != "FailedDocumentsOnly" && value != "AllDocuments" {
									errors = append(errors, fmt.Errorf(
										"%q must be one of 'FailedDocumentsOnly', 'AllDocuments'", k))
								}
								return
							},
						},

						"type_name": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
								value := v.(string)
								if len(value) > 100 {
									errors = append(errors, fmt.Errorf(
										"%q cannot be longer than 100 characters", k))
								}
								return
							},
						},
					},
				},
			},

			"arn": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"version_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"destination_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
		},
	}
}

func createS3Config(d *schema.ResourceData) *firehose.S3DestinationConfiguration {
	s3 := d.Get("s3_configuration").([]interface{})[0].(map[string]interface{})

	return &firehose.S3DestinationConfiguration{
		BucketARN: aws.String(s3["bucket_arn"].(string)),
		RoleARN:   aws.String(s3["role_arn"].(string)),
		BufferingHints: &firehose.BufferingHints{
			IntervalInSeconds: aws.Int64(int64(s3["buffer_interval"].(int))),
			SizeInMBs:         aws.Int64(int64(s3["buffer_size"].(int))),
		},
		CloudWatchLoggingOptions: &firehose.CloudWatchLoggingOptions{
			Enabled:       aws.Bool(s3["log_enabled"].(bool)),
			LogGroupName:  aws.String(s3["log_group_name"].(string)),
			LogStreamName: aws.String(s3["log_stream_name"].(string)),
		},
		Prefix:                  extractPrefixConfiguration(s3),
		CompressionFormat:       aws.String(s3["compression_format"].(string)),
		EncryptionConfiguration: extractEncryptionConfiguration(s3),
	}
}

func updateS3Config(d *schema.ResourceData) *firehose.S3DestinationUpdate {
	s3 := d.Get("s3_configuration").([]interface{})[0].(map[string]interface{})

	return &firehose.S3DestinationUpdate{
		BucketARN: aws.String(s3["bucket_arn"].(string)),
		RoleARN:   aws.String(s3["role_arn"].(string)),
		BufferingHints: &firehose.BufferingHints{
			IntervalInSeconds: aws.Int64((int64)(s3["buffer_interval"].(int))),
			SizeInMBs:         aws.Int64((int64)(s3["buffer_size"].(int))),
		},
		CloudWatchLoggingOptions: &firehose.CloudWatchLoggingOptions{
			Enabled:       aws.Bool(s3["log_enabled"].(bool)),
			LogGroupName:  aws.String(s3["log_group_name"].(string)),
			LogStreamName: aws.String(s3["log_stream_name"].(string)),
		},
		Prefix:                  extractPrefixConfiguration(s3),
		CompressionFormat:       aws.String(s3["compression_format"].(string)),
		EncryptionConfiguration: extractEncryptionConfiguration(s3),
	}
}

func extractEncryptionConfiguration(s3 map[string]interface{}) *firehose.EncryptionConfiguration {
	if key, ok := s3["kms_key_arn"]; ok && len(key.(string)) > 0 {
		return &firehose.EncryptionConfiguration{
			KMSEncryptionConfig: &firehose.KMSEncryptionConfig{
				AWSKMSKeyARN: aws.String(key.(string)),
			},
		}
	}

	return &firehose.EncryptionConfiguration{
		NoEncryptionConfig: aws.String("NoEncryption"),
	}
}

func extractPrefixConfiguration(s3 map[string]interface{}) *string {
	if v, ok := s3["prefix"]; ok {
		return aws.String(v.(string))
	}

	return nil
}

func createRedshiftConfig(d *schema.ResourceData, s3Config *firehose.S3DestinationConfiguration) (*firehose.RedshiftDestinationConfiguration, error) {
	redshiftRaw, ok := d.GetOk("redshift_configuration")
	if !ok {
		return nil, fmt.Errorf("[ERR] Error loading Redshift Configuration for Kinesis Firehose: redshift_configuration not found")
	}
	rl := redshiftRaw.([]interface{})

	redshift := rl[0].(map[string]interface{})

	return &firehose.RedshiftDestinationConfiguration{
		ClusterJDBCURL:  aws.String(redshift["cluster_jdbcurl"].(string)),
		Password:        aws.String(redshift["password"].(string)),
		Username:        aws.String(redshift["username"].(string)),
		RoleARN:         aws.String(redshift["role_arn"].(string)),
		CopyCommand:     extractCopyCommandConfiguration(redshift),
		S3Configuration: s3Config,
	}, nil
}

func updateRedshiftConfig(d *schema.ResourceData, s3Update *firehose.S3DestinationUpdate) (*firehose.RedshiftDestinationUpdate, error) {
	redshiftRaw, ok := d.GetOk("redshift_configuration")
	if !ok {
		return nil, fmt.Errorf("[ERR] Error loading Redshift Configuration for Kinesis Firehose: redshift_configuration not found")
	}
	rl := redshiftRaw.([]interface{})

	redshift := rl[0].(map[string]interface{})

	return &firehose.RedshiftDestinationUpdate{
		ClusterJDBCURL: aws.String(redshift["cluster_jdbcurl"].(string)),
		Password:       aws.String(redshift["password"].(string)),
		Username:       aws.String(redshift["username"].(string)),
		RoleARN:        aws.String(redshift["role_arn"].(string)),
		CopyCommand:    extractCopyCommandConfiguration(redshift),
		S3Update:       s3Update,
	}, nil
}

func createElasticsearchConfig(d *schema.ResourceData, s3Config *firehose.S3DestinationConfiguration) (*firehose.ElasticsearchDestinationConfiguration, error) {
	esConfig, ok := d.GetOk("elasticsearch_configuration")
	if !ok {
		return nil, fmt.Errorf("[ERR] Error loading Elasticsearch Configuration for Kinesis Firehose: elasticsearch_configuration not found")
	}
	esList := esConfig.([]interface{})

	es := esList[0].(map[string]interface{})

	config := &firehose.ElasticsearchDestinationConfiguration{
		BufferingHints:  extractBufferingHints(es),
		DomainARN:       aws.String(es["domain_arn"].(string)),
		IndexName:       aws.String(es["index_name"].(string)),
		RetryOptions:    extractRetryOptions(es),
		RoleARN:         aws.String(es["role_arn"].(string)),
		TypeName:        aws.String(es["type_name"].(string)),
		S3Configuration: s3Config,
	}

	if indexRotationPeriod, ok := es["index_rotation_period"]; ok {
		config.IndexRotationPeriod = aws.String(indexRotationPeriod.(string))
	}
	if s3BackupMode, ok := es["s3_backup_mode"]; ok {
		config.S3BackupMode = aws.String(s3BackupMode.(string))
	}

	return config, nil
}

func updateElasticsearchConfig(d *schema.ResourceData, s3Update *firehose.S3DestinationUpdate) (*firehose.ElasticsearchDestinationUpdate, error) {
	esConfig, ok := d.GetOk("elasticsearch_configuration")
	if !ok {
		return nil, fmt.Errorf("[ERR] Error loading Elasticsearch Configuration for Kinesis Firehose: elasticsearch_configuration not found")
	}
	esList := esConfig.([]interface{})

	es := esList[0].(map[string]interface{})

	update := &firehose.ElasticsearchDestinationUpdate{
		BufferingHints: extractBufferingHints(es),
		DomainARN:      aws.String(es["domain_arn"].(string)),
		IndexName:      aws.String(es["index_name"].(string)),
		RetryOptions:   extractRetryOptions(es),
		RoleARN:        aws.String(es["role_arn"].(string)),
		TypeName:       aws.String(es["type_name"].(string)),
		S3Update:       s3Update,
	}

	if indexRotationPeriod, ok := es["index_rotation_period"]; ok {
		update.IndexRotationPeriod = aws.String(indexRotationPeriod.(string))
	}

	return update, nil
}

func extractBufferingHints(es map[string]interface{}) *firehose.ElasticsearchBufferingHints {
	bufferingHints := &firehose.ElasticsearchBufferingHints{}

	if bufferingInterval, ok := es["buffering_hints"].(int); ok {
		bufferingHints.IntervalInSeconds = aws.Int64(int64(bufferingInterval))
	}
	if bufferingSize, ok := es["buffering_size"].(int); ok {
		bufferingHints.SizeInMBs = aws.Int64(int64(bufferingSize))
	}

	return bufferingHints

}

func extractRetryOptions(es map[string]interface{}) *firehose.ElasticsearchRetryOptions {
	retryOptions := &firehose.ElasticsearchRetryOptions{}

	if retryDuration, ok := es["retry_duration"].(int); ok {
		retryOptions.DurationInSeconds = aws.Int64(int64(retryDuration))
	}

	return retryOptions
}

func extractCopyCommandConfiguration(redshift map[string]interface{}) *firehose.CopyCommand {
	cmd := &firehose.CopyCommand{
		DataTableName: aws.String(redshift["data_table_name"].(string)),
	}
	if copyOptions, ok := redshift["copy_options"]; ok {
		cmd.CopyOptions = aws.String(copyOptions.(string))
	}
	if columns, ok := redshift["data_table_columns"]; ok {
		cmd.DataTableColumns = aws.String(columns.(string))
	}

	return cmd
}

func resourceAwsKinesisFirehoseDeliveryStreamCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).firehoseconn

	sn := d.Get("name").(string)
	s3Config := createS3Config(d)

	createInput := &firehose.CreateDeliveryStreamInput{
		DeliveryStreamName: aws.String(sn),
	}

	if d.Get("destination").(string) == "s3" {
		createInput.S3DestinationConfiguration = s3Config
	} else if d.Get("destination").(string) == "elasticsearch" {
		esConfig, err := createElasticsearchConfig(d, s3Config)
		if err != nil {
			return err
		}
		createInput.ElasticsearchDestinationConfiguration = esConfig
	} else {
		rc, err := createRedshiftConfig(d, s3Config)
		if err != nil {
			return err
		}
		createInput.RedshiftDestinationConfiguration = rc
	}

	var lastError error
	err := resource.Retry(1*time.Minute, func() *resource.RetryError {
		_, err := conn.CreateDeliveryStream(createInput)
		if err != nil {
			log.Printf("[DEBUG] Error creating Firehose Delivery Stream: %s", err)
			lastError = err

			if awsErr, ok := err.(awserr.Error); ok {
				// IAM roles can take ~10 seconds to propagate in AWS:
				// http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/iam-roles-for-amazon-ec2.html#launch-instance-with-role-console
				if awsErr.Code() == "InvalidArgumentException" && strings.Contains(awsErr.Message(), "Firehose is unable to assume role") {
					log.Printf("[DEBUG] Firehose could not assume role referenced, retrying...")
					return resource.RetryableError(awsErr)
				}
			}
			// Not retryable
			return resource.NonRetryableError(err)
		}

		return nil
	})
	if err != nil {
		if awsErr, ok := lastError.(awserr.Error); ok {
			return fmt.Errorf("[WARN] Error creating Kinesis Firehose Delivery Stream: \"%s\", code: \"%s\"", awsErr.Message(), awsErr.Code())
		}
		return err
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"CREATING"},
		Target:     []string{"ACTIVE"},
		Refresh:    firehoseStreamStateRefreshFunc(conn, sn),
		Timeout:    20 * time.Minute,
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

	return resourceAwsKinesisFirehoseDeliveryStreamRead(d, meta)
}

func resourceAwsKinesisFirehoseDeliveryStreamUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).firehoseconn

	sn := d.Get("name").(string)
	s3Config := updateS3Config(d)

	updateInput := &firehose.UpdateDestinationInput{
		DeliveryStreamName:             aws.String(sn),
		CurrentDeliveryStreamVersionId: aws.String(d.Get("version_id").(string)),
		DestinationId:                  aws.String(d.Get("destination_id").(string)),
	}

	if d.Get("destination").(string) == "s3" {
		updateInput.S3DestinationUpdate = s3Config
	} else if d.Get("destination").(string) == "elasticsearch" {
		esUpdate, err := updateElasticsearchConfig(d, s3Config)
		if err != nil {
			return err
		}
		updateInput.ElasticsearchDestinationUpdate = esUpdate
	} else {
		rc, err := updateRedshiftConfig(d, s3Config)
		if err != nil {
			return err
		}
		updateInput.RedshiftDestinationUpdate = rc
	}

	_, err := conn.UpdateDestination(updateInput)
	if err != nil {
		return fmt.Errorf(
			"Error Updating Kinesis Firehose Delivery Stream: \"%s\"\n%s",
			sn, err)
	}

	return resourceAwsKinesisFirehoseDeliveryStreamRead(d, meta)
}

func resourceAwsKinesisFirehoseDeliveryStreamRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).firehoseconn

	resp, err := conn.DescribeDeliveryStream(&firehose.DescribeDeliveryStreamInput{
		DeliveryStreamName: aws.String(d.Get("name").(string)),
	})

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
	d.Set("version_id", s.VersionId)
	d.Set("arn", *s.DeliveryStreamARN)
	if len(s.Destinations) > 0 {
		destination := s.Destinations[0]
		d.Set("destination_id", *destination.DestinationId)
	}

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
		Target:     []string{"DESTROYED"},
		Refresh:    firehoseStreamStateRefreshFunc(conn, sn),
		Timeout:    20 * time.Minute,
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
