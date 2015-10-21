---
layout: "aws"
page_title: "AWS: aws_codedeploy_app"
sidebar_current: "docs-aws-resource-codedeploy-app"
description: |-
  Provides a CodeDeploy application.
---

# aws\_codedeploy\_app

Provides a CodeDeploy application to be used as a basis for deployments

## Example Usage

```
resource "aws_codedeploy_app" "foo" {
  name = "foo"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the application.

## Attribute Reference

The following arguments are exported:

* `id` - Amazon's assigned ID for the application.
* `name` - The application's name.
