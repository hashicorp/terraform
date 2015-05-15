---
layout: "aws"
page_title: "AWS: aws_sqs"
sidebar_current: "docs-aws-resource-sqs"
description: |-
  Provides a SQS resource.
---

# aws\_sqs

Provides a SQS resource.

## Example Usage

```
resource "aws_sqs" "queue" {
    name = "queue-name"
}
```

## Argument Reference

The following arguments are supported:

* `visibility_timeout` - (Optional) The time in seconds that the delivery of all messages in the queue will be delayed. An integer from 0 to 900 (15 minutes). The default for this attribute is 0 (zero)
* `retention_period` - (Optional) The number of seconds Amazon SQS retains a message. Integer representing seconds, from 60 (1 minute) to 1209600 (14 days). The default for this attribute is 345600 (4 days).
* `max_message_size` - (Optional) The limit of how many bytes a message can contain before Amazon SQS rejects it. An integer from 1024 bytes (1 KiB) up to 262144 bytes (256 KiB). The default for this attribute is 262144 (256 KiB).
* `delivery_delay` - (Optional) The visibility timeout for the queue. An integer from 0 to 43200 (12 hours). The default for this attribute is 30. For more information about visibility timeout.
* `receive_wait_time` - (Optional) The time for which a ReceiveMessage call will wait for a message to arrive. An integer from 0 to 20 (seconds). The default for this attribute is 0.

## Attributes Reference

The following attributes are exported:

* `queue_url` - The URL for the created Amazon SQS queue.
