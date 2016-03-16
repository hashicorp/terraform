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

* `rest_api_id` - (Required) API Gateway ID
* `resource_id` - (Required) API Gateway Resource ID
* `http_method` - (Required) HTTP Method (GET, POST, PUT, DELETE, HEAD, OPTION)
* `type` - (Required) Specifies a put integration input's type (HTTP, MOCK, AWS)
* `uri` - (Optional) Input's  Uniform Resource Identifier (HTTP, AWS)
* `integration_http_method` - (Optional) Integration HTTP Method (GET, POST, PUT, DELETE, HEAD, OPTION)

