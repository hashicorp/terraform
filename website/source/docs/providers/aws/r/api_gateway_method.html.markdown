---
layout: "aws"
page_title: "AWS: aws_api_gateway_method"
sidebar_current: "docs-aws-resource-api-gateway-method"
description: |-
  Provides a HTTP Method for an API Gateway Resource.
---

# aws\_api\_gateway\_method

Provides a HTTP Method for an API Gateway Resource.

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

* `rest_api_id` - (Required) The ID of the associated REST API
* `resource_id` - (Required) The API resource ID
* `http_method` - (Required) The HTTP Method (`GET`, `POST`, `PUT`, `DELETE`, `HEAD`, `OPTION`)
* `authorization` - (Required) The type of authorization used for the method
* `api_key_required` - (Optional) Specify if the method requires an API key
* `request_models` - (Optional) A map of the API models used for the request's content type
  where key is the content type (e.g. `application/json`)
  and value is either `Error`, `Empty` (built-in models) or `aws_api_gateway_model`'s `name`.
* `request_parameters_in_json` - (Optional) A map written as a JSON string specifying
  the request query string parameters and headers that should be passed to the integration
