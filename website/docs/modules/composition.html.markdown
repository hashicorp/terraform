---
layout: "docs"
page_title: "Module Composition"
sidebar_current: "docs-modules-composition"
description: |-
  Module composition allows infrastructure to be described from modular
  building blocks.
---

# Module Composition

-> This section is written for **Terraform v0.12 or later**. The general patterns
   described in this section _do_ apply to earlier versions, but the examples
   shown are using v0.12-only syntax and features. For general information
   on module usage in prior versions, see
   [/docs/configuration-0-11/modules.html](the v0.11 documentation about modules).

In a simple Terraform configuration with only one root module, we create a
flat set of resources and use Terraform's expression syntax to describe the
relationships between these resources:

```hcl
resource "aws_vpc" "example" {
  cidr_block = "10.1.0.0/16"
}

resource "aws_subnet" "example" {
  vpc_id = aws_vpc.example.id

  availability_zone = "us-west-2b"
  cidr_block        = cidrsubnet(aws_vpc.example.cidr_block, 4, 1)
}
```

When we introduce `module` blocks, our configuration becomes heirarchical
rather than flat: each module contains its own set of resources, and possibly
its own child modules, which can potentially create a deep, complex tree of
resource configurations.

However, in most cases we strongly recommend keeping the module tree flat,
with only one level of child modules, and use a technique similar to the
above of using expressions to describe the relationships between the modules:

```hcl
module "network" {
  source = "./modules/aws-network"

  base_cidr_block = "10.0.0.0/8"
}

module "consul_cluster" {
  source = "./modules/aws-consul-cluster"

  vpc_id     = module.network.vpc_id
  subnet_ids = module.network.subnet_ids
}
```

We call this flat style of module usage _module composition_, because it
takes multiple [composable](https://en.wikipedia.org/wiki/Composability)
building-block modules and assembles them together to produce a larger system.
Instead of a module _embedding_ its dependencies, creating and managing its
own copy, the module _receives_ its dependencies from the root module, which
can therefore connect the same modules in different ways to produce different
results.

The rest of this page discusses some more specific composition patterns that
may be useful when describing larger systems with Terraform.

## Dependency Inversion

In the example above, we saw a `consul_cluster` module that presumably describes
a cluster of [HashiCorp Consul](https://www.consul.io/) servers running in
an AWS VPC network, and thus it requires as arguments the identifiers of both
the VPC itself and of the subnets within that VPC.

An alternative design would be to have the `consul_cluster` module describe
its _own_ network resources, but if we did that then it would be hard for
the Consul cluster to coexist with other infrastructure in the same network,
and so where possible we prefer to keep modules relatively small and pass in
their dependencies.

This [dependency inversion](https://en.wikipedia.org/wiki/Dependency_inversion_principle)
approach also improves flexibility for future
refactoring, because the `consul_cluster` module doesn't know or care how
those identifiers are obtained by the calling module. A future refactor may
separate the network creation into its own configuration, and thus we may
pass those values into the module from data sources instead:

```hcl
data "aws_vpc" "main" {
  tags {
    Environment = "production"
  }
}

data "aws_subnet_ids" "main" {
  vpc_id = data.aws_vpc.main.id
}

module "consul_cluster" {
  source = "./modules/aws-consul-cluster"

  vpc_id     = data.aws_vpc.main.id
  subnet_ids = data.aws_subnet_ids.main.ids
}
```

This technique is also an answer to the common situation where in one
configuration a particular object already exists and just needs to be queried,
while in another configuration the equivalent object must be managed directly.
This commonly arises, for example, in development environment scenarios where
certain infrastructure may be shared across development environments for cost
reasons but managed directly inline in production for isolation.

This is the best way to model such situations in Terraform. We do not recommend
attempting to construct "if this already exists then use it, otherwise create it"
conditional configurations; instead, decompose your system into modules and
decide for each configuration whether each modular object is directly managed or
merely used by reference. This makes each configuration a more direct description
of your intent, allowing Terraform to produce a more accurate plan, and making
it clearer to future maintainers how each configuration is expected to behave.

## Multi-cloud Abstractions

Terraform itself intentionally does not attempt to abstract over similar
services offered by different vendors, because we want to expose the full
functionality in each offering and yet unifying multiple offerings behind a
single interface will tend to require a "lowest common denominator" approach.

However, through composition of Terraform modules it is possible to create
your own lightweight multi-cloud abstractions by making your own tradeoffs
about which platform features are important to you.

Opportunities for such abstractions arise in any situation where multiple
vendors implement the same concept, protocol, or open standard. For example,
the basic capabilities of the domain name system are common across all vendors,
and although some vendors differentiate themselves with unique features such
as geolocation and smart load balancing, you may conclude that in your use-case
you are willing to eschew those features in return for creating modules that
abstract the common DNS concepts across multiple vendors:

```hcl
module "webserver" {
  source = "./modules/webserver"
}

locals {
  fixed_recordsets = [
    {
      name = "www"
      type = "CNAME"
      ttl  = 3600
      records = [
        "webserver01",
        "webserver02",
        "webserver03",
      ]
    },
  ]
  server_recordsets = [
    for i, addr in module.webserver.public_ip_addrs : {
      name    = format("webserver%02d", i)
      type    = "A"
      records = [addr]
    }
  ]
}

module "dns_records" {
  source = "./modules/route53-dns-records"

  route53_zone_id = var.route53_zone_id
  recordsets      = concat(local.fixed_recordsets, local.server_recordsets)
}
```

In the above example, we've created a lightweight abstraction in the form of
a "recordset" object. This contains the attributes that describe the general
idea of a DNS recordset that should be mappable onto any DNS provider.

We then instantiate one specific _implementation_ of that abstraction as a
module, in this case deploying our recordsets to Amazon Route53.

If we later wanted to switch to a different DNS provider, we'd need only to
replace the `dns_records` module with a new implementation targeting that
provider, and all of the configuration that _produces_ the recordset
definitions can remain unchanged.

We can create lightweight abstractions like these by defining Terraform object
types representing the concepts involved and then using these object types
for module input variables. In this case, all of our "DNS records"
implementations would have the following variable declared:

```hcl
variable "recordsets" {
  type = object({
    name    = string
    type    = string
    ttl     = number
    records = list(string)
  })
}
```

While DNS serves as a simple example, there are many more opportunities to
exploit common elements across vendors. A more complex example is Kubernetes,
where there are now many different vendors offering hosted Kubernetes clusters
and even more ways to run Kubernetes yourself.

If the common functionality across all of these implementations is sufficient
for your needs, you may choose to implement a set of different modules that
describe a particular Kubernetes cluster implementation and all have the common
trait of exporting the hostname of the cluster as an output value:

```hcl
output "hostname" {
  value = azurerm_kubernetes_cluster.main.fqdn
}
```

You can then write _other_ modules that expect only a Kubernetes cluster
hostname as input and use them interchangably with any of your Kubernetes
cluster modules:

```hcl
module "k8s_cluster" {
  source = "modules/azurerm-k8s-cluster"

  # (Azure-specific configuration arguments)
}

module "monitoring_tools" {
  source = "modules/monitoring_tools"

  cluster_hostname = module.k8s_cluster.hostname
}
```

## Data-only Modules

Most modules contain `resource` blocks and thus describe infrastructure to be
created and managed. It may sometimes be useful to write modules that do not
describe any new infrastructure at all, but merely retrieve information about
existing infrastructure that was created elsewhere using
[data sources](/docs/configuration/data-sources.html).

As with conventional modules, we suggest using this technique only when the
module raises the level of abstraction in some way, in this case by
encapsulating exactly how the data is retrieved.

A common use of this technique is when a system has been decomposed into several
subsystem configurations but there is certain infrastructure that is shared
across all of the subsystems, such as a common IP network. In this situation,
we might write a shared module called `join-network-aws` which can be called
by any configuration that needs information about the shared network when
deployed in AWS:

```
module "network" {
  source = "./modules/join-network-aws"

  environment = "production"
}

module "k8s_cluster" {
  source = "./modules/aws-k8s-cluster"

  subnet_ids = module.network.aws_subnet_ids
}
```

The `network` module itself could retrieve this data in a number of different
ways: it could query the AWS API directly using
[`aws_vpc`](/docs/providers/aws/d/vpc.html)
and
[`aws_subnet_ids`](/docs/providers/aws/d/subnet_ids.html)
data sources, or it could read saved information from a Consul cluster using
[`consul_keys`](https://www.terraform.io/docs/providers/consul/d/keys.html),
or it might read the outputs directly from the state of the configuration that
manages the network using
[`terraform_remote_state`](https://www.terraform.io/docs/providers/terraform/d/remote_state.html).

The key benefit of this approach is that the source of this information can
change over time without updating every configuration that depends on it.
Furthermore, if you design your data-only module with a similar set of outputs
as a corresponding management module, you can swap between the two relatively
easily when refactoring.
