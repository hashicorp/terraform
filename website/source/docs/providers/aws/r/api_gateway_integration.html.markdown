---
layout: "aws"
page_title: "AWS: aws_api_gateway_integration"
sidebar_current: "docs-aws-resource-api-gateway-integration"
description: |-
  Provides an HTTP Method Integration for an API Gateway Resource.
---

# aws\_api\_gateway\_integration

Provides an HTTP Method Integration for an API Gateway Resource.

## Example Usage

```
resource "aws_api_gateway_rest_api" "MyDemoAPI" {
  name = "MyDemoAPI"
  description = "This is my API for demonstration purposes"
}

resource "aws_api_gateway_resource" "MyDemoResource" {
  rest_api_id = "${aws_api_gateway_rest_api.MyDemoAPI.id}"
  parent_id = "${aws_api_gateway_rest_api.MyDemoAPI.root_resource_id}"
  path_part = "mydemoresource"
}

resource "aws_api_gateway_method" "MyDemoMethod" {
  rest_api_id = "${aws_api_gateway_rest_api.MyDemoAPI.id}"
  resource_id = "${aws_api_gateway_resource.MyDemoResource.id}"
  http_method = "GET"
  authorization = "NONE"
}

resource "aws_api_gateway_integration" "MyDemoIntegration" {
  rest_api_id = "${aws_api_gateway_rest_api.MyDemoAPI.id}"
  resource_id = "${aws_api_gateway_resource.MyDemoResource.id}"
  http_method = "${aws_api_gateway_method.MyDemoMethod.http_method}"
  type = "MOCK"
}
```

## Argument Reference

The following arguments are supported:

* `rest_api_id` - (Required) The ID of the associated REST API
* `resource_id` - (Required) The API resource ID
* `http_method` - (Required) The HTTP method (`GET`, `POST`, `PUT`, `DELETE`, `HEAD`, `OPTION`)
* `type` - (Required) The integration input's type (HTTP, MOCK, AWS)
* `uri` - (Optional) The input's URI (HTTP, AWS). **Required** if `type` is `HTTP` or `AWS`.
* `credentials` - (Optional) The credentials required for the integration. For `AWS` integrations, 2 options are available. To specify an IAM Role for Amazon API Gateway to assume, use the role's ARN. To require that the caller's identity be passed through from the request, specify the string `arn:aws:iam::\*:user/\*`.
* `integration_http_method` - (Optional) The integration HTTP method
  (`GET`, `POST`, `PUT`, `DELETE`, `HEAD`, `OPTION`).
* `request_templates` - (Optional) A map of the integration's request templates.
