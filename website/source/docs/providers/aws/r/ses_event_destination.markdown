---
layout: "aws"
page_title: "AWS: ses_event_destination"
sidebar_current: "docs-aws-resource-ses-event-destination"
description: |-
  Provides an SES event destination
---

# aws\_ses\_event_destination

Provides an SES event destination

## Example Usage

```hcl
# Add a firehose event destination to a configuration set
resource "aws_ses_event_destination" "kinesis" {
  name                   = "event-destination-kinesis"
  configuration_set_name = "${aws_ses_configuration_set.test.name}"
  enabled                = true
  matching_types         = ["bounce", "send"]

  kinesis_destination = {
    stream_arn = "${aws_kinesis_firehose_delivery_stream.test_stream.arn}"
    role_arn   = "${aws_iam_role.firehose_role.arn}"
  }
}

# CloudWatch event destination
resource "aws_ses_event_destination" "cloudwatch" {
  name                   = "event-destination-cloudwatch"
  configuration_set_name = "${aws_ses_configuration_set.test.name}"
  enabled                = true
  matching_types         = ["bounce", "send"]

  cloudwatch_destination = {
    default_value  = "default"
    dimension_name = "dimension"
    value_source   = "emailHeader"
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the event destination
* `configuration_set_name` - (Required) The name of the configuration set
* `enabled` - (Optional) If true, the event destination will be enabled
* `matching_types` - (Required) A list of matching types. May be any of `"send"`, `"reject"`, `"bounce"`, `"complaint"`, or `"delivery"`.
* `cloudwatch_destination` - (Optional) CloudWatch destination for the events
* `kinesis_destination` - (Optional) Send the events to a kinesis firehose destination

~> **NOTE:** You can specify `"cloudwatch_destination"` or `"kinesis_destination"` but not both

CloudWatch Destination requires the following:

* `default_value` - (Required) The default value for the event
* `dimension_name` - (Required) The name for the dimension
* `value_source` - (Required) The source for the value. It can be either `"messageTag"` or `"emailHeader"`

Kinesis Destination requires the following:

* `stream_arn` - (Required) The ARN of the Kinesis Stream
* `role_arn` - (Required) The ARN of the role that has permissions to access the Kinesis Stream

