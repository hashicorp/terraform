---
layout: "aws"
page_title: "AWS: aws_kinesis_stream"
sidebar_current: "docs-aws-resource-kinesis-stream"
description: |-
  Provides a AWS Kinesis Stream
---

# aws\_kinesis\_stream

Provides a Kinesis Stream resource. Amazon Kinesis is a managed service that 
scales elastically for real-time processing of streaming big data.

For more details, see the [Amazon Kinesis Documentation][1].

## Example Usage

```
resource "aws_kinesis_stream" "test_stream" {
	name = "terraform-kinesis-test"
	shard_count = 1
	tags {
		Environment = "test"
	}
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) A name to identify the stream. This is unique to the 
AWS account and region the Stream is created in.
* `shard_count` – (Required) The number of shards that the stream will use.
Amazon has guidlines for specifying the Stream size that should be referenced 
when creating a Kinesis stream. See [Amazon Kinesis Streams][2] for more.
* `tags` - (Optional) A mapping of tags to assign to the resource.

## Attributes Reference

* `id` - The unique Stream id
* `name` - The unique Stream name (same as `id`)
* `shard_count` - The count of Shards for this Stream
* `arn` - The Amazon Resource Name (ARN) specifying the Stream


[1]: http://aws.amazon.com/documentation/kinesis/
[2]: http://docs.aws.amazon.com/kinesis/latest/dev/amazon-kinesis-streams.html
