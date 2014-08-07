---
layout: "aws"
page_title: "AWS: aws_elb"
sidebar_current: "docs-aws-resource-elb"
---

# aws\_elb

Provides an Elastic Load Balancer resource.

## Example Usage

```
# Create a new load balancer
resource "aws_elb" "bar" {
  name = "foobar-terraform-elb"
  availability_zones = ["us-west-2a", "us-west-2b", "us-west-2c"]

  listener {
    instance_port = 8000
    instance_protocol = "http"
    lb_port = 80
    lb_protocol = "http"
  }

  health_check {
    healthy_threshold = 2
    unhealthy_threshold = 2
    timeout = 3
    target = "HTTP:8000/"
    interval = 30
  }

  instances = ["${aws_instance.foo.id}"]
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the ELB
* `availability_zones` - (Optional) The AZ's to serve traffic in.
* `instances` - (Optional) A list of instance ids to place in the ELB pool.
* `listener` - (Required) A list of listener blocks. Listeners documented below.
* `health_check` - (Required) A health_check block. Health Check documented below.

Listeners support the following:

* `instance_port` - (Required) The port on the instance to route to
* `instance_protocol` - (Required) The the protocol to use to the instance.
* `lb_port` - (Required) The port to listen on for the load balancer
* `lb_protocol` - (Required) The protocol to listen on.

Health Check supports the following:

* `healthy_threshold` - (Required) The number of checks before the instance is declared healthy.
* `unhealthy_threshold` - (Required) The number of checks before the instance is declared unhealthy.
* `target` - (Required) The target of the check.
* `interval` - (Required) The interval between checks.
* `timeout` - (Required) The length of time before the check times out.


## Attributes Reference

The following attributes are exported:

* `id` - The name of the ELB
* `name` - The name of the ELB
* `dns_name` - The DNS name of the ELB
* `instances` - The list of instances in the ELB
