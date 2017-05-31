---
layout: "aws"
page_title: "AWS: aws_elb_listener"
sidebar_current: "docs-aws-resource-elb-listener"
description: |-
  Provides an Elastic Load Balancer Listener resource.
---

# aws\_elb\_listener

Provides an Elastic Load Balancer Listener resource.

~> **NOTE on ELB Listeners and ELB Listener:** Terraform currently provides
both a standalone ELB Listener resource, and an [Elastic Load Balancer resource](elb.html) with
`listener` defined in-line. At this time you cannot use an ELB with in-line
listeners in conjunction with an ELB Listener resource. Doing so will cause a
conflict and will overwrite attachments.
## Example Usage

```
resource "aws_elb" "foo" {
  availability_zones = ["us-west-2a", "us-west-2b", "us-west-2c"]

  listener {
    instance_port = 8000
    instance_protocol = "http"
    lb_port = 80
    lb_protocol = "http"
  }
}

resource "aws_elb" "bar" {
  availability_zones = ["us-west-2a", "us-west-2b", "us-west-2c"]

  listener {
    instance_port = 8000
    instance_protocol = "http"
    lb_port = 80
    lb_protocol = "http"
  }
}

resource "aws_elb_listener" "another_listener" {
  loadbalancer_names = ["${aws_elb.foo.name}", "${aws_elb.bar.name}"]
  instance_port = 8500
  instance_protocol = "http"
  lb_port = 8500
  lb_protocol = "http"
}
```

## Argument Reference

The following arguments are supported:

* `loadbalancer_names` - (Required) A list of the loadbalancers to which to
attach the listener
* `instance_port` - (Required) The port on the instance to route to
* `instance_protocol` - (Required) The protocol to use to the instance. Valid
  values are `HTTP`, `HTTPS`, `TCP`, or `SSL`
* `lb_port` - (Required) The port to listen on for the load balancer
* `lb_protocol` - (Required) The protocol to listen on. Valid values are `HTTP`,
  `HTTPS`, `TCP`, or `SSL`
* `ssl_certificate_id` - (Optional) The ARN of an SSL certificate you have
uploaded to AWS IAM. **Note ECDSA-specific restrictions below.  Only valid when `lb_protocol` is either HTTPS or SSL**

## Note on ECDSA Key Algorithm

If the ARN of the `ssl_certificate_id` that is pointed to references a
certificate that was signed by an ECDSA key, note that ELB only supports the
P256 and P384 curves.  Using a certificate signed by a key using a different
curve could produce the error `ERR_SSL_VERSION_OR_CIPHER_MISMATCH` in your
browser.