---
layout: "aws"
page_title: "AWS: aws_iam_policy_document"
sidebar_current: "docs-aws-datasource-iam-policy-document"
description: |-
  Generates an IAM policy document in JSON format
---

# aws\_iam\_policy\_document

Generates an IAM policy document in JSON format.

This is a data source which can be used to construct a JSON representation of
an IAM policy document, for use with resources which expect policy documents,
such as the `aws_iam_policy` resource.

```hcl
data "aws_iam_policy_document" "example" {
  statement {
    sid = "1"

    actions = [
      "s3:ListAllMyBuckets",
      "s3:GetBucketLocation",
    ]

    resources = [
      "arn:aws:s3:::*",
    ]
  }

  statement {
    actions = [
      "s3:ListBucket",
    ]

    resources = [
      "arn:aws:s3:::${var.s3_bucket_name}",
    ]

    condition {
      test     = "StringLike"
      variable = "s3:prefix"

      values = [
        "",
        "home/",
        "home/&{aws:username}/",
      ]
    }
  }

  statement {
    actions = [
      "s3:*",
    ]

    resources = [
      "arn:aws:s3:::${var.s3_bucket_name}/home/&{aws:username}",
      "arn:aws:s3:::${var.s3_bucket_name}/home/&{aws:username}/*",
    ]
  }
}

resource "aws_iam_policy" "example" {
  name   = "example_policy"
  path   = "/"
  policy = "${data.aws_iam_policy_document.example.json}"
}
```

Using this data source to generate policy documents is *optional*. It is also
valid to use literal JSON strings within your configuration, or to use the
`file` interpolation function to read a raw JSON policy document from a file.

## Argument Reference

The following arguments are supported:

* `policy_id` (Optional) - An ID for the policy document.
* `statement` (Required) - A nested configuration block (described below)
  configuring one *statement* to be included in the policy document.

Each document configuration must have one or more `statement` blocks, which
each accept the following arguments:

* `sid` (Optional) - An ID for the policy statement.
* `effect` (Optional) - Either "Allow" or "Deny", to specify whether this
  statement allows or denies the given actions. The default is "Allow".
* `actions` (Optional) - A list of actions that this statement either allows
  or denies. For example, ``["ec2:RunInstances", "s3:*"]``.
* `not_actions` (Optional) - A list of actions that this statement does *not*
  apply to. Used to apply a policy statement to all actions *except* those
  listed.
* `resources` (Optional) - A list of resource ARNs that this statement applies
  to. This is required by AWS if used for an IAM policy.
* `not_resources` (Optional) - A list of resource ARNs that this statement
  does *not* apply to. Used to apply a policy statement to all resources
  *except* those listed.
* `principals` (Optional) - A nested configuration block (described below)
  specifying a resource (or resource pattern) to which this statement applies.
* `not_principals` (Optional) - Like `principals` except gives resources that
  the statement does *not* apply to.
* `condition` (Optional) - A nested configuration block (described below)
  that defines a further, possibly-service-specific condition that constrains
  whether this statement applies.

Each policy may have either zero or more `principals` blocks or zero or more
`not_principals` blocks, both of which each accept the following arguments:

* `type` (Required) The type of principal. For AWS accounts this is "AWS".
* `identifiers` (Required) List of identifiers for principals. When `type`
  is "AWS", these are IAM user or role ARNs.

Each policy statement may have zero or more `condition` blocks, which each
accept the following arguments:

* `test` (Required) The name of the
  [IAM condition type](http://docs.aws.amazon.com/IAM/latest/UserGuide/reference_policies_elements.html#AccessPolicyLanguage_ConditionType)
  to evaluate.
* `variable` (Required) The name of a
  [Context Variable](http://docs.aws.amazon.com/IAM/latest/UserGuide/reference_policies_elements.html#AvailableKeys)
  to apply the condition to. Context variables may either be standard AWS
  variables starting with `aws:`, or service-specific variables prefixed with
  the service name.
* `values` (Required) The values to evaluate the condition against. If multiple
  values are provided, the condition matches if at least one of them applies.
  (That is, the tests are combined with the "OR" boolean operation.)

When multiple `condition` blocks are provided, they must *all* evaluate to true
for the policy statement to apply. (In other words, the conditions are combined
with the "AND" boolean operation.)

## Context Variable Interpolation

The IAM policy document format allows context variables to be interpolated
into various strings within a statement. The native IAM policy document format
uses `${...}`-style syntax that is in conflict with Terraform's interpolation
syntax, so this data source instead uses `&{...}` syntax for interpolations that
should be processed by AWS rather than by Terraform.

## Attributes Reference

The following attribute is exported:

* `json` - The above arguments serialized as a standard JSON policy document.

## Example with Multiple Principals

Showing how you can use this as an assume role policy as well as showing how you can specify multiple principal blocks with different types.

```hcl
data "aws_iam_policy_document" "event_stream_bucket_role_assume_role_policy" {
  statement {
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["firehose.amazonaws.com"]
    }

    principals {
      type        = "AWS"
      identifiers = ["${var.trusted_role_arn}"]
    }
  }
}
```
