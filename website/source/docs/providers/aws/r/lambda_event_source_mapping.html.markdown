---
layout: "aws"
page_title: "AWS: aws_lambda_event_source_mapping"
sidebar_current: "docs-aws-resource-aws-lambda-event-source-mapping"
description: |-
  Provides a Lambda event source mapping. This allows Lambda functions to get events from Kinesis and DynamoDB.
---

# aws\_lambda\_event\_source\_mapping

Provides a Lambda event source mapping. This allows Lambda functions to get events from Kinesis and DynamoDB.

For information about Lambda and how to use it, see [What is AWS Lambda?][1]
For information about event source mappings, see [CreateEventSourceMapping][2] in the API docs.

## Example Usage

```
resource "aws_lambda_event_source_mapping" "event_source_mapping" {
    batch_size = 100
    event_source_arn = "arn:aws:kinesis:REGION:123456789012:stream/stream_name"
    enabled = true
    function_name = "arn:aws:lambda:REGION:123456789012:function:function_name"
    starting_position = "TRIM_HORIZON|LATEST"
}
```

## Argument Reference

* `batch_size` - (Optional) The largest number of records that Lambda will retrieve from your event source at the time of invocation. Defaults to `100`.
* `event_source_arn` - (Required) The event source ARN - can either be a Kinesis or DynamoDB stream.
* `enabled` - (Optional) Determines if the mapping will be enabled on creation. Defaults to `true`.
* `function_name` - (Required) The name or the ARN of the Lambda function that will be subscribing to events.
* `starting_position` - (Required) The position in the stream where AWS Lambda should start reading. Can be one of either `TRIM_HORIZON` or `LATEST`.

## Attributes Reference

* `function_arn` - The the ARN of the Lambda function the event source mapping is sending events to. (Note: this is a computed value that differs from `function_name` above.)
* `last_modified` - The date this resource was last modified.
* `last_processing_result` - The result of the last AWS Lambda invocation of your Lambda function.
* `state` - The state of the event source mapping.
* `state_transition_reason` - The reason the event source mapping is in its current state.
* `uuid` - The UUID of the created event source mapping.


[1]: http://docs.aws.amazon.com/lambda/latest/dg/welcome.html
[2]: http://docs.aws.amazon.com/lambda/latest/dg/API_CreateEventSourceMapping.html
