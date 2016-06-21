---
layout: "aws"
page_title: "AWS: aws_sqs_queue"
sidebar_current: "docs-aws-resource-sqs-queue"
description: |-
  Provides a SQS resource.
---

# aws\_sqs\_queue

## Example Usage

```
resource "aws_sqs_queue" "terraform_queue" {
  name = "terraform-example-queue"
  delay_seconds = 90
  max_message_size = 2048
  message_retention_seconds = 86400
  receive_wait_time_seconds = 10
  redrive_policy = "{\"deadLetterTargetArn\":\"${aws_sqs_queue.terraform_queue_deadletter.arn}\",\"maxReceiveCount\":4}"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) This is the human-readable name of the queue
* `visibility_timeout_seconds` - (Optional) The visibility timeout for the queue. An integer from 0 to 43200 (12 hours). The default for this attribute is 30. For more information about visibility timeout, see [AWS docs](https://docs.aws.amazon.com/AWSSimpleQueueService/latest/SQSDeveloperGuide/AboutVT.html).
* `message_retention_seconds` - (Optional) The number of seconds Amazon SQS retains a message. Integer representing seconds, from 60 (1 minute) to 1209600 (14 days). The default for this attribute is 345600 (4 days).
* `max_message_size` - (Optional) The limit of how many bytes a message can contain before Amazon SQS rejects it. An integer from 1024 bytes (1 KiB) up to 262144 bytes (256 KiB). The default for this attribute is 262144 (256 KiB).
* `delay_seconds` - (Optional) The time in seconds that the delivery of all messages in the queue will be delayed. An integer from 0 to 900 (15 minutes). The default for this attribute is 0 seconds.
* `receive_wait_time_seconds` - (Optional) The time for which a ReceiveMessage call will wait for a message to arrive (long polling) before returning. An integer from 0 to 20 (seconds). The default for this attribute is 0, meaning that the call will return immediately.
* `policy` - (Optional) The JSON policy for the SQS queue
* `redrive_policy` - (Optional) The JSON policy to set up the Dead Letter Queue, see [AWS docs](https://docs.aws.amazon.com/AWSSimpleQueueService/latest/SQSDeveloperGuide/SQSDeadLetterQueue.html). **Note:** when specifying `maxReceiveCount`, you must specify it as an integer (`5`), and not a string (`"5"`). 

## Attributes Reference

The following attributes are exported:

* `id` - The URL for the created Amazon SQS queue.
* `arn` - The ARN of the SQS queue
