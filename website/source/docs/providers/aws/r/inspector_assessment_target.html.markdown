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

```hcl
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
* `resource_group_arn` (Required )- The resource group ARN stating tags for instance matching.

## Attributes Reference

The following attributes are exported:

* `arn` - The target assessment ARN.
