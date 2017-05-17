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

```hcl
# Create a new load balancer attachment
resource "aws_autoscaling_attachment" "asg_attachment_bar" {
  autoscaling_group_name = "${aws_autoscaling_group.asg.id}"
  elb                    = "${aws_elb.bar.id}"
}
```

```hcl
# Create a new ALB Target Group attachment
resource "aws_autoscaling_attachment" "asg_attachment_bar" {
  autoscaling_group_name = "${aws_autoscaling_group.asg.id}"
  alb_target_group_arn   = "${aws_alb_target_group.test.arn}"
}
```

## Argument Reference

The following arguments are supported:

* `autoscaling_group_name` - (Required) Name of ASG to associate with the ELB.
* `elb` - (Optional) The name of the ELB.
* `alb_target_group_arn` - (Optional) The ARN of an ALB Target Group.

