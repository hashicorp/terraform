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
such as an AWS VPC and import all of it. This workflow will be improved in a
future version of Terraform.

To import a resource, first write a resource block for it in your
configuration, establishing the name by which it will be known to Terraform:

```
resource "aws_instance" "example" {
  # ...instance configuration...
}
```

The name "example" here is local to the module where it is declared and is
chosen by the configuration author. This is distinct from any ID issued by
the remote system, which may change over time while the resource name
remains constant.

If desired, you can leave the body of the resource block blank for now and
return to fill it in once the instance is imported.

Now `terraform import` can be run to attach an existing instance to this
resource configuration:

```shell
$ terraform import aws_instance.example i-abcd1234
```

This command locates the AWS instance with ID `i-abcd1234`. Then it attaches
the existing settings of the instance, as described by the EC2 API, to the
name `aws_instance.example` of a module. In this example the module path
implies that the root module is used. Finally, the mapping is saved in the
Terraform state.

It is also possible to import to resources in child modules, using their paths,
and to single instances of a resource with `count` or `for_each` set. See
[_Resource Addressing_](/docs/internals/resource-addressing.html) for more
details on how to specify a target resource.

The syntax of the given ID is dependent on the resource type being imported.
For example, AWS instances use an opaque ID issued by the EC2 API, but
AWS Route53 Zones use the domain name itself. Consult the documentation for
each importable resource for details on what form of ID is required.

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

If you want to rename or otherwise move the imported resources, the
[state management commands](/docs/commands/state/index.html) can be used.
