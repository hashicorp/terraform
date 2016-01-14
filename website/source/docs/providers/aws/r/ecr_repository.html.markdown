---
layout: "aws"
page_title: "AWS: aws_ecr_repository"
sidebar_current: "docs-aws-resource-ecr-repository"
description: |-
  Provides an EC2 Container Registry Repository.
---

# aws\_ecr\_repository

Provides an EC2 Container Registry Repository.

~> **NOTE on ECR Availability**: The EC2 Container Registry has an [initial
launch region of
`us-east-1`](https://aws.amazon.com/blogs/aws/ec2-container-registry-now-generally-available/).
As more regions become available, they will be listed [in the AWS
Docs](https://docs.aws.amazon.com/general/latest/gr/rande.html#ecr_region)

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
