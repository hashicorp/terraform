---
layout: "docs"
page_title: "Configuring Resources"
sidebar_current: "docs-config-resources"
description: |-
  The most important thing you'll configure with Terraform are resources. Resources are a component of your infrastructure. It might be some low level component such as a physical server, virtual machine, or container. Or it can be a higher level component such as an email provider, DNS record, or database provider.
---

# Resource Configuration

The most important thing you'll configure with Terraform are
resources. Resources are a component of your infrastructure.
It might be some low level component such as a physical server,
virtual machine, or container. Or it can be a higher level
component such as an email provider, DNS record, or database
provider.

This page assumes you're familiar with the
[configuration syntax](/docs/configuration/syntax.html)
already.

## Example

A resource configuration looks like the following:

```hcl
resource "aws_instance" "web" {
  ami           = "ami-408c7f28"
  instance_type = "t1.micro"
}
```

## Description

The `resource` block creates a resource of the given `TYPE` (first
parameter) and `NAME` (second parameter). The combination of the type
and name must be unique.

Within the block (the `{ }`) is configuration for the resource. The
configuration is dependent on the type, and is documented for each
resource type in the
[providers section](/docs/providers/index.html).

### Meta-parameters

There are **meta-parameters** available to all resources:

- `count` (int) - The number of identical resources to create. This doesn't
  apply to all resources. For details on using variables in conjunction with
  count, see [Using Variables with `count`](#using-variables-with-count) below.

    -> Modules don't currently support the `count` parameter.

- `depends_on` (list of strings) - Explicit dependencies that this resource has.
  These dependencies will be created before this resource. For syntax and other
  details, see the section below on [explicit
  dependencies](#explicit-dependencies).

- `provider` (string) - The name of a specific provider to use for this
  resource. The name is in the format of `TYPE.ALIAS`, for example, `aws.west`.
  Where `west` is set using the `alias` attribute in a provider. See [multiple
  provider instances](#multi-provider-instances).

- `lifecycle` (configuration block) - Customizes the lifecycle behavior of the
  resource. The specific options are documented below.

    The `lifecycle` block allows the following keys to be set:

  - `create_before_destroy` (bool) - This flag is used to ensure the replacement
    of a resource is created before the original instance is destroyed. As an
    example, this can be used to create an new DNS record before removing an old
    record.

        ~> Resources that utilize the `create_before_destroy` key can only
        depend on other resources that also include `create_before_destroy`.
        Referencing a resource that does not include `create_before_destroy`
        will result in a dependency graph cycle.

  - `prevent_destroy` (bool) - This flag provides extra protection against the
    destruction of a given resource. When this is set to `true`, any plan that
    includes a destroy of this resource will return an error message.

  - `ignore_changes` (list of strings) - Customizes how diffs are evaluated for
    resources, allowing individual attributes to be ignored through changes. As
    an example, this can be used to ignore dynamic changes to the resource from
    external resources. Other meta-parameters cannot be ignored.

        ~> Ignored attribute names can be matched by their name, not state ID.
        For example, if an `aws_route_table` has two routes defined and the
        `ignore_changes` list contains "route", both routes will be ignored.
        Additionally you can also use a single entry with a wildcard (e.g. `"*"`)
        which will match all attribute names. Using a partial string together
        with a wildcard (e.g. `"rout*"`) is **not** supported.

### Timeouts

Individual Resources may provide a `timeouts` block to enable users to configure the
amount of time a specific operation is allowed to take before being considered
an error. For example, the
[aws_db_instance](/docs/providers/aws/r/db_instance.html#timeouts)
resource provides configurable timeouts for the
`create`, `update`, and `delete` operations. Any Resource that provies Timeouts
will document the default values for that operation, and users can overwrite
them in their configuration.

Example overwriting the `create` and `delete` timeouts:

```hcl
resource "aws_db_instance" "timeout_example" {
  allocated_storage = 10
  engine            = "mysql"
  engine_version    = "5.6.17"
  instance_class    = "db.t1.micro"
  name              = "mydb"

  # ...

  timeouts {
    create = "60m"
    delete = "2h"
  }
}
```

Individual Resources must opt-in to providing configurable Timeouts, and
attempting to configure the timeout for a Resource that does not support
Timeouts, or overwriting a specific action that the Resource does not specify as
an option, will result in an error. Valid units of time are  `s`, `m`, `h`.

### Explicit Dependencies

Terraform ensures that dependencies are successfully created before a
resource is created. During a destroy operation, Terraform ensures that
this resource is destroyed before its dependencies.

A resource automatically depends on anything it references via
[interpolations](/docs/configuration/interpolation.html). The automatically
determined dependencies are all that is needed most of the time. You can also
use the `depends_on` parameter to explicitly define a list of additional
dependencies.

The primary use case of explicit `depends_on` is to depend on a _side effect_
of another operation. For example: if a provisioner creates a file, and your
resource reads that file, then there is no interpolation reference for Terraform
to automatically connect the two resources. However, there is a causal
ordering that needs to be represented. This is an ideal case for `depends_on`.
In most cases, however, `depends_on` should be avoided and Terraform should
be allowed to determine dependencies automatically.

The syntax of `depends_on` is a list of resources and modules:

- Resources are `TYPE.NAME`, such as `aws_instance.web`.
- Modules are `module.NAME`, such as `module.foo`.

When a resource depends on a module, _everything_ in that module must be
created before the resource is created.

An example of a resource depending on both a module and resource is shown
below. Note that `depends_on` can contain any number of dependencies:

```hcl
resource "aws_instance" "web" {
  depends_on = ["aws_instance.leader", "module.vpc"]
}
```

-> **Use sparingly!** `depends_on` is rarely necessary.
In almost every case, Terraform's automatic dependency system is the best-case
scenario by having your resources depend only on what they explicitly use.
Please think carefully before you use `depends_on` to determine if Terraform
could automatically do this a better way.

### Connection block

Within a resource, you can optionally have a **connection block**.
Connection blocks describe to Terraform how to connect to the
resource for
[provisioning](/docs/provisioners/index.html). This block doesn't
need to be present if you're using only local provisioners, or
if you're not provisioning at all.

Resources provide some data on their own, such as an IP address,
but other data must be specified by the user.

The full list of settings that can be specified are listed on
the [provisioner connection page](/docs/provisioners/connection.html).

### Provisioners

Within a resource, you can specify zero or more **provisioner
blocks**. Provisioner blocks configure
[provisioners](/docs/provisioners/index.html).

Within the provisioner block is provisioner-specific configuration,
much like resource-specific configuration.

Provisioner blocks can also contain a connection block
(documented above). This connection block can be used to
provide more specific connection info for a specific provisioner.
An example use case might be to use a different user to log in
for a single provisioner.

## Using Variables With `count`

When declaring multiple instances of a resource using [`count`](#count), it is
common to want each instance to have a different value for a given attribute.

You can use the `${count.index}`
[interpolation](/docs/configuration/interpolation.html) along with a map
[variable](/docs/configuration/variables.html) to accomplish this.

For example, here's how you could create three [AWS
Instances](/docs/providers/aws/r/instance.html) each with their own
static IP address:

```hcl
variable "instance_ips" {
  default = {
    "0" = "10.11.12.100"
    "1" = "10.11.12.101"
    "2" = "10.11.12.102"
  }
}

resource "aws_instance" "app" {
  count = "3"
  private_ip = "${lookup(var.instance_ips, count.index)}"
  # ...
}
```

## Multiple Provider Instances

By default, a resource targets the provider based on its type. For example
an `aws_instance` resource will target the "aws" provider. As of Terraform
0.5.0, a resource can target any provider by name.

The primary use case for this is to target a specific configuration of
a provider that is configured multiple times to support multiple regions, etc.

To target another provider, set the `provider` field:

```hcl
resource "aws_instance" "foo" {
	provider = "aws.west"

	# ...
}
```

The value of the field should be `TYPE` or `TYPE.ALIAS`. The `ALIAS` value
comes from the `alias` field value when configuring the
[provider](/docs/configuration/providers.html).

```hcl
provider "aws" {
  alias = "west"

  # ...
}
```

If no `provider` field is specified, the default provider is used.

## Syntax

The full syntax is:

```text
resource TYPE NAME {
	CONFIG ...
	[count = COUNT]
	[depends_on = [NAME, ...]]
	[provider = PROVIDER]

    [LIFECYCLE]

	[CONNECTION]
	[PROVISIONER ...]
}
```

where `CONFIG` is:

```text
KEY = VALUE

KEY {
	CONFIG
}
```

where `LIFECYCLE` is:

```text
lifecycle {
    [create_before_destroy = true|false]
    [prevent_destroy = true|false]
    [ignore_changes = [ATTRIBUTE NAME, ...]]
}
```

where `CONNECTION` is:

```text
connection {
	KEY = VALUE
	...
}
```

where `PROVISIONER` is:

```text
provisioner NAME {
	CONFIG ...

	[when = "create"|"destroy"]
	[on_failure = "continue"|"fail"]

	[CONNECTION]
}
```
