---
layout: "guides"
page_title: "Design Patterns for Larger Systems - Guides"
sidebar_current: "guides-design-patterns"
description: |-
  Terraform provides many options for modelling more complex systems, each
  with different tradeoffs. There is no perfect solution for all situations,
  but this guide describes some design patterns that are useful in many
  common situations as systems grow in size and complexity.
---

# Running Terraform in Automation

~> **This is an advanced guide!** When getting started with Terraform, it's
recommended to get comfortable working with its workflow with a single, simple
configuration. The techniques discussed in this guide are for modelling
larger systems and this guide assumes familiarity with basic Terraform concepts
and experience using Terraform to describe simpler systems.

When Terraform is applied to larger and more complex systems it becomes
unweildy to represent the entire system as a single Terraform module or even
as a single Terraform configuration. Terraform's configuration language
provides mechanisms to decompose systems into smaller units, but these
mechanisms can be used in many different ways, with different tradeoffs.

There is no perfect design for all situations, but this guide describes some
design patterns that have proven useful in many common situations as systems
have grown in size and complexity.

The patterns described attempt to maximize the following characteristics:

* **Encapsulation**: for larger systems, it is difficult or impossible for each
  individual to understand the entire system in detail. Encapsulation allows
  certain subsystems to be treated as a "black box" when constructing the
  system, and known-good configurations to be reused in multiple contexts.

* **Composability**: once a system has been decomposed, it is ideal if the
  individual components can be combined together in various ways, used
  to solve multiple problems, and recombined in new ways as needs change.
  This requires a good
  [separation of concerns](https://en.wikipedia.org/wiki/Separation_of_concerns)
  between components so that their reliance on the details of a specific
  system configuration is minimized.

* **Explicitness**: infrastructure can be expensive and mistakes updating it
  even moreso. While encapsulation is helpful in developing a high-level mental
  model of a system, _too much_ abstraction can make it hard to understand the
  implications of a change and hard to evaluate a proposed execution plan,
  leading to costly mistakes.

* **Testability**: although Terraform configurations are declarative and so
  are not usually tested in the usual sense that imperative software is, it is
  nonetheless helpful to be able to develop and manually test each component in
  isolation, without disturbing any "real" infrastructure, to minimize
  unexpected impacts as multi-use components are improved.

## Modules as Components

[Modules](/docs/modules/index.html) are the primary unit of decomposition in
Terraform, and as such are a key part of any attempt to organize a complex
system.

(include something about testing these isolated modules in here too)

## Dependency Injection

The general idea of
[dependency injection](https://en.wikipedia.org/wiki/Dependency_injection)
is to provide a component's dependencies as arguments rather than having it
create or obtain its own. Within Terraform, the dependency injection
approach can be achieved by distinguishing a configuration's _root module_
from its _child modules_.

While root and child modules are structurally equivalent, in a typical
configuration the root module has the following special responsibilities:

* Obtain externally-defined contextual data such as credentials or account
  identifiers, either through [variables](/docs/configuration/variables.html)
  or from [data sources](/docs/configuration/data-sources.html).

* Configure any providers that are used by child modules.

* Instantiate one or more _component modules_ and connect their inputs and
  outputs via expressions.

* Publish the resulting settings, either through
  [outputs](/docs/configuration/outputs.html) or into an explicit configuration
  store such as [Consul](https://www.consul.io/) for direct consumption.

The child modules, then, represent the individual components of the system,
each of which has a number of settings and dependencies and which produces
a number of results. These modules assume _no_ context except that passed
in from their caller, including both variables and provider configurations,
which ensures that they can be instantiated as many times as needed without
conflicts -- at least, as long as the root module passes in an appropriate
set of inputs.

A typical use of dependency injection in Terraform is to define a virtual
network topology consisting of regions, networks and subnetworks. In such
a system, the regions are pre-existing context established by the vendor,
while the networks and subnetworks are components of our own system, built
on infrastructure elements provided by the vendor. In AWS, for example,
networks are called _Virtual Private Clouds_ or "VPCs" and each one has
one or more _subnets_.

One approach for this would be to define a module representing a network
and a module representing a subnetwork, and have the root module instantiate
the network module and the network module in turn instantiate the subnet
module. While this would work, it forces the network module to inherit
all of the concerns of the subnetwork module, anticipating all of the possible
ways that subnetworks may be set up within a module.

A simpler and more flexible approach is for the root module to instantiate
both the network module _and_ the subnet module, and then use expressions
to pass necessary settings from each network to its associated subnets.
This is _dependency injection_ style, because the root module controls
the set of components and provides the necessary details about each network
to its associated subnetworks.

The root module for a system that has these components might contain something
like the following:

```hcl
# Provider configurations for the two regions we operate in
provider "aws" {
  alias  = "use1"
  region = "us-east-1"
}
provider "aws" {
  alias  = "usw2"
  region = "us-west-2"
}

variable "base_block" {
  default = "192.168.0.0/16"
}

# Network components: each region has one network
module "network_use1" {
  source    = "./network"
  providers = {
    aws = "aws.use1"
  }

  cidr_block = "${cidrsubnet(var.base_block, 3, 1)}"
}
module "network_usw2" {
  source    = "./network"
  providers = {
    aws = "aws.usw2"
  }

  cidr_block = "${cidrsubnet(var.base_block, 3, 2)}"
}

# Subnetwork components: each region has both a public and a private subnet
# (In practice there would probably be multiple of each spread over
# multiple availability zones, but we'll keep this simple for the sake
# of example.)
module "subnet_use1_public" {
  source    = "./subnetwork"
  providers = {
    aws = "aws.use1"
  }

  cidr_block = "${cidrsubnet(module.network_use1.cidr_block, 1, 0)}"
  aws_vpc_id = "${module.network_use1.aws_vpc_id}"
  aws_az     = "us-east-1b"
  public     = true
}
module "subnet_use1_private" {
  source    = "./subnetwork"
  providers = {
    aws = "aws.use1"
  }

  cidr_block = "${cidrsubnet(module.network_use1.cidr_block, 1, 1)}"
  aws_vpc_id = "${module.network_use1.aws_vpc_id}"
  aws_az     = "us-east-1b"
  public     = false
}
module "subnet_usw2_public" {
  source    = "./subnetwork"
  providers = {
    aws = "aws.usw2"
  }

  cidr_block = "${cidrsubnet(module.network_usw2.cidr_block, 1, 0)}"
  aws_vpc_id = "${module.network_usw2.aws_vpc_id}"
  aws_az     = "us-west-2c"
  public     = true
}
module "subnet_usw2_private" {
  source    = "./subnetwork"
  providers = {
    aws = "aws.usw2"
  }

  cidr_block = "${cidrsubnet(module.network_usw2.cidr_block, 1, 1)}"
  aws_vpc_id = "${module.network_usw2.aws_vpc_id}"
  aws_az     = "us-west-2c"
  public     = false
}

output "public_subnet_ids" {
  value = [
    "${module.subnet_use1_public.aws_subnet_id}",
    "${module.subnet_usw2_public.aws_subnet_id}",
  ]
}
output "private_subnet_ids" {
  value = [
    "${module.subnet_use1_private.aws_subnet_id}",
    "${module.subnet_usw2_private.aws_subnet_id}",
  ]
}
```

Although in this initial state each network as a uniform set of subnetworks,
using the dependency injection style makes this easy to adapt over time
as needs change. If a particular region needs an additional subnetwork in
its network, this can be added without altering either of the component
modules by just adding a new `module` block to the root module.

## Pattern Modules

While dependency injection creates flexibility, it also tends to lead to
duplication.

In the above example, a network topology was defined from
its component parts such that it can in principle have different addressing
schemes and subnetwork divisions per region, have a region with only
a private subnet, and various other variations. On the other hand, it
is easy to see how over time such inconsistencies could prove to be a burden
rather than a benefit.

As common structural patterns emerge within your organization, such as a
battle-tested per-region network layout, it can be useful to describe these
patterns via _pattern modules_, which share some of the traits of
_component modules_ -- they use only state provided directly by the caller --
but also intentionally combine several concerns in a manner similar to a
root module in order to create a broader abstraction.

Continuing our network topology example, we might create a "standard network
topology" pattern module that combines the network and subnetwork component
modules in a repeatable way. The root module can then be simplified:

```hcl
provider "aws" {
  alias  = "use1"
  region = "us-east-1"
}
provider "aws" {
  alias  = "usw2"
  region = "us-west-2"
}

variable "base_block" {
  default = "192.168.0.0/16"
}

module "network_use1" {
  source    = "./standard-network-topology"
  providers = {
    aws = "aws.use1"
  }

  cidr_block = "${cidrsubnet(var.base_block, 3, 1)}"
  aws_az     = "us-east-1b"
}
module "network_usw2" {
  source    = "./standard-network-topology"
  providers = {
    aws = "aws.usw2"
  }

  cidr_block = "${cidrsubnet(var.base_block, 3, 2)}"
  aws_az     = "us-west-2c"
}


output "public_subnet_ids" {
  value = [
    "${module.network_use1.public_aws_subnet_id}",
    "${module.network_usw2.public_aws_subnet_id}",
  ]
}
output "private_subnet_ids" {
  value = [
    "${module.network_use1.private_aws_subnet_id}",
    "${module.network_usw2.private_aws_subnet_id}",
  ]
}
```

The content of the `./standard-network-topology` module might include
the following:

```hcl
variable "cidr_block" {
}
variable "aws_az" {
}

module "network" {
  source    = "../network"

  cidr_block = "${var.cidr_block}"
}
module "subnet_public" {
  source    = "../subnetwork"

  cidr_block = "${cidrsubnet(module.network.cidr_block, 1, 0)}"
  aws_vpc_id = "${module.network.aws_vpc_id}"
  aws_az     = "${var.aws_az}"
  public     = true
}
module "subnet_private" {
  source    = "../subnetwork"

  cidr_block = "${cidrsubnet(module.network.cidr_block, 1, 1)}"
  aws_vpc_id = "${module.network.aws_vpc_id}"
  aws_az     = "${var.aws_az}"
  public     = false
}

output "public_aws_subnet_id" {
  value = "${module.subnet_public.aws_subnet_id}"
}
output "private_aws_subnet_id" {
  value = "${module.subnet_private.aws_subnet_id}"
}
```

The pattern module now represents the pattern of having both a public
and a private subnet in each network. This module can now be used to
apply a consistent network topology across many regions as the
root module evolves.

Pattern modules should generally contain only instantiations of component
modules that do the "real work", and should be used sparingly. It can be
tempting to create pattern modules for every use-case, or to try to represent
a large set of components via a single module with complex-typed variables.
Our goal of _explicitness_ is best served by using flat module structures
with explicit instantiations of individual resources, rather than complex
dynamic configurations (e.g. using `count` over complex arrays) that make
it hard to predict the effect of a change.

## Separate Module Repositories

## Separated Subsystem Lifecycles

(techniques for breaking a large system into multiple configurations)
