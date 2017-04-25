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

Using `terraform import` is simple. An example is shown below:

```shell
$ terraform import aws_instance.bar i-abcd1234
```

The above command imports an AWS instance with the given ID to the
address `aws_instance.bar`. You can also import resources into modules.
See the [resource addressing](/docs/internals/resource-addressing.html)
page for more details on the full range of addresses supported.

The ID given is dependent on the resource type being imported. For example,
AWS instances use their direct IDs. However, AWS Route53 zones use the
domain name itself. Reference the resource documentation for details on
what the ID it expects is.

As a result of the above command, the resource is put into the state file.
If you run `terraform plan`, you should see Terraform plan your resource
for destruction. You now have to create a matching configuration so that
Terraform doesn't plan a destroy.

## Complex Imports

The above import is considered a "simple import": one resource is imported
into the state file. An import may also result in a "complex import" where
multiple resources are imported. For example, an AWS security group imports
an `aws_security_group` but also one `aws_security_group_rule` for each rule.

In this case, the name of the resource is shown as part of the import output.
You'll have to create a configuration for each resource imported. If you want
to rename or otherwise modify the imported resources, the
[state management commands](/docs/commands/state/index.html) should be used.
