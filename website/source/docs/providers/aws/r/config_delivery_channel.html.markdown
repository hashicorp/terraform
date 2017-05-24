---
layout: "aws"
page_title: "AWS: aws_config_delivery_channel"
sidebar_current: "docs-aws-resource-config-delivery-channel"
description: |-
  Provides an AWS Config Delivery Channel.
---

# aws\_config\_delivery\_channel

Provides an AWS Config Delivery Channel.

~> **Note:** Delivery Channel requires a [Configuration Recorder](/docs/providers/aws/r/config_configuration_recorder.html) to be present. Use of `depends_on` (as shown below) is recommended to avoid race conditions.

## Example Usage

```hcl
resource "aws_config_delivery_channel" "foo" {
  name           = "example"
  s3_bucket_name = "${aws_s3_bucket.b.bucket}"
  depends_on     = ["aws_config_configuration_recorder.foo"]
}

resource "aws_s3_bucket" "b" {
  bucket        = "example-awsconfig"
  force_destroy = true
}

resource "aws_config_configuration_recorder" "foo" {
  name     = "example"
  role_arn = "${aws_iam_role.r.arn}"
}

resource "aws_iam_role" "r" {
  name = "awsconfig-example"

  assume_role_policy = <<POLICY
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "config.amazonaws.com"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
POLICY
}

resource "aws_iam_role_policy" "p" {
  name = "awsconfig-example"
  role = "${aws_iam_role.r.id}"

  policy = <<POLICY
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": [
        "s3:*"
      ],
      "Effect": "Allow",
      "Resource": [
        "${aws_s3_bucket.b.arn}",
        "${aws_s3_bucket.b.arn}/*"
      ]
    }
  ]
}
POLICY
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Optional) The name of the delivery channel. Defaults to `default`.
* `s3_bucket_name` - (Required) The name of the S3 bucket used to store the configuration history.
* `s3_key_prefix` - (Optional) The prefix for the specified S3 bucket.
* `sns_topic_arn` - (Optional) The ARN of the SNS topic that AWS Config delivers notifications to.
* `snapshot_delivery_properties` - (Optional) Options for how AWS Config delivers configuration snapshots. See below

### `snapshot_delivery_properties`

* `delivery_frequency` - (Optional) - The frequency with which a AWS Config recurringly delivers configuration snapshots.
	e.g. `One_Hour` or `Three_Hours`

## Attributes Reference

The following attributes are exported:

* `id` - The name of the delivery channel.

## Import

Delivery Channel can be imported using the name, e.g.

```
$ terraform import aws_config_delivery_channel.foo example
```
