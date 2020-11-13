---
layout: "language"
page_title: "Resources - Configuration Language"
sidebar_current: "docs-config-resources"
description: |-
  Resources are the most important element in a Terraform configuration.
  Each resource corresponds to an infrastructure object, such as a virtual
  network or compute instance.
---

# Resource Blocks

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Resources](../configuration-0-11/resources.html).

> **Hands-on:** Try the [Terraform: Get Started](https://learn.hashicorp.com/collections/terraform/aws-get-started?utm_source=WEBSITE&utm_medium=WEB_IO&utm_offer=ARTICLE_PAGE&utm_content=DOCS) collection on HashiCorp Learn.

_Resources_ are the most important element in the Terraform language.
Each resource block describes one or more infrastructure objects, such
as virtual networks, compute instances, or higher-level components such
as DNS records.

## Resource Syntax

Resource declarations can include a number of advanced features, but only
a small subset are required for initial use. More advanced syntax features,
such as single resource declarations that produce multiple similar remote
objects, are described later in this page.

```hcl
resource "aws_instance" "web" {
  ami           = "ami-a1b2c3d4"
  instance_type = "t2.micro"
}
```

A `resource` block declares a resource of a given type ("aws_instance")
with a given local name ("web"). The name is used to refer to this resource
from elsewhere in the same Terraform module, but has no significance outside
that module's scope.

The resource type and name together serve as an identifier for a given
resource and so must be unique within a module.

Within the block body (between `{` and `}`) are the configuration arguments
for the resource itself. Most arguments in this section depend on the
resource type, and indeed in this example both `ami` and `instance_type` are
arguments defined specifically for [the `aws_instance` resource type](/docs/providers/aws/r/instance.html).

-> **Note:** Resource names must start with a letter or underscore, and may
contain only letters, digits, underscores, and dashes.

## Resource Types

Each resource is associated with a single _resource type_, which determines
the kind of infrastructure object it manages and what arguments and other
attributes the resource supports.

### Providers

Each resource type is implemented by a [provider](/docs/configuration/provider-requirements.html),
which is a plugin for Terraform that offers a collection of resource types. A
provider usually provides resources to manage a single cloud or on-premises
infrastructure platform. Providers are distributed separately from Terraform
itself, but Terraform can automatically install most providers when initializing
a working directory.

In order to manage resources, a Terraform module must specify which providers it
requires. Additionally, most providers need some configuration in order to
access their remote APIs, and the root module must provide that configuration.

For more information, see:

- [Provider Requirements](/docs/configuration/provider-requirements.html), for declaring which
  providers a module uses.
- [Provider Configuration](/docs/configuration/providers.html), for configuring provider settings.

Terraform usually automatically determines which provider to use based on a
resource type's name. (By convention, resource type names start with their
provider's preferred local name.) When using multiple configurations of a
provider (or non-preferred local provider names), you must use the `provider`
meta-argument to manually choose an alternate provider configuration. See
[the `provider` meta-argument](/docs/configuration/meta-arguments/resource-provider.html) for more details.

### Resource Arguments

Most of the arguments within the body of a `resource` block are specific to the
selected resource type. The resource type's documentation lists which arguments
are available and how their values should be formatted.

The values for resource arguments can make full use of
[expressions](/docs/configuration/expressions/index.html) and other dynamic Terraform
language features.

There are also some _meta-arguments_ that are defined by Terraform itself
and apply across all resource types. (See [Meta-Arguments](#meta-arguments) below.)

### Documentation for Resource Types

Every Terraform provider has its own documentation, describing its resource
types and their arguments.

Most publicly available providers are distributed on the
[Terraform Registry](https://registry.terraform.io/browse/providers), which also
hosts their documentation. When viewing a provider's page on the Terraform
Registry, you can click the "Documentation" link in the header to browse its
documentation. Provider documentation on the registry is versioned, and you can
use the dropdown version menu in the header to switch which version's
documentation you are viewing.

To browse the publicly available providers and their documentation, see
[the providers section of the Terraform Registry](https://registry.terraform.io/browse/providers).

-> **Note:** Provider documentation used to be hosted directly on terraform.io,
as part of Terraform's core documentation. Although some provider documentation
might still be hosted here, the Terraform Registry is now the main home for all
public provider docs. (The exception is the built-in
[`terraform` provider](/docs/providers/terraform/index.html) for reading state
data, since it is not available on the Terraform Registry.)

## Resource Behavior

For more information about how Terraform manages resources when applying a
configuration, see
[Resource Behavior](/docs/configuration/blocks/resources/behavior.html).

## Meta-Arguments

The Terraform language defines several meta-arguments, which can be used with
any resource type to change the behavior of resources.

The following meta-arguments are documented on separate pages:

- [`depends_on`, for specifying hidden dependencies](/docs/configuration/meta-arguments/depends_on.html)
- [`count`, for creating multiple resource instances according to a count](/docs/configuration/meta-arguments/count.html)
- [`for_each`, to create multiple instances according to a map, or set of strings](/docs/configuration/meta-arguments/for_each.html)
- [`provider`, for selecting a non-default provider configuration](/docs/configuration/meta-arguments/resource-provider.html)
- [`lifecycle`, for lifecycle customizations](/docs/configuration/meta-arguments/lifecycle.html)
- [`provisioner` and `connection`, for taking extra actions after resource creation](/docs/configuration/blocks/resources/provisioners/index.html)

## Operation Timeouts

Some resource types provide a special `timeouts` nested block argument that
allows you to customize how long certain operations are allowed to take
before being considered to have failed.
For example, [`aws_db_instance`](/docs/providers/aws/r/db_instance.html)
allows configurable timeouts for `create`, `update` and `delete` operations.

Timeouts are handled entirely by the resource type implementation in the
provider, but resource types offering these features follow the convention
of defining a child block called `timeouts` that has a nested argument
named after each operation that has a configurable timeout value.
Each of these arguments takes a string representation of a duration, such
as `"60m"` for 60 minutes, `"10s"` for ten seconds, or `"2h"` for two hours.

```hcl
resource "aws_db_instance" "example" {
  # ...

  timeouts {
    create = "60m"
    delete = "2h"
  }
}
```

The set of configurable operations is chosen by each resource type. Most
resource types do not support the `timeouts` block at all. Consult the
documentation for each resource type to see which operations it offers
for configuration, if any.
