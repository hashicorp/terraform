---
layout: "docs"
page_title: "Configuring Resources"
sidebar_current: "docs-config-resources"
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

```
resource "aws_instance" "web" {
    ami = "ami-123456"
    instance_type = "m1.small"
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

There are **meta-parameters** available to all resources:

  * `count` (int) - The number of identical resources to create.
      This doesn't apply to all resources. You can use the `${count.index}`
      [interpolation](/docs/configuration/interpolation.html) to reference
      the current count index in your resource.

  * `depends_on` (list of strings) - Explicit dependencies that this
      resource has. These dependencies will be created before this
      resource. The dependencies are in the format of `TYPE.NAME`,
      for example `aws_instance.web`.

  * `lifecycle` (configuration block) - Customizes the lifecycle
      behavior of the resource. The specific options are documented
      below.

The `lifecycle` block allows the following keys to be set:

  * `create_before_destroy` (bool) - This flag is used to ensure
      the replacement of a resource is created before the original
      instance is destroyed. As an example, this can be used to
      create an new DNS record before removing an old record.

-------------

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

-------------

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

## Syntax

The full syntax is:

```
resource TYPE NAME {
	CONFIG ...
	[count = COUNT]
    [depends_on = [RESOURCE NAME, ...]]
    [LIFECYCLE]

	[CONNECTION]
	[PROVISIONER ...]
}
```

where `CONFIG` is:

```
KEY = VALUE

KEY {
	CONFIG
}
```

where `LIFECYCLE` is:

```
lifecycle {
    [create_before_destroy = true|false]
}
```

where `CONNECTION` is:

```
connection {
	KEY = VALUE
	...
}
```

where `PROVISIONER` is:

```
provisioner NAME {
	CONFIG ...

	[CONNECTION]
}
```
