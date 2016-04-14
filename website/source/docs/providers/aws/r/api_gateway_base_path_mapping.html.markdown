---
layout: "aws"
page_title: "AWS: aws_api_gateway_base_path_mapping"
sidebar_current: "docs-aws-resource-api-gateway-base-path-mapping"
description: |-
  Provides an API Gateway Base Path Mapping
---

# aws\_api\_gateway\_base\_path\_mapping

Provides an API Gateway Base Path Mapping.

## Example Usage
```
resource "aws_api_gateway_rest_api" "MyDemoAPI" {
  name = "MyDemoAPI"
  description = "This is my API for demonstration purposes"
}

resource "aws_api_gateway_resource" "test" {
  rest_api_id = "${aws_api_gateway_rest_api.MyDemoAPI.id}"
  parent_id = "${aws_api_gateway_rest_api.MyDemoAPI.root_resource_id}"
  path_part = "test"
}

resource "aws_api_gateway_method" "test" {
  rest_api_id = "${aws_api_gateway_rest_api.MyDemoAPI.id}"
  resource_id = "${aws_api_gateway_resource.test.id}"
  http_method = "GET"
  authorization = "NONE"
}

resource "aws_api_gateway_deployment" "MyDemoDeployment" {
  depends_on = ["aws_api_gateway_integration.test"]

  rest_api_id = "${aws_api_gateway_rest_api.MyDemoAPI.id}"

  variables = {
    "answer" = "42"
  }
}

resource "aws_api_gateway_domain" "MyDemoDomain" {
  domain_name = "your.doma.in"
  certificate_name = "demo_api_cert"
  certificate_body = "${file("certs/your.doma.in.crt")}"
  certificate_private_key = "${file("certs/your.doma.in.pkey")}"
  certificate_chain = "${file("certs/your.doma.in.chain")}"
}

resource "aws_api_gateway_base_path_mapping" "test" {
  api_id = "${aws_api_gateway_rest_api.MyDemoAPI.id}"
  path = ""
  stage = "${aws_api_gateway_deployment.MyDemoDeployment.stage_name}"
  domain_name = "${aws_api_gateway_domain.MyDemoDomain.id}"
}

```

## Argument Reference

The following arguments are supported:

* `domain_name` - (Required) The domain name of the BasePathMapping resource to create.
* `api_id` - (Required) The name of the API that you want to apply this mapping to.
* `stage` - (Required) The name of the API's stage that you want to use for this mapping. Leave this blank if you do not want callers to explicitly specify the stage name after any base path name.
* `base_path` - (Required) The base path name that callers of the API must provide as part of the URL after the domain name. This value must be unique for all of the mappings across a single API. Use an empty string if you do not want callers to specify a base path name after the domain name.
