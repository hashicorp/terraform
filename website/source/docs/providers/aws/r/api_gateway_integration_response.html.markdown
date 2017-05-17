---
layout: "aws"
page_title: "AWS: aws_api_gateway_integration_response"
sidebar_current: "docs-aws-resource-api-gateway-integration-response"
description: |-
  Provides an HTTP Method Integration Response for an API Gateway Resource.
---

# aws\_api\_gateway\_integration\_response

Provides an HTTP Method Integration Response for an API Gateway Resource.

-> **Note:** Depends on having `aws_api_gateway_integration` inside your rest api. To ensure this
you might need to add an explicit `depends_on` for clean runs.

## Example Usage

```hcl
resource "aws_api_gateway_rest_api" "MyDemoAPI" {
  name        = "MyDemoAPI"
  description = "This is my API for demonstration purposes"
}

resource "aws_api_gateway_resource" "MyDemoResource" {
  rest_api_id = "${aws_api_gateway_rest_api.MyDemoAPI.id}"
  parent_id   = "${aws_api_gateway_rest_api.MyDemoAPI.root_resource_id}"
  path_part   = "mydemoresource"
}

resource "aws_api_gateway_method" "MyDemoMethod" {
  rest_api_id   = "${aws_api_gateway_rest_api.MyDemoAPI.id}"
  resource_id   = "${aws_api_gateway_resource.MyDemoResource.id}"
  http_method   = "GET"
  authorization = "NONE"
}

resource "aws_api_gateway_integration" "MyDemoIntegration" {
  rest_api_id = "${aws_api_gateway_rest_api.MyDemoAPI.id}"
  resource_id = "${aws_api_gateway_resource.MyDemoResource.id}"
  http_method = "${aws_api_gateway_method.MyDemoMethod.http_method}"
  type        = "MOCK"
}

resource "aws_api_gateway_method_response" "200" {
  rest_api_id = "${aws_api_gateway_rest_api.MyDemoAPI.id}"
  resource_id = "${aws_api_gateway_resource.MyDemoResource.id}"
  http_method = "${aws_api_gateway_method.MyDemoMethod.http_method}"
  status_code = "200"
}

resource "aws_api_gateway_integration_response" "MyDemoIntegrationResponse" {
  rest_api_id = "${aws_api_gateway_rest_api.MyDemoAPI.id}"
  resource_id = "${aws_api_gateway_resource.MyDemoResource.id}"
  http_method = "${aws_api_gateway_method.MyDemoMethod.http_method}"
  status_code = "${aws_api_gateway_method_response.200.status_code}"

  # Transforms the backend JSON response to XML
  response_templates {
    "application/xml" = <<EOF
#set($inputRoot = $input.path('$'))
<?xml version="1.0" encoding="UTF-8"?>
<message>
    $inputRoot.body
</message>
EOF
  }
}
```

## Argument Reference

The following arguments are supported:

* `rest_api_id` - (Required) The ID of the associated REST API
* `resource_id` - (Required) The API resource ID
* `http_method` - (Required) The HTTP method (`GET`, `POST`, `PUT`, `DELETE`, `HEAD`, `OPTION`, `ANY`)
* `status_code` - (Required) The HTTP status code
* `selection_pattern` - (Optional) Specifies the regular expression pattern used to choose
  an integration response based on the response from the backend.
  If the backend is an `AWS` Lambda function, the AWS Lambda function error header is matched.
  For all other `HTTP` and `AWS` backends, the HTTP status code is matched.
* `response_templates` - (Optional) A map specifying the templates used to transform the integration response body
* `response_parameters` - (Optional) A map of response parameters that can be read from the backend response.
  For example: `response_parameters = { "method.response.header.X-Some-Header" = "integration.response.header.X-Some-Other-Header" }`,
* `response_parameters_in_json` - **Deprecated**, use `response_parameters` instead.
* `content_handling` - (Optional) Specifies how to handle request payload content type conversions. Supported values are `CONVERT_TO_BINARY` and `CONVERT_TO_TEXT`. If this property is not defined, the response payload will be passed through from the integration response to the method response without modification.
