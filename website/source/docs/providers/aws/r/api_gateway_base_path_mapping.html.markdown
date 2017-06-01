---
layout: "aws"
page_title: "AWS: aws_api_gateway_base_path_mapping"
sidebar_current: "docs-aws-resource-api-gateway-base-path-mapping"
description: |-
  Connects a custom domain with a deployed API
---

# aws\_api\_gateway\_base\_path\_mapping

Connects a custom domain name registered via `aws_api_gateway_domain_name`
with a deployed API so that its methods can be called via the
custom domain name.

## Example Usage

```hcl
resource "aws_api_gateway_deployment" "example" {
  # See aws_api_gateway_rest_api_docs for how to create this
  rest_api_id = "${aws_api_gateway_rest_api.MyDemoAPI.id}"
  stage_name  = "live"
}

resource "aws_api_gateway_domain_name" "example" {
  domain_name = "example.com"

  certificate_name        = "example-api"
  certificate_body        = "${file("${path.module}/example.com/example.crt")}"
  certificate_chain       = "${file("${path.module}/example.com/ca.crt")}"
  certificate_private_key = "${file("${path.module}/example.com/example.key")}"
}

resource "aws_api_gateway_base_path_mapping" "test" {
  api_id      = "${aws_api_gateway_rest_api.MyDemoAPI.id}"
  stage_name  = "${aws_api_gateway_deployment.example.stage_name}"
  domain_name = "${aws_api_gateway_domain_name.example.domain_name}"
}
```

## Argument Reference

The following arguments are supported:

* `domain_name` - (Required) The already-registered domain name to connect the API to.
* `api_id` - (Required) The id of the API to connect.
* `stage_name` - (Optional) The name of a specific deployment stage to expose at the given path. If omitted, callers may select any stage by including its name as a path element after the base path.
* `base_path` - (Optional) Path segment that must be prepended to the path when accessing the API via this mapping. If omitted, the API is exposed at the root of the given domain.
