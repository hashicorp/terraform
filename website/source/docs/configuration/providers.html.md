---
layout: "docs"
page_title: "Configuring Providers"
sidebar_current: "docs-config-providers"
---

# Provider Configuration

Providers are responsible in Terraform for managing the lifecycle
of a [resource](/docs/configuration/resource.html): create,
read, update, delete.

Every resource in Terraform is mapped to a provider based
on longest-prefix matching. For example the `aws_instance`
resource type would map to the `aws` provider (if that exists).

Most providers require some sort of configuration to provide
authentication information, endpoint URLs, etc. Provider configuration
blocks are a way to set this information globally for all
matching resources.

This page assumes you're familiar with the
[configuration syntax](/docs/configuration/syntax.html)
already.

## Example

A provider configuration looks like the following:

```
provider "aws" {
	access_key = "foo"
	secret_key = "bar"
	region = "us-east-1"
}
```

## Description

The `provider` block configures the provider of the given `NAME`.
Multiple provider blocks can be used to configure multiple providers.

Terraform matches providers to resources by matching two criteria.
Both criteria must be matched for a provider to manage a resource:

  * They must share a common prefix. Longest matching prefixes are
    tried first. For example, `aws_instance` would choose the
    `aws` provider.

  * The provider must report that it supports the given resource
    type. Providers internally tell Terraform the list of resources
    they support.

Within the block (the `{ }`) is configuration for the resource.
The configuration is dependent on the type, and is documented
[for each provider](/docs/providers/index.html).

## Syntax

The full syntax is:

```
provider NAME {
	CONFIG ...
}
```

where `CONFIG` is:

```
KEY = VALUE

KEY {
	CONFIG
}
```
