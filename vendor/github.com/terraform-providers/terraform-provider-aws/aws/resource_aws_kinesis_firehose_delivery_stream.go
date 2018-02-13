package aws

import (
	"bytes"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/firehose"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func cloudWatchLoggingOptionsSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeSet,
		MaxItems: 1,
		Optional: true,
		Computed: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"enabled": {
					Type:     schema.TypeBool,
					Optional: true,
					Default:  false,
				},

				"log_group_name": {
					Type:     schema.TypeString,
					Optional: true,
				},

				"log_stream_name": {
					Type:     schema.TypeString,
					Optional: true,
				},
			},
		},
	}
}

func s3ConfigurationSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		MaxItems: 1,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"bucket_arn": {
					Type:     schema.TypeString,
					Required: true,
				},

				"buffer_size": {
					Type:     schema.TypeInt,
					Optional: true,
					Default:  5,
				},

				"buffer_interval": {
					Type:     schema.TypeInt,
					Optional: true,
					Default:  300,
				},

				"compression_format": {
					Type:     schema.TypeString,
					Optional: true,
					Default:  "UNCOMPRESSED",
				},

				"kms_key_arn": {
					Type:         schema.TypeString,
					Optional:     true,
					ValidateFunc: validateArn,
				},

				"role_arn": {
					Type:     schema.TypeString,
					Required: true,
				},

				"prefix": {
					Type:     schema.TypeString,
					Optional: true,
				},

				"cloudwatch_logging_options": cloudWatchLoggingOptionsSchema(),
			},
		},
	}
}

func processingConfigurationSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		MaxItems: 1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"enabled": {
					Type:     schema.TypeBool,
					Optional: true,
				},
				"processors": {
					Type:     schema.TypeList,
					Optional: true,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"parameters": {
								Type:     schema.TypeList,
								Optional: true,
								Elem: &schema.Resource{
									Schema: map[string]*schema.Schema{
										"parameter_name": {
											Type:     schema.TypeString,
											Required: true,
											ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
												value := v.(string)
												if value != "LambdaArn" && value != "NumberOfRetries" {
													errors = append(errors, fmt.Errorf(
														"%q must be one of 'LambdaArn', 'NumberOfRetries'", k))
												}
												return
											},
										},
										"parameter_value": {
											Type:     schema.TypeString,
											Required: true,
											ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
												value := v.(string)
												if len(value) < 1 || len(value) > 512 {
													errors = append(errors, fmt.Errorf(
														"%q must be at least one character long and at most 512 characters long", k))
												}
												return
											},
										},
									},
								},
							},
							"type": {
								Type:     schema.TypeString,
								Required: true,
								ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
									value := v.(string)
									if value != "Lambda" {
										errors = append(errors, fmt.Errorf(
											"%q must be 'Lambda'", k))
									}
									return
								},
							},
						},
					},
				},
			},
		},
	}
}

func cloudwatchLoggingOptionsHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%t-", m["enabled"].(bool)))
	if m["enabled"].(bool) {
		buf.WriteString(fmt.Sprintf("%s-", m["log_group_name"].(string)))
		buf.WriteString(fmt.Sprintf("%s-", m["log_stream_name"].(string)))
	}
	return hashcode.String(buf.String())
}

func flattenCloudwatchLoggingOptions(clo firehose.CloudWatchLoggingOptions) *schema.Set {
	cloudwatchLoggingOptions := map[string]interface{}{
		"enabled": *clo.Enabled,
	}
	if *clo.Enabled {
		cloudwatchLoggingOptions["log_group_name"] = *clo.LogGroupName
		cloudwatchLoggingOptions["log_stream_name"] = *clo.LogStreamName
	}
	return schema.NewSet(cloudwatchLoggingOptionsHash, []interface{}{cloudwatchLoggingOptions})
}

func flattenFirehoseS3Configuration(s3 firehose.S3DestinationDescription) []interface{} {
	s3Configuration := map[string]interface{}{
		"role_arn":           *s3.RoleARN,
		"bucket_arn":         *s3.BucketARN,
		"buffer_size":        *s3.BufferingHints.SizeInMBs,
		"buffer_interval":    *s3.BufferingHints.IntervalInSeconds,
		"compression_format": *s3.CompressionFormat,
	}
	if s3.CloudWatchLoggingOptions != nil {
		s3Configuration["cloudwatch_logging_options"] = flattenCloudwatchLoggingOptions(*s3.CloudWatchLoggingOptions)
	}
	if s3.EncryptionConfiguration.KMSEncryptionConfig != nil {
		s3Configuration["kms_key_arn"] = *s3.EncryptionConfiguration.KMSEncryptionConfig.AWSKMSKeyARN
	}
	if s3.Prefix != nil {
		s3Configuration["prefix"] = *s3.Prefix
	}
	return []interface{}{s3Configuration}
}

func flattenProcessingConfiguration(pc firehose.ProcessingConfiguration, roleArn string) []map[string]interface{} {
	processingConfiguration := make([]map[string]interface{}, 1)

	// It is necessary to explicitely filter this out
	// to prevent diffs during routine use and retain the ability
	// to show diffs if any field has drifted
	defaultLambdaParams := map[string]string{
		"NumberOfRetries":         "3",
		"RoleArn":                 roleArn,
		"BufferSizeInMBs":         "3",
		"BufferIntervalInSeconds": "60",
	}

	processors := make([]interface{}, len(pc.Processors), len(pc.Processors))
	for i, p := range pc.Processors {
		t := *p.Type
		parameters := make([]interface{}, 0)

		for _, params := range p.Parameters {
			name, value := *params.ParameterName, *params.ParameterValue

			if t == firehose.ProcessorTypeLambda {
				// Ignore defaults
				if v, ok := defaultLambdaParams[name]; ok && v == value {
					continue
				}
			}

			parameters = append(parameters, map[string]interface{}{
				"parameter_name":  name,
				"parameter_value": value,
			})
		}

		processors[i] = map[string]interface{}{
			"type":       t,
			"parameters": parameters,
		}
	}
	processingConfiguration[0] = map[string]interface{}{
		"enabled":    *pc.Enabled,
		"processors": processors,
	}
	return processingConfiguration
}

func flattenKinesisFirehoseDeliveryStream(d *schema.ResourceData, s *firehose.DeliveryStreamDescription) error {
	d.Set("version_id", s.VersionId)
	d.Set("arn", *s.DeliveryStreamARN)
	d.Set("name", s.DeliveryStreamName)
	if len(s.Destinations) > 0 {
		destination := s.Destinations[0]
		if destination.RedshiftDestinationDescription != nil {
			d.Set("destination", "redshift")
			password := d.Get("redshift_configuration.0.password").(string)

			redshiftConfiguration := map[string]interface{}{
				"cluster_jdbcurl":    *destination.RedshiftDestinationDescription.ClusterJDBCURL,
				"role_arn":           *destination.RedshiftDestinationDescription.RoleARN,
				"username":           *destination.RedshiftDestinationDescription.Username,
				"password":           password,
				"data_table_name":    *destination.RedshiftDestinationDescription.CopyCommand.DataTableName,
				"copy_options":       *destination.RedshiftDestinationDescription.CopyCommand.CopyOptions,
				"data_table_columns": *destination.RedshiftDestinationDescription.CopyCommand.DataTableColumns,
				"s3_backup_mode":     *destination.RedshiftDestinationDescription.S3BackupMode,
				"retry_duration":     *destination.RedshiftDestinationDescription.RetryOptions.DurationInSeconds,
			}

			if v := destination.RedshiftDestinationDescription.CloudWatchLoggingOptions; v != nil {
				redshiftConfiguration["cloudwatch_logging_options"] = flattenCloudwatchLoggingOptions(*v)
			}

			if v := destination.RedshiftDestinationDescription.S3BackupDescription; v != nil {
				redshiftConfiguration["s3_backup_configuration"] = flattenFirehoseS3Configuration(*v)
			}

			redshiftConfList := make([]map[string]interface{}, 1)
			redshiftConfList[0] = redshiftConfiguration
			d.Set("redshift_configuration", redshiftConfList)
			d.Set("s3_configuration", flattenFirehoseS3Configuration(*destination.RedshiftDestinationDescription.S3DestinationDescription))

		} else if destination.ElasticsearchDestinationDescription != nil {
			d.Set("destination", "elasticsearch")

			elasticsearchConfiguration := map[string]interface{}{
				"buffering_interval":    *destination.ElasticsearchDestinationDescription.BufferingHints.IntervalInSeconds,
				"buffering_size":        *destination.ElasticsearchDestinationDescription.BufferingHints.SizeInMBs,
				"domain_arn":            *destination.ElasticsearchDestinationDescription.DomainARN,
				"role_arn":              *destination.ElasticsearchDestinationDescription.RoleARN,
				"type_name":             *destination.ElasticsearchDestinationDescription.TypeName,
				"index_name":            *destination.ElasticsearchDestinationDescription.IndexName,
				"s3_backup_mode":        *destination.ElasticsearchDestinationDescription.S3BackupMode,
				"retry_duration":        *destination.ElasticsearchDestinationDescription.RetryOptions.DurationInSeconds,
				"index_rotation_period": *destination.ElasticsearchDestinationDescription.IndexRotationPeriod,
			}

			if v := destination.ElasticsearchDestinationDescription.CloudWatchLoggingOptions; v != nil {
				elasticsearchConfiguration["cloudwatch_logging_options"] = flattenCloudwatchLoggingOptions(*v)
			}

			elasticsearchConfList := make([]map[string]interface{}, 1)
			elasticsearchConfList[0] = elasticsearchConfiguration
			d.Set("elasticsearch_configuration", elasticsearchConfList)
			d.Set("s3_configuration", flattenFirehoseS3Configuration(*destination.ElasticsearchDestinationDescription.S3DestinationDescription))
		} else if destination.SplunkDestinationDescription != nil {
			d.Set("destination", "splunk")

			splunkConfiguration := map[string]interface{}{
				"hec_acknowledgment_timeout": *destination.SplunkDestinationDescription.HECAcknowledgmentTimeoutInSeconds,
				"hec_endpoint":               *destination.SplunkDestinationDescription.HECEndpoint,
				"hec_endpoint_type":          *destination.SplunkDestinationDescription.HECEndpointType,
				"hec_token":                  *destination.SplunkDestinationDescription.HECToken,
				"s3_backup_mode":             *destination.SplunkDestinationDescription.S3BackupMode,
				"retry_duration":             *destination.SplunkDestinationDescription.RetryOptions.DurationInSeconds,
			}

			if v := destination.SplunkDestinationDescription.CloudWatchLoggingOptions; v != nil {
				splunkConfiguration["cloudwatch_logging_options"] = flattenCloudwatchLoggingOptions(*v)
			}

			splunkConfList := make([]map[string]interface{}, 1)
			splunkConfList[0] = splunkConfiguration
			d.Set("splunk_configuration", splunkConfList)
			d.Set("s3_configuration", flattenFirehoseS3Configuration(*destination.SplunkDestinationDescription.S3DestinationDescription))
		} else if d.Get("destination").(string) == "s3" {
			d.Set("destination", "s3")
			d.Set("s3_configuration", flattenFirehoseS3Configuration(*destination.S3DestinationDescription))
		} else {
			d.Set("destination", "extended_s3")

			roleArn := *destination.ExtendedS3DestinationDescription.RoleARN
			extendedS3Configuration := map[string]interface{}{
				"buffer_interval":            *destination.ExtendedS3DestinationDescription.BufferingHints.IntervalInSeconds,
				"buffer_size":                *destination.ExtendedS3DestinationDescription.BufferingHints.SizeInMBs,
				"bucket_arn":                 *destination.ExtendedS3DestinationDescription.BucketARN,
				"role_arn":                   roleArn,
				"compression_format":         *destination.ExtendedS3DestinationDescription.CompressionFormat,
				"prefix":                     *destination.ExtendedS3DestinationDescription.Prefix,
				"cloudwatch_logging_options": flattenCloudwatchLoggingOptions(*destination.ExtendedS3DestinationDescription.CloudWatchLoggingOptions),
			}

			if v := destination.ExtendedS3DestinationDescription.EncryptionConfiguration.KMSEncryptionConfig; v != nil {
				extendedS3Configuration["kms_key_arn"] = *v.AWSKMSKeyARN
			}

			if v := destination.ExtendedS3DestinationDescription.ProcessingConfiguration; v != nil {
				extendedS3Configuration["processing_configuration"] = flattenProcessingConfiguration(*v, roleArn)
			}

			extendedS3ConfList := make([]map[string]interface{}, 1)
			extendedS3ConfList[0] = extendedS3Configuration

			err := d.Set("extended_s3_configuration", extendedS3ConfList)
			if err != nil {
				return err
			}
		}
		d.Set("destination_id", *destination.DestinationId)
	}
	return nil
}

func resourceAwsKinesisFirehoseDeliveryStream() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsKinesisFirehoseDeliveryStreamCreate,
		Read:   resourceAwsKinesisFirehoseDeliveryStreamRead,
		Update: resourceAwsKinesisFirehoseDeliveryStreamUpdate,
		Delete: resourceAwsKinesisFirehoseDeliveryStreamDelete,

		Importer: &schema.ResourceImporter{
			State: func(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				resARN, err := arn.Parse(d.Id())
				if err != nil {
					return nil, err
				}
				d.Set("name", strings.Split(resARN.Resource, "/")[1])
				return []*schema.ResourceData{d}, nil
			},
		},

		SchemaVersion: 1,
		MigrateState:  resourceAwsKinesisFirehoseMigrateState,
		Schema: map[string]*schema.Schema{
			"name": {
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

			"kinesis_source_configuration": {
				Type:     schema.TypeList,
				ForceNew: true,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"kinesis_stream_arn": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateArn,
						},

						"role_arn": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateArn,
						},
					},
				},
			},

			"destination": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				StateFunc: func(v interface{}) string {
					value := v.(string)
					return strings.ToLower(value)
				},
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(string)
					if value != "s3" && value != "extended_s3" && value != "redshift" && value != "elasticsearch" && value != "splunk" {
						errors = append(errors, fmt.Errorf(
							"%q must be one of 's3', 'extended_s3', 'redshift', 'elasticsearch', 'splunk'", k))
					}
					return
				},
			},

			"s3_configuration": s3ConfigurationSchema(),

			"extended_s3_configuration": {
				Type:          schema.TypeList,
				Optional:      true,
				ConflictsWith: []string{"s3_configuration"},
				MaxItems:      1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"bucket_arn": {
							Type:     schema.TypeString,
							Required: true,
						},

						"buffer_size": {
							Type:     schema.TypeInt,
							Optional: true,
							Default:  5,
						},

						"buffer_interval": {
							Type:     schema.TypeInt,
							Optional: true,
							Default:  300,
						},

						"compression_format": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "UNCOMPRESSED",
						},

						"kms_key_arn": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validateArn,
						},

						"role_arn": {
							Type:     schema.TypeString,
							Required: true,
						},

						"prefix": {
							Type:     schema.TypeString,
							Optional: true,
						},

						"cloudwatch_logging_options": cloudWatchLoggingOptionsSchema(),

						"processing_configuration": processingConfigurationSchema(),
					},
				},
			},

			"redshift_configuration": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"cluster_jdbcurl": {
							Type:     schema.TypeString,
							Required: true,
						},

						"username": {
							Type:     schema.TypeString,
							Required: true,
						},

						"password": {
							Type:      schema.TypeString,
							Required:  true,
							Sensitive: true,
						},

						"role_arn": {
							Type:     schema.TypeString,
							Required: true,
						},

						"s3_backup_mode": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "Disabled",
							ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
								value := v.(string)
								if value != "Disabled" && value != "Enabled" {
									errors = append(errors, fmt.Errorf(
										"%q must be one of 'Disabled', 'Enabled'", k))
								}
								return
							},
						},

						"s3_backup_configuration": s3ConfigurationSchema(),

						"retry_duration": {
							Type:     schema.TypeInt,
							Optional: true,
							Default:  3600,
							ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
								value := v.(int)
								if value < 0 || value > 7200 {
									errors = append(errors, fmt.Errorf(
										"%q must be in the range from 0 to 7200 seconds.", k))
								}
								return
							},
						},

						"copy_options": {
							Type:     schema.TypeString,
							Optional: true,
						},

						"data_table_columns": {
							Type:     schema.TypeString,
							Optional: true,
						},

						"data_table_name": {
							Type:     schema.TypeString,
							Required: true,
						},

						"cloudwatch_logging_options": cloudWatchLoggingOptionsSchema(),
					},
				},
			},

			"elasticsearch_configuration": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"buffering_interval": {
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

						"buffering_size": {
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

						"domain_arn": {
							Type:     schema.TypeString,
							Required: true,
						},

						"index_name": {
							Type:     schema.TypeString,
							Required: true,
						},

						"index_rotation_period": {
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

						"retry_duration": {
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

						"role_arn": {
							Type:     schema.TypeString,
							Required: true,
						},

						"s3_backup_mode": {
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

						"type_name": {
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

						"cloudwatch_logging_options": cloudWatchLoggingOptionsSchema(),
					},
				},
			},

			"splunk_configuration": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"hec_acknowledgment_timeout": {
							Type:         schema.TypeInt,
							Optional:     true,
							Default:      180,
							ValidateFunc: validateIntegerInRange(180, 600),
						},

						"hec_endpoint": {
							Type:     schema.TypeString,
							Required: true,
						},

						"hec_endpoint_type": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  firehose.HECEndpointTypeRaw,
							ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
								value := v.(string)
								if value != firehose.HECEndpointTypeRaw && value != firehose.HECEndpointTypeEvent {
									errors = append(errors, fmt.Errorf(
										"%q must be one of 'Raw', 'Event'", k))
								}
								return
							},
						},

						"hec_token": {
							Type:     schema.TypeString,
							Required: true,
						},

						"s3_backup_mode": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  firehose.SplunkS3BackupModeFailedEventsOnly,
							ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
								value := v.(string)
								if value != firehose.SplunkS3BackupModeFailedEventsOnly && value != firehose.SplunkS3BackupModeAllEvents {
									errors = append(errors, fmt.Errorf(
										"%q must be one of 'FailedEventsOnly', 'AllEvents'", k))
								}
								return
							},
						},

						"retry_duration": {
							Type:         schema.TypeInt,
							Optional:     true,
							Default:      3600,
							ValidateFunc: validateIntegerInRange(0, 7200),
						},

						"cloudwatch_logging_options": cloudWatchLoggingOptionsSchema(),

						"processing_configuration": processingConfigurationSchema(),
					},
				},
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

			"destination_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
		},
	}
}

func createSourceConfig(source map[string]interface{}) *firehose.KinesisStreamSourceConfiguration {

	configuration := &firehose.KinesisStreamSourceConfiguration{
		KinesisStreamARN: aws.String(source["kinesis_stream_arn"].(string)),
		RoleARN:          aws.String(source["role_arn"].(string)),
	}

	return configuration
}

func createS3Config(d *schema.ResourceData) *firehose.S3DestinationConfiguration {
	s3 := d.Get("s3_configuration").([]interface{})[0].(map[string]interface{})

	configuration := &firehose.S3DestinationConfiguration{
		BucketARN: aws.String(s3["bucket_arn"].(string)),
		RoleARN:   aws.String(s3["role_arn"].(string)),
		BufferingHints: &firehose.BufferingHints{
			IntervalInSeconds: aws.Int64(int64(s3["buffer_interval"].(int))),
			SizeInMBs:         aws.Int64(int64(s3["buffer_size"].(int))),
		},
		Prefix:                  extractPrefixConfiguration(s3),
		CompressionFormat:       aws.String(s3["compression_format"].(string)),
		EncryptionConfiguration: extractEncryptionConfiguration(s3),
	}

	if _, ok := s3["cloudwatch_logging_options"]; ok {
		configuration.CloudWatchLoggingOptions = extractCloudWatchLoggingConfiguration(s3)
	}

	return configuration
}

func expandS3BackupConfig(d map[string]interface{}) *firehose.S3DestinationConfiguration {
	config := d["s3_backup_configuration"].([]interface{})
	if len(config) == 0 {
		return nil
	}

	s3 := config[0].(map[string]interface{})

	configuration := &firehose.S3DestinationConfiguration{
		BucketARN: aws.String(s3["bucket_arn"].(string)),
		RoleARN:   aws.String(s3["role_arn"].(string)),
		BufferingHints: &firehose.BufferingHints{
			IntervalInSeconds: aws.Int64(int64(s3["buffer_interval"].(int))),
			SizeInMBs:         aws.Int64(int64(s3["buffer_size"].(int))),
		},
		Prefix:                  extractPrefixConfiguration(s3),
		CompressionFormat:       aws.String(s3["compression_format"].(string)),
		EncryptionConfiguration: extractEncryptionConfiguration(s3),
	}

	if _, ok := s3["cloudwatch_logging_options"]; ok {
		configuration.CloudWatchLoggingOptions = extractCloudWatchLoggingConfiguration(s3)
	}

	return configuration
}

func createExtendedS3Config(d *schema.ResourceData) *firehose.ExtendedS3DestinationConfiguration {
	s3 := d.Get("extended_s3_configuration").([]interface{})[0].(map[string]interface{})

	configuration := &firehose.ExtendedS3DestinationConfiguration{
		BucketARN: aws.String(s3["bucket_arn"].(string)),
		RoleARN:   aws.String(s3["role_arn"].(string)),
		BufferingHints: &firehose.BufferingHints{
			IntervalInSeconds: aws.Int64(int64(s3["buffer_interval"].(int))),
			SizeInMBs:         aws.Int64(int64(s3["buffer_size"].(int))),
		},
		Prefix:                  extractPrefixConfiguration(s3),
		CompressionFormat:       aws.String(s3["compression_format"].(string)),
		EncryptionConfiguration: extractEncryptionConfiguration(s3),
	}

	if _, ok := s3["processing_configuration"]; ok {
		configuration.ProcessingConfiguration = extractProcessingConfiguration(s3)
	}

	if _, ok := s3["cloudwatch_logging_options"]; ok {
		configuration.CloudWatchLoggingOptions = extractCloudWatchLoggingConfiguration(s3)
	}

	return configuration
}

func updateS3Config(d *schema.ResourceData) *firehose.S3DestinationUpdate {
	s3 := d.Get("s3_configuration").([]interface{})[0].(map[string]interface{})

	configuration := &firehose.S3DestinationUpdate{
		BucketARN: aws.String(s3["bucket_arn"].(string)),
		RoleARN:   aws.String(s3["role_arn"].(string)),
		BufferingHints: &firehose.BufferingHints{
			IntervalInSeconds: aws.Int64((int64)(s3["buffer_interval"].(int))),
			SizeInMBs:         aws.Int64((int64)(s3["buffer_size"].(int))),
		},
		Prefix:                   extractPrefixConfiguration(s3),
		CompressionFormat:        aws.String(s3["compression_format"].(string)),
		EncryptionConfiguration:  extractEncryptionConfiguration(s3),
		CloudWatchLoggingOptions: extractCloudWatchLoggingConfiguration(s3),
	}

	if _, ok := s3["cloudwatch_logging_options"]; ok {
		configuration.CloudWatchLoggingOptions = extractCloudWatchLoggingConfiguration(s3)
	}

	return configuration
}

func updateS3BackupConfig(d map[string]interface{}) *firehose.S3DestinationUpdate {
	config := d["s3_backup_configuration"].([]interface{})
	if len(config) == 0 {
		return nil
	}

	s3 := config[0].(map[string]interface{})

	configuration := &firehose.S3DestinationUpdate{
		BucketARN: aws.String(s3["bucket_arn"].(string)),
		RoleARN:   aws.String(s3["role_arn"].(string)),
		BufferingHints: &firehose.BufferingHints{
			IntervalInSeconds: aws.Int64((int64)(s3["buffer_interval"].(int))),
			SizeInMBs:         aws.Int64((int64)(s3["buffer_size"].(int))),
		},
		Prefix:                   extractPrefixConfiguration(s3),
		CompressionFormat:        aws.String(s3["compression_format"].(string)),
		EncryptionConfiguration:  extractEncryptionConfiguration(s3),
		CloudWatchLoggingOptions: extractCloudWatchLoggingConfiguration(s3),
	}

	if _, ok := s3["cloudwatch_logging_options"]; ok {
		configuration.CloudWatchLoggingOptions = extractCloudWatchLoggingConfiguration(s3)
	}

	return configuration
}

func updateExtendedS3Config(d *schema.ResourceData) *firehose.ExtendedS3DestinationUpdate {
	s3 := d.Get("extended_s3_configuration").([]interface{})[0].(map[string]interface{})

	configuration := &firehose.ExtendedS3DestinationUpdate{
		BucketARN: aws.String(s3["bucket_arn"].(string)),
		RoleARN:   aws.String(s3["role_arn"].(string)),
		BufferingHints: &firehose.BufferingHints{
			IntervalInSeconds: aws.Int64((int64)(s3["buffer_interval"].(int))),
			SizeInMBs:         aws.Int64((int64)(s3["buffer_size"].(int))),
		},
		Prefix:                   extractPrefixConfiguration(s3),
		CompressionFormat:        aws.String(s3["compression_format"].(string)),
		EncryptionConfiguration:  extractEncryptionConfiguration(s3),
		CloudWatchLoggingOptions: extractCloudWatchLoggingConfiguration(s3),
		ProcessingConfiguration:  extractProcessingConfiguration(s3),
	}

	if _, ok := s3["cloudwatch_logging_options"]; ok {
		configuration.CloudWatchLoggingOptions = extractCloudWatchLoggingConfiguration(s3)
	}

	return configuration
}

func extractProcessingConfiguration(s3 map[string]interface{}) *firehose.ProcessingConfiguration {
	config := s3["processing_configuration"].([]interface{})
	if len(config) == 0 {
		return nil
	}

	processingConfiguration := config[0].(map[string]interface{})

	return &firehose.ProcessingConfiguration{
		Enabled:    aws.Bool(processingConfiguration["enabled"].(bool)),
		Processors: extractProcessors(processingConfiguration["processors"].([]interface{})),
	}
}

func extractProcessors(processingConfigurationProcessors []interface{}) []*firehose.Processor {
	processors := []*firehose.Processor{}

	for _, processor := range processingConfigurationProcessors {
		processors = append(processors, extractProcessor(processor.(map[string]interface{})))
	}

	return processors
}

func extractProcessor(processingConfigurationProcessor map[string]interface{}) *firehose.Processor {
	return &firehose.Processor{
		Type:       aws.String(processingConfigurationProcessor["type"].(string)),
		Parameters: extractProcessorParameters(processingConfigurationProcessor["parameters"].([]interface{})),
	}
}

func extractProcessorParameters(processorParameters []interface{}) []*firehose.ProcessorParameter {
	parameters := []*firehose.ProcessorParameter{}

	for _, attr := range processorParameters {
		parameters = append(parameters, extractProcessorParameter(attr.(map[string]interface{})))
	}

	return parameters
}

func extractProcessorParameter(processorParameter map[string]interface{}) *firehose.ProcessorParameter {
	parameter := &firehose.ProcessorParameter{
		ParameterName:  aws.String(processorParameter["parameter_name"].(string)),
		ParameterValue: aws.String(processorParameter["parameter_value"].(string)),
	}

	return parameter
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

func extractCloudWatchLoggingConfiguration(s3 map[string]interface{}) *firehose.CloudWatchLoggingOptions {
	config := s3["cloudwatch_logging_options"].(*schema.Set).List()
	if len(config) == 0 {
		return nil
	}

	loggingConfig := config[0].(map[string]interface{})
	loggingOptions := &firehose.CloudWatchLoggingOptions{
		Enabled: aws.Bool(loggingConfig["enabled"].(bool)),
	}

	if v, ok := loggingConfig["log_group_name"]; ok {
		loggingOptions.LogGroupName = aws.String(v.(string))
	}

	if v, ok := loggingConfig["log_stream_name"]; ok {
		loggingOptions.LogStreamName = aws.String(v.(string))
	}

	return loggingOptions

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

	configuration := &firehose.RedshiftDestinationConfiguration{
		ClusterJDBCURL:  aws.String(redshift["cluster_jdbcurl"].(string)),
		RetryOptions:    extractRedshiftRetryOptions(redshift),
		Password:        aws.String(redshift["password"].(string)),
		Username:        aws.String(redshift["username"].(string)),
		RoleARN:         aws.String(redshift["role_arn"].(string)),
		CopyCommand:     extractCopyCommandConfiguration(redshift),
		S3Configuration: s3Config,
	}

	if _, ok := redshift["cloudwatch_logging_options"]; ok {
		configuration.CloudWatchLoggingOptions = extractCloudWatchLoggingConfiguration(redshift)
	}
	if s3BackupMode, ok := redshift["s3_backup_mode"]; ok {
		configuration.S3BackupMode = aws.String(s3BackupMode.(string))
		configuration.S3BackupConfiguration = expandS3BackupConfig(d.Get("redshift_configuration").([]interface{})[0].(map[string]interface{}))
	}

	return configuration, nil
}

func updateRedshiftConfig(d *schema.ResourceData, s3Update *firehose.S3DestinationUpdate) (*firehose.RedshiftDestinationUpdate, error) {
	redshiftRaw, ok := d.GetOk("redshift_configuration")
	if !ok {
		return nil, fmt.Errorf("[ERR] Error loading Redshift Configuration for Kinesis Firehose: redshift_configuration not found")
	}
	rl := redshiftRaw.([]interface{})

	redshift := rl[0].(map[string]interface{})

	configuration := &firehose.RedshiftDestinationUpdate{
		ClusterJDBCURL: aws.String(redshift["cluster_jdbcurl"].(string)),
		RetryOptions:   extractRedshiftRetryOptions(redshift),
		Password:       aws.String(redshift["password"].(string)),
		Username:       aws.String(redshift["username"].(string)),
		RoleARN:        aws.String(redshift["role_arn"].(string)),
		CopyCommand:    extractCopyCommandConfiguration(redshift),
		S3Update:       s3Update,
	}

	if _, ok := redshift["cloudwatch_logging_options"]; ok {
		configuration.CloudWatchLoggingOptions = extractCloudWatchLoggingConfiguration(redshift)
	}
	if s3BackupMode, ok := redshift["s3_backup_mode"]; ok {
		configuration.S3BackupMode = aws.String(s3BackupMode.(string))
		configuration.S3BackupUpdate = updateS3BackupConfig(d.Get("redshift_configuration").([]interface{})[0].(map[string]interface{}))
	}

	return configuration, nil
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
		RetryOptions:    extractElasticSearchRetryOptions(es),
		RoleARN:         aws.String(es["role_arn"].(string)),
		TypeName:        aws.String(es["type_name"].(string)),
		S3Configuration: s3Config,
	}

	if _, ok := es["cloudwatch_logging_options"]; ok {
		config.CloudWatchLoggingOptions = extractCloudWatchLoggingConfiguration(es)
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
		RetryOptions:   extractElasticSearchRetryOptions(es),
		RoleARN:        aws.String(es["role_arn"].(string)),
		TypeName:       aws.String(es["type_name"].(string)),
		S3Update:       s3Update,
	}

	if _, ok := es["cloudwatch_logging_options"]; ok {
		update.CloudWatchLoggingOptions = extractCloudWatchLoggingConfiguration(es)
	}

	if indexRotationPeriod, ok := es["index_rotation_period"]; ok {
		update.IndexRotationPeriod = aws.String(indexRotationPeriod.(string))
	}

	return update, nil
}

func createSplunkConfig(d *schema.ResourceData, s3Config *firehose.S3DestinationConfiguration) (*firehose.SplunkDestinationConfiguration, error) {
	splunkRaw, ok := d.GetOk("splunk_configuration")
	if !ok {
		return nil, fmt.Errorf("[ERR] Error loading Splunk Configuration for Kinesis Firehose: splunk_configuration not found")
	}
	sl := splunkRaw.([]interface{})

	splunk := sl[0].(map[string]interface{})

	configuration := &firehose.SplunkDestinationConfiguration{
		HECToken:                          aws.String(splunk["hec_token"].(string)),
		HECEndpointType:                   aws.String(splunk["hec_endpoint_type"].(string)),
		HECEndpoint:                       aws.String(splunk["hec_endpoint"].(string)),
		HECAcknowledgmentTimeoutInSeconds: aws.Int64(int64(splunk["hec_acknowledgment_timeout"].(int))),
		RetryOptions:                      extractSplunkRetryOptions(splunk),
		S3Configuration:                   s3Config,
	}

	if _, ok := splunk["cloudwatch_logging_options"]; ok {
		configuration.CloudWatchLoggingOptions = extractCloudWatchLoggingConfiguration(splunk)
	}
	if s3BackupMode, ok := splunk["s3_backup_mode"]; ok {
		configuration.S3BackupMode = aws.String(s3BackupMode.(string))
	}

	return configuration, nil
}

func updateSplunkConfig(d *schema.ResourceData, s3Update *firehose.S3DestinationUpdate) (*firehose.SplunkDestinationUpdate, error) {
	splunkRaw, ok := d.GetOk("splunk_configuration")
	if !ok {
		return nil, fmt.Errorf("[ERR] Error loading Splunk Configuration for Kinesis Firehose: splunk_configuration not found")
	}
	sl := splunkRaw.([]interface{})

	splunk := sl[0].(map[string]interface{})

	configuration := &firehose.SplunkDestinationUpdate{
		HECToken:                          aws.String(splunk["hec_token"].(string)),
		HECEndpointType:                   aws.String(splunk["hec_endpoint_type"].(string)),
		HECEndpoint:                       aws.String(splunk["hec_endpoint"].(string)),
		HECAcknowledgmentTimeoutInSeconds: aws.Int64(int64(splunk["hec_acknowledgment_timeout"].(int))),
		RetryOptions:                      extractSplunkRetryOptions(splunk),
		S3Update:                          s3Update,
	}

	if _, ok := splunk["cloudwatch_logging_options"]; ok {
		configuration.CloudWatchLoggingOptions = extractCloudWatchLoggingConfiguration(splunk)
	}
	if s3BackupMode, ok := splunk["s3_backup_mode"]; ok {
		configuration.S3BackupMode = aws.String(s3BackupMode.(string))
	}

	return configuration, nil
}

func extractBufferingHints(es map[string]interface{}) *firehose.ElasticsearchBufferingHints {
	bufferingHints := &firehose.ElasticsearchBufferingHints{}

	if bufferingInterval, ok := es["buffering_interval"].(int); ok {
		bufferingHints.IntervalInSeconds = aws.Int64(int64(bufferingInterval))
	}
	if bufferingSize, ok := es["buffering_size"].(int); ok {
		bufferingHints.SizeInMBs = aws.Int64(int64(bufferingSize))
	}

	return bufferingHints
}

func extractElasticSearchRetryOptions(es map[string]interface{}) *firehose.ElasticsearchRetryOptions {
	retryOptions := &firehose.ElasticsearchRetryOptions{}

	if retryDuration, ok := es["retry_duration"].(int); ok {
		retryOptions.DurationInSeconds = aws.Int64(int64(retryDuration))
	}

	return retryOptions
}

func extractRedshiftRetryOptions(redshift map[string]interface{}) *firehose.RedshiftRetryOptions {
	retryOptions := &firehose.RedshiftRetryOptions{}

	if retryDuration, ok := redshift["retry_duration"].(int); ok {
		retryOptions.DurationInSeconds = aws.Int64(int64(retryDuration))
	}

	return retryOptions
}

func extractSplunkRetryOptions(splunk map[string]interface{}) *firehose.SplunkRetryOptions {
	retryOptions := &firehose.SplunkRetryOptions{}

	if retryDuration, ok := splunk["retry_duration"].(int); ok {
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
	validateError := validateAwsKinesisFirehoseSchema(d)

	if validateError != nil {
		return validateError
	}

	conn := meta.(*AWSClient).firehoseconn

	sn := d.Get("name").(string)

	createInput := &firehose.CreateDeliveryStreamInput{
		DeliveryStreamName: aws.String(sn),
	}

	if v, ok := d.GetOk("kinesis_source_configuration"); ok {
		sourceConfig := createSourceConfig(v.([]interface{})[0].(map[string]interface{}))
		createInput.KinesisStreamSourceConfiguration = sourceConfig
		createInput.DeliveryStreamType = aws.String(firehose.DeliveryStreamTypeKinesisStreamAsSource)
	} else {
		createInput.DeliveryStreamType = aws.String(firehose.DeliveryStreamTypeDirectPut)
	}

	if d.Get("destination").(string) == "extended_s3" {
		extendedS3Config := createExtendedS3Config(d)
		createInput.ExtendedS3DestinationConfiguration = extendedS3Config
	} else {
		s3Config := createS3Config(d)

		if d.Get("destination").(string) == "s3" {
			createInput.S3DestinationConfiguration = s3Config
		} else if d.Get("destination").(string) == "elasticsearch" {
			esConfig, err := createElasticsearchConfig(d, s3Config)
			if err != nil {
				return err
			}
			createInput.ElasticsearchDestinationConfiguration = esConfig
		} else if d.Get("destination").(string) == "redshift" {
			rc, err := createRedshiftConfig(d, s3Config)
			if err != nil {
				return err
			}
			createInput.RedshiftDestinationConfiguration = rc
		} else if d.Get("destination").(string) == "splunk" {
			rc, err := createSplunkConfig(d, s3Config)
			if err != nil {
				return err
			}
			createInput.SplunkDestinationConfiguration = rc
		}
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
			return fmt.Errorf("[WARN] Error creating Kinesis Firehose Delivery Stream: %s", awsErr.Error())
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

func validateAwsKinesisFirehoseSchema(d *schema.ResourceData) error {

	_, s3Exists := d.GetOk("s3_configuration")
	_, extendedS3Exists := d.GetOk("extended_s3_configuration")

	if d.Get("destination").(string) == "extended_s3" {
		if !extendedS3Exists {
			return fmt.Errorf(
				"When destination is 'extended_s3', extended_s3_configuration is required",
			)
		} else if s3Exists {
			return fmt.Errorf(
				"When destination is 'extended_s3', s3_configuration must not be set",
			)
		}
	} else {
		if !s3Exists {
			return fmt.Errorf(
				"When destination is %s, s3_configuration is required",
				d.Get("destination").(string),
			)
		} else if extendedS3Exists {
			return fmt.Errorf(
				"extended_s3_configuration can only be used when destination is 'extended_s3'",
			)
		}
	}

	return nil
}

func resourceAwsKinesisFirehoseDeliveryStreamUpdate(d *schema.ResourceData, meta interface{}) error {
	validateError := validateAwsKinesisFirehoseSchema(d)

	if validateError != nil {
		return validateError
	}

	conn := meta.(*AWSClient).firehoseconn

	sn := d.Get("name").(string)

	updateInput := &firehose.UpdateDestinationInput{
		DeliveryStreamName:             aws.String(sn),
		CurrentDeliveryStreamVersionId: aws.String(d.Get("version_id").(string)),
		DestinationId:                  aws.String(d.Get("destination_id").(string)),
	}

	if d.Get("destination").(string) == "extended_s3" {
		extendedS3Config := updateExtendedS3Config(d)
		updateInput.ExtendedS3DestinationUpdate = extendedS3Config
	} else {
		s3Config := updateS3Config(d)

		if d.Get("destination").(string) == "s3" {
			updateInput.S3DestinationUpdate = s3Config
		} else if d.Get("destination").(string) == "elasticsearch" {
			esUpdate, err := updateElasticsearchConfig(d, s3Config)
			if err != nil {
				return err
			}
			updateInput.ElasticsearchDestinationUpdate = esUpdate
		} else if d.Get("destination").(string) == "redshift" {
			rc, err := updateRedshiftConfig(d, s3Config)
			if err != nil {
				return err
			}
			updateInput.RedshiftDestinationUpdate = rc
		} else if d.Get("destination").(string) == "splunk" {
			rc, err := updateSplunkConfig(d, s3Config)
			if err != nil {
				return err
			}
			updateInput.SplunkDestinationUpdate = rc
		}
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
			return fmt.Errorf("[WARN] Error reading Kinesis Firehose Delivery Stream: %s", awsErr.Error())
		}
		return err
	}

	s := resp.DeliveryStreamDescription
	err = flattenKinesisFirehoseDeliveryStream(d, s)
	if err != nil {
		return err
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
