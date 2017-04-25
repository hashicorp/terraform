---
layout: "aws"
page_title: "AWS: aws_api_gateway_account"
sidebar_current: "docs-aws-resource-api-gateway-account"
description: |-
  Provides a settings of an API Gateway Account.
---

# aws\_api\_gateway\_account

Provides a settings of an API Gateway Account. Settings is applied region-wide per `provider` block.

-> **Note:** As there is no API method for deleting account settings or resetting it to defaults, destroying this resource will keep your account settings intact

## Example Usage

```hcl
resource "aws_api_gateway_account" "demo" {
  cloudwatch_role_arn = "${aws_iam_role.cloudwatch.arn}"
}

resource "aws_iam_role" "cloudwatch" {
  name = "api_gateway_cloudwatch_global"

  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "",
      "Effect": "Allow",
      "Principal": {
        "Service": "apigateway.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
EOF
}

resource "aws_iam_role_policy" "cloudwatch" {
  name = "default"
  role = "${aws_iam_role.cloudwatch.id}"

  policy = <<EOF
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "logs:CreateLogGroup",
                "logs:CreateLogStream",
                "logs:DescribeLogGroups",
                "logs:DescribeLogStreams",
                "logs:PutLogEvents",
                "logs:GetLogEvents",
                "logs:FilterLogEvents"
            ],
            "Resource": "*"
        }
    ]
}
EOF
}
```

## Argument Reference

The following argument is supported:

* `cloudwatch_role_arn` - (Optional) The ARN of an IAM role for CloudWatch (to allow logging & monitoring).
	See more [in AWS Docs](https://docs.aws.amazon.com/apigateway/latest/developerguide/how-to-stage-settings.html#how-to-stage-settings-console).
	Logging & monitoring can be enabled/disabled and otherwise tuned on the API Gateway Stage level.

## Attribute Reference

The following attribute is exported:

* `throttle_settings` - Account-Level throttle settings. See exported fields below.

`throttle_settings` block exports the following:

* `burst_limit` - The absolute maximum number of times API Gateway allows the API to be called per second (RPS).
* `rate_limit` - The number of times API Gateway allows the API to be called per second on average (RPS).


## Import

API Gateway Accounts can be imported using the word `api-gateway-account`, e.g.

```
$ terraform import aws_api_gateway_account.demo api-gateway-account
```