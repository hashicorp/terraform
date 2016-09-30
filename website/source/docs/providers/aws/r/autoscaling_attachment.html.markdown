---
layout: "aws"
page_title: "AWS: aws_autoscaling_attachment"
sidebar_current: "docs-aws-resource-autoscaling-attachment"
description: |-
  Provides an AutoScaling Group Attachment resource.
---

# aws\_autoscaling\_attachment

Provides an AutoScaling Attachment resource.

~> **NOTE on AutoScaling Groups and ASG Attachments:** Terraform currently provides
both a standalone ASG Attachment resource (describing an ASG attached to
an ELB), and an [AutoScaling Group resource](autoscaling_group.html) with
`load_balancers` defined in-line. At this time you cannot use an ASG with in-line
load balancers in conjunction with an ASG Attachment resource. Doing so will cause a
conflict and will overwrite attachments.
## Example Usage

```
# Create a new load balancer attachment
resource "aws_autoscaling_attachment" "asg_attachment_bar" {
  elb        = "${aws_elb.bar.id}"
  group_name = "${aws_autoscaling_group.asg.id}"
}
```

## Argument Reference

The following arguments are supported:

* `elb` - (Required) The name of the ELB.
* `group_name` - (Required) Name of ASG to associate with the ELB.
