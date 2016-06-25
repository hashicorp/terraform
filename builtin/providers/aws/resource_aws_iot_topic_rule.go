package aws

import (
	//"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iot"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsIotTopicRule() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsIotTopicRuleCreate,
		Read:   resourceAwsIotTopicRuleRead,
		Update: resourceAwsIotTopicRuleUpdate,
		Delete: resourceAwsIotTopicRuleDelete,
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"enabled": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"sql": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"sql_version": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"cloudwatch_alarm": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"alarm_name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"role_arn": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"state_reason": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"state_value": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"cloudwatch_metric": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"metric_name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"metric_namespace": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"metric_timestamp": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"metric_unit": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"metric_value": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"role_arn": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"dynamodb": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"hash_key_field": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"hash_key_value": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"payload_field": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"range_key_field": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"range_key_value": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"role_arn": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"table_name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"elasticsearch": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"endpoint": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"id": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"index": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"role_arn": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"type": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"firehose": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"delivery_stream_name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"role_arn": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"kinesis": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"partition_key": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"role_arn": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"stream_name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"lambda": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"function_arn": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"republish": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"role_arn": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"topic": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"s3": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"bucket_name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"key": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"role_arn": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"sns": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"message_format": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"target_arn": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"role_arn": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"sqs": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"queue_url": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"role_arn": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"use_base64": &schema.Schema{
							Type:     schema.TypeBool,
							Required: true,
						},
					},
				},
			},
		},
	}
}

func resourceAwsIotTopicRuleCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iotconn

	ruleName := d.Get("name").(string)

	cloudwatchAlarmActions := d.Get("cloudwatch_alarm").(*schema.Set).List()
	cloudwatchMetricActions := d.Get("cloudwatch_metric").(*schema.Set).List()
	dynamoDbActions := d.Get("dynamodb").(*schema.Set).List()
	elasticsearchActions := d.Get("elasticsearch").(*schema.Set).List()
	firehoseActions := d.Get("firehose").(*schema.Set).List()
	kinesisActions := d.Get("kinesis").(*schema.Set).List()
	lambdaActions := d.Get("lambda").(*schema.Set).List()
	republishActions := d.Get("republish").(*schema.Set).List()
	s3Actions := d.Get("s3").(*schema.Set).List()
	snsActions := d.Get("sns").(*schema.Set).List()
	sqsActions := d.Get("sqs").(*schema.Set).List()

	numActions := len(cloudwatchAlarmActions) + len(cloudwatchMetricActions) +
		len(dynamoDbActions) + len(elasticsearchActions) + len(firehoseActions) +
		len(kinesisActions) + len(lambdaActions) + len(republishActions) +
		len(s3Actions) + len(snsActions) + len(sqsActions)
	actions := make([]*iot.Action, numActions)

	i := 0
	// Add Cloudwatch Alarm actions
	for _, a := range cloudwatchAlarmActions {
		raw := a.(map[string]interface{})
		actions[i] = &iot.Action{
			CloudwatchAlarm: &iot.CloudwatchAlarmAction{
				AlarmName:   aws.String(raw["alarm_name"].(string)),
				RoleArn:     aws.String(raw["role_arn"].(string)),
				StateReason: aws.String(raw["state_reason"].(string)),
				StateValue:  aws.String(raw["state_value"].(string)),
			},
		}
		i++
	}

	// Add Cloudwatch Metric actions
	for _, a := range cloudwatchMetricActions {
		raw := a.(map[string]interface{})
		actions[i] = &iot.Action{
			CloudwatchMetric: &iot.CloudwatchMetricAction{
				MetricName:      aws.String(raw["metric_name"].(string)),
				MetricNamespace: aws.String(raw["metric_namespace"].(string)),
				MetricUnit:      aws.String(raw["metric_unit"].(string)),
				MetricValue:     aws.String(raw["metric_value"].(string)),
				RoleArn:         aws.String(raw["role_arn"].(string)),
				MetricTimestamp: aws.String(raw["metric_timestamp"].(string)),
			},
		}
		i++
	}

	// Add DynamoDB actions
	for _, a := range dynamoDbActions {
		raw := a.(map[string]interface{})
		// TODO: add hash_key_type
		// TODO: add range_key_type
		actions[i] = &iot.Action{
			DynamoDB: &iot.DynamoDBAction{
				HashKeyField:  aws.String(raw["hash_key_field"].(string)),
				HashKeyValue:  aws.String(raw["hash_key_value"].(string)),
				RangeKeyField: aws.String(raw["range_key_field"].(string)),
				RangeKeyValue: aws.String(raw["range_key_value"].(string)),
				RoleArn:       aws.String(raw["role_arn"].(string)),
				TableName:     aws.String(raw["table_name"].(string)),
				PayloadField:  aws.String(raw["payload_field"].(string)),
			},
		}
		i++
	}

	// Add Elasticsearch actions

	for _, a := range elasticsearchActions {
		raw := a.(map[string]interface{})
		actions[i] = &iot.Action{
			Elasticsearch: &iot.ElasticsearchAction{
				Endpoint: aws.String(raw["endpoint"].(string)),
				Id:       aws.String(raw["id"].(string)),
				Index:    aws.String(raw["index"].(string)),
				RoleArn:  aws.String(raw["role_arn"].(string)),
				Type:     aws.String(raw["type"].(string)),
			},
		}
		i++
	}

	// Add Firehose actions

	for _, a := range firehoseActions {
		raw := a.(map[string]interface{})
		actions[i] = &iot.Action{
			Firehose: &iot.FirehoseAction{
				DeliveryStreamName: aws.String(raw["delivery_stream_name"].(string)),
				RoleArn:            aws.String(raw["role_arn"].(string)),
			},
		}
		i++
	}

	// Add Kinesis actions

	for _, a := range kinesisActions {
		raw := a.(map[string]interface{})
		actions[i] = &iot.Action{
			Kinesis: &iot.KinesisAction{
				RoleArn:      aws.String(raw["role_arn"].(string)),
				StreamName:   aws.String(raw["stream_name"].(string)),
				PartitionKey: aws.String(raw["partition_key"].(string)),
			},
		}
		i++
	}

	// Add Lambda actions

	for _, a := range lambdaActions {
		raw := a.(map[string]interface{})
		actions[i] = &iot.Action{
			Lambda: &iot.LambdaAction{
				FunctionArn: aws.String(raw["function_arn"].(string)),
			},
		}
		i++
	}

	// Add Republish actions

	for _, a := range republishActions {
		raw := a.(map[string]interface{})
		actions[i] = &iot.Action{
			Republish: &iot.RepublishAction{
				RoleArn: aws.String(raw["role_arn"].(string)),
				Topic:   aws.String(raw["topic"].(string)),
			},
		}
		i++
	}

	// Add S3 actions

	for _, a := range s3Actions {
		raw := a.(map[string]interface{})
		actions[i] = &iot.Action{
			S3: &iot.S3Action{
				BucketName: aws.String(raw["bucket_name"].(string)),
				Key:        aws.String(raw["key"].(string)),
				RoleArn:    aws.String(raw["role_arn"].(string)),
			},
		}
		i++
	}

	// Add SNS actions

	for _, a := range snsActions {
		raw := a.(map[string]interface{})
		actions[i] = &iot.Action{
			Sns: &iot.SnsAction{
				RoleArn:       aws.String(raw["role_arn"].(string)),
				TargetArn:     aws.String(raw["target_arn"].(string)),
				MessageFormat: aws.String(raw["message_format"].(string)),
			},
		}
		i++
	}

	// Add SQS actions

	for _, a := range sqsActions {
		raw := a.(map[string]interface{})
		actions[i] = &iot.Action{
			Sqs: &iot.SqsAction{
				QueueUrl:  aws.String(raw["queue_url"].(string)),
				RoleArn:   aws.String(raw["role_arn"].(string)),
				UseBase64: aws.Bool(raw["use_base64"].(bool)),
			},
		}
		i++
	}

	_, err := conn.CreateTopicRule(&iot.CreateTopicRuleInput{
		RuleName: aws.String(ruleName),
		TopicRulePayload: &iot.TopicRulePayload{
			Description:      aws.String(d.Get("description").(string)),
			RuleDisabled:     aws.Bool(!d.Get("enabled").(bool)),
			Sql:              aws.String(d.Get("sql").(string)),
			AwsIotSqlVersion: aws.String(d.Get("sql_version").(string)),
			Actions:          actions,
		},
	})

	if err != nil {
		return err
	}

	d.SetId(ruleName)

	return nil
}

func resourceAwsIotTopicRuleRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iotconn

	out, err := conn.GetTopicRule(&iot.GetTopicRuleInput{
		RuleName: aws.String(d.Id()),
	})

	if err != nil {
		return err
	}

	d.SetId(*out.Rule.RuleName)
	d.Set("arn", *out.RuleArn)

	return nil
}

func resourceAwsIotTopicRuleUpdate(d *schema.ResourceData, meta interface{}) error {
	//TODO: implement
	return nil
}

func resourceAwsIotTopicRuleDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iotconn

	_, err := conn.DeleteTopicRule(&iot.DeleteTopicRuleInput{
		RuleName: aws.String(d.Id()),
	})

	if err != nil {
		return err
	}

	return nil
}
