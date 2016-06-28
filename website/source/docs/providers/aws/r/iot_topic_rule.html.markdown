---
layout: "aws"
page_title: "AWS: aws_iot_topic_rule"
sidebar_current: "docs-aws-resource-iot-topic-rule"
description: |-
    Creates and manages an AWS IoT topic rule
---

# aws\_iot\_topic_rule

## Example Usage

```
resource "aws_iot_topic_rule" "rule" {
  name = "MyRule"
  description = "Example rule"
  enabled = true
  sql = "SELECT * FROM 'topic/test'";
  sql_version = ""

  cloudwatch_alarm {
    alarm_name = ""
    role_arn = ""
    state_reason = ""
    state_value = ""
  }

  cloudwatch_metric {
    metric_name = ""
    metric_namespace = ""
    metric_timestamp = ""
    metric_unit = ""
    metric_value = ""
    role_arn = ""
  }

  dynamodb {
    hash_key_field = ""
    hash_key_value = ""
    payload_field = ""
    range_key_field = ""
    range_key_value = ""
    role_arn = ""
    table_name = ""
  }

  elasticsearch {
    endpoint = ""
    id = ""
    index = ""
    role_arn = ""
    type = ""
  }

  firehose {
    delivery_stream_name = ""
    role_arn = ""
  }

  kinesis {
    partition_key = ""
    role_arn = ""
    stream_name = ""
  }

  lambda {
    function_arn = ""
  }

  republish {
    role_arn = ""
    topic = ""
  }

  s3 {
    bucket_name = ""
    key = ""
    role_arn = ""
  }

  sns {
    message_format = ""
    role_arn = ""
    target_arn = ""
  }

  sqs {
    queue_url = ""
    role_arn = ""
    use_base64 = false
  }
}
```

## Argument Reference

* `name` - Name of the topic rule
* `description` - Human readable description of the topic rule
* `enabled` - Boolean flag to indicate if the topic rule is enabled
* `sql` - The SQL statement of the topic rule
* `sql_version` - Version of the SQL statement

The `cloudwatch_alarm` object takes the following arguments:

* `alarm_name` - The CloudWatch alarm name
* `role_arn` - The IAM role arn that allows to access the CloudWatch alarm
* `state_reason` - The reason for the alarm change
* `state_value` - The value of the alarm state. Acceptable values are: OK, ALARM, INSUFFICIENT_DATA

The `cloudwatch_metric` object takes the following arguments:

* `metric_name` - The CloudWatch metric name
* `metric_namespace` - The CloudWatch metric namespace
* `metric_timestamp` - The CloudWatch metric timestamp
* `metric_unit` - The CloudWatch metric unit (supported units can be found here: http://docs.aws.amazon.com/AmazonCloudWatch/latest/DeveloperGuide/cloudwatch_concepts.html#Unit)
* `metric_value` - The CloudWatch metric value
* `role_arn` - The IAM role arn that allows to access the CloudWatch metric

The `dynamodb` object takes the following arguments:

* `hash_key_field` - The hash key field
* `hash_key_type` - The hash key type, can be STRING or NUMBER
* `hash_key_value` - The hash key value
* `payload_field` - The action payload
* `range_key_field` - The range key field
* `range_key_type` - The range key type, can be STRING or NUMBer
* `range_key_value` - The range key value
* `role_arn` - The IAM role arn that allows to access the DynamoDB table
* `table_name` - The DynamoDB table name

The `elasticsearch` object takes the following arguments:

* `endpoint` - The ElasticSearch endpoint
* `id` - Unique ID for the document
* `index` - The ElasticSearch index
* `role_arn` - The IAM role arn that allows to access the ElasticSearch domain
* `type` - The type of the document

The `firehose` object takes the following arguments:

* `delivery_stream_name` - The name of the Firehose delivery stream
* `role_arn` - The IAM role arn that allows to access the Firehose delivery stream

The `kinesis` object takes the following arguments:

* `partition_key` - The partition key
* `role_arn` - The IAM role arn that allows to access the Kinesis stream
* `stream_name` - The Kinesis stream name

The `lambda` object takes the following arguments:

* `function_arn` - The arn of the lambda function

The `republish` object takes the following arguments:

* `role_arn` - The IAM role arn that allows to access the topic
* `topic` - The topic the message should be republished to

The `s3` object takes the following arguments:

* `bucket_name` - The name of the S3 bucket
* `key` - The key of the object
* `role_arn` - The IAM role arn that allows to access the S3 bucket

The `sns` object takes the following arguments:

* `message_format` - The message format, allowed values are "RAW" or "JSON"
* `role_arn` - The IAM role arn that allows to access the SNS topic
* `target_arn` - The arn of the SNS topic

The `sqs` object takes the following arguments:

* `queue_url` - The URL of the SQS queue
* `role_arn` - The IAM role arn that allows to access the SQS queue
* `use_base64` - Boolean to enable base64 encoding

## Attributes Reference

* `arn` - The ARN of the topic rul
