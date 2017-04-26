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

```hcl
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
    topic_arn     = "${aws_sns_topic.topic.arn}"
    events        = ["s3:ObjectCreated:*"]
    filter_suffix = ".log"
  }
}
```

### Add notification configuration to SQS Queue

```hcl
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
    queue_arn     = "${aws_sqs_queue.queue.arn}"
    events        = ["s3:ObjectCreated:*"]
    filter_suffix = ".log"
  }
}
```

### Add notification configuration to Lambda Function

```hcl
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
  statement_id  = "AllowExecutionFromS3Bucket"
  action        = "lambda:InvokeFunction"
  function_name = "${aws_lambda_function.func.arn}"
  principal     = "s3.amazonaws.com"
  source_arn    = "${aws_s3_bucket.bucket.arn}"
}

resource "aws_lambda_function" "func" {
  filename      = "your-function.zip"
  function_name = "example_lambda_name"
  role          = "${aws_iam_role.iam_for_lambda.arn}"
  handler       = "exports.example"
}

resource "aws_s3_bucket" "bucket" {
  bucket = "your_bucket_name"
}

resource "aws_s3_bucket_notification" "bucket_notification" {
  bucket = "${aws_s3_bucket.bucket.id}"

  lambda_function {
    lambda_function_arn = "${aws_lambda_function.func.arn}"
    events              = ["s3:ObjectCreated:*"]
    filter_prefix       = "AWSLogs/"
    filter_suffix       = ".log"
  }
}
```

### Trigger multiple Lambda functions

```hcl
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

resource "aws_lambda_permission" "allow_bucket1" {
  statement_id  = "AllowExecutionFromS3Bucket1"
  action        = "lambda:InvokeFunction"
  function_name = "${aws_lambda_function.func1.arn}"
  principal     = "s3.amazonaws.com"
  source_arn    = "${aws_s3_bucket.bucket.arn}"
}

resource "aws_lambda_function" "func1" {
  filename      = "your-function1.zip"
  function_name = "example_lambda_name1"
  role          = "${aws_iam_role.iam_for_lambda.arn}"
  handler       = "exports.example"
}

resource "aws_lambda_permission" "allow_bucket2" {
  statement_id  = "AllowExecutionFromS3Bucket2"
  action        = "lambda:InvokeFunction"
  function_name = "${aws_lambda_function.func2.arn}"
  principal     = "s3.amazonaws.com"
  source_arn    = "${aws_s3_bucket.bucket.arn}"
}

resource "aws_lambda_function" "func2" {
  filename      = "your-function2.zip"
  function_name = "example_lambda_name2"
  role          = "${aws_iam_role.iam_for_lambda.arn}"
  handler       = "exports.example"
}

resource "aws_s3_bucket" "bucket" {
  bucket = "your_bucket_name"
}

resource "aws_s3_bucket_notification" "bucket_notification" {
  bucket = "${aws_s3_bucket.bucket.id}"

  lambda_function {
    lambda_function_arn = "${aws_lambda_function.func1.arn}"
    events              = ["s3:ObjectCreated:*"]
    filter_prefix       = "AWSLogs/"
    filter_suffix       = ".log"
  }

  lambda_function {
    lambda_function_arn = "${aws_lambda_function.func2.arn}"
    events              = ["s3:ObjectCreated:*"]
    filter_prefix       = "OtherLogs/"
    filter_suffix       = ".log"
  }
}
```

### Add multiple notification configurations to SQS Queue

```hcl
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
    id            = "image-upload-event"
    queue_arn     = "${aws_sqs_queue.queue.arn}"
    events        = ["s3:ObjectCreated:*"]
    filter_prefix = "images/"
  }

  queue {
    id            = "video-upload-event"
    queue_arn     = "${aws_sqs_queue.queue.arn}"
    events        = ["s3:ObjectCreated:*"]
    filter_prefix = "videos/"
  }
}
```

For Terraform's [JSON syntax](https://www.terraform.io/docs/configuration/syntax.html), use an array instead of defining the `queue` key twice.

```json
{
	"bucket": "${aws_s3_bucket.bucket.id}",
	"queue": [
		{
			"id": "image-upload-event",
			"queue_arn": "${aws_sqs_queue.queue.arn}",
			"events": ["s3:ObjectCreated:*"],
			"filter_prefix": "images/"
		},
		{
			"id": "video-upload-event",
			"queue_arn": "${aws_sqs_queue.queue.arn}",
			"events": ["s3:ObjectCreated:*"],
			"filter_prefix": "videos/"
		}
	]
}
```

## Argument Reference

The following arguments are supported:

* `bucket` - (Required) The name of the bucket to put notification configuration.
* `topic` - (Optional) The notification configuration to SNS Topic (documented below).
* `queue` - (Optional) The notification configuration to SQS Queue (documented below).
* `lambda_function` - (Optional, Multiple) Used to configure notifications to a Lambda Function (documented below).

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

## Import

S3 bucket notification can be imported using the `bucket`, e.g.

```
$ terraform import aws_s3_bucket_notification.bucket_notification bucket-name
```
