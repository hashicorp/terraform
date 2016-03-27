---
layout: "aws"
page_title: "AWS: aws_lambda_permission"
sidebar_current: "docs-aws-resource-lambda-permission"
description: |-
  Creates a Lambda function permission.
---

# aws\_lambda\_permission

Creates a Lambda permission to allow external sources invoking the Lambda function
(e.g. CloudWatch Event Rule, SNS or S3).

## Example Usage

```
resource "aws_lambda_permission" "allow_cloudwatch" {
    statement_id = "AllowExecutionFromCloudWatch"
    action = "lambda:InvokeFunction"
    function_name = "${aws_lambda_function.test_lambda.arn}"
    principal = "events.amazonaws.com"
    source_account = "111122223333"
    source_arn = "arn:aws:events:eu-west-1:111122223333:rule/RunDaily"
    qualifier = "${aws_lambda_alias.test_alias.name}"
}

resource "aws_lambda_alias" "test_alias" {
    name = "testalias"
    description = "a sample description"
    function_name = "${aws_lambda_function.test_lambda.arn}"
    function_version = "$LATEST"
}

resource "aws_lambda_function" "test_lambda" {
    filename = "lambdatest.zip"
    function_name = "lambda_function_name"
    role = "${aws_iam_role.iam_for_lambda.arn}"
    handler = "exports.handler"
}

resource "aws_iam_role" "iam_for_lambda" {
    name = "iam_for_lambda"
    assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "lambda.amazonaws.com"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
EOF
}
```

## Usage with SNS

```
resource "aws_lambda_permission" "with_sns" {
    statement_id = "AllowExecutionFromSNS"
    action = "lambda:InvokeFunction"
    function_name = "${aws_lambda_function.my-func.arn}"
    principal = "sns.amazonaws.com"
    source_arn = "${aws_sns_topic.default.arn}"
}

resource "aws_sns_topic" "default" {
  name = "call-lambda-maybe"
}

resource "aws_sns_topic_subscription" "lambda" {
    topic_arn = "${aws_sns_topic.default.arn}"
    protocol  = "lambda"
    endpoint  = "${aws_lambda_function.func.arn}"
}

resource "aws_lambda_function" "func" {
    filename = "lambdatest.zip"
    function_name = "lambda_called_from_sns"
    role = "${aws_iam_role.default.arn}"
    handler = "exports.handler"
}

resource "aws_iam_role" "default" {
    name = "iam_for_lambda_with_sns"
    assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "lambda.amazonaws.com"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
EOF
}
```

## Argument Reference

 * `action` - (Required) The AWS Lambda action you want to allow in this statement. (e.g. `lambda:InvokeFunction`)
 * `function_name` - (Required) Name of the Lambda function whose resource policy you are updating
 * `principal` - (Required) The principal who is getting this permission.
 	e.g. `s3.amazonaws.com`, an AWS account ID, or any valid AWS service principal
 	such as `events.amazonaws.com` or `sns.amazonaws.com`.
 * `statement_id` - (Required) A unique statement identifier.
 * `qualifier` - (Optional) Query parameter to specify function version or alias name.
 	The permission will then apply to the specific qualified ARN.
 	e.g. `arn:aws:lambda:aws-region:acct-id:function:function-name:2`
 * `source_account` - (Optional) The AWS account ID (without a hyphen) of the source owner.
 * `source_arn` - (Optional) When granting Amazon S3 permission to invoke your function,
 	you should specify this field with the bucket Amazon Resource Name (ARN) as its value.
 	This ensures that only events generated from the specified bucket can invoke the function.
