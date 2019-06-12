---
layout: "docs"
page_title: "Modules - Configuration Language"
sidebar_current: "docs-config-modules"
description: |-
  Modules allow multiple resources to be grouped together and encapsulated.
---

# Modules

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Modules](../configuration-0-11/modules.html).

A _module_ is a container for multiple resources that are used together.

Every Terraform configuration has at least one module, known as its
_root module_, which consists of the resources defined in the `.tf` files in
the main working directory.

A module can call other modules, which lets you include the child module's
resources into the configuration in a concise way. Modules
can also be called multiple times, either within the same configuration or
in separate configurations, allowing resource configurations to be packaged
and re-used.

This page describes how to call one module from another. Other pages in this
section of the documentation describe the different elements that make up
modules, and there is further information about how modules can be used,
created, and published in [the dedicated _Modules_ section](/docs/modules/index.html).

## Calling a Child Module

To _call_ a module means to include the contents of that module into the
configuration with specific values for its
[input variables](./variables.html). Modules are called
from within other modules using `module` blocks:

```hcl
module "servers" {
  source = "./app-cluster"

  servers = 5
}
```

A module that includes a `module` block like this is the _calling module_ of the
child module.

The label immediately after the `module` keyword is a local name, which the
calling module can use to refer to this instance of the module.

Within the block body (between `{` and `}`) are the arguments for the module.
Most of the arguments correspond to [input variables](./variables.html)
defined by the module, including the `servers` argument in the above example.
Terraform also defines a few meta-arguments that are reserved by Terraform
and used for its own purposes; we will discuss those throughout the rest of
this section.

All modules require a `source` argument, which is a meta-argument defined by
Terraform CLI. Its value is either the path to a local directory of the
module's configuration files, or a remote module source that Terraform should
download and use. This value must be a literal string with no template
sequences; arbitrary expressions are not allowed. For more information on
possible values for this argument, see [Module Sources](/docs/modules/sources.html).

The same source address can be specified in multiple `module` blocks to create
multiple copies of the resources defined within, possibly with different
variable values.

After adding, removing, or modifying `module` blocks, you must re-run
`terraform init` to allow Terraform the opportunity to adjust the installed
modules. By default this command will not upgrade an already-installed module;
use the `-upgrade` option to instead upgrade to the newest available version.

## Accessing Module Output Values

The resources defined in a module are encapsulated, so the calling module
cannot access their attributes directly. However, the child module can
declare [output values](./outputs.html) to selectively
export certain values to be accessed by the calling module.

For example, if the `./app-cluster` module referenced in the example above
exported an output value named `instance_ids` then the calling module
can reference that result using the expression `module.servers.instance_ids`:

```hcl
resource "aws_elb" "example" {
  # ...

  instances = module.servers.instance_ids
}
```

For more information about referring to named values, see
[Expressions](./expressions.html).

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
[Terraform Cloud's private module registry](/docs/cloud/registry/index.html).
Other module sources can provide their own versioning mechanisms within the
source string itself, or might not support versions at all. In particular,
modules sourced from local file paths do not support `version`; since
they're loaded from the same source repository, they always share the same
version as their caller.

## Other Meta-arguments

Along with the `source` meta-argument described above, module blocks have
some more meta-arguments that have special meaning across all modules,
described in more detail in other sections:

* `version` - (Optional) A [version constraint](#module-versions)
  string that specifies which versions of the referenced module are acceptable.
  The newest version matching the constraint will be used. `version` is supported
  only for modules retrieved from module registries.

* `providers` - (Optional) A map whose keys are provider configuration names
  that are expected by child module and whose values are corresponding
  provider names in the calling module. This allows
  [provider configurations to be passed explicitly to child modules](#passing-providers-explicitly).
  If not specified, the child module inherits all of the default (un-aliased)
  provider configurations from the calling module.

In addition to the above, the argument names `count`, `for_each` and
`lifecycle` are not currently used by Terraform but are reserved for planned
future features.

Since modules are a complex feature in their own right, further detail
about how modules can be used, created, and published is included in
[the dedicated section on modules](/docs/modules/index.html).

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
    aws.src = "aws.usw1"
    aws.dst = "aws.usw2"
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

A proxy configuration block is one that is either completely empty or that
contains only the `alias` argument. It serves as a placeholder for
provider configurations passed between modules. Although an empty proxy
configuration block is valid, it is not necessary: proxy configuration blocks
are needed only to establish which _alias_ provider configurations a child
module is expecting.

A proxy configuration block must not include the `version` argument. To specify
version constraints for a particular child module without creating a local
module configuration, use the [`required_providers`](/docs/configuration/terraform.html#specifying-required-provider-versions)
setting inside a `terraform` block.

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

## Tainting resources within a module

The [taint command](/docs/commands/taint.html) can be used to _taint_ specific
resources within a module:

```shell
$ terraform taint -module=salt_master aws_instance.salt_master
```

It is not possible to taint an entire module. Instead, each resource within
the module must be tainted separately.
