---
layout: "aws"
page_title: "AWS: aws_ecr_repository_policy_statement"
sidebar_current: "docs-aws-resource-ecr-repository-policy-statement"
description: |-
  Provides an ECR Repository Policy.
---

# aws\_ecr\_repository\_policy\_statement

Provides a single ECR repository policy statement. This is useful when you need to inject statements from different modules to the same ecr_repository, as only one policy document is allowed by AWS.

Please note that this can't be used with aws_ecr_repository_policy, as one might override the other.

~> **NOTE on ECR Availability**: The EC2 Container Registry is not yet rolled out
in all regions - available regions are listed  
[the AWS Docs](https://docs.aws.amazon.com/general/latest/gr/rande.html#ecr_region).

## Example Usage

```
resource "aws_ecr_repository" "foo" {
  name = "bar"
}

resource "aws_ecr_repository_policy_statement" "foostatement" {
  repository = "${aws_ecr_repository.foo.name}"
  sid = "new policy"

  statement = <<EOF
{
    "Effect": "Allow",
    "Principal": "*",
    "Action": [
        "ecr:GetDownloadUrlForLayer",
        "ecr:BatchGetImage",
        "ecr:BatchCheckLayerAvailability",
        "ecr:PutImage",
        "ecr:InitiateLayerUpload",
        "ecr:UploadLayerPart",
        "ecr:CompleteLayerUpload",
        "ecr:DescribeRepositories",
        "ecr:GetRepositoryPolicy",
        "ecr:ListImages",
        "ecr:DeleteRepository",
        "ecr:BatchDeleteImage",
        "ecr:SetRepositoryPolicy",
        "ecr:DeleteRepositoryPolicy"
    ]
}
EOF
}
```

## Argument Reference

The following arguments are supported:

* `sid` - (Optional) Sid of the statement to apply to the repository police. If omitted, Terraform will
assign a random, unique name.
* `sid_prefix` - (Optional, Forces new resource) Creates a unique Sid beginning with the specified
  prefix. Conflicts with `sid`.
* `repository` - (Required) Name of the repository to apply the policy.
* `statement` - (Required) The statement definition. This is a JSON formatted string.

## Attributes Reference

The following attributes are exported:

* `sid` - The Sid of the statement.
* `repository` - The name of the repository.
* `registry_id` - The registry ID where the repository was created.
