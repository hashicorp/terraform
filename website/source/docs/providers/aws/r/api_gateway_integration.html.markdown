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
  For HTTP integrations, the URI must be a fully formed, encoded HTTP(S) URL according to the RFC-3986 specification . For AWS integrations, the URI should be of the form `arn:aws:apigateway:{region}:{subdomain.service|service}:{path|action}/{service_api}`. `region`, `subdomain` and `service` are used to determine the right endpoint.
  e.g. `arn:aws:apigateway:eu-west-1:lambda:path/2015-03-31/functions/arn:aws:lambda:eu-west-1:012345678901:function:my-func/invocations`
* `credentials` - (Optional) The credentials required for the integration. For `AWS` integrations, 2 options are available. To specify an IAM Role for Amazon API Gateway to assume, use the role's ARN. To require that the caller's identity be passed through from the request, specify the string `arn:aws:iam::\*:user/\*`.
* `integration_http_method` - (Optional) The integration HTTP method
  (`GET`, `POST`, `PUT`, `DELETE`, `HEAD`, `OPTION`). **Required** if `type` is `AWS` or `HTTP`.
  Not all methods are compatible with all `AWS` integrations.
  e.g. Lambda function [can only be invoked](https://github.com/awslabs/aws-apigateway-importer/issues/9#issuecomment-129651005) via `POST`.
* `request_templates` - (Optional) A map of the integration's request templates.
* `request_parameters_in_json` - (Optional) A map written as a JSON string specifying
  the request query string parameters and headers that should be passed to the
  backend responder
