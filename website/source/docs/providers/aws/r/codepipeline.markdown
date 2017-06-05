---
layout: "aws"
page_title: "AWS: aws_codepipeline"
sidebar_current: "docs-aws-resource-codepipeline"
description: |-
  Provides a CodePipeline
---

# aws\_codepipeline

Provides a CodePipeline.

~> **NOTE on `aws_codepipeline`:** - the `GITHUB_TOKEN` environment variable must be set if the GitHub provider is specified.

## Example Usage

```hcl
resource "aws_s3_bucket" "foo" {
  bucket = "test-bucket"
  acl    = "private"
}

resource "aws_iam_role" "foo" {
  name = "test-role"

  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Service": "codepipeline.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
EOF
}

resource "aws_iam_role_policy" "codepipeline_policy" {
  name = "codepipeline_policy"
  role = "${aws_iam_role.codepipeline_role.id}"
  policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect":"Allow",
      "Action": [
        "s3:GetObject",
        "s3:GetObjectVersion",
        "s3:GetBucketVersioning"
      ],
      "Resource": [
        "${aws_s3_bucket.foo.arn}",
        "${aws_s3_bucket.foo.arn}/*"
      ]
    },
    {
      "Effect": "Allow",
      "Action": [
        "codebuild:BatchGetBuilds",
        "codebuild:StartBuild"
      ],
      "Resource": "*"
    }
  ]
}
EOF
}

resource "aws_codepipeline" "foo" {
  name     = "tf-test-pipeline"
  role_arn = "${aws_iam_role.foo.arn}"

  artifact_store {
    location = "${aws_s3_bucket.foo.bucket}"
    type     = "S3"
  }

  stage {
    name = "Source"

    action {
      name             = "Source"
      category         = "Source"
      owner            = "ThirdParty"
      provider         = "GitHub"
      version          = "1"
      output_artifacts = ["test"]

      configuration {
        Owner      = "my-organization"
        Repo       = "test"
        Branch     = "master"
      }
    }
  }

  stage {
    name = "Build"

    action {
      name            = "Build"
      category        = "Build"
      owner           = "AWS"
      provider        = "CodeBuild"
      input_artifacts = ["test"]
      version         = "1"

      configuration {
        ProjectName = "test"
      }
    }
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the pipeline.
* `role_arn` - (Required) A service role Amazon Resource Name (ARN) that grants AWS CodePipeline permission to make calls to AWS services on your behalf.
* `artifact_store` (Required) An artifact_store block. Artifact stores are documented below.
* `stage` (Required) A stage block. Stages are documented below.


An `artifact_store` block supports the following arguments:

* `location` - (Required) The location where AWS CodePipeline stores artifacts for a pipeline, such as an S3 bucket.
* `type` - (Required) The type of the artifact store, such as Amazon S3
* `encryption_key` - (Optional) The encryption key AWS CodePipeline uses to encrypt the data in the artifact store, such as an AWS Key Management Service (AWS KMS) key. If you don't specify a key, AWS CodePipeline uses the default key for Amazon Simple Storage Service (Amazon S3).


A `stage` block supports the following arguments:

* `name` - (Required) The name of the stage.
* `action` - (Required) The action(s) to include in the stage. Defined as an `action` block below

A `action` block supports the following arguments:

* `category` - (Required) A category defines what kind of action can be taken in the stage, and constrains the provider type for the action. Possible values are `Approval`, `Build`, `Deploy`, `Invoke`, `Source` and `Test`.
* `owner` - (Required) The creator of the action being called. Possible values are `AWS`, `Custom` and `ThirdParty`.
* `name` - (Required) The action declaration's name.
* `provider` - (Required) The provider of the service being called by the action. Valid providers are determined by the action category. For example, an action in the Deploy category type might have a provider of AWS CodeDeploy, which would be specified as CodeDeploy.
* `version` - (Required) A string that identifies the action type.
* `configuration` - (Optional) A Map of the action declaration's configuration.
* `input_artifacts` - (Optional) A list of artifact names to be worked on.
* `output_artifacts` - (Optional) A list of artifact names to output. Output artifact names must be unique within a pipeline.
* `role_arn` - (Optional) The ARN of the IAM service role that will perform the declared action. This is assumed through the roleArn for the pipeline.
* `run_order` - (Optional) The order in which actions are run.

~> **Note:** The input artifact of an action must exactly match the output artifact declared in a preceding action, but the input artifact does not have to be the next action in strict sequence from the action that provided the output artifact. Actions in parallel can declare different output artifacts, which are in turn consumed by different following actions.

## Attributes Reference

The following attributes are exported:

* `id` - The codepipeline ID.

## Import

CodePipelines can be imported using the name, e.g.

```
$ terraform import aws_codepipeline.foo example
```
