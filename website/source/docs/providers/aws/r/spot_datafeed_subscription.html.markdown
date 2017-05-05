---
layout: "aws"
page_title: "AWS: aws_spot_datafeed_subscription"
sidebar_current: "docs-aws-resource-spot-datafeed-subscription"
description: |-
  Provides a Spot Datafeed Subscription resource.
---

# aws\_spot\_datafeed\_subscription

-> **Note:** There is only a single subscription allowed per account.

To help you understand the charges for your Spot instances, Amazon EC2 provides a data feed that describes your Spot instance usage and pricing.
This data feed is sent to an Amazon S3 bucket that you specify when you subscribe to the data feed.

## Example Usage

```hcl
resource "aws_s3_bucket" "default" {
  bucket = "tf-spot-datafeed"
}

resource "aws_spot_datafeed_subscription" "default" {
  bucket = "${aws_s3_bucket.default.bucket}"
  prefix = "my_subdirectory"
}
```

## Argument Reference
* `bucket` - (Required) The Amazon S3 bucket in which to store the Spot instance data feed.
* `prefix` - (Optional) Path of folder inside bucket to place spot pricing data.

## Import

A Spot Datafeed Subscription can be imported using the word `spot-datafeed-subscription`, e.g.

```
$ terraform import aws_spot_datafeed_subscription.mysubscription spot-datafeed-subscription
```
