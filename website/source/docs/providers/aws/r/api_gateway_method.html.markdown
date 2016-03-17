---
layout: "aws"
page_title: "AWS: aws_api_gateway_method"
sidebar_current: "docs-aws-resource-api-gateway-method"
description: |-
  Provides an HTTP Method for an API Gateway Resource.
---

# aws\_api\_gateway\_method

Provides an HTTP Method for an API Gateway Resource.

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

```

## Argument Reference

The following arguments are supported:

* `rest_api_id` - (Required) API Gateway ID
* `resource_id` - (Required) API Gateway Resource ID
* `http_method` - (Required) HTTP Method (GET, POST, PUT, DELETE, HEAD, OPTION)
* `authorization` - (Required) Type of authorization used for the method.
* `api_key_required` - (Optional) Specify if the method required an ApiKey
* `request_models` - (Optional) Specifies the Model resources used for the request's content type description
* `request_parameters` - (Optional) Represents  requests parameters that are sent with the backend request

