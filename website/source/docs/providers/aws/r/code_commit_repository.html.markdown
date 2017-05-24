---
layout: "aws"
page_title: "AWS: aws_codecommit_repository"
sidebar_current: "docs-aws-resource-codecommit-repository"
description: |-
  Provides a CodeCommit Repository Resource.
---

# aws\_codecommit\_repository

Provides a CodeCommit Repository Resource.

~> **NOTE on CodeCommit Availability**: The CodeCommit is not yet rolled out
in all regions - available regions are listed
[the AWS Docs](https://docs.aws.amazon.com/general/latest/gr/rande.html#codecommit_region).

## Example Usage

```hcl
resource "aws_codecommit_repository" "test" {
  repository_name = "MyTestRepository"
  description     = "This is the Sample App Repository"
}
```

## Argument Reference

The following arguments are supported:

* `repository_name` - (Required) The name for the repository. This needs to be less than 100 characters.
* `description` - (Optional) The description of the repository. This needs to be less than 1000 characters
* `default_branch` - (Optional) The default branch of the repository. The branch specified here needs to exist.

## Attributes Reference

The following attributes are exported:

* `repository_id` - The ID of the repository
* `arn` - The ARN of the repository
* `clone_url_http` - The URL to use for cloning the repository over HTTPS.
* `clone_url_ssh` - The URL to use for cloning the repository over SSH.