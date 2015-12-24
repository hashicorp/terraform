---
layout: "aws"
page_title: "AWS: aws_route53_health_check"
sidebar_current: "docs-aws-resource-route53-health-check"
description: |-
  Provides a Route53 health check.
---
# aws\_route53\_health\_check

Provides a Route53 health check.

## Example Usage

```
resource "aws_route53_health_check" "foo" {
  fqdn = "foobar.terraform.com"
  port = 80
  type = "HTTP"
  resource_path = "/"
  failure_threshold = "5"
  request_interval = "30"

  tags = {
    Name = "tf-test-health-check"
   }
}
```

## Argument Reference

The following arguments are supported:

* `fqdn` - (Optional) The fully qualified domain name of the endpoint to be checked.
* `ip_address` - (Optional) The IP address of the endpoint to be checked.
* `failure_threshold` - (Required) The number of consecutive health checks that an endpoint must pass or fail.
* `request_interval` - (Required) The number of seconds between the time that Amazon Route 53 gets a response from your endpoint and the time that it sends the next health-check request.
* `resource_path` - (Optional) The path that you want Amazon Route 53 to request when performing health checks.
* `search_string` - (Optional) String searched in respoonse body for check to considered healthy.
* `tags` - (Optional) A mapping of tags to assign to the health check.

At least one of either `fqdn` or `ip_address` must be specified.

