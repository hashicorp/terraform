package aws

import (
	"fmt"
	"log"
	"reflect"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/kinesisanalytics"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceAwsKinesisAnalyticsApplication() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsKinesisAnalyticsApplicationCreate,
		Read:   resourceAwsKinesisAnalyticsApplicationRead,
		Update: resourceAwsKinesisAnalyticsApplicationUpdate,
		Delete: resourceAwsKinesisAnalyticsApplicationDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"code": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"create_timestamp": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"last_update_timestamp": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"status": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"version": {
				Type:     schema.TypeInt,
				Computed: true,
			},

			"cloudwatch_logging_options": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"log_stream_arn": {
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

			"inputs": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"kinesis_firehose": {
							Type:     schema.TypeList,
							Optional: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"resource_arn": {
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

						"kinesis_stream": {
							Type:     schema.TypeList,
							Optional: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"resource_arn": {
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

						"name_prefix": {
							Type:     schema.TypeString,
							Required: true,
						},

						"parallelism": {
							Type:     schema.TypeList,
							Optional: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"count": {
										Type:     schema.TypeInt,
										Required: true,
									},
								},
							},
						},

						"processing_configuration": {
							Type:     schema.TypeList,
							Optional: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"lambda": {
										Type:     schema.TypeList,
										Required: true,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"resource_arn": {
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
								},
							},
						},

						"schema": {
							Type:     schema.TypeList,
							Required: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"record_columns": {
										Type:     schema.TypeList,
										Required: true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"mapping": {
													Type:     schema.TypeString,
													Optional: true,
												},

												"name": {
													Type:     schema.TypeString,
													Required: true,
												},

												"sql_type": {
													Type:     schema.TypeString,
													Required: true,
												},
											},
										},
									},

									"record_encoding": {
										Type:     schema.TypeString,
										Optional: true,
									},

									"record_format": {
										Type:     schema.TypeList,
										Required: true,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"mapping_parameters": {
													Type:     schema.TypeList,
													Optional: true,
													MaxItems: 1,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"csv": {
																Type:     schema.TypeList,
																Optional: true,
																MaxItems: 1,
																Elem: &schema.Resource{
																	Schema: map[string]*schema.Schema{
																		"record_column_delimiter": {
																			Type:     schema.TypeString,
																			Required: true,
																		},

																		"record_row_delimiter": {
																			Type:     schema.TypeString,
																			Required: true,
																		},
																	},
																},
															},

															"json": {
																Type:     schema.TypeList,
																Optional: true,
																MaxItems: 1,
																Elem: &schema.Resource{
																	Schema: map[string]*schema.Schema{
																		"record_row_path": {
																			Type:     schema.TypeString,
																			Required: true,
																		},
																	},
																},
															},
														},
													},
												},

												"record_format_type": {
													Type:     schema.TypeString,
													Computed: true,
												},
											},
										},
									},
								},
							},
						},

						"starting_position_configuration": {
							Type:     schema.TypeList,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"starting_position": {
										Type:     schema.TypeString,
										Computed: true,
									},
								},
							},
						},

						"stream_names": {
							Type:     schema.TypeSet,
							Computed: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},

			"outputs": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 3,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"kinesis_firehose": {
							Type:     schema.TypeList,
							Optional: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"resource_arn": {
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

						"kinesis_stream": {
							Type:     schema.TypeList,
							Optional: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"resource_arn": {
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

						"lambda": {
							Type:     schema.TypeList,
							Optional: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"resource_arn": {
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

						"name": {
							Type:     schema.TypeString,
							Required: true,
						},

						"schema": {
							Type:     schema.TypeList,
							Required: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"record_format_type": {
										Type:     schema.TypeString,
										Optional: true,
										ValidateFunc: validation.StringInSlice([]string{
											kinesisanalytics.RecordFormatTypeCsv,
											kinesisanalytics.RecordFormatTypeJson,
										}, false),
									},
								},
							},
						},
					},
				},
			},

			"reference_data_sources": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"s3": {
							Type:     schema.TypeList,
							Required: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"bucket_arn": {
										Type:         schema.TypeString,
										Required:     true,
										ValidateFunc: validateArn,
									},

									"file_key": {
										Type:     schema.TypeString,
										Required: true,
									},

									"role_arn": {
										Type:         schema.TypeString,
										Required:     true,
										ValidateFunc: validateArn,
									},
								},
							},
						},

						"schema": {
							Type:     schema.TypeList,
							Required: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"record_columns": {
										Type:     schema.TypeList,
										Required: true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"mapping": {
													Type:     schema.TypeString,
													Optional: true,
												},

												"name": {
													Type:     schema.TypeString,
													Required: true,
												},

												"sql_type": {
													Type:     schema.TypeString,
													Required: true,
												},
											},
										},
									},

									"record_encoding": {
										Type:     schema.TypeString,
										Optional: true,
									},

									"record_format": {
										Type:     schema.TypeList,
										Required: true,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"mapping_parameters": {
													Type:     schema.TypeList,
													Optional: true,
													MaxItems: 1,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"csv": {
																Type:     schema.TypeList,
																Optional: true,
																MaxItems: 1,
																Elem: &schema.Resource{
																	Schema: map[string]*schema.Schema{
																		"record_column_delimiter": {
																			Type:     schema.TypeString,
																			Required: true,
																		},

																		"record_row_delimiter": {
																			Type:     schema.TypeString,
																			Required: true,
																		},
																	},
																},
															},

															"json": {
																Type:     schema.TypeList,
																Optional: true,
																MaxItems: 1,
																Elem: &schema.Resource{
																	Schema: map[string]*schema.Schema{
																		"record_row_path": {
																			Type:     schema.TypeString,
																			Required: true,
																		},
																	},
																},
															},
														},
													},
												},

												"record_format_type": {
													Type:     schema.TypeString,
													Computed: true,
												},
											},
										},
									},
								},
							},
						},

						"table_name": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
		},
	}
}

func resourceAwsKinesisAnalyticsApplicationCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).kinesisanalyticsconn
	name := d.Get("name").(string)
	createOpts := &kinesisanalytics.CreateApplicationInput{
		ApplicationName: aws.String(name),
	}

	if v, ok := d.GetOk("code"); ok && v.(string) != "" {
		createOpts.ApplicationCode = aws.String(v.(string))
	}

	if v, ok := d.GetOk("cloudwatch_logging_options"); ok {
		clo := v.([]interface{})[0].(map[string]interface{})
		cloudwatchLoggingOption := expandKinesisAnalyticsCloudwatchLoggingOption(clo)
		createOpts.CloudWatchLoggingOptions = []*kinesisanalytics.CloudWatchLoggingOption{cloudwatchLoggingOption}
	}

	if v, ok := d.GetOk("inputs"); ok {
		i := v.([]interface{})[0].(map[string]interface{})
		inputs := expandKinesisAnalyticsInputs(i)
		createOpts.Inputs = []*kinesisanalytics.Input{inputs}
	}

	if v, ok := d.GetOk("outputs"); ok {
		o := v.([]interface{})[0].(map[string]interface{})
		outputs := expandKinesisAnalyticsOutputs(o)
		createOpts.Outputs = []*kinesisanalytics.Output{outputs}
	}

	err := resource.Retry(1*time.Minute, func() *resource.RetryError {
		output, err := conn.CreateApplication(createOpts)
		if err != nil {
			if isAWSErr(err, kinesisanalytics.ErrCodeInvalidArgumentException, "Kinesis Analytics service doesn't have sufficient privileges") {
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(err)
		}
		d.SetId(aws.StringValue(output.ApplicationSummary.ApplicationARN))
		return nil
	})
	if err != nil {
		return fmt.Errorf("Unable to create Kinesis Analytics application: %s", err)
	}

	return resourceAwsKinesisAnalyticsApplicationUpdate(d, meta)
}

func resourceAwsKinesisAnalyticsApplicationRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).kinesisanalyticsconn
	name := d.Get("name").(string)

	describeOpts := &kinesisanalytics.DescribeApplicationInput{
		ApplicationName: aws.String(name),
	}
	resp, err := conn.DescribeApplication(describeOpts)
	if isAWSErr(err, kinesisanalytics.ErrCodeResourceNotFoundException, "") {
		log.Printf("[WARN] Kinesis Analytics Application (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}
	if err != nil {
		return fmt.Errorf("error reading Kinesis Analytics Application (%s): %s", d.Id(), err)
	}

	d.Set("name", aws.StringValue(resp.ApplicationDetail.ApplicationName))
	d.Set("arn", aws.StringValue(resp.ApplicationDetail.ApplicationARN))
	d.Set("code", aws.StringValue(resp.ApplicationDetail.ApplicationCode))
	d.Set("create_timestamp", aws.TimeValue(resp.ApplicationDetail.CreateTimestamp).Format(time.RFC3339))
	d.Set("description", aws.StringValue(resp.ApplicationDetail.ApplicationDescription))
	d.Set("last_update_timestamp", aws.TimeValue(resp.ApplicationDetail.LastUpdateTimestamp).Format(time.RFC3339))
	d.Set("status", aws.StringValue(resp.ApplicationDetail.ApplicationStatus))
	d.Set("version", int(aws.Int64Value(resp.ApplicationDetail.ApplicationVersionId)))

	if err := d.Set("cloudwatch_logging_options", flattenKinesisAnalyticsCloudwatchLoggingOptions(resp.ApplicationDetail.CloudWatchLoggingOptionDescriptions)); err != nil {
		return fmt.Errorf("error setting cloudwatch_logging_options: %s", err)
	}

	if err := d.Set("inputs", flattenKinesisAnalyticsInputs(resp.ApplicationDetail.InputDescriptions)); err != nil {
		return fmt.Errorf("error setting inputs: %s", err)
	}

	if err := d.Set("outputs", flattenKinesisAnalyticsOutputs(resp.ApplicationDetail.OutputDescriptions)); err != nil {
		return fmt.Errorf("error setting outputs: %s", err)
	}

	if err := d.Set("reference_data_sources", flattenKinesisAnalyticsReferenceDataSources(resp.ApplicationDetail.ReferenceDataSourceDescriptions)); err != nil {
		return fmt.Errorf("error setting reference_data_sources: %s", err)
	}

	return nil
}

func resourceAwsKinesisAnalyticsApplicationUpdate(d *schema.ResourceData, meta interface{}) error {
	var version int
	conn := meta.(*AWSClient).kinesisanalyticsconn
	name := d.Get("name").(string)

	if v, ok := d.GetOk("version"); ok {
		version = v.(int)
	} else {
		version = 1
	}

	if !d.IsNewResource() {
		updateApplicationOpts := &kinesisanalytics.UpdateApplicationInput{
			ApplicationName:             aws.String(name),
			CurrentApplicationVersionId: aws.Int64(int64(version)),
		}

		applicationUpdate, err := createApplicationUpdateOpts(d)
		if err != nil {
			return err
		}

		if !reflect.DeepEqual(applicationUpdate, &kinesisanalytics.ApplicationUpdate{}) {
			updateApplicationOpts.SetApplicationUpdate(applicationUpdate)
			_, updateErr := conn.UpdateApplication(updateApplicationOpts)
			if updateErr != nil {
				return updateErr
			}
			version = version + 1
		}

		oldLoggingOptions, newLoggingOptions := d.GetChange("cloudwatch_logging_options")
		if len(oldLoggingOptions.([]interface{})) == 0 && len(newLoggingOptions.([]interface{})) > 0 {
			if v, ok := d.GetOk("cloudwatch_logging_options"); ok {
				clo := v.([]interface{})[0].(map[string]interface{})
				cloudwatchLoggingOption := expandKinesisAnalyticsCloudwatchLoggingOption(clo)
				addOpts := &kinesisanalytics.AddApplicationCloudWatchLoggingOptionInput{
					ApplicationName:             aws.String(name),
					CurrentApplicationVersionId: aws.Int64(int64(version)),
					CloudWatchLoggingOption:     cloudwatchLoggingOption,
				}
				err := resource.Retry(1*time.Minute, func() *resource.RetryError {
					_, err := conn.AddApplicationCloudWatchLoggingOption(addOpts)
					if err != nil {
						if isAWSErr(err, kinesisanalytics.ErrCodeInvalidArgumentException, "Kinesis Analytics service doesn't have sufficient privileges") {
							return resource.RetryableError(err)
						}
						return resource.NonRetryableError(err)
					}
					return nil
				})
				if err != nil {
					return fmt.Errorf("Unable to add CloudWatch logging options: %s", err)
				}
				version = version + 1
			}
		}

		oldInputs, newInputs := d.GetChange("inputs")
		if len(oldInputs.([]interface{})) == 0 && len(newInputs.([]interface{})) > 0 {
			if v, ok := d.GetOk("inputs"); ok {
				i := v.([]interface{})[0].(map[string]interface{})
				input := expandKinesisAnalyticsInputs(i)
				addOpts := &kinesisanalytics.AddApplicationInputInput{
					ApplicationName:             aws.String(name),
					CurrentApplicationVersionId: aws.Int64(int64(version)),
					Input:                       input,
				}
				err := resource.Retry(1*time.Minute, func() *resource.RetryError {
					_, err := conn.AddApplicationInput(addOpts)
					if err != nil {
						if isAWSErr(err, kinesisanalytics.ErrCodeInvalidArgumentException, "Kinesis Analytics service doesn't have sufficient privileges") {
							return resource.RetryableError(err)
						}
						return resource.NonRetryableError(err)
					}
					return nil
				})
				if err != nil {
					return fmt.Errorf("Unable to add application inputs: %s", err)
				}
				version = version + 1
			}
		}

		oldOutputs, newOutputs := d.GetChange("outputs")
		if len(oldOutputs.([]interface{})) == 0 && len(newOutputs.([]interface{})) > 0 {
			if v, ok := d.GetOk("outputs"); ok {
				o := v.([]interface{})[0].(map[string]interface{})
				output := expandKinesisAnalyticsOutputs(o)
				addOpts := &kinesisanalytics.AddApplicationOutputInput{
					ApplicationName:             aws.String(name),
					CurrentApplicationVersionId: aws.Int64(int64(version)),
					Output:                      output,
				}
				err := resource.Retry(1*time.Minute, func() *resource.RetryError {
					_, err := conn.AddApplicationOutput(addOpts)
					if err != nil {
						if isAWSErr(err, kinesisanalytics.ErrCodeInvalidArgumentException, "Kinesis Analytics service doesn't have sufficient privileges") {
							return resource.RetryableError(err)
						}
						return resource.NonRetryableError(err)
					}
					return nil
				})
				if err != nil {
					return fmt.Errorf("Unable to add application outputs: %s", err)
				}
				version = version + 1
			}
		}
	}

	oldReferenceData, newReferenceData := d.GetChange("reference_data_sources")
	if len(oldReferenceData.([]interface{})) == 0 && len(newReferenceData.([]interface{})) > 0 {
		if v := d.Get("reference_data_sources").([]interface{}); len(v) > 0 {
			for _, r := range v {
				rd := r.(map[string]interface{})
				referenceData := expandKinesisAnalyticsReferenceData(rd)
				addOpts := &kinesisanalytics.AddApplicationReferenceDataSourceInput{
					ApplicationName:             aws.String(name),
					CurrentApplicationVersionId: aws.Int64(int64(version)),
					ReferenceDataSource:         referenceData,
				}
				err := resource.Retry(1*time.Minute, func() *resource.RetryError {
					_, err := conn.AddApplicationReferenceDataSource(addOpts)
					if err != nil {
						if isAWSErr(err, kinesisanalytics.ErrCodeInvalidArgumentException, "Kinesis Analytics service doesn't have sufficient privileges") {
							return resource.RetryableError(err)
						}
						return resource.NonRetryableError(err)
					}
					return nil
				})
				if err != nil {
					return fmt.Errorf("Unable to add application reference data source: %s", err)
				}
				version = version + 1
			}
		}
	}

	return resourceAwsKinesisAnalyticsApplicationRead(d, meta)
}

func resourceAwsKinesisAnalyticsApplicationDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).kinesisanalyticsconn
	name := d.Get("name").(string)
	createTimestamp, parseErr := time.Parse(time.RFC3339, d.Get("create_timestamp").(string))
	if parseErr != nil {
		return parseErr
	}

	log.Printf("[DEBUG] Kinesis Analytics Application destroy: %v", d.Id())
	deleteOpts := &kinesisanalytics.DeleteApplicationInput{
		ApplicationName: aws.String(name),
		CreateTimestamp: aws.Time(createTimestamp),
	}
	_, deleteErr := conn.DeleteApplication(deleteOpts)
	if isAWSErr(deleteErr, kinesisanalytics.ErrCodeResourceNotFoundException, "") {
		return nil
	}
	deleteErr = waitForDeleteKinesisAnalyticsApplication(conn, d.Id(), d.Timeout(schema.TimeoutDelete))
	if deleteErr != nil {
		return fmt.Errorf("error waiting for deletion of Kinesis Analytics Application (%s): %s", d.Id(), deleteErr)
	}

	log.Printf("[DEBUG] Kinesis Analytics Application deleted: %v", d.Id())
	return nil
}

func expandKinesisAnalyticsCloudwatchLoggingOption(clo map[string]interface{}) *kinesisanalytics.CloudWatchLoggingOption {
	cloudwatchLoggingOption := &kinesisanalytics.CloudWatchLoggingOption{
		LogStreamARN: aws.String(clo["log_stream_arn"].(string)),
		RoleARN:      aws.String(clo["role_arn"].(string)),
	}
	return cloudwatchLoggingOption
}

func expandKinesisAnalyticsInputs(i map[string]interface{}) *kinesisanalytics.Input {
	input := &kinesisanalytics.Input{
		NamePrefix: aws.String(i["name_prefix"].(string)),
	}

	if v := i["kinesis_firehose"].([]interface{}); len(v) > 0 {
		kf := v[0].(map[string]interface{})
		kfi := &kinesisanalytics.KinesisFirehoseInput{
			ResourceARN: aws.String(kf["resource_arn"].(string)),
			RoleARN:     aws.String(kf["role_arn"].(string)),
		}
		input.KinesisFirehoseInput = kfi
	}

	if v := i["kinesis_stream"].([]interface{}); len(v) > 0 {
		ks := v[0].(map[string]interface{})
		ksi := &kinesisanalytics.KinesisStreamsInput{
			ResourceARN: aws.String(ks["resource_arn"].(string)),
			RoleARN:     aws.String(ks["role_arn"].(string)),
		}
		input.KinesisStreamsInput = ksi
	}

	if v := i["parallelism"].([]interface{}); len(v) > 0 {
		p := v[0].(map[string]interface{})

		if c, ok := p["count"]; ok {
			ip := &kinesisanalytics.InputParallelism{
				Count: aws.Int64(int64(c.(int))),
			}
			input.InputParallelism = ip
		}
	}

	if v := i["processing_configuration"].([]interface{}); len(v) > 0 {
		pc := v[0].(map[string]interface{})

		if l := pc["lambda"].([]interface{}); len(l) > 0 {
			lp := l[0].(map[string]interface{})
			ipc := &kinesisanalytics.InputProcessingConfiguration{
				InputLambdaProcessor: &kinesisanalytics.InputLambdaProcessor{
					ResourceARN: aws.String(lp["resource_arn"].(string)),
					RoleARN:     aws.String(lp["role_arn"].(string)),
				},
			}
			input.InputProcessingConfiguration = ipc
		}
	}

	if v := i["schema"].([]interface{}); len(v) > 0 {
		vL := v[0].(map[string]interface{})
		ss := expandKinesisAnalyticsSourceSchema(vL)
		input.InputSchema = ss
	}

	return input
}

func expandKinesisAnalyticsSourceSchema(vL map[string]interface{}) *kinesisanalytics.SourceSchema {
	ss := &kinesisanalytics.SourceSchema{}
	if v := vL["record_columns"].([]interface{}); len(v) > 0 {
		var rcs []*kinesisanalytics.RecordColumn

		for _, rc := range v {
			rcD := rc.(map[string]interface{})
			rc := &kinesisanalytics.RecordColumn{
				Name:    aws.String(rcD["name"].(string)),
				SqlType: aws.String(rcD["sql_type"].(string)),
			}

			if v, ok := rcD["mapping"]; ok {
				rc.Mapping = aws.String(v.(string))
			}

			rcs = append(rcs, rc)
		}

		ss.RecordColumns = rcs
	}

	if v, ok := vL["record_encoding"]; ok && v.(string) != "" {
		ss.RecordEncoding = aws.String(v.(string))
	}

	if v := vL["record_format"].([]interface{}); len(v) > 0 {
		vL := v[0].(map[string]interface{})
		rf := &kinesisanalytics.RecordFormat{}

		if v := vL["mapping_parameters"].([]interface{}); len(v) > 0 {
			vL := v[0].(map[string]interface{})
			mp := &kinesisanalytics.MappingParameters{}

			if v := vL["csv"].([]interface{}); len(v) > 0 {
				cL := v[0].(map[string]interface{})
				cmp := &kinesisanalytics.CSVMappingParameters{
					RecordColumnDelimiter: aws.String(cL["record_column_delimiter"].(string)),
					RecordRowDelimiter:    aws.String(cL["record_row_delimiter"].(string)),
				}
				mp.CSVMappingParameters = cmp
				rf.RecordFormatType = aws.String("CSV")
			}

			if v := vL["json"].([]interface{}); len(v) > 0 {
				jL := v[0].(map[string]interface{})
				jmp := &kinesisanalytics.JSONMappingParameters{
					RecordRowPath: aws.String(jL["record_row_path"].(string)),
				}
				mp.JSONMappingParameters = jmp
				rf.RecordFormatType = aws.String("JSON")
			}
			rf.MappingParameters = mp
		}

		ss.RecordFormat = rf
	}
	return ss
}

func expandKinesisAnalyticsOutputs(o map[string]interface{}) *kinesisanalytics.Output {
	output := &kinesisanalytics.Output{
		Name: aws.String(o["name"].(string)),
	}

	if v := o["kinesis_firehose"].([]interface{}); len(v) > 0 {
		kf := v[0].(map[string]interface{})
		kfo := &kinesisanalytics.KinesisFirehoseOutput{
			ResourceARN: aws.String(kf["resource_arn"].(string)),
			RoleARN:     aws.String(kf["role_arn"].(string)),
		}
		output.KinesisFirehoseOutput = kfo
	}

	if v := o["kinesis_stream"].([]interface{}); len(v) > 0 {
		ks := v[0].(map[string]interface{})
		kso := &kinesisanalytics.KinesisStreamsOutput{
			ResourceARN: aws.String(ks["resource_arn"].(string)),
			RoleARN:     aws.String(ks["role_arn"].(string)),
		}
		output.KinesisStreamsOutput = kso
	}

	if v := o["lambda"].([]interface{}); len(v) > 0 {
		l := v[0].(map[string]interface{})
		lo := &kinesisanalytics.LambdaOutput{
			ResourceARN: aws.String(l["resource_arn"].(string)),
			RoleARN:     aws.String(l["role_arn"].(string)),
		}
		output.LambdaOutput = lo
	}

	if v := o["schema"].([]interface{}); len(v) > 0 {
		ds := v[0].(map[string]interface{})
		dso := &kinesisanalytics.DestinationSchema{
			RecordFormatType: aws.String(ds["record_format_type"].(string)),
		}
		output.DestinationSchema = dso
	}

	return output
}

func expandKinesisAnalyticsReferenceData(rd map[string]interface{}) *kinesisanalytics.ReferenceDataSource {
	referenceData := &kinesisanalytics.ReferenceDataSource{
		TableName: aws.String(rd["table_name"].(string)),
	}

	if v := rd["s3"].([]interface{}); len(v) > 0 {
		s3 := v[0].(map[string]interface{})
		s3rds := &kinesisanalytics.S3ReferenceDataSource{
			BucketARN:        aws.String(s3["bucket_arn"].(string)),
			FileKey:          aws.String(s3["file_key"].(string)),
			ReferenceRoleARN: aws.String(s3["role_arn"].(string)),
		}
		referenceData.S3ReferenceDataSource = s3rds
	}

	if v := rd["schema"].([]interface{}); len(v) > 0 {
		ss := expandKinesisAnalyticsSourceSchema(v[0].(map[string]interface{}))
		referenceData.ReferenceSchema = ss
	}

	return referenceData
}

func createApplicationUpdateOpts(d *schema.ResourceData) (*kinesisanalytics.ApplicationUpdate, error) {
	applicationUpdate := &kinesisanalytics.ApplicationUpdate{}

	if d.HasChange("code") {
		if v, ok := d.GetOk("code"); ok && v.(string) != "" {
			applicationUpdate.ApplicationCodeUpdate = aws.String(v.(string))
		}
	}

	oldLoggingOptions, newLoggingOptions := d.GetChange("cloudwatch_logging_options")
	if len(oldLoggingOptions.([]interface{})) > 0 && len(newLoggingOptions.([]interface{})) > 0 {
		if v, ok := d.GetOk("cloudwatch_logging_options"); ok {
			clo := v.([]interface{})[0].(map[string]interface{})
			cloudwatchLoggingOption := expandKinesisAnalyticsCloudwatchLoggingOptionUpdate(clo)
			applicationUpdate.CloudWatchLoggingOptionUpdates = []*kinesisanalytics.CloudWatchLoggingOptionUpdate{cloudwatchLoggingOption}
		}
	}

	oldInputs, newInputs := d.GetChange("inputs")
	if len(oldInputs.([]interface{})) > 0 && len(newInputs.([]interface{})) > 0 {
		if v, ok := d.GetOk("inputs"); ok {
			vL := v.([]interface{})[0].(map[string]interface{})
			inputUpdate := expandKinesisAnalyticsInputUpdate(vL)
			applicationUpdate.InputUpdates = []*kinesisanalytics.InputUpdate{inputUpdate}
		}
	}

	oldOutputs, newOutputs := d.GetChange("outputs")
	if len(oldOutputs.([]interface{})) > 0 && len(newOutputs.([]interface{})) > 0 {
		if v, ok := d.GetOk("outputs"); ok {
			vL := v.([]interface{})[0].(map[string]interface{})
			outputUpdate := expandKinesisAnalyticsOutputUpdate(vL)
			applicationUpdate.OutputUpdates = []*kinesisanalytics.OutputUpdate{outputUpdate}
		}
	}

	oldReferenceData, newReferenceData := d.GetChange("reference_data_sources")
	if len(oldReferenceData.([]interface{})) > 0 && len(newReferenceData.([]interface{})) > 0 {
		if v := d.Get("reference_data_sources").([]interface{}); len(v) > 0 {
			var rdsus []*kinesisanalytics.ReferenceDataSourceUpdate
			for _, rd := range v {
				rdL := rd.(map[string]interface{})
				rdsu := &kinesisanalytics.ReferenceDataSourceUpdate{
					ReferenceId:     aws.String(rdL["id"].(string)),
					TableNameUpdate: aws.String(rdL["table_name"].(string)),
				}

				if v := rdL["s3"].([]interface{}); len(v) > 0 {
					vL := v[0].(map[string]interface{})
					s3rdsu := &kinesisanalytics.S3ReferenceDataSourceUpdate{
						BucketARNUpdate:        aws.String(vL["bucket_arn"].(string)),
						FileKeyUpdate:          aws.String(vL["file_key"].(string)),
						ReferenceRoleARNUpdate: aws.String(vL["role_arn"].(string)),
					}
					rdsu.S3ReferenceDataSourceUpdate = s3rdsu
				}

				if v := rdL["schema"].([]interface{}); len(v) > 0 {
					vL := v[0].(map[string]interface{})
					ss := expandKinesisAnalyticsSourceSchema(vL)
					rdsu.ReferenceSchemaUpdate = ss
				}

				rdsus = append(rdsus, rdsu)
			}
			applicationUpdate.ReferenceDataSourceUpdates = rdsus
		}
	}

	return applicationUpdate, nil
}

func expandKinesisAnalyticsInputUpdate(vL map[string]interface{}) *kinesisanalytics.InputUpdate {
	inputUpdate := &kinesisanalytics.InputUpdate{
		InputId:          aws.String(vL["id"].(string)),
		NamePrefixUpdate: aws.String(vL["name_prefix"].(string)),
	}

	if v := vL["kinesis_firehose"].([]interface{}); len(v) > 0 {
		kf := v[0].(map[string]interface{})
		kfiu := &kinesisanalytics.KinesisFirehoseInputUpdate{
			ResourceARNUpdate: aws.String(kf["resource_arn"].(string)),
			RoleARNUpdate:     aws.String(kf["role_arn"].(string)),
		}
		inputUpdate.KinesisFirehoseInputUpdate = kfiu
	}

	if v := vL["kinesis_stream"].([]interface{}); len(v) > 0 {
		ks := v[0].(map[string]interface{})
		ksiu := &kinesisanalytics.KinesisStreamsInputUpdate{
			ResourceARNUpdate: aws.String(ks["resource_arn"].(string)),
			RoleARNUpdate:     aws.String(ks["role_arn"].(string)),
		}
		inputUpdate.KinesisStreamsInputUpdate = ksiu
	}

	if v := vL["parallelism"].([]interface{}); len(v) > 0 {
		p := v[0].(map[string]interface{})

		if c, ok := p["count"]; ok {
			ipu := &kinesisanalytics.InputParallelismUpdate{
				CountUpdate: aws.Int64(int64(c.(int))),
			}
			inputUpdate.InputParallelismUpdate = ipu
		}
	}

	if v := vL["processing_configuration"].([]interface{}); len(v) > 0 {
		pc := v[0].(map[string]interface{})

		if l := pc["lambda"].([]interface{}); len(l) > 0 {
			lp := l[0].(map[string]interface{})
			ipc := &kinesisanalytics.InputProcessingConfigurationUpdate{
				InputLambdaProcessorUpdate: &kinesisanalytics.InputLambdaProcessorUpdate{
					ResourceARNUpdate: aws.String(lp["resource_arn"].(string)),
					RoleARNUpdate:     aws.String(lp["role_arn"].(string)),
				},
			}
			inputUpdate.InputProcessingConfigurationUpdate = ipc
		}
	}

	if v := vL["schema"].([]interface{}); len(v) > 0 {
		ss := &kinesisanalytics.InputSchemaUpdate{}
		vL := v[0].(map[string]interface{})

		if v := vL["record_columns"].([]interface{}); len(v) > 0 {
			var rcs []*kinesisanalytics.RecordColumn

			for _, rc := range v {
				rcD := rc.(map[string]interface{})
				rc := &kinesisanalytics.RecordColumn{
					Name:    aws.String(rcD["name"].(string)),
					SqlType: aws.String(rcD["sql_type"].(string)),
				}

				if v, ok := rcD["mapping"]; ok {
					rc.Mapping = aws.String(v.(string))
				}

				rcs = append(rcs, rc)
			}

			ss.RecordColumnUpdates = rcs
		}

		if v, ok := vL["record_encoding"]; ok && v.(string) != "" {
			ss.RecordEncodingUpdate = aws.String(v.(string))
		}

		if v := vL["record_format"].([]interface{}); len(v) > 0 {
			vL := v[0].(map[string]interface{})
			rf := &kinesisanalytics.RecordFormat{}

			if v := vL["mapping_parameters"].([]interface{}); len(v) > 0 {
				vL := v[0].(map[string]interface{})
				mp := &kinesisanalytics.MappingParameters{}

				if v := vL["csv"].([]interface{}); len(v) > 0 {
					cL := v[0].(map[string]interface{})
					cmp := &kinesisanalytics.CSVMappingParameters{
						RecordColumnDelimiter: aws.String(cL["record_column_delimiter"].(string)),
						RecordRowDelimiter:    aws.String(cL["record_row_delimiter"].(string)),
					}
					mp.CSVMappingParameters = cmp
					rf.RecordFormatType = aws.String("CSV")
				}

				if v := vL["json"].([]interface{}); len(v) > 0 {
					jL := v[0].(map[string]interface{})
					jmp := &kinesisanalytics.JSONMappingParameters{
						RecordRowPath: aws.String(jL["record_row_path"].(string)),
					}
					mp.JSONMappingParameters = jmp
					rf.RecordFormatType = aws.String("JSON")
				}
				rf.MappingParameters = mp
			}
			ss.RecordFormatUpdate = rf
		}
		inputUpdate.InputSchemaUpdate = ss
	}

	return inputUpdate
}

func expandKinesisAnalyticsOutputUpdate(vL map[string]interface{}) *kinesisanalytics.OutputUpdate {
	outputUpdate := &kinesisanalytics.OutputUpdate{
		OutputId:   aws.String(vL["id"].(string)),
		NameUpdate: aws.String(vL["name"].(string)),
	}

	if v := vL["kinesis_firehose"].([]interface{}); len(v) > 0 {
		kf := v[0].(map[string]interface{})
		kfou := &kinesisanalytics.KinesisFirehoseOutputUpdate{
			ResourceARNUpdate: aws.String(kf["resource_arn"].(string)),
			RoleARNUpdate:     aws.String(kf["role_arn"].(string)),
		}
		outputUpdate.KinesisFirehoseOutputUpdate = kfou
	}

	if v := vL["kinesis_stream"].([]interface{}); len(v) > 0 {
		ks := v[0].(map[string]interface{})
		ksou := &kinesisanalytics.KinesisStreamsOutputUpdate{
			ResourceARNUpdate: aws.String(ks["resource_arn"].(string)),
			RoleARNUpdate:     aws.String(ks["role_arn"].(string)),
		}
		outputUpdate.KinesisStreamsOutputUpdate = ksou
	}

	if v := vL["lambda"].([]interface{}); len(v) > 0 {
		l := v[0].(map[string]interface{})
		lou := &kinesisanalytics.LambdaOutputUpdate{
			ResourceARNUpdate: aws.String(l["resource_arn"].(string)),
			RoleARNUpdate:     aws.String(l["role_arn"].(string)),
		}
		outputUpdate.LambdaOutputUpdate = lou
	}

	if v := vL["schema"].([]interface{}); len(v) > 0 {
		ds := v[0].(map[string]interface{})
		dsu := &kinesisanalytics.DestinationSchema{
			RecordFormatType: aws.String(ds["record_format_type"].(string)),
		}
		outputUpdate.DestinationSchemaUpdate = dsu
	}

	return outputUpdate
}

func expandKinesisAnalyticsCloudwatchLoggingOptionUpdate(clo map[string]interface{}) *kinesisanalytics.CloudWatchLoggingOptionUpdate {
	cloudwatchLoggingOption := &kinesisanalytics.CloudWatchLoggingOptionUpdate{
		CloudWatchLoggingOptionId: aws.String(clo["id"].(string)),
		LogStreamARNUpdate:        aws.String(clo["log_stream_arn"].(string)),
		RoleARNUpdate:             aws.String(clo["role_arn"].(string)),
	}
	return cloudwatchLoggingOption
}

func flattenKinesisAnalyticsCloudwatchLoggingOptions(options []*kinesisanalytics.CloudWatchLoggingOptionDescription) []interface{} {
	s := []interface{}{}
	for _, v := range options {
		option := map[string]interface{}{
			"id":             aws.StringValue(v.CloudWatchLoggingOptionId),
			"log_stream_arn": aws.StringValue(v.LogStreamARN),
			"role_arn":       aws.StringValue(v.RoleARN),
		}
		s = append(s, option)
	}
	return s
}

func flattenKinesisAnalyticsInputs(inputs []*kinesisanalytics.InputDescription) []interface{} {
	s := []interface{}{}

	if len(inputs) > 0 {
		id := inputs[0]

		input := map[string]interface{}{
			"id":          aws.StringValue(id.InputId),
			"name_prefix": aws.StringValue(id.NamePrefix),
		}

		list := schema.NewSet(schema.HashString, nil)
		for _, sn := range id.InAppStreamNames {
			list.Add(aws.StringValue(sn))
		}
		input["stream_names"] = list

		if id.InputParallelism != nil {
			input["parallelism"] = []interface{}{
				map[string]interface{}{
					"count": int(aws.Int64Value(id.InputParallelism.Count)),
				},
			}
		}

		if id.InputProcessingConfigurationDescription != nil {
			ipcd := id.InputProcessingConfigurationDescription

			if ipcd.InputLambdaProcessorDescription != nil {
				input["processing_configuration"] = []interface{}{
					map[string]interface{}{
						"lambda": []interface{}{
							map[string]interface{}{
								"resource_arn": aws.StringValue(ipcd.InputLambdaProcessorDescription.ResourceARN),
								"role_arn":     aws.StringValue(ipcd.InputLambdaProcessorDescription.RoleARN),
							},
						},
					},
				}
			}
		}

		if id.InputSchema != nil {
			inputSchema := id.InputSchema
			is := []interface{}{}
			rcs := []interface{}{}
			ss := map[string]interface{}{
				"record_encoding": aws.StringValue(inputSchema.RecordEncoding),
			}

			for _, rc := range inputSchema.RecordColumns {
				rcM := map[string]interface{}{
					"mapping":  aws.StringValue(rc.Mapping),
					"name":     aws.StringValue(rc.Name),
					"sql_type": aws.StringValue(rc.SqlType),
				}
				rcs = append(rcs, rcM)
			}
			ss["record_columns"] = rcs

			if inputSchema.RecordFormat != nil {
				rf := inputSchema.RecordFormat
				rfM := map[string]interface{}{
					"record_format_type": aws.StringValue(rf.RecordFormatType),
				}

				if rf.MappingParameters != nil {
					mps := []interface{}{}
					if rf.MappingParameters.CSVMappingParameters != nil {
						cmp := map[string]interface{}{
							"csv": []interface{}{
								map[string]interface{}{
									"record_column_delimiter": aws.StringValue(rf.MappingParameters.CSVMappingParameters.RecordColumnDelimiter),
									"record_row_delimiter":    aws.StringValue(rf.MappingParameters.CSVMappingParameters.RecordRowDelimiter),
								},
							},
						}
						mps = append(mps, cmp)
					}

					if rf.MappingParameters.JSONMappingParameters != nil {
						jmp := map[string]interface{}{
							"json": []interface{}{
								map[string]interface{}{
									"record_row_path": aws.StringValue(rf.MappingParameters.JSONMappingParameters.RecordRowPath),
								},
							},
						}
						mps = append(mps, jmp)
					}

					rfM["mapping_parameters"] = mps
				}
				ss["record_format"] = []interface{}{rfM}
			}

			is = append(is, ss)
			input["schema"] = is
		}

		if id.InputStartingPositionConfiguration != nil && id.InputStartingPositionConfiguration.InputStartingPosition != nil {
			input["starting_position_configuration"] = []interface{}{
				map[string]interface{}{
					"starting_position": aws.StringValue(id.InputStartingPositionConfiguration.InputStartingPosition),
				},
			}
		}

		if id.KinesisFirehoseInputDescription != nil {
			input["kinesis_firehose"] = []interface{}{
				map[string]interface{}{
					"resource_arn": aws.StringValue(id.KinesisFirehoseInputDescription.ResourceARN),
					"role_arn":     aws.StringValue(id.KinesisFirehoseInputDescription.RoleARN),
				},
			}
		}

		if id.KinesisStreamsInputDescription != nil {
			input["kinesis_stream"] = []interface{}{
				map[string]interface{}{
					"resource_arn": aws.StringValue(id.KinesisStreamsInputDescription.ResourceARN),
					"role_arn":     aws.StringValue(id.KinesisStreamsInputDescription.RoleARN),
				},
			}
		}

		s = append(s, input)
	}
	return s
}

func flattenKinesisAnalyticsOutputs(outputs []*kinesisanalytics.OutputDescription) []interface{} {
	s := []interface{}{}

	if len(outputs) > 0 {
		id := outputs[0]

		output := map[string]interface{}{
			"id":   aws.StringValue(id.OutputId),
			"name": aws.StringValue(id.Name),
		}

		if id.KinesisFirehoseOutputDescription != nil {
			output["kinesis_firehose"] = []interface{}{
				map[string]interface{}{
					"resource_arn": aws.StringValue(id.KinesisFirehoseOutputDescription.ResourceARN),
					"role_arn":     aws.StringValue(id.KinesisFirehoseOutputDescription.RoleARN),
				},
			}
		}

		if id.KinesisStreamsOutputDescription != nil {
			output["kinesis_stream"] = []interface{}{
				map[string]interface{}{
					"resource_arn": aws.StringValue(id.KinesisStreamsOutputDescription.ResourceARN),
					"role_arn":     aws.StringValue(id.KinesisStreamsOutputDescription.RoleARN),
				},
			}
		}

		if id.LambdaOutputDescription != nil {
			output["lambda"] = []interface{}{
				map[string]interface{}{
					"resource_arn": aws.StringValue(id.LambdaOutputDescription.ResourceARN),
					"role_arn":     aws.StringValue(id.LambdaOutputDescription.RoleARN),
				},
			}
		}

		if id.DestinationSchema != nil {
			output["schema"] = []interface{}{
				map[string]interface{}{
					"record_format_type": aws.StringValue(id.DestinationSchema.RecordFormatType),
				},
			}
		}

		s = append(s, output)
	}

	return s
}

func flattenKinesisAnalyticsReferenceDataSources(dataSources []*kinesisanalytics.ReferenceDataSourceDescription) []interface{} {
	s := []interface{}{}

	if len(dataSources) > 0 {
		for _, ds := range dataSources {
			dataSource := map[string]interface{}{
				"id":         aws.StringValue(ds.ReferenceId),
				"table_name": aws.StringValue(ds.TableName),
			}

			if ds.S3ReferenceDataSourceDescription != nil {
				dataSource["s3"] = []interface{}{
					map[string]interface{}{
						"bucket_arn": aws.StringValue(ds.S3ReferenceDataSourceDescription.BucketARN),
						"file_key":   aws.StringValue(ds.S3ReferenceDataSourceDescription.FileKey),
						"role_arn":   aws.StringValue(ds.S3ReferenceDataSourceDescription.ReferenceRoleARN),
					},
				}
			}

			if ds.ReferenceSchema != nil {
				rs := ds.ReferenceSchema
				rcs := []interface{}{}
				ss := map[string]interface{}{
					"record_encoding": aws.StringValue(rs.RecordEncoding),
				}

				for _, rc := range rs.RecordColumns {
					rcM := map[string]interface{}{
						"mapping":  aws.StringValue(rc.Mapping),
						"name":     aws.StringValue(rc.Name),
						"sql_type": aws.StringValue(rc.SqlType),
					}
					rcs = append(rcs, rcM)
				}
				ss["record_columns"] = rcs

				if rs.RecordFormat != nil {
					rf := rs.RecordFormat
					rfM := map[string]interface{}{
						"record_format_type": aws.StringValue(rf.RecordFormatType),
					}

					if rf.MappingParameters != nil {
						mps := []interface{}{}
						if rf.MappingParameters.CSVMappingParameters != nil {
							cmp := map[string]interface{}{
								"csv": []interface{}{
									map[string]interface{}{
										"record_column_delimiter": aws.StringValue(rf.MappingParameters.CSVMappingParameters.RecordColumnDelimiter),
										"record_row_delimiter":    aws.StringValue(rf.MappingParameters.CSVMappingParameters.RecordRowDelimiter),
									},
								},
							}
							mps = append(mps, cmp)
						}

						if rf.MappingParameters.JSONMappingParameters != nil {
							jmp := map[string]interface{}{
								"json": []interface{}{
									map[string]interface{}{
										"record_row_path": aws.StringValue(rf.MappingParameters.JSONMappingParameters.RecordRowPath),
									},
								},
							}
							mps = append(mps, jmp)
						}

						rfM["mapping_parameters"] = mps
					}
					ss["record_format"] = []interface{}{rfM}
				}

				dataSource["schema"] = []interface{}{ss}
			}

			s = append(s, dataSource)
		}
	}

	return s
}

func waitForDeleteKinesisAnalyticsApplication(conn *kinesisanalytics.KinesisAnalytics, applicationId string, timeout time.Duration) error {
	stateConf := resource.StateChangeConf{
		Pending: []string{
			kinesisanalytics.ApplicationStatusRunning,
			kinesisanalytics.ApplicationStatusDeleting,
		},
		Target:  []string{""},
		Timeout: timeout,
		Refresh: refreshKinesisAnalyticsApplicationStatus(conn, applicationId),
	}
	application, err := stateConf.WaitForState()
	if err != nil {
		if isAWSErr(err, kinesisanalytics.ErrCodeResourceNotFoundException, "") {
			return nil
		}
	}
	if application == nil {
		return nil
	}
	return err
}

func refreshKinesisAnalyticsApplicationStatus(conn *kinesisanalytics.KinesisAnalytics, applicationId string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		output, err := conn.DescribeApplication(&kinesisanalytics.DescribeApplicationInput{
			ApplicationName: aws.String(applicationId),
		})
		if err != nil {
			return nil, "", err
		}
		application := output.ApplicationDetail
		if application == nil {
			return application, "", fmt.Errorf("Kinesis Analytics Application (%s) could not be found.", applicationId)
		}
		return application, aws.StringValue(application.ApplicationStatus), nil
	}
}
