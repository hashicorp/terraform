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

Just like a resource, a module's configuration can be deleted to destroy the
resources belonging to the module.

## Source

The only required configuration key for a module is the `source` parameter. The
value of this tells Terraform where to download the module's source code.
Terraform comes with support for a variety of module sources.

We recommend using modules from the public [Terraform Registry](/docs/registry/index.html)
or from [Terraform Enterprise's private module registry](/docs/enterprise/registry/index.html).
These sources support version constraints for a more reliable experience, and
provide a searchable marketplace for finding the modules you need.

Registry modules are specified using a simple slash-separated path like the
`hashicorp/consul/aws` path used in the above example. The full source string
for each registry module can be found from the registry website.

Terraform also supports modules in local directories, identified by a relative
path starting with either `./` or `../`. Such local modules are useful to
organize code in more complex repositories, and are described in more detail
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
modules, but you can use the `-upgrade` option to check for available upgrades.
When version constraints are specified (as described in the following section)
a newer version will be used only if it is within the given constraint.

## Module Versions

We recommend explicitly constraining the acceptable version numbers for
each external module to avoid unexpected or unwanted changes.

Use the `version` attribute in the `module` block to specify versions:

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
* `~> 1.2.0`: any non-beta version `>= 1.2.0` and `< 1.3.0`, e.g. `1.2.X`
* `~> 1.2`: any non-beta version `>= 1.2.0` and `< 2.0.0`, e.g. `1.X.Y`
* `>= 1.0.0, <= 2.0.0`: any version between 1.0.0 and 2.0.0 inclusive

When depending on third-party modules, references to specific versions are
recommended since this ensures that updates only happen when convenient to you.

For modules maintained within your organization, a version range strategy
may be appropriate if a semantic versioning methodology is used consistently
or if there is a well-defined release process that avoids unwanted updates.

Version constraints are supported only for modules installed from a module
registry, such as the [Terraform Registry](https://registry.terraform.io/) or
[Terraform Enterprise's private module registry](/docs/enterprise/registry/index.html).
Other module sources can provide their own versioning mechanisms within the
source string itself, or might not support versions at all. In particular,
modules sourced from local file paths do not support `version`; since
they're loaded from the same source repository, they always share the same
version as their caller.

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

In a configuration with multiple modules, there are some special considerations
for how resources are associated with provider configurations.

While in principle `provider` blocks can appear in any module, it is recommended
that they be placed only in the _root_ module of a configuration, since this
approach allows users to configure providers just once and re-use them across
all descendent modules.

Each resource in the configuration must be associated with one provider
configuration, which may either be within the same module as the resource
or be passed from the parent module. Providers can be passed down to descendent
modules in two ways: either _implicitly_ through inheritance, or _explicitly_
via the `providers` argument within a `module` block. These two options are
discussed in more detail in the following sections.

In all cases it is recommended to keep explicit provider configurations only in
the root module and pass them (whether implicitly or explicitly) down to
descendent modules. This avoids the provider configurations from being "lost"
when descendent modules are removed from the configuration. It also allows
the user of a configuration to determine which providers require credentials
by inspecting only the root module.

Provider configurations are used for all operations on associated resources,
including destroying remote objects and refreshing state. Terraform retains, as
part of its state, a reference to the provider configuration that was most
recently used to apply changes to each resource. When a `resource` block is
removed from the configuration, this record in the state is used to locate the
appropriate configuration because the resource's `provider` argument (if any)
is no longer present in the configuration.

As a consequence, it is required that all resources created for a particular
provider configuration must be destroyed before that provider configuration is
removed, unless the related resources are re-configured to use a different
provider configuration first.

### Implicit Provider Inheritance

For convenience in simple configurations, a child module automatically inherits
default (un-aliased) provider configurations from its parent. This means that
explicit `provider` blocks appear only in the root module, and downstream
modules can simply declare resources for that provider and have them
automatically associated with the root provider configurations.

For example, the root module might contain only a `provider` block and a
`module` block to instantiate a child module:

```hcl
provider "aws" {
  region = "us-west-1"
}

module "child" {
  source = "./child"
}
```

The child module can then use any resource from this provider with no further
provider configuration required:

```hcl
resource "aws_s3_bucket" "example" {
  bucket = "provider-inherit-example"
}
```

This approach is recommended in the common case where only a single
configuration is needed for each provider across the entire configuration.

In more complex situations there may be [multiple provider instances](/docs/configuration/providers.html#multiple-provider-instances),
or a child module may need to use different provider settings than
its parent. For such situations, it's necessary to pass providers explicitly
as we will see in the next section.

## Passing Providers Explicitly

When child modules each need a different configuration of a particular
provider, or where the child module requires a different provider configuration
than its parent, the `providers` argument within a `module` block can be
used to define explicitly which provider configs are made available to the
child module. For example:

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

The `providers` argument within a `module` block is similar to
the `provider` argument within a resource as described for
[multiple provider instances](/docs/configuration/providers.html#multiple-provider-instances),
but is a map rather than a single string because a module may contain resources
from many different providers.

Once the `providers` argument is used in a `module` block, it overrides all of
the default inheritance behavior, so it is necessary to enumerate mappings
for _all_ of the required providers. This is to avoid confusion and surprises
that may result when mixing both implicit and explicit provider passing.

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
_proxy configuration blocks_ like the following, to declare that it
requires configurations to be passed with these from the `providers` block in
the parent's `module` block:

```hcl
provider "aws" {
  alias = "src"
}

provider "aws" {
  alias = "dst"
}
```

Each resource should then have its own `provider` attribute set to either
`"aws.src"` or `"aws.dst"` to choose which of the two provider instances to use.

At this time it is required to write an explicit proxy configuration block
even for default (un-aliased) provider configurations when they will be passed
via an explicit `providers` block:

```hcl
provider "aws" {
}
```

If such a block is not present, the child module will behave as if it has no
configurations of this type at all, which may cause input prompts to supply
any required provider configuration arguments. This limitation will be
addressed in a future version of Terraform.

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
