---
layout: "aws"
page_title: "AWS: aws_alb"
sidebar_current: "docs-aws-resource-alb"
description: |-
  Provides an Application Load Balancer resource.
---

# aws\_alb

Provides an Application Load Balancer resource.

## Example Usage

```
# Create a new load balancer
resource "aws_alb" "test" {
  name            = "test-alb-tf"
  internal        = false
  security_groups = ["${aws_security_group.alb_sg.id}"]
  subnets         = ["${aws_subnet.public.*.id}"]

  enable_deletion_protection = true

  access_logs {
    bucket = "${aws_s3_bucket.alb_logs.bucket}"
    prefix = "test-alb"
  }

  tags {
    Environment = "production"
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Optional) The name of the ALB. This name must be unique within your AWS account, can have a maximum of 32 characters, 
must contain only alphanumeric characters or hyphens, and must not begin or end with a hyphen. If not specified, 
Terraform will autogenerate a name beginning with `tf-lb`.
* `name_prefix` - (Optional) Creates a unique name beginning with the specified prefix. Conflicts with `name`.
* `internal` - (Optional) If true, the ALB will be internal.
* `security_groups` - (Optional) A list of security group IDs to assign to the ELB.
* `access_logs` - (Optional) An Access Logs block. Access Logs documented below.
* `subnets` - (Required) A list of subnet IDs to attach to the ELB.
* `idle_timeout` - (Optional) The time in seconds that the connection is allowed to be idle. Default: 60.
* `enable_deletion_protection` - (Optional) If true, deletion of the load balancer will be disabled via
   the AWS API. This will prevent Terraform from deleting the load balancer.
* `tags` - (Optional) A mapping of tags to assign to the resource.

Access Logs (`access_logs`) support the following:

* `bucket` - (Required) The S3 bucket name to store the logs in.
* `prefix` - (Optional) The S3 bucket prefix. Logs are stored in the root if not configured.

## Attributes Reference

The following attributes are exported in addition to the arguments listed above:

* `id` - The ARN of the load balancer (matches `arn`)
* `arn` - The ARN of the load balancer (matches `id`)
* `dns_name` - The DNS name of the load balancer
* `canonical_hosted_zone_id` - The canonical hosted zone ID of the load balancer.
* `zone_id` - The canonical hosted zone ID of the load balancer (to be used in a Route 53 Alias record)

## Import

ALBs can be imported using their ARN, e.g.

```
$ terraform import aws_alb.bar arn:aws:elasticloadbalancing:us-west-2:123456789012:loadbalancer/app/my-load-balancer/50dc6c495c0c9188
```
