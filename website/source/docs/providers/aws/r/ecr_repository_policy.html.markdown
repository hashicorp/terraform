---
layout: "aws"
page_title: "AWS: aws_ecr_repository_policy"
sidebar_current: "docs-aws-resource-ecr-repository-policy"
description: |-
  Provides an ECR Repository Policy.
---

# aws\_ecr\_repository\_policy

Provides an ECR repository policy.

Note that currently only one policy may be applied to a repository.

~> **NOTE on ECR Availability**: The EC2 Container Registry has an [initial
launch region of
`us-east-1`](https://aws.amazon.com/blogs/aws/ec2-container-registry-now-generally-available/).
As more regions become available, they will be listed [in the AWS
Docs](https://docs.aws.amazon.com/general/latest/gr/rande.html#ecr_region)

## Example Usage

```
resource "aws_ecr_repository" "foo" {
  repository = "bar"
}

resource "aws_ecr_repository_policy" "foopolicy" {
  repository = "${aws_ecr_repository.foo.name}"
  policy = <<EOF
{
    "Version": "2008-10-17",
    "Statement": [
        {
            "Sid": "new policy",
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
    ]
}
EOF
}
```

## Argument Reference

The following arguments are supported:

* `repository` - (Required) Name of the repository to apply the policy.
* `policy` - (Required) The policy document. This is a JSON formatted string.

## Attributes Reference

The following attributes are exported:

* `repository` - The name of the repository.
* `registry_id` - The registry ID where the repository was created.
