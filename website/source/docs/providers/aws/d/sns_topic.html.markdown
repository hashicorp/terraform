---
layout: "aws"
page_title: "AWS: aws_sns_topic"
sidebar_current: "docs-aws-datasource-sns-topic"
description: |-
  Get information on a Amazon Simple Notification Service (SNS) Topic
---

# aws\_sns\_topic

Use this data source to get the ARN of a topic in AWS Simple Notification
Service (SNS). By using this data source, you can reference SNS topics
without having to hard code the ARNs as input.

## Example Usage

```hcl
data "aws_sns_topic" "example" {
  name = "an_example_topic"
}
```

## Argument Reference

* `name` - (Required) The friendly name of the topic to match.

## Attributes Reference

* `arn` - Set to the ARN of the found topic, suitable for referencing in other resources that support SNS topics.
