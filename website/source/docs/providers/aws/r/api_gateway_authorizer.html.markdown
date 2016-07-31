---
layout: "aws"
page_title: "AWS: aws_api_gateway_authorizer"
sidebar_current: "docs-aws-resource-api-gateway-authorizer"
description: |-
  Provides an API Gateway Authorizer.
---

# aws\_api\_gateway\_authorizer

Provides an API Gateway Authorizer.

## Example Usage

```
resource "aws_api_gateway_authorizer" "demo" {
  name = "demo"
  rest_api_id = "${aws_api_gateway_rest_api.demo.id}"
  authorizer_uri = "arn:aws:apigateway:region:lambda:path/2015-03-31/functions/${aws_lambda_function.authorizer.arn}/invocations"
  authorizer_credentials = "${aws_iam_role.invocation_role.arn}"
}

resource "aws_api_gateway_rest_api" "demo" {
  name = "auth-demo"
}

resource "aws_iam_role" "invocation_role" {
  name = "api_gateway_auth_invocation"
  path = "/"
  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "apigateway.amazonaws.com"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
EOF
}

resource "aws_iam_role_policy" "invocation_policy" {
  name = "default"
  role = "${aws_iam_role.invocation_role.id}"
  policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "lambda:InvokeFunction",
      "Effect": "Allow",
      "Resource": "${aws_lambda_function.authorizer.arn}"
    }
  ]
}
EOF
}

resource "aws_iam_role" "lambda" {
  name = "demo-lambda"
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

resource "aws_lambda_function" "authorizer" {
  filename = "lambda-function.zip"
  source_code_hash = "${base64sha256(file("lambda-function.zip"))}"
  function_name = "api_gateway_authorizer"
  role = "${aws_iam_role.lambda.arn}"
  handler = "exports.example"
}
```

## Argument Reference

The following arguments are supported:

* `authorizer_uri` - (Required) The authorizer's Uniform Resource Identifier (URI).
	For `TOKEN` type, this must be a well-formed Lambda function URI in the form of
	`arn:aws:apigateway:{region}:lambda:path/{service_api}`. e.g. `arn:aws:apigateway:region:lambda:path/2015-03-31/functions/arn:aws:lambda:us-west-2:012345678912:function:my-function/invocations`
* `name` - (Required) The name of the authorizer
* `rest_api_id` - (Required) The ID of the associated REST API
* `identity_source` - (Optional) The source of the identity in an incoming request.
	Defaults to `method.request.header.Authorization`.
* `type` - (Optional) The type of the authorizer. `TOKEN` is currently the only allowed value.
	Defaults to `TOKEN`.
* `authorizer_credentials` - (Optional) The credentials required for the authorizer.
	To specify an IAM Role for API Gateway to assume, use the IAM Role ARN.
* `authorizer_result_ttl_in_seconds` - (Optional) The TTL of cached authorizer results in seconds.
	Defaults to `300`.
* `identity_validation_expression` - (Optional) A validation expression for the incoming identity.
	For `TOKEN` type, this value should be a regular expression. The incoming token from the client is matched
	against this expression, and will proceed if the token matches. If the token doesn't match,
	the client receives a 401 Unauthorized response.
