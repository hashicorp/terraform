---
layout: "aws"
page_title: "AWS: aws_ssm_activation"
sidebar_current: "docs-aws-resource-ssm-activation"
description: |-
  Registers an on-premises server or virtual machine with Amazon EC2 so that it can be managed using Run Command.
---

# aws\_ssm\_activation

Registers an on-premises server or virtual machine with Amazon EC2 so that it can be managed using Run Command.

## Example Usage

```hcl
resource "aws_iam_role" "test_role" {
  name = "test_role"

  assume_role_policy = <<EOF
  {
    "Version": "2012-10-17",
    "Statement": {
      "Effect": "Allow",
      "Principal": {"Service": "ssm.amazonaws.com"},
      "Action": "sts:AssumeRole"
    }
  }
EOF
}

resource "aws_iam_role_policy_attachment" "test_attach" {
  role       = "${aws_iam_role.test_role.name}"
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonEC2RoleforSSM"
}

resource "aws_ssm_activation" "foo" {
  name               = "test_ssm_activation"
  description        = "Test"
  iam_role           = "${aws_iam_role.test_role.id}"
  registration_limit = "5"
  depends_on         = ["aws_iam_role_policy_attachment.test_attach"]
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Optional) The default name of the registerd managed instance.
* `description` - (Optional) The description of the resource that you want to register.
* `expiration_date` - (Optional) The date by which this activation request should expire. The default value is 24 hours.
* `iam_role` - (Required) The IAM Role to attach to the managed instance.
* `registration_limit` - (Optional) The maximum number of managed instances you want to register. The default value is 1 instance.

## Attributes Reference

The following attributes are exported:

* `name` - The default name of the registerd managed instance.
* `description` - The description of the resource that was registered.
* `expired` - If the current activation has expired.
* `expiration_date` - The date by which this activation request should expire. The default value is 24 hours.
* `iam_role` - The IAM Role attached to the managed instance.
* `registration_limit` - The maximum number of managed instances you want to be registered. The default value is 1 instance.
* `registration_count` - The number of managed instances that are currently registered using this activation.
