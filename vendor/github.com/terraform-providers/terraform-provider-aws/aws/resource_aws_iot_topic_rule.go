package aws

import (
	"log"

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

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateIoTTopicRuleName,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"enabled": {
				Type:     schema.TypeBool,
				Required: true,
			},
			"sql": {
				Type:     schema.TypeString,
				Required: true,
			},
			"sql_version": {
				Type:     schema.TypeString,
				Required: true,
			},
			"cloudwatch_alarm": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"alarm_name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"role_arn": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateArn,
						},
						"state_reason": {
							Type:     schema.TypeString,
							Required: true,
						},
						"state_value": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateIoTTopicRuleCloudWatchAlarmStateValue,
						},
					},
				},
			},
			"cloudwatch_metric": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"metric_name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"metric_namespace": {
							Type:     schema.TypeString,
							Required: true,
						},
						"metric_timestamp": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validateIoTTopicRuleCloudWatchMetricTimestamp,
						},
						"metric_unit": {
							Type:     schema.TypeString,
							Required: true,
						},
						"metric_value": {
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
			"dynamodb": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"hash_key_field": {
							Type:     schema.TypeString,
							Required: true,
						},
						"hash_key_value": {
							Type:     schema.TypeString,
							Required: true,
						},
						"hash_key_type": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"payload_field": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"range_key_field": {
							Type:     schema.TypeString,
							Required: true,
						},
						"range_key_value": {
							Type:     schema.TypeString,
							Required: true,
						},
						"range_key_type": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"role_arn": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateArn,
						},
						"table_name": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"elasticsearch": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"endpoint": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateIoTTopicRuleElasticSearchEndpoint,
						},
						"id": {
							Type:     schema.TypeString,
							Required: true,
						},
						"index": {
							Type:     schema.TypeString,
							Required: true,
						},
						"role_arn": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateArn,
						},
						"type": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"firehose": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"delivery_stream_name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"role_arn": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateArn,
						},
						"separator": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validateIoTTopicRuleFirehoseSeparator,
						},
					},
				},
			},
			"kinesis": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"partition_key": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"role_arn": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateArn,
						},
						"stream_name": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"lambda": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"function_arn": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateArn,
						},
					},
				},
			},
			"republish": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"role_arn": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateArn,
						},
						"topic": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"s3": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"bucket_name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"key": {
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
			"sns": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"message_format": {
							Type:     schema.TypeString,
							Default:  iot.MessageFormatRaw,
							Optional: true,
						},
						"target_arn": {
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
			"sqs": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"queue_url": {
							Type:     schema.TypeString,
							Required: true,
						},
						"role_arn": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateArn,
						},
						"use_base64": {
							Type:     schema.TypeBool,
							Required: true,
						},
					},
				},
			},
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func createTopicRulePayload(d *schema.ResourceData) *iot.TopicRulePayload {
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
		act := &iot.Action{
			CloudwatchMetric: &iot.CloudwatchMetricAction{
				MetricName:      aws.String(raw["metric_name"].(string)),
				MetricNamespace: aws.String(raw["metric_namespace"].(string)),
				MetricUnit:      aws.String(raw["metric_unit"].(string)),
				MetricValue:     aws.String(raw["metric_value"].(string)),
				RoleArn:         aws.String(raw["role_arn"].(string)),
			},
		}
		if v, ok := raw["metric_timestamp"].(string); ok && v != "" {
			act.CloudwatchMetric.MetricTimestamp = aws.String(v)
		}
		actions[i] = act
		i++
	}

	// Add DynamoDB actions
	for _, a := range dynamoDbActions {
		raw := a.(map[string]interface{})
		act := &iot.Action{
			DynamoDB: &iot.DynamoDBAction{
				HashKeyField:  aws.String(raw["hash_key_field"].(string)),
				HashKeyValue:  aws.String(raw["hash_key_value"].(string)),
				RangeKeyField: aws.String(raw["range_key_field"].(string)),
				RangeKeyValue: aws.String(raw["range_key_value"].(string)),
				RoleArn:       aws.String(raw["role_arn"].(string)),
				TableName:     aws.String(raw["table_name"].(string)),
			},
		}
		if v, ok := raw["hash_key_type"].(string); ok && v != "" {
			act.DynamoDB.HashKeyType = aws.String(v)
		}
		if v, ok := raw["range_key_type"].(string); ok && v != "" {
			act.DynamoDB.RangeKeyType = aws.String(v)
		}
		if v, ok := raw["payload_field"].(string); ok && v != "" {
			act.DynamoDB.PayloadField = aws.String(v)
		}
		actions[i] = act
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
		act := &iot.Action{
			Firehose: &iot.FirehoseAction{
				DeliveryStreamName: aws.String(raw["delivery_stream_name"].(string)),
				RoleArn:            aws.String(raw["role_arn"].(string)),
			},
		}
		if v, ok := raw["separator"].(string); ok && v != "" {
			act.Firehose.Separator = aws.String(raw["separator"].(string))
		}
		actions[i] = act
		i++
	}

	// Add Kinesis actions

	for _, a := range kinesisActions {
		raw := a.(map[string]interface{})
		act := &iot.Action{
			Kinesis: &iot.KinesisAction{
				RoleArn:    aws.String(raw["role_arn"].(string)),
				StreamName: aws.String(raw["stream_name"].(string)),
			},
		}
		if v, ok := raw["partition_key"].(string); ok && v != "" {
			act.Kinesis.PartitionKey = aws.String(v)
		}
		actions[i] = act
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

	return &iot.TopicRulePayload{
		Description:      aws.String(d.Get("description").(string)),
		RuleDisabled:     aws.Bool(!d.Get("enabled").(bool)),
		Sql:              aws.String(d.Get("sql").(string)),
		AwsIotSqlVersion: aws.String(d.Get("sql_version").(string)),
		Actions:          actions,
	}
}

func resourceAwsIotTopicRuleCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iotconn

	ruleName := d.Get("name").(string)

	params := &iot.CreateTopicRuleInput{
		RuleName:         aws.String(ruleName),
		TopicRulePayload: createTopicRulePayload(d),
	}
	log.Printf("[DEBUG] Creating IoT Topic Rule: %s", params)
	_, err := conn.CreateTopicRule(params)

	if err != nil {
		return err
	}

	d.SetId(ruleName)

	return resourceAwsIotTopicRuleRead(d, meta)
}

func resourceAwsIotTopicRuleRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iotconn

	params := &iot.GetTopicRuleInput{
		RuleName: aws.String(d.Id()),
	}
	log.Printf("[DEBUG] Reading IoT Topic Rule: %s", params)
	out, err := conn.GetTopicRule(params)

	if err != nil {
		return err
	}

	d.Set("arn", out.RuleArn)
	d.Set("name", out.Rule.RuleName)
	d.Set("description", out.Rule.Description)
	d.Set("enabled", !(*out.Rule.RuleDisabled))
	d.Set("sql", out.Rule.Sql)
	d.Set("sql_version", out.Rule.AwsIotSqlVersion)
	d.Set("cloudwatch_alarm", flattenIoTRuleCloudWatchAlarmActions(out.Rule.Actions))
	d.Set("cloudwatch_metric", flattenIoTRuleCloudWatchMetricActions(out.Rule.Actions))
	d.Set("dynamodb", flattenIoTRuleDynamoDbActions(out.Rule.Actions))
	d.Set("elasticsearch", flattenIoTRuleElasticSearchActions(out.Rule.Actions))
	d.Set("firehose", flattenIoTRuleFirehoseActions(out.Rule.Actions))
	d.Set("kinesis", flattenIoTRuleKinesisActions(out.Rule.Actions))
	d.Set("lambda", flattenIoTRuleLambdaActions(out.Rule.Actions))
	d.Set("republish", flattenIoTRuleRepublishActions(out.Rule.Actions))
	d.Set("s3", flattenIoTRuleS3Actions(out.Rule.Actions))
	d.Set("sns", flattenIoTRuleSnsActions(out.Rule.Actions))
	d.Set("sqs", flattenIoTRuleSqsActions(out.Rule.Actions))

	return nil
}

func resourceAwsIotTopicRuleUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iotconn

	params := &iot.ReplaceTopicRuleInput{
		RuleName:         aws.String(d.Get("name").(string)),
		TopicRulePayload: createTopicRulePayload(d),
	}
	log.Printf("[DEBUG] Updating IoT Topic Rule: %s", params)
	_, err := conn.ReplaceTopicRule(params)

	if err != nil {
		return err
	}

	return resourceAwsIotTopicRuleRead(d, meta)
}

func resourceAwsIotTopicRuleDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).iotconn

	params := &iot.DeleteTopicRuleInput{
		RuleName: aws.String(d.Id()),
	}
	log.Printf("[DEBUG] Deleting IoT Topic Rule: %s", params)
	_, err := conn.DeleteTopicRule(params)

	if err != nil {
		return err
	}

	return nil
}
