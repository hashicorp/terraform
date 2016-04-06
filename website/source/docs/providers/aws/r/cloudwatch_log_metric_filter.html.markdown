---
layout: "aws"
page_title: "AWS: aws_cloudwatch_log_metric_filter"
sidebar_current: "docs-aws-resource-cloudwatch-log-metric-filter"
description: |-
  Provides a CloudWatch Log Metric Filter resource.
---

# aws\_cloudwatch\_log\_metric\_filter

Provides a CloudWatch Log Metric Filter resource.

## Example Usage

```
resource "aws_cloudwatch_log_metric_filter" "yada" {
  name = "MyAppAccessCount"
  pattern = ""
  log_group_name = "${aws_cloudwatch_log_group.dada.name}"

  metric_transformation {
  	name = "EventCount"
  	namespace = "YourNamespace"
  	value = "1"
  }
}

resource "aws_cloudwatch_log_group" "dada" {
	name = "MyApp/access.log"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) A name for the metric filter.
* `pattern` - (Required) A valid [CloudWatch Logs filter pattern](https://docs.aws.amazon.com/AmazonCloudWatch/latest/DeveloperGuide/FilterAndPatternSyntax.html)
  for extracting metric data out of ingested log events.
* `log_group_name` - (Required) The name of the log group to associate the metric filter with.
* `metric_transformation` - (Required) A block defining collection of information
	needed to define how metric data gets emitted. See below.

The `metric_transformation` block supports the following arguments:

* `name` - (Required) The name of the CloudWatch metric to which the monitored log information should be published (e.g. `ErrorCount`)
* `namespace` - (Required) The destination namespace of the CloudWatch metric.
* `value` - (Required) What to publish to the metric. For example, if you're counting the occurrences of a particular term like "Error", the value will be "1" for each occurrence. If you're counting the bytes transferred the published value will be the value in the log event.

## Attributes Reference

The following attributes are exported:

* `id` - The name of the metric filter.
