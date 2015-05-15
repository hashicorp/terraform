---
layout: "aws"
page_title: "AWS: aws_sns_topic"
sidebar_current: "docs-aws-resource-sns-topic"
description: |-
  Provides an SNS topic
---

# aws\_sns\_topic

Provides an SNS topic.

## Example Usage

```
resource "aws_sns_topic" "topic" {
  name = "topic"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The SNS topic's name.

## Attribute Reference

The following attributes are exported:

* `name` - The name of the topic.
* `arn` - The ARN assigned by AWS to this topic.
