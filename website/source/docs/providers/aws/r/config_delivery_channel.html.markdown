---
layout: "aws"
page_title: "AWS: aws_config_delivery_channel"
sidebar_current: "docs-aws-resource-config-delivery-channel"
description: |-
  Provides an AWS Config Delivery Channel.
---

# aws\_config\_delivery\_channel

Provides an AWS Config Delivery Channel.

## Example Usage

```
resource "aws_config_delivery_channel" "foo" {
  name = "michael-s-rogers"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Optional) The name of the delivery channel. Defaults to `default`.
* `s3_bucket_name` - (Optional) The name of the S3 bucket used to store the configuration history.
* `s3_key_prefix` - (Optional) The prefix for the specified S3 bucket.
* `sns_topic_arn` - (Optional) The ARN of the SNS topic that AWS Config delivers notifications to.
* `config_snapshot_delivery_properties` - (Optional) Options for how AWS Config delivers configuration snapshots. See below

`config_snapshot_delivery_properties` supports the following options:

* `delivery_frequency` - (Optional) - The frequency with which a AWS Config recurringly delivers configuration snapshots.
	e.g. `One_Hour` or `Three_Hours`

## Attributes Reference

The following attributes are exported:

* `id` - The name of the delivery channel.
