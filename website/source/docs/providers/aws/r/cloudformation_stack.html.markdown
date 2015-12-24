---
layout: "aws"
page_title: "AWS: aws_cloudformation_stack"
sidebar_current: "docs-aws-resource-cloudformation-stack"
description: |-
  Provides a CloudFormation Stack resource.
---

# aws\_cloudformation\_stack

Provides a CloudFormation Stack resource.

## Example Usage

```
resource "aws_cloudformation_stack" "network" {
  name = "networking-stack"
  template_body = <<STACK
{
  "Resources" : {
    "my-vpc": {
      "Type" : "AWS::EC2::VPC",
      "Properties" : {
        "CidrBlock" : "10.0.0.0/16",
        "Tags" : [
          {"Key": "Name", "Value": "Primary_CF_VPC"}
        ]
      }
    }
  }
}
STACK
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) Stack name.
* `template_body` - (Optional) Structure containing the template body (max size: 51,200 bytes).
* `template_url` - (Optional) Location of a file containing the template body (max size: 460,800 bytes).
* `capabilities` - (Optional) A list of capabilities.
  Currently, the only valid value is `CAPABILITY_IAM`
* `disable_rollback` - (Optional) Set to true to disable rollback of the stack if stack creation failed.
  Conflicts with `on_failure`.
* `notification_arns` - (Optional) A list of SNS topic ARNs to publish stack related events.
* `on_failure` - (Optional) Action to be taken if stack creation fails. This must be
  one of: `DO_NOTHING`, `ROLLBACK`, or `DELETE`. Conflicts with `disable_rollback`.
* `parameters` - (Optional) A list of Parameter structures that specify input parameters for the stack.
* `policy_body` - (Optional) Structure containing the stack policy body.
  Conflicts w/ `policy_url`.
* `policy_url` - (Optional) Location of a file containing the stack policy.
  Conflicts w/ `policy_body`.
* `tags` - (Optional) A list of tags to associate with this stack.
* `timeout_in_minutes` - (Optional) The amount of time that can pass before the stack status becomes `CREATE_FAILED`.

## Attributes Reference

The following attributes are exported:

* `arn` - A unique identifier of the stack.
* `outputs` - A list of output structures.
