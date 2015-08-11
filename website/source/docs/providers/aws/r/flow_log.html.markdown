---
layout: "aws"
page_title: "AWS: aws_flow_log"
sidebar_current: "docs-aws-resource-flow-log"
description: |-
  Provides a VPC/Subnet/ENI Flow Log
---

# aws\_flow\_log

Provides a VPC/Subnet/ENI Flow Log to capture IP traffic for a specific network
interface, subnet, or VPC. Logs are sent to a CloudWatch Log Group.

```
resource "aws_flow_log" "test_flow_log" {
  # log_group_name needs to exist before hand
  # until we have a CloudWatch Log Group Resource
  log_group_name = "tf-test-log-group"
  iam_role_arn = "${aws_iam_role.test_role.arn}"
  vpc_id = "${aws_vpc.default.id}"
  traffic_type = "ALL"
}

resource "aws_iam_role" "test_role" {
    name = "test_role"
    assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "",
      "Effect": "Allow",
      "Principal": {
        "Service": "vpc-flow-logs.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
} 
EOF
}

resource "aws_iam_role_policy" "test_policy" {
    name = "test_policy"
    role = "${aws_iam_role.test_role.id}"
    policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": [
        "logs:CreateLogGroup",
        "logs:CreateLogStream",
        "logs:PutLogEvents",
        "logs:DescribeLogGroups",
        "logs:DescribeLogStreams"
      ],
      "Effect": "Allow",
      "Resource": "*"
    }
  ]
}   
EOF
}
```

## Argument Reference

The following arguments are supported:

* `log_group_name` - (Required) The name of the CloudWatch log group
* `iam_role_arn` - (Required) The ARN for the IAM role that's used to post flow
  logs to a CloudWatch Logs log group
* `vpc_id` - (Optional) VPC ID to attach to
* `subnet_id` - (Optional) Subnet ID to attach to
* `eni_id` - (Optional) Elastic Network Interface ID to attach to
* `traffic_type` - (Required) The type of traffic to capture. Valid values:
  `ACCEPT`,`REJECT`, `ALL`

## Attributes Reference

The following attributes are exported:

* `id` - The Flow Log ID
