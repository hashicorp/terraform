---
layout: "docs"
page_title: "Configuration Language"
sidebar_current: "docs-config-index"
description: |-
  Terraform uses text files to describe infrastructure and to set variables.
  These text files are called Terraform _configurations_ and are
  written in the Terraform language.
---

# Configuration Language

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language](../configuration-0-11/index.html).

Terraform uses its own configuration language, designed to allow concise
descriptions of infrastructure. The Terraform language is declarative,
describing an intended goal rather than the steps to reach that goal.

## Resources and Modules

The main purpose of the Terraform language is declaring [resources](./resources.html).
All other language features exist only to make the definition of resources
more flexible and convenient.

A group of resources can be gathered into a [module](./modules.html),
which creates a larger unit of configuration. A resource describes a single
infrastructure object, while a module might describe a set of objects and the
necessary relationships between them in order to create a higher-level system.

A _Terraform configuration_ consists of a _root module_, where evaluation
begins, along with a tree of child modules created when one module calls
another.

## Arguments, Blocks, and Expressions

The syntax of the Terraform language consists of only a few basic elements:

```hcl
resource "aws_vpc" "main" {
  cidr_block = var.base_cidr_block
}

<BLOCK TYPE> "<BLOCK LABEL>" "<BLOCK LABEL>" {
  # Block body
  <IDENTIFIER> = <EXPRESSION> # Argument
}
```

- _Blocks_ are containers for other content and usually represent the
  configuration of some kind of object, like a resource. Blocks have a
  _block type,_ can have zero or more _labels,_ and have a _body_ that contains
  any number of arguments and nested blocks. Most of Terraform's features are
  controlled by top-level blocks in a configuration file.
- _Arguments_ assign a value to a name. They appear within blocks.
- _Expressions_ represent a value, either literally or by referencing and
  combining other values. They appear as values for arguments, or within other
  expressions.

For full details about Terraform's syntax, see:

- [Configuration Syntax](./syntax.html)
- [Expressions](./expressions.html)

## Code Organization

The Terraform language uses configuration files that are named with the `.tf`
file extension. There is also [a JSON-based variant of the language](./syntax-json.html)
that is named with the `.tf.json` file extension.

Configuration files must always use UTF-8 encoding, and by convention are
usually maintained with Unix-style line endings (LF) rather than Windows-style
line endings (CRLF), though both are accepted.

A _module_ is a collection of `.tf` or `.tf.json` files kept together in a
directory. The root module is built from the configuration files in the
current working directory when Terraform is run, and this module may reference
child modules in other directories, which can in turn reference other modules,
etc.

The simplest Terraform configuration is a single root module containing only
a single `.tf` file. A configuration can grow gradually as more resources
are added, either by creating new configuration files within the root module
or by organizing sets of resources into child modules.

## Configuration Ordering

Because Terraform's configuration language is declarative, the ordering of
blocks is generally not significant. (The order of `provisioner` blocks within a
resource is the only major feature where block order matters.)

Terraform automatically processes resources in the correct order based on
relationships defined between them in configuration, and so you can organize
resources into source files in whatever way makes sense for your infrastructure.

## Terraform CLI vs. Providers

The Terraform command line interface (CLI) is a general engine for evaluating
and applying Terraform configurations. It defines the Terraform language syntax
and overall structure, and coordinates sequences of changes that must be made to
make remote infrastructure match the given configuration.

This general engine has no knowledge about specific types of infrastructure
objects. Instead, Terraform uses plugins called
[providers](./providers.html) that each define and manage a
set of resource types. Most providers are associated with a particular cloud or
on-premises infrastructure service, allowing Terraform to manage infrastructure
objects within that service.

Terraform doesn't have a concept of platform-independent resource types
— resources are always tied to a provider, since the features of similar
resources can vary greatly from provider to provider. But Terraform CLI's shared
configuration engine ensures that the same language constructs and syntax are
available across all services and allows resource types from different services
to be combined as needed.

## Example

The following simple example describes a simple network topology for Amazon Web
Services, just to give a sense of the overall structure and syntax of the
Terraform language. Similar configurations can be created for other virtual
network services, using resource types defined by other providers, and a
practical network configuration will often contain additional elements not
shown here.

```hcl
variable "aws_region" {}

variable "base_cidr_block" {
  description = "A /16 CIDR range definition, such as 10.1.0.0/16, that the VPC will use"
  default = "10.1.0.0/16"
}

variable "availability_zones" {
  description = "A list of availability zones in which to create subnets"
  type = list(string)
}

provider "aws" {
  region = var.aws_region
}

resource "aws_vpc" "main" {
  # Referencing the base_cidr_block variable allows the network address
  # to be changed without modifying the configuration.
  cidr_block = var.base_cidr_block
}

resource "aws_subnet" "az" {
  # Create one subnet for each given availability zone.
  count = length(var.availability_zones)

  # For each subnet, use one of the specified availability zones.
  availability_zone = var.availability_zones[count.index]

  # By referencing the aws_vpc.main object, Terraform knows that the subnet
  # must be created only after the VPC is created.
  vpc_id = aws_vpc.main.id

  # Built-in functions and operators can be used for simple transformations of
  # values, such as computing a subnet address. Here we create a /20 prefix for
  # each subnet, using consecutive addresses for each availability zone,
  # such as 10.1.16.0/20 .
  cidr_block = cidrsubnet(aws_vpc.main.cidr_block, 4, count.index+1)
}
```

For more information on the configuration elements shown here, use the
site navigation to explore the Terraform language documentation sub-sections.
To start, see [_Resource Configuration_](./resources.html).
