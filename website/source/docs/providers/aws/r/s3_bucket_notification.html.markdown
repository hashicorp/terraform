---
layout: "aws"
page_title: "AWS: aws_s3_bucket_notification"
side_bar_current: "docs-aws-resource-s3-bucket-notification"
description: |-
  Provides a S3 bucket notification resource.
---

# aws\_s3\_bucket\_notification

Provides a S3 bucket notification resource.

## Example Usage

### Add notification configuration to SNS Topic

```
resource "aws_sns_topic" "topic" {
    name = "s3-event-notification-topic"
    policy = <<POLICY
{
    "Version":"2012-10-17",
    "Statement":[{
        "Effect": "Allow",
        "Principal": {"AWS":"*"},
        "Action": "SNS:Publish",
        "Resource": "arn:aws:sns:*:*:s3-event-notification-topic",
        "Condition":{
            "ArnLike":{"aws:SourceArn":"${aws_s3_bucket.bucket.arn}"}
        }
    }]
}
POLICY
}

resource "aws_s3_bucket" "bucket" {
	bucket = "your_bucket_name"
}

resource "aws_s3_bucket_notification" "bucket_notification" {
	bucket = "${aws_s3_bucket.bucket.id}"
	topic {
		topic_arn = "${aws_sns_topic.topic.arn}"
		events = ["s3:ObjectCreated:*"]
		filter_suffix = ".log"
	}
}
```

### Add notification configuration to SQS Queue

```
resource "aws_sqs_queue" "queue" {
    name = "s3-event-notification-queue"
    policy = <<POLICY
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": "*",
      "Action": "sqs:SendMessage",
	  "Resource": "arn:aws:sqs:*:*:s3-event-notification-queue",
      "Condition": {
        "ArnEquals": { "aws:SourceArn": "${aws_s3_bucket.bucket.arn}" }
      }
    }
  ]
}
POLICY
}

resource "aws_s3_bucket" "bucket" {
	bucket = "your_bucket_name"
}

resource "aws_s3_bucket_notification" "bucket_notification" {
	bucket = "${aws_s3_bucket.bucket.id}"
	queue {
		queue_arn = "${aws_sqs_queue.queue.arn}"
		events = ["s3:ObjectCreated:*"]
		filter_suffix = ".log"
	}
}
```

### Add notification configuration to Lambda Function

```
resource "aws_iam_role" "iam_for_lambda" {
    name = "iam_for_lambda"
    assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "lambda.amazonaws.com"
      },
      "Effect": "Allow"
    }
  ]
}
EOF
}

resource "aws_lambda_permission" "allow_bucket" {
    statement_id = "AllowExecutionFromS3Bucket"
    action = "lambda:InvokeFunction"
    function_name = "${aws_lambda_function.func.arn}"
    principal = "s3.amazonaws.com"
    source_arn = "${aws_s3_bucket.bucket.arn}"
}

resource "aws_lambda_function" "func" {
    filename = "your-function.zip"
    function_name = "example_lambda_name"
    role = "${aws_iam_role.iam_for_lambda.arn}"
    handler = "exports.example"
}

resource "aws_s3_bucket" "bucket" {
	bucket = "your_bucket_name"
}

resource "aws_s3_bucket_notification" "bucket_notification" {
	bucket = "${aws_s3_bucket.bucket.id}"
	lambda_function {
		lambda_function_arn = "${aws_lambda_function.func.arn}"
		events = ["s3:ObjectCreated:*"]
		filter_prefix = "AWSLogs/"
		filter_suffix = ".log"
	}
}
```

## Argument Reference

The following arguments are supported:

* `bucket` - (Required) The name of the bucket to put notification configuration.
* `topic` - (Optional) The notification configuration to SNS Topic (documented below).
* `queue` - (Optional) The notification configuration to SQS Queue (documented below).
* `lambda_function` - (Optional) The notification configuration to Lambda Function (documented below).

The `topic` notification configuration supports the following:

* `id` - (Optional) Specifies unique identifier for each of the notification configurations.
* `topic_arn` - (Required) Specifies Amazon SNS topic ARN.
* `events` - (Required) Specifies [event](http://docs.aws.amazon.com/AmazonS3/latest/dev/NotificationHowTo.html#notification-how-to-event-types-and-destinations) for which to send notifications.
* `filter_prefix` - (Optional) Specifies object key name prefix.
* `filter_suffix` - (Optional) Specifies object key name suffix.

The `queue` notification configuration supports the following:

* `id` - (Optional) Specifies unique identifier for each of the notification configurations.
* `queue_arn` - (Required) Specifies Amazon SQS queue ARN.
* `events` - (Required) Specifies [event](http://docs.aws.amazon.com/AmazonS3/latest/dev/NotificationHowTo.html#notification-how-to-event-types-and-destinations) for which to send notifications.
* `filter_prefix` - (Optional) Specifies object key name prefix.
* `filter_suffix` - (Optional) Specifies object key name suffix.

The `lambda_function` notification configuration supports the following:

* `id` - (Optional) Specifies unique identifier for each of the notification configurations.
* `lambda_function_arn` - (Required) Specifies Amazon Lambda function ARN.
* `events` - (Required) Specifies [event](http://docs.aws.amazon.com/AmazonS3/latest/dev/NotificationHowTo.html#notification-how-to-event-types-and-destinations) for which to send notifications.
* `filter_prefix` - (Optional) Specifies object key name prefix.
* `filter_suffix` - (Optional) Specifies object key name suffix.

