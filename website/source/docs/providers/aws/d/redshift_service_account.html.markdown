---
layout: "aws"
page_title: "AWS: aws_redshift_service_account"
sidebar_current: "docs-aws-datasource-redshift-service-account"
description: |-
  Get AWS Redshift Service Account ID for storing audit data in S3.
---

# aws\_redshift\_service\_account

Use this data source to get the Service Account ID of the [AWS Redshift Account](http://docs.aws.amazon.com/redshift/latest/mgmt/db-auditing.html#db-auditing-enable-logging)
in a given region for the purpose of allowing Redshift to store audit data in S3.

## Example Usage

```hcl
data "aws_redshift_service_account" "main" {}

resource "aws_s3_bucket" "bucket" {
  bucket        = "tf-redshift-logging-test-bucket"
  force_destroy = true

  policy = <<EOF
{
	"Version": "2008-10-17",
	"Statement": [
		{
        			"Sid": "Put bucket policy needed for audit logging",
        			"Effect": "Allow",
        			"Principal": {
        				"AWS": "arn:aws:iam:${data.aws_redshift_service_account.main.id}:user/logs"
        			},
        			"Action": "s3:PutObject",
        			"Resource": "arn:aws:s3:::tf-redshift-logging-test-bucket/*"
        		},
        		{
        			"Sid": "Get bucket policy needed for audit logging ",
        			"Effect": "Allow",
        			"Principal": {
        				"AWS": "arn:aws:iam:${data.aws_redshift_service_account.main.id}:user/logs"
        			},
        			"Action": "s3:GetBucketAcl",
        			"Resource": "arn:aws:s3:::tf-redshift-logging-test-bucket"
        		}
	]
}
EOF
}
```

## Argument Reference

* `region` - (Optional) Name of the Region whose Redshift account id is desired. If not specified, default's to the region from the AWS provider configuration.


## Attributes Reference

* `id` - The ID of the Redshift service Account in the selected region.
