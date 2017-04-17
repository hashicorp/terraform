---
layout: "aws"
page_title: "AWS: aws_billing_service_account"
sidebar_current: "docs-aws-datasource-billing-service-account"
description: |-
  Get AWS Billing Service Account
---

# aws\_billing\_service\_account

Use this data source to get the Account ID of the [AWS Billing and Cost Management Service Account](http://docs.aws.amazon.com/awsaccountbilling/latest/aboutv2/billing-getting-started.html#step-2) for the purpose of whitelisting in S3 bucket policy.

## Example Usage

```hcl
data "aws_billing_service_account" "main" {}

resource "aws_s3_bucket" "billing_logs" {
  bucket = "my-billing-tf-test-bucket"
  acl    = "private"

  policy = <<POLICY
{
  "Id": "Policy",
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": [
        "s3:GetBucketAcl", "s3:GetBucketPolicy"
      ],
      "Effect": "Allow",
      "Resource": "arn:aws:s3:::my-billing-tf-test-bucket",
      "Principal": {
        "AWS": [
          "${data.aws_billing_service_account.main.id}"
        ]
      }
    },
    {
      "Action": [
        "s3:PutObject"
      ],
      "Effect": "Allow",
      "Resource": "arn:aws:s3:::my-billing-tf-test-bucket/AWSLogs/*",
      "Principal": {
        "AWS": [
          "${data.aws_billing_service_account.main.id}"
        ]
      }
    }
  ]
}
POLICY
}
```


## Attributes Reference

* `id` - The ID of the AWS billing service account.
* `arn` - The ARN of the AWS billing service account.
