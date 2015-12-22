---
layout: "aws"
page_title: "AWS: aws_ecr_repository"
sidebar_current: "docs-aws-resource-ecr-repository"
description: |-
  Provides an ECR Repository.
---

# aws\_ecr\_repository

Provides an ECR repository.

## Example Usage

```
resource "aws_ecr_repository" "foo" {
  name = "bar"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) Name of the repository.

## Attributes Reference

The following attributes are exported:

* `arn` - Full ARN of the repository.
* `name` - The name of the repository.
* `registry_id` - The registry ID where the repository was created.
