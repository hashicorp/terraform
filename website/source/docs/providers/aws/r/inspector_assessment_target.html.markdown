---
layout: "aws"
page_title: "AWS: aws_inspector_assessment_target"
sidebar_current: "docs-aws-resource-inspector-assessment-target"
description: |-
  Provides a Inspector assessment target.
---

# aws\_inspector\_assessment\_target

Provides a Inspector assessment target

## Example Usage

```
resource "aws_inspector_resource_group" "bar" {
  tags {
    Name = "foo"
    Env  = "bar"
  }
}

resource "aws_inspector_assessment_target" "foo" {
  name               = "assessment target"
  resource_group_arn = "${aws_inspector_resource_group.bar.arn}"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the assessment target.
* `resource_group_arn` (Required) - The resource group ARN stating tags for instance matching.
* `subscribe_to_event` (Optional) - A list of objects representing a subscription of an SNS topic to life-cycle events. The keys are documented below.

The `subscribe_to_event` object supports the following arguments:
* `event` - (Required) The name (or names) of the event (or events) the SNS topic is subscribing to. Allowed values are: `ASSESSMENT_RUN_STARTED`, `ASSESSMENT_RUN_COMPLETED`, `ASSESSMENT_RUN_STATE_CHANGED`, and `FINDING_REPORTED`.
* `topic_arn` - (Required) The ARN of the subscribing SNS topic.

## Attributes Reference

The following attributes are exported:

* `arn` - The target assessment ARN.
