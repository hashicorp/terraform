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
resource "aws_api_gateway_rest_api" "MyDemoAPI" {
  name = "MyDemoAPI"
  description = "This is my API for demonstration purposes"
}

resource "aws_api_gateway_authorizer" "test" {
  rest_api_id = "${aws_api_gateway_rest_api.test.id}"
  name = "my_authorizer"
  authorizer_uri = "arn:aws:apigateway:eu-west-1:lambda:path/2015-03-31/functions/arn:aws:lambda:eu-west-1:123456789012:function:auth_function/invocations"
  credentials = "arn:aws:iam::123456789012:role/lambda_auth"
  identity_source = "method.request.header.Authorization"
  type = "TOKEN"
  result_in_ttl = 300
}
```

## Argument Reference

The following arguments are supported:

* `type` - (Required) The type of the authorizer.
* `rest_api_id` - (Required) The ID of the associated REST API
* `name` - (Required) The name of the authorizer.
* `authorizer_uri` - (Required) Specifies the authorizer's Uniform Resource Identifier (URI).
* `identity_source` - (Required) The source of the identity in an incoming request.
* `identity_validation_expression` - (Optional) The TTL of cached authorizer results.
* `result_in_ttl` - (Optional) The TTL of cached authorizer results.
* `credentials` - (Optional) Specifies the credentials required for the authorizer, if any.

## Attribute Reference

The following attributes are exported:

* `id` - The ID of the authorizer
