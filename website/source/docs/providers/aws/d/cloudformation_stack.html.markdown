---
layout: "aws"
page_title: "AWS: aws_cloudformation_stack"
sidebar_current: "docs-aws-datasource-cloudformation-stack"
description: |-
    Provides metadata of a CloudFormation stack (e.g. outputs)
---

# aws\_cloudformation\_stack

The CloudFormation Stack data source allows access to stack
outputs and other useful data including the template body.

## Example Usage

```hcl
data "aws_cloudformation_stack" "network" {
  name = "my-network-stack"
}

resource "aws_instance" "web" {
  ami           = "ami-abb07bcb"
  instance_type = "t1.micro"
  subnet_id     = "${data.aws_cloudformation_stack.network.outputs["SubnetId"]}"

  tags {
    Name = "HelloWorld"
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the stack

## Attributes Reference

The following attributes are exported:

* `capabilities` - A list of capabilities
* `description` - Description of the stack
* `disable_rollback` - Whether the rollback of the stack is disabled when stack creation fails
* `notification_arns` - A list of SNS topic ARNs to publish stack related events
* `outputs` - A map of outputs from the stack.
* `parameters` - A map of parameters that specify input parameters for the stack.
* `tags` - A map of tags associated with this stack.
* `template_body` - Structure containing the template body.
* `iam_role_arn` - The ARN of the IAM role used to create the stack.
* `timeout_in_minutes` - The amount of time that can pass before the stack status becomes `CREATE_FAILED`
