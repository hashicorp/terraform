---
layout: "pluginpatterns"
page_title: "Provider Design Patterns"
sidebar_current: "docs-plugins-patterns-providers"
description: |-
  Design patterns for writing Terraform providers.
---

# Provider Design Patterns

In Terraform, a *provider* represents a third-party service. Providers are
containers for other features related to their services, and they are also
responsible for establishing a connection to their services that can then
be shared between the implementations of those features.

This section is about the user experience design patterns for the provider
itself. Providers contain Resources, so
[the Resource Design Patterns](resources.html) are also relevant for those who
are writing provider plugins.

## Provider Configuration

Many providers require some sort of configuration to specify authentication
credentials, endpoint URLs, etc. These are often specified within a `provider`
block in the configuration:

```
provider "example" {
    url       = "https://api.example.com/"
    api_token = "abcd1234"
}
```

When naming these arguments, it's desirable to follow the terminology used
within the UI or API of the target service. For example, some services refer
to their authentication secrets as "passwords", while others use "auth token",
"API token", "access token", etc. Keeping consistent with the target service
will help users to understand how to map what they see in the UI or API
documentation to what Terraform is requesting.

Almost all provider attributes should be configurable also via environment
variables, with the names conventionally being uppercase versions of the
attribute names prefixed by the provider name. In this example, such
variables might be called `EXAMPLE_URL` and `EXAMPLE_API_TOKEN`.
This is particularly important for authentication arguments like usernames
and API tokens, since teams can set these in the environment to allow
multiple users to collaborate on the same infrastructure without sharing
credentials.

## Provider Granularity

In most cases each provider represents a single service provider or service,
but sometimes the correct granularity is not obvious. For example, there is
a single *Amazon Web Services* provider that contains resources covering much
of their broad line of distinct services; this could equally have been
a separate provider for each AWS service, giving `ec2`, `rds` and `route53`
providers.

Having fewer distinct providers that each cover a broad set of resources is
easier for the user, since it reduces the amount of provider configuration
overhead when writing configurations. A good rule of thumb is to combine sets
of services from the same vendor that can share a common configuration.

For example, Since a user can access all of
the different AWS services with a single set of credentials (access
restrictions notwithstanding), it is convenient to model the whole of AWS as a
single provider that accepts those common credentials.

Conversely, the support for *Microsoft Azure* is split into two distinct
providers, because the product has two distinct API models (one of which
is deprecated) that each require different configuration structures
and use incompatible resource schemas.

## Services with Multiple Endpoints

Some providers represent services that can be accessed at multiple endpoints,
with different data and resources at each. For example, with AWS and
Google Cloud there are multiple *regions* that are each operationally distinct,
and open source services like MySQL can have distinct instances across
many different servers, whether run in-house or outsourced to cloud hosting
providers. Within this section, we'll refer to these various similar
concepts as "regions" for the sake of simplicity.

In most cases a provider is configured for a *specific* region.
For example, the AWS provider has an argument `region`. Because of this, a
user wishing to create resources across multiple regions must instantiate
multiple instances of the provider:

```
provider "aws" {
    alias  = "oregon"     // arbitrary alias chosen by the user
    region = "us-west-2"
}
provider "aws" {
    alias  = "virginia"   // arbitrary alias chosen by the user
    region = "us-east-1"
}
```

Provider developers must decide on the appropriate scope for their provider
based on the expected use-case. The AWS provider *could* have instead expected
`region` to be specified as an argument for each distinct resource, which
could make multi-region architectures simpler to build but would complicate
the the single-region case by requiring an additional redundant attribute in
every resource block.

Most existing providers have one provider instance per distinct region, so
it's likely that future versions of Terraform will continue to
improve the user experience for this case. This model is this preferred
unless a multi-region usage model is expected to be the common case for
a particular service.
