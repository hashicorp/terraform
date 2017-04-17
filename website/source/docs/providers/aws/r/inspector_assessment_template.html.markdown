---
layout: "aws"
page_title: "AWS: aws_inspector_assessment_template"
sidebar_current: "docs-aws-resource-inspector-assessment-template"
description: |-
  Provides a Inspector assessment template.
---

# aws\_inspector\_assessment\_template

Provides a Inspector assessment template

## Example Usage

```hcl
resource "aws_inspector_assessment_template" "foo" {
  name       = "bar template"
  target_arn = "${aws_inspector_assessment_target.foo.arn}"
  duration   = 3600

  rules_package_arns = [
    "arn:aws:inspector:us-west-2:758058086616:rulespackage/0-9hgA516p",
    "arn:aws:inspector:us-west-2:758058086616:rulespackage/0-H5hpSawc",
    "arn:aws:inspector:us-west-2:758058086616:rulespackage/0-JJOtZiqQ",
    "arn:aws:inspector:us-west-2:758058086616:rulespackage/0-vg5GGHSD",
  ]
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the assessment template.
* `target_arn` - (Required) The assessment target ARN to attach the template to.
* `duration` - (Required) The duration of the inspector run.
* `rules_package_arns` - (Required) The rules to be used during the run.

## Attributes Reference

The following attributes are exported:

* `arn` - The template assessment ARN.
