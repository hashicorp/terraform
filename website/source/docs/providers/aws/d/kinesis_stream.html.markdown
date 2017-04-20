---
layout: "aws"
page_title: "AWS: aws_kinesis_stream"
sidebar_current: "docs-aws-datasource-kinesis-stream"
description: |-
  Provides a Kinesis Stream data source.
---

# aws\_kinesis\_stream

Use this data source to get information about a Kinesis Stream for use in other
resources.

For more details, see the [Amazon Kinesis Documentation][1].

## Example Usage

```
data "aws_kinesis_stream" "stream" {
  name = "stream-name"
}
```

## Argument Reference

* `name` - (Required) The name of the Kinesis Stream.

## Attributes Reference

`id` is set to the Amazon Resource Name (ARN) of the Kinesis Stream. In addition, the following attributes
are exported:

* `arn` - The Amazon Resource Name (ARN) of the Kinesis Stream (same as id).
* `name` - The name of the Kinesis Stream.
* `creation_timestamp` - The approximate UNIX timestamp that the stream was created.
* `status` - The current status of the stream. The stream status is one of CREATING, DELETING, ACTIVE, or UPDATING.
* `retention_period` - Length of time (in hours) data records are accessible after they are added to the stream.
* `open_shards` - The list of shard ids in the OPEN state. See [Shard State][2] for more.
* `closed_shards` - The list of shard ids in the CLOSED state. See [Shard State][2] for more.
* `shard_level_metrics` - A list of shard-level CloudWatch metrics which are enabled for the stream. See [Monitoring with CloudWatch][3] for more.
* `tags` - A mapping of tags to assigned to the stream.

[1]: https://aws.amazon.com/documentation/kinesis/
[2]: https://docs.aws.amazon.com/streams/latest/dev/kinesis-using-sdk-java-after-resharding.html#kinesis-using-sdk-java-resharding-data-routing
[3]: https://docs.aws.amazon.com/streams/latest/dev/monitoring-with-cloudwatch.html