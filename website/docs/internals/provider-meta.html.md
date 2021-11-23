---
layout: "docs"
page_title: "Provider Metadata"
sidebar_current: "docs-internals-provider-meta"
description: |-
  For advanced use cases, modules can provide some pre-defined metadata for providers.
---

# Provider Metadata

In some situations it's beneficial for a provider to offer an interface
through which modules can pass it information unrelated to the resources
in the module, but scoped on a per-module basis.

Provider Metadata allows a provider to declare metadata fields it expects,
which individual modules can then populate independently of any provider
configuration. While provider configurations are often shared between modules,
provider metadata is always module-specific.

Provider Metadata is intended primarily for the situation where an official
module is developed by the same vendor that produced the provider it is
intended to work with, to allow the vendor to indirectly obtain usage
statistics for each module via the provider. For that reason, this
documentation is presented from the perspective of the provider developer
rather than the module developer.

~> **Advanced Topic!** This page covers technical details
of Terraform. You don't need to understand these details to
effectively use Terraform. The details are documented here for
module authors and provider developers working on advanced
functionality.

~> **Experimental Feature!** This functionality is still considered
experimental, and anyone taking advantage of it should [coordinate
with the Terraform team](https://github.com/hashicorp/terraform/issues/new)
to help the team understand how the feature is being used and to make
sure their use case is taken into account as the feature develops.

## Defining the Schema

Before a provider can receive information from a module, the provider
must strictly define the data it can accept. You can do this by setting
the `ProviderMeta` property on your `schema.Provider` struct. Its value
functions similarly to the provider config: a map of strings to the
`schema.Schema` describing the values those strings accept.

## Using the Data

When Terraform calls your provider, you can use the `schema.ResourceData`
that your `Create`, `Read`, and `Update` functions already use to get
access to the provider metadata being passed. First define a struct
that matches your schema, then call the `GetProviderSchema` method on
your `schema.ResourceData`, passing a pointer to a variable of that type.
The variable will be populated with the provider metadata, and will return
an error if there was an issue with parsing the data into the struct.

## Specifying Data in Modules

To include data in your modules, create a `provider_meta` nested block under
your module's `terraform` block, with the name of the provider it's trying
to pass information to:

```hcl
terraform {
  provider_meta "my-provider" {
    hello = "world"
  }
}
```

The `provider_meta` block must match the schema the provider has defined.

## Versioning Your Modules

Any module taking advantage of this functionality must make sure that the
provider metadata supplied matches the schema defined in the provider, and
that the version of Terraform that is being run has support for the provider
metadata functionality. It's therefore recommended that any module taking
advantage of this functionality should specify a minimum Terraform version of
0.13.0 or higher, and a minimum version of each of the providers it specifies
metadata as the first version the schema being used was supported by the
provider.
