---
layout: "docs"
page_title: "Import: Usage"
sidebar_current: "docs-import-usage"
description: |-
  The `terraform import` command is used to import existing infrastructure.
---

# Import Usage

The `terraform import` command is used to import existing infrastructure.

The command currently can only import one resource at a time. This means
you can't yet point Terraform import to an entire collection of resources
such as an AWS VPC and import all of it. A future version of Terraform will
be able to do this.

To import a resource, first write a resource block for it in your
configuration, establishing the name by which it will be known in Terraform:

```
resource "aws_instance" "bar" {
  # ...instance configuration...
}
```

If desired, you can leave the body of the resource block blank for now and
return to fill it in once the instance is imported.

Now `terraform import` can be run to attach an existing instance to this
resource configuration:

```shell
$ terraform import aws_instance.bar i-abcd1234
```

The above command imports an AWS instance with the given ID and attaches
it to the name `aws_instance.bar`. You can also import resources into modules.
See the [resource addressing](/docs/internals/resource-addressing.html)
page for more details on the full range of addresses supported.

The ID given is dependent on the resource type being imported. For example,
AWS instances use their direct IDs. However, AWS Route53 zones use the
domain name itself. Console the resource documentation for details on what
form of ID each resource expects.

As a result of the above command, the resource is recorded in the state file.
You can now run `terraform plan` to see how the configuration compares to
the imported resource, and make any adjustments to the configuration to
align with the current (or desired) state of the imported object.

## Complex Imports

The above import is considered a "simple import": one resource is imported
into the state file. An import may also result in a "complex import" where
multiple resources are imported. For example, an AWS security group imports
an `aws_security_group` but also one `aws_security_group_rule` for each rule.

In this scenario, the secondary resources will not already exist in
configuration, so it is necessary to consult the import output and create
a `resource` block in configuration for each secondary resource. If this is
not done, Terraform will plan to destroy the imported objects on the next run.

If you want to rename or otherwise modify the imported resources, the
[state management commands](/docs/commands/state/index.html) can be used.
