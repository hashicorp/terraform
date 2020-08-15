---
layout: "language"
page_title: "Modules - Configuration Language"
sidebar_current: "docs-config-modules"
description: |-
  Modules allow multiple resources to be grouped together and encapsulated.
---

# Modules

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Modules](../configuration-0-11/modules.html).

> **Hands-on:** Try the [Reuse Configuration with Modules](https://learn.hashicorp.com/collections/terraform/modules?utm_source=WEBSITE&utm_medium=WEB_IO&utm_offer=ARTICLE_PAGE&utm_content=DOCS) collection on HashiCorp Learn.

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
created, and published in [the dedicated _Modules_
section](/docs/modules/index.html).

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
Terraform. Its value is either the path to a local directory containing the
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

## Transferring Resource State Into Modules

When refactoring an existing configuration to split code into child modules,
moving resource blocks between modules causes Terraform to see the new location
as an entirely different resource from the old. Always check the execution plan
after moving code across modules to ensure that no resources are deleted by
surprise.

If you want to make sure an existing resource is preserved, use
[the `terraform state mv` command](/docs/commands/state/mv.html) to inform
Terraform that it has moved to a different module.

When passing resource addresses to `terraform state mv`, resources within child
modules must be prefixed with `module.<MODULE NAME>.`. If a module was called
with `count` or `for_each` ([see below][inpage-multiple]), its resource
addresses must be prefixed with `module.<MODULE NAME>[<INDEX>].` instead, where
`<INDEX>` matches the `count.index` or `each.key` value of a particular module
instance.

Full resource addresses for module contents are used within the UI and on the
command line, but cannot be used within a Terraform configuration. Only
[outputs](./outputs.html) from a module can be referenced from
elsewhere in your configuration.

## Other Meta-arguments

Along with the `source` meta-argument described above, module blocks have
some optional meta-arguments that have special meaning across all modules,
described in more detail below:

- `version` - A [version constraint string](./version-constraints.html)
  that specifies acceptable versions of the module. Described in detail under
  [Module Versions][inpage-versions] below.

- `count` and `for_each` - Both of these arguments create multiple instances of a
  module from a single `module` block. Described in detail under
  [Multiple Instances of a Module][inpage-multiple] below.

- `providers` - A map whose keys are provider configuration names
  that are expected by child module and whose values are the corresponding
  provider configurations in the calling module. This allows
  [provider configurations to be passed explicitly to child modules](#passing-providers-explicitly).
  If not specified, the child module inherits all of the default (un-aliased)
  provider configurations from the calling module. Described in detail under
  [Providers Within Modules][inpage-providers]

- `depends_on` - Creates explicit dependencies between the entire
  module and the listed targets. This will delay the final evaluation of the
  module, and any sub-modules, until after the dependencies have been applied.
  Modules have the same dependency resolution behavior
  [as defined for managed resources](./resources.html#resource-dependencies).

In addition to the above, the `lifecycle` argument is not currently used by
Terraform but is reserved for planned future features.

Since modules are a complex feature in their own right, further detail
about how modules can be used, created, and published is included in
[the dedicated section on modules](/docs/modules/index.html).

## Module Versions

[inpage-versions]: #module-versions

When using modules installed from a module registry, we recommend explicitly
constraining the acceptable version numbers to avoid unexpected or unwanted
changes.

Use the `version` attribute in the `module` block to specify versions:

```shell
module "consul" {
  source  = "hashicorp/consul/aws"
  version = "0.0.5"

  servers = 3
}
```

The `version` attribute accepts a [version constraint string](./version-constraints.html).
Terraform will use the newest installed version of the module that meets the
constraint; if no acceptable versions are installed, it will download the newest
version that meets the constraint.

Version constraints are supported only for modules installed from a module
registry, such as the public [Terraform Registry](https://registry.terraform.io/)
or [Terraform Cloud's private module registry](/docs/cloud/registry/index.html).
Other module sources can provide their own versioning mechanisms within the
source string itself, or might not support versions at all. In particular,
modules sourced from local file paths do not support `version`; since
they're loaded from the same source repository, they always share the same
version as their caller.

## Multiple Instances of a Module

[inpage-multiple]: #multiple-instances-of-a-module

-> **Note:** Module support for the `for_each` and `count` meta-arguments was
added in Terraform 0.13. Previous versions can only use these arguments with
individual resources.

Use the `for_each` or the `count` argument to create multiple instances of a
module from a single `module` block. These arguments have the same syntax and
type constraints as
[`for_each`](./resources.html#for_each-multiple-resource-instances-defined-by-a-map-or-set-of-strings)
and
[`count`](./resources.html#count-multiple-resource-instances-by-count)
when used with resources.

```hcl
# my_buckets.tf
module "bucket" {
  for_each = toset(["assets", "media"])
  source   = "./publish_bucket"
  name     = "${each.key}_bucket"
}
```

```hcl
# publish_bucket/bucket-and-cloudfront.tf
variable "name" {} # this is the input parameter of the module

resource "aws_s3_bucket" "example" {
  # Because var.name includes each.key in the calling
  # module block, its value will be different for
  # each instance of this module.
  bucket = var.name

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

We declare multiple module instances by using the `for_each` attribute,
which accepts a map (with string keys) or a set of strings as its value. Additionally,
we use the special `each.key` value in our module block, because the
[`each`](/docs/configuration/resources.html#the-each-object) object is available when
we have declared `for_each` on the module block. When using the `count` argument, the
[`count`](/docs/configuration/resources.html#the-count-object) object is available.

Resources from child modules are prefixed with `module.module_name[module index]`
when displayed in plan output and elsewhere in the UI. For a module with without
`count` or `for_each`, the address will not contain the module index as the module's
name suffices to reference the module.

In our example, the `./publish_bucket` module contains `aws_s3_bucket.example`, and so the two
instances of this module produce S3 bucket resources with [resource addresses](/docs/internals/resource-addressing.html) of `module.bucket["assets"].aws_s3_bucket.example`
and `module.bucket["media"].aws_s3_bucket.example` respectively.

## Providers Within Modules

[inpage-providers]: #providers-within-modules

In a configuration with multiple modules, there are some special considerations
for how resources are associated with provider configurations.

Each resource in the configuration must be associated with one provider
configuration. Provider configurations, unlike most other concepts in
Terraform, are global to an entire Terraform configuration and can be shared
across module boundaries. Provider configurations can be defined only in a
root Terraform module.

Providers can be passed down to descendent modules in two ways: either
_implicitly_ through inheritance, or _explicitly_ via the `providers` argument
within a `module` block. These two options are discussed in more detail in the
following sections.

A module intended to be called by one or more other modules must not contain
any `provider` blocks, with the exception of the special
"proxy provider blocks" discussed under
_[Passing Providers Explicitly](#passing-providers-explicitly)_
below.

For backward compatibility with configurations targeting Terraform v0.10 and
earlier Terraform does not produce an error for a `provider` block in a shared
module if the `module` block only uses features available in Terraform v0.10,
but that is a legacy usage pattern that is no longer recommended. A legacy
module containing its own provider configurations is not compatible with the
`for_each`, `count`, and `depends_on` arguments that were introduced in
Terraform v0.13. For more information, see
[Legacy Shared Modules with Provider Configurations](#legacy-shared-modules-with-provider-configurations).

Provider configurations are used for all operations on associated resources,
including destroying remote objects and refreshing state. Terraform retains, as
part of its state, a reference to the provider configuration that was most
recently used to apply changes to each resource. When a `resource` block is
removed from the configuration, this record in the state will be used to locate
the appropriate configuration because the resource's `provider` argument
(if any) will no longer be present in the configuration.

As a consequence, you must ensure that all resources that belong to a
particular provider configuration are destroyed before you can remove that
provider configuration's block from your configuration. If Terraform finds
a resource instance tracked in the state whose provider configuration block is
no longer available then it will return an error during planning, prompting you
to reintroduce the provider configuration.

### Provider Version Constraints in Modules

Although provider _configurations_ are shared between modules, each module must
declare its own [provider requirements](provider-requirements.html), so that
Terraform can ensure that there is a single version of the provider that is
compatible with all modules in the configuration and to specify the
[source address](provider-requirements.html#source-addresses) that serves as
the global (module-agnostic) identifier for a provider.

To declare that a module requires particular versions of a specific provider,
use a `required_providers` block inside a `terraform` block:

```hcl
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = ">= 2.7.0"
    }
  }
}
```

A provider requirement says, for example, "This module requires version v2.7.0
of the provider `hashicorp/aws` and will refer to it as `aws`." It doesn't,
however, specify any of the configuration settings that determine what remote
endpoints the provider will access, such as an AWS region; configuration
settings come from provider _configurations_, and a particular overall Terraform
configuration can potentially have
[several different configurations for the same provider](providers.html#alias-multiple-provider-instances).

If you are writing a shared Terraform module, constrain only the minimum
required provider version using a `>=` constraint. This should specify the
minimum version containing the features your module relies on, and thus allow a
user of your module to potentially select a newer provider version if other
features are needed by other parts of their overall configuration.

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

We recommend using this approach when a single configuration for each provider
is sufficient for an entire configuration.

~> **Note:** Only provider configurations are inherited by child modules, not provider source or version requirements. Each module must [declare its own provider requirements](provider-requirements.html). This is especially important for non-HashiCorp providers.

In more complex situations there may be
[multiple provider configurations](/docs/configuration/providers.html#alias-multiple-provider-configurations),
or a child module may need to use different provider settings than
its parent. For such situations, you must pass providers explicitly.

### Passing Providers Explicitly

When child modules each need a different configuration of a particular
provider, or where the child module requires a different provider configuration
than its parent, you can use the `providers` argument within a `module` block
to explicitly define which provider configurations are available to the
child module. For example:

```hcl
# The default "aws" configuration is used for AWS resources in the root
# module where no explicit provider instance is selected.
provider "aws" {
  region = "us-west-1"
}

# An alternate configuration is also defined for a different
# region, using the alias "usw2".
provider "aws" {
  alias  = "usw2"
  region = "us-west-2"
}

# An example child module is instantiated with the alternate configuration,
# so any AWS resources it defines will use the us-west-2 region.
module "example" {
  source    = "./example"
  providers = {
    aws = aws.usw2
  }
}
```

The `providers` argument within a `module` block is similar to
[the `provider` argument](resources.html#provider-selecting-a-non-default-provider-configuration)
within a resource, but is a map rather than a single string because a module may
contain resources from many different providers.

The keys of the `providers` map are provider configuration names as expected by
the child module, and the values are the names of corresponding configurations
in the _current_ module.

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
    aws.src = aws.usw1
    aws.dst = aws.usw2
  }
}
```

The subdirectory `./tunnel` must then contain _proxy configuration blocks_ like
the following, to declare that it requires its calling module to pass
configurations with these names in its `providers` argument:

```hcl
provider "aws" {
  alias = "src"
}

provider "aws" {
  alias = "dst"
}
```

Each resource should then have its own `provider` attribute set to either
`aws.src` or `aws.dst` to choose which of the two provider configurations to
use.

### Proxy Configuration Blocks

A proxy configuration block is one that contains only the `alias` argument. It
serves as a placeholder for provider configurations passed between modules, and
declares that a module expects to be explicitly passed an additional (aliased)
provider configuration.

-> **Note:** Although a completely empty proxy configuration block is also
valid, it is not necessary: proxy configuration blocks are needed only to
establish which _aliased_ provider configurations a child module expects.
Don't use a proxy configuration block if a module only needs a single default
provider configuration, and don't use proxy configuration blocks only to imply
[provider requirements](./provider-requirements.html).

## Legacy Shared Modules with Provider Configurations

In Terraform v0.10 and earlier there was no explicit way to use different
configurations of a provider in different modules in the same configuration,
and so module authors commonly worked around this by writing `provider` blocks
directly inside their modules, making the module have its own separate
provider configurations separate from those declared in the root module.

However, that pattern had a significant drawback: because a provider
configuration is required to destroy the remote object associated with a
resource instance as well as to create or update it, a provider configuration
must always stay present in the overall Terraform configuration for longer
than all of the resources it manages. If a particular module includes
both resources and the provider configurations for those resources then
removing the module from its caller would violate that constraint: both the
resources and their associated providers would, in effect, be removed
simultaneously.

Terraform v0.11 introduced the mechanisms described in earlier sections to
allow passing provider configurations between modules in a structured way, and
thus we explicitly recommended against writing a child module with its own
provider configuration blocks. However, that legacy pattern continued to work
for compatibility purposes -- though with the same drawback -- until Terraform
v0.13.

Terraform v0.13 introduced the possibility for a module itself to use the
`for_each`, `count`, and `depends_on` arguments, but the implementation of
those unfortunately conflicted with the support for the legacy pattern.

To retain the backward compatibility as much as possible, Terraform v0.13
continues to support the legacy pattern for module blocks that do not use these
new features, but a module with its own provider configurations is not
compatible with `for_each`, `count`, or `depends_on`. Terraform will produce an
error if you attempt to combine these features. For example:

```
Error: Module does not support count

  on main.tf line 15, in module "child":
  15:   count = 2

Module "child" cannot be used with count because it contains a nested provider
configuration for "aws", at child/main.tf:2,10-15.

This module can be made compatible with count by changing it to receive all of
its provider configurations from the calling module, by using the "providers"
argument in the calling module block.
```

To make a module compatible with the new features, you must either remove all
of the `provider` blocks from its definition or, if you need multiple
configurations for the same provider, replace them with
_proxy configuration blocks_ as described in
[Passing Providers Explicitly](#passing-providers-explicitly).

If the new version of the module uses proxy configuration blocks, or if the
calling module needs the child module to use different provider configurations
than its own default provider configurations, the calling module must then
include an explicit `providers` argument to describe which provider
configurations the child module will use:

```hcl
provider "aws" {
  region = "us-west-1"
}

provider "aws" {
  region = "us-east-1"
  alias  = "east"
}

module "child" {
  count = 2
  providers = {
    # By default, the child module would use the
    # default (unaliased) AWS provider configuration
    # using us-west-1, but this will override it
    # to use the additional "east" configuration
    # for its resources instead.
    aws = aws.east
  }
}
```

Since the association between resources and provider configurations is
static, module calls using `for_each` or `count` cannot pass different
provider configurations to different instances. If you need different
instances of your module to use different provider configurations then you
must use a separate `module` block for each distinct set of provider
configurations:

```hcl
provider "aws" {
  alias  = "usw1"
  region = "us-west-1"
}

provider "aws" {
  alias  = "usw2"
  region = "us-west-2"
}

provider "google" {
  alias       = "usw1"
  credentials = "${file("account.json")}"
  project     = "my-project-id"
  region      = "us-west1"
  zone        = "us-west1-a"
}

provider "google" {
  alias       = "usw2"
  credentials = "${file("account.json")}"
  project     = "my-project-id"
  region      = "us-west2"
  zone        = "us-west2-a"
}

module "bucket_w1" {
  source    = "./publish_bucket"
  providers = {
    aws.src    = aws.usw1
    google.src = google.usw2
  }
}

module "bucket_w2" {
  source    = "./publish_bucket"
  providers = {
    aws.src    = aws.usw2
    google.src = google.usw2
  }
}
```

## Tainting resources within a module

The [taint command](/docs/commands/taint.html) can be used to _taint_ specific
resources within a module:

```shell
$ terraform taint module.salt_master.aws_instance.salt_master
```

It is not possible to taint an entire module. Instead, each resource within
the module must be tainted separately.
