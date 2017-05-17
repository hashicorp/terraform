---
layout: "aws"
page_title: "AWS: aws_elb_attachment"
sidebar_current: "docs-aws-resource-elb-attachment"
description: |-
  Provides an Elastic Load Balancer Attachment resource.
---

# aws\_elb\_attachment

Provides an Elastic Load Balancer Attachment resource.

~> **NOTE on ELB Instances and ELB Attachments:** Terraform currently provides
both a standalone ELB Attachment resource (describing an instance attached to
an ELB), and an [Elastic Load Balancer resource](elb.html) with
`instances` defined in-line. At this time you cannot use an ELB with in-line
instances in conjunction with an ELB Attachment resource. Doing so will cause a
conflict and will overwrite attachments.
## Example Usage

```hcl
# Create a new load balancer attachment
resource "aws_elb_attachment" "baz" {
  elb      = "${aws_elb.bar.id}"
  instance = "${aws_instance.foo.id}"
}
```

## Argument Reference

The following arguments are supported:

* `elb` - (Required) The name of the ELB.
* `instance` - (Required) Instance ID to place in the ELB pool.
