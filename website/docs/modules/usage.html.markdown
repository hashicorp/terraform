---
layout: "docs"
page_title: "Using Modules"
sidebar_current: "docs-modules-usage"
description: Using modules in Terraform is very similar to defining resources.
---

# Module Usage

Using child modules in Terraform is very similar to defining resources:

```shell
module "consul" {
  source  = "hashicorp/consul/aws"
  servers = 3
}
```

You can view the full documentation for configuring modules in the [Module Configuration](/docs/configuration/modules.html) section.

In modules we only specify a name, rather than a name and a type as for resources.
This name is used elsewhere in the configuration to reference the module and
its outputs.

The source tells Terraform what to create. In this example, we instantiate
the [Consul module for AWS](https://registry.terraform.io/modules/hashicorp/consul/aws)
from the [Terraform Registry](https://registry.terraform.io). Other source
types are supported, as described in the following section.

Just like a resource, the a module's configuration can be deleted to destroy the
resources belonging to the module.

## Source

The only required configuration key for a module is the `source` parameter. The
value of this tells Terraform where to download the module's source code.
Terraform comes with support for a variety of module sources.

The recommended source for external modules is a
[Terraform Registry](/docs/registry/index.html), which provides the full
capabilities of modules such as version constraints.
Registry modules are specified using a simple slash-separated path like the
`hashicorp/consul/aws` path used in the above example. The full source string
for each registry module can be found from the registry website.

Terraform also supports modules in local directories, identified by a relative
path starting with either `./` or `../`. Such local modules are useful to
organize code more complex repositories, and are described in more detail
in [_Creating Modules_](/docs/modules/create.html).

Finally, Terraform can download modules directly from various storage providers
and version control systems. These sources do not support versioning and other
registry benefits, but can be convenient for getting started when already
available within an organization. The full list of available sources
are documented in [the module sources documentation](/docs/modules/sources.html).

When a configuration uses modules, they must first be installed by running
[`terraform init`](/docs/commands/init.html):

```shell
$ terraform init
```

This command will download any modules that haven't been updated already,
as well as performing other Terraform working directory initialization such
as installing providers.

By default the command will not check for available updates to already-installed
modules, but you can use the `-update` option to check for available upgrades.
When version constraints are specified (as described in the following section)
a newer version will be used only if it is within the given constraint.

## Module Versions

It is recommended to explicitly constrain the acceptable version numbers for
each external module so that upstream changes aren't automatically adopted,
since this may result in unexpected or unwanted changes changes.

The `version` attribute within the `module` block is used for this purpose:

```shell
module "consul" {
  source  = "hashicorp/consul/aws"
  version = "0.0.5"

  servers = 3
}
```

The `version` attribute value may either be a single explicit version or
a version constraint expression. Constraint expressions use the following
syntax to specify a _range_ of versions that are acceptable:

* `>= 1.2.0`: version 1.2.0 or newer
* `<= 1.2.0`: version 1.2.0 or older
* `~> 1.2`: any non-beta patch release within the `1.2` range
* `>= 1.0.0, <= 2.0.0`: any version between 1.0.0 and 2.0.0 inclusive

When depending on third-party modules, references to specific versions are
recommended since this ensures that updates only happen when convenient to you.

For modules maintained within your organization, a version range strategy
may be appropriate if a semantic versioning methodology is used consistently
or if there is a well-defined release process that avoids unwanted updates.

Version constraints are supported only for modules installed from a module
registry, such as the [Terraform Registry](https://registry.terraform.io/).
Other module sources may provide their own versioning mechanisms within the
source string itself, or they may not support versions at all. In particular,
modules whose sources are local file paths do not support `version` because
they are constrained to share the same version as their caller by being
obtained by the same source repository.

## Configuration

The arguments used in a `module` block, such as the `servers` parameter above,
correspond to [variables](/docs/configuration/variables.html) within the module
itself. You can therefore discover all the available variables for a module by
inspecting the source of it.

The special arguments `source`, `version` and `providers` are exceptions. These
are used for special purposes by Terraform and should therefore not be used
as variable names within a module.

## Outputs

Modules encapsulate their resources. A resource in one module cannot directly depend on resources or attributes in other modules, unless those are exported through [outputs](/docs/configuration/outputs.html). These outputs can be referenced in other places in your configuration, for example:

```hcl
resource "aws_instance" "client" {
  ami               = "ami-408c7f28"
  instance_type     = "t1.micro"
  availability_zone = "${module.consul.server_availability_zone}"
}
```

This is deliberately very similar to accessing resource attributes. Instead of
referencing a resource attribute, however, the expression in this case
references an output of the module.

Just like with resources, interpolation expressions can create implicit
dependencies on resources and other modules. Since modules encapsulate
other resources, however, the dependency is not on the module as a whole
but rather on the `server_availability_zone` output specifically, which
allows Terraform to work on resources in different modules concurrently rather
than waiting for the entire module to be complete before proceeding.

## Providers within Modules

For convenience in simple configurations, child modules automatically inherit
default (un-aliased) provider configurations from their parent. This means that
in most cases only the root module needs explicit `provider` blocks, and then
any defined provider can be freely used with the same settings in child modules.

In more complex situations it may be necessary for a child module to use
different provider settings than its parent. In this situation it is
possible to define
[multiple provider instances](/docs/configuration/providers.html#multiple-provider-instances)
and pass them explicitly and selectively to a child module:

```hcl
# The default "aws" configuration is used for AWS resources in the root
# module where no explicit provider instance is selected.
provider "aws" {
  region = "us-west-1"
}

# A non-default, or "aliased" configuration is also defined for a different
# region.
provider "aws" {
  alias  = "usw2"
  region = "us-west-2"
}

# An example child module is instantiated with the _aliased_ configuration,
# so any AWS resources it defines will use the us-west-2 region.
module "example" {
  source    = "./example"
  providers = {
    aws = "aws.usw2"
  }
}
```

The `providers` argument within a `module` block serves the same purpose as
the `provider` argument within a resource as described for
[multiple provider instances](/docs/configuration/providers.html#multiple-provider-instances),
but is a map rather than a single string because a module may contain resources
from many different providers.

Once the `providers` argument is used in a `module` block it overrides all of
the default inheritance behavior, so it is necessary to enumerate mappings
for _all_ of the required providers. This is to avoid confusion and surprises
when mixing both implicit and explicit provider passing.

Additional provider configurations (those with the `alias` argument set) are
_never_ inherited automatically by child modules, and so must always be passed
explicitly using the `providers` map. For example, a module
that configures connectivity between networks in two AWS regions is likely
to need both a source and a destination region. In that case, the root module
may look something like this:

```hcl
provider "aws" {
  alias  = "usw1"
  region = "us-west-1"
}

provider "aws" {
  alias  = "usw2"
  region = "us-west-2"
}

module "tunnel" {
  source    = "./tunnel"
  providers = {
    "aws.src" = "aws.usw1"
    "aws.dst" = "aws.usw2"
  }
}
```

In the `providers` map, the keys are provider names as expected by the child
module, while the values are the names of corresponding configurations in
the _current_ module. The subdirectory `./tunnel` must then contain
`alias`-only configuration blocks like the following, to declare that it
requires these names to be passed from a `providers` block in the parent's
`module` block:

```
provider "aws" {
  alias = "src"
}

provider "aws" {
  alias = "dst"
}
```

Each resource should then have its own `provider` attribute set to either
`"aws.src"` or `"aws.dst"` to choose which of the two provider instances to use.

It is recommended to use the default inheritance behavior in most cases where
only a single default instance of each provider is used, and switch to
passing providers explicitly only if multiple instances are needed.

In all cases it is recommended to keep explicit provider configurations only in
the root module and pass them (either implicitly or explicitly) down to
descendent modules. This avoids the provider configurations being "lost"
when descendent providers are removed from the configuration. It also allows
the user of a configuration to determine which providers require credentials
by inspecting only the root module.

Provider configurations are used for all operations on resources, including
destroying remote objects and refreshing state. Terraform retains, as part of
its state, a reference to the provider configuration that was most recently
used to apply changes to each resource. When a resource is removed from the
configuration, this record in state is used to locate the appropriate
configuration because the resource's `provider` argument is no longer present
in the configuration.

As a consequence, it is required that all resources created for a particular
provider configuration must be destroyed before that provider configuration is
removed, unless the related resources are re-configured to use a different
provider configuration first.

## Multiple Instances of a Module

A particular module source can be instantiated multiple times:

```hcl
# my_buckets.tf

module "assets_bucket" {
  source = "./publish_bucket"
  name   = "assets"
}

module "media_bucket" {
  source = "./publish_bucket"
  name   = "media"
}
```

```hcl
# publish_bucket/bucket-and-cloudfront.tf

variable "name" {} # this is the input parameter of the module

resource "aws_s3_bucket" "example" {
  # ...
}

resource "aws_iam_user" "deploy_user" {
  # ...
}
```

This example defines a local child module in the `./publish_bucket`
subdirectory. That module has configuration to create an S3 bucket. The module
wraps the bucket and all the other implementation details required to configure
a bucket.

We can then instantiate the module multiple times in our configuration by
giving each instance a unique name -- here `module "assets_bucket"` and
`module "media_bucket"` -- whilst specifying the same `source` value.

Resources from child modules are prefixed with `module.<module-instance-name>`
when displayed in plan output and elsewhere in the UI. For example, the
`./publish_bucket` module contains `aws_s3_bucket.example`, and so the two
instances of this module produce S3 bucket resources with [_resource addresses_](/docs/internals/resource-addressing.html)
`module.assets_bucket.aws_s3_bucket.example` and `module.media_bucket.aws_s3_bucket.example`
respectively. These full addresses are used within the UI and on the command
line, but are not valid within interpolation expressions due to the
encapsulation behavior described above.

When refactoring an existing configuration to introduce modules, moving
resource blocks between modules causes Terraform to see the new location
as an entirely separate resource to the old. Always check the execution plan
after performing such actions to ensure that no resources are surprisingly
deleted.

Each instance of a module may optionally have different providers passed to it
using the `providers` argument described above. This can be useful in situations
where, for example, a duplicated set of resources must be created across
several regions or datacenters.

## Summarizing Modules in the UI

By default, commands such as the [plan command](/docs/commands/plan.html) and
[graph command](/docs/commands/graph.html) will show each resource in a nested
module to represent the full scope of the configuration. For more complex
configurations, the `-module-depth` option may be useful to summarize some or all
of the modules as single objects.

For example, with a configuration similar to what we've built above, the default
graph output looks like the following:

![Terraform Expanded Module Graph](docs/module_graph_expand.png)

If we instead set `-module-depth=0`, the graph will look like this:

![Terraform Module Graph](docs/module_graph.png)

Other commands work similarly with modules. Note that `-module-depth` only
affects how modules are presented in the UI; it does not affect how modules
and their contained resources are processed by Terraform operations.

## Tainting resources within a module

The [taint command](/docs/commands/taint.html) can be used to _taint_ specific
resources within a module:

```shell
$ terraform taint -module=salt_master aws_instance.salt_master
```

It is not possible to taint an entire module. Instead, each resource within
the module must be tainted separately.
