---
layout: "aws"
page_title: "AWS: aws_elb_account_id"
sidebar_current: "docs-aws-datasource-elb-account-id"
description: |-
  Get AWS Elastic Load Balancing Account ID
---

# aws\_elb\_account\_id

Use this data source to get the Account ID of the [AWS Elastic Load Balancing Account](http://docs.aws.amazon.com/elasticloadbalancing/latest/classic/enable-access-logs.html#attach-bucket-policy)
in a given region for the purpose of whitelisting in S3 bucket policy.

## Example Usage

```
data "aws_elb_account_id" "main" { }

resource "aws_s3_bucket" "elb_logs" {
    bucket = "my-elb-tf-test-bucket"
    acl = "private"
    policy = <<POLICY
{
  "Id": "Policy",
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": [
        "s3:PutObject"
      ],
      "Effect": "Allow",
      "Resource": "arn:aws:s3:::my-elb-tf-test-bucket/AWSLogs/*",
      "Principal": {
        "AWS": [
          "${data.aws_elb_account_id.main.id}"
        ]
      }
    }
  ]
}
POLICY
}

resource "aws_elb" "bar" {
  name = "my-foobar-terraform-elb"
  availability_zones = ["us-west-2a"]

  access_logs {
    bucket = "${aws_s3_bucket.elb_logs.bucket}"
    interval = 5
  }

  listener {
    instance_port = 8000
    instance_protocol = "http"
    lb_port = 80
    lb_protocol = "http"
  }
}
```

## Argument Reference

* `region` - (Optional) Region of a given AWS ELB Account


## Attributes Reference

* `id` - Account ID
