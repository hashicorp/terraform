---
layout: "docs"
page_title: "Data Sources - 0.11 Configuration Language"
sidebar_current: "docs-conf-old-data-sources"
description: |-
  Data sources allow data to be fetched or computed for use elsewhere in Terraform configuration.
---

# Data Sources

-> **Note:** This page is about Terraform 0.11 and earlier. For Terraform 0.12
and later, see
[Configuration Language: Configuring Data Sources](../configuration/data-sources.html).

*Data sources* allow data to be fetched or computed for use elsewhere
in Terraform configuration. Use of data sources allows a Terraform
configuration to build on information defined outside of Terraform,
or defined by another separate Terraform configuration.

[Providers](./providers.html) are responsible in
Terraform for defining and implementing data sources. Whereas
a [resource](./resources.html) causes Terraform
to create and manage a new infrastructure component, data sources
present read-only views into pre-existing data, or they compute
new values on the fly within Terraform itself.

For example, a data source may retrieve remote state data from a
Terraform Cloud workspace, configuration information from Consul, or look up a pre-existing
AWS resource by filtering on its attributes and tags.

Every data source in Terraform is mapped to a provider based
on longest-prefix matching. For example the `aws_ami`
data source would map to the `aws` provider (if that exists).

This page assumes you're familiar with the
[configuration syntax](./syntax.html)
already.

## Example

A data source configuration looks like the following:

```hcl
# Find the latest available AMI that is tagged with Component = web
data "aws_ami" "web" {
  filter {
    name   = "state"
    values = ["available"]
  }

  filter {
    name   = "tag:Component"
    values = ["web"]
  }

  most_recent = true
}
```

## Description

The `data` block creates a data instance of the given `TYPE` (first
parameter) and `NAME` (second parameter). The combination of the type
and name must be unique.

Within the block (the `{ }`) is configuration for the data instance. The
configuration is dependent on the type, and is documented for each
data source in the [providers section](/docs/providers/index.html).

Each data instance will export one or more attributes, which can be
interpolated into other resources using variables of the form
`data.TYPE.NAME.ATTR`. For example:

```hcl
resource "aws_instance" "web" {
  ami           = "${data.aws_ami.web.id}"
  instance_type = "t1.micro"
}
```

### Meta-parameters

As data sources are essentially a read only subset of resources they also support the same [meta-parameters](./resources.html#meta-parameters) of resources except for the [`lifecycle` configuration block](./resources.html#lifecycle).

## Multiple Provider Instances

Similarly to [resources](./resources.html), the
`provider` meta-parameter can be used where a configuration has
multiple aliased instances of the same provider:

```hcl
data "aws_ami" "web" {
  provider = "aws.west"

  # ...
}
```

See the ["Multiple Provider Instances"](./resources.html#multiple-provider-instances) documentation for resources
for more information.

## Data Source Lifecycle

If the arguments of a data instance contain no references to computed values,
such as attributes of resources that have not yet been created, then the
data instance will be read and its state updated during Terraform's "refresh"
phase, which by default runs prior to creating a plan. This ensures that the
retrieved data is available for use during planning and the diff will show
the real values obtained.

Data instance arguments may refer to computed values, in which case the
attributes of the instance itself cannot be resolved until all of its
arguments are defined. In this case, refreshing the data instance will be
deferred until the "apply" phase, and all interpolations of the data instance
attributes will show as "computed" in the plan since the values are not yet
known.
