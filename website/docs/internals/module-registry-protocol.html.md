---
layout: "docs"
page_title: "Module Registry Protocol"
sidebar_current: "docs-internals-module-registry-protocol"
description: |-
  The module registry protocol is implemented by a host intending to be the
  host of one or more Terraform modules, specifying which modules are available
  and where to find their distribution packages.
---

# Module Registry Protocol

-> Third-party module registries are supported only in Terraform CLI 0.11 and later. Prior versions do not support this protocol.

The module registry protocol is what Terraform CLI uses to discover metadata
about modules available for installation and to locate the distribution
package for a selected module.

The primary implementation of this protocol is the public
[Terraform Registry](https://registry.terraform.io/) at `registry.terraform.io`.
By writing and deploying your own implementation of this protocol, you can
create a separate registry to distribute your own modules, as an alternative to
publishing them on the public Terraform Registry.

The public Terraform Registry implements a superset of the API described on
this page, in order to capture additional information used in the registry UI.
For information on those extensions, see
[Terraform Registry HTTP API](/docs/registry/api.html). Third-party registry
implementations may choose to implement those extensions if desired, but
Terraform CLI itself does not use them.

## Module Addresses

Each Terraform module has an associated address. A module address has the
syntax `hostname/namespace/name/system`, where:

* `hostname` is the hostname of the module registry that serves this module.
* `namespace` is the name of a namespace, unique on a particular hostname, that
  can contain one or more modules that are somehow related. On the public
  Terraform Registry the "namespace" represents the organization that is
  packaging and distributing the module.
* `name` is the module name, which generally names the abstraction that the
  module is intending to create.
* `system` is the name of a remote system that the module is primarily written
  to target. For multi-cloud abstractions, there can be multiple modules with
  addresses that differ only in "system" to reflect provider-specific
  implementations of the abstraction, like
  `registry.terraform.io/hashicorp/consul/aws` vs.
  `registry.terraform.io/hashicorp/consul/azurerm`. The system name commonly
  matches the type portion of the address of an official provider, like `aws`
  or `azurerm` in the above examples, but that is not required and so you can
  use whichever system keywords make sense for the organization of your
  particular registry.

The `hostname/` portion of a module address (including its slash delimiter)
is optional, and if omitted defaults to `registry.terraform.io/`.

For example:

* `hashicorp/consul/aws` is a shorthand for
  `registry.terraform.io/hashicorp/consul/aws`, which is a module on the
  public registry for deploying Consul clusters in Amazon Web Services.
* `example.com/awesomecorp/consul/happycloud` is a hypothetical module published
  on a third-party registry.

If you intend to share a module you've developed for use by all Terraform
users, please consider publishing it into the public
[Terraform Registry](https://registry.terraform.io/) to make your module more
discoverable. You only need to implement this module registry protocol if you
wish to publish modules whose addresses include a different hostname that is
under your control.

## Module Versions

Each distinct module address has associated with it a set of versions, each
of which has an associated version number. Terraform assumes version numbers
follow the [Semantic Versioning 2.0](https://semver.org/) conventions, with
the user-facing behavior of the module serving as the "public API".

Each `module` block may select a distinct version of a module, even if multiple
blocks have the same source address.

## Service Discovery

The module registry protocol begins with Terraform CLI using
[Terraform's remote service discovery protocol](./remote-service-discovery.html),
with the hostname in the module address acting as the "User-facing Hostname".

The service identifier for the module registry protocol is `modules.v1`.
Its associated string value is the base URL for the relative URLs defined in
the sections that follow.

For example, the service discovery document for a host that _only_ implements
the module registry protocol might contain the following:

```json
{
  "modules.v1": "/terraform/modules/v1/"
}
```

If the given URL is a relative URL then Terraform will interpret it as relative
to the discovery document itself. The specific module registry protocol
endpoints are defined as URLs relative to the given base URL, and so the
specified base URL should generally end with a slash to ensure that those
relative paths will be resolved as expected.

The following sections describe the various operations that a module
registry must implement to be compatible with Terraform CLI's module
installer. The indicated URLs are all relative to the URL resulting from
service discovery, as described above. We use the current URLs on
Terraform Registry as working examples, assuming that the caller already
performed service discovery on `registry.terraform.io` to learn the base URL.

The URLs are shown with the convention that a path portion with a colon `:`
prefix is a placeholder for a dynamically-selected value, while all other
path portions are literal. For example, in `:namespace/:type/versions`,
the first two path portions are placeholders while the third is literally
the string "versions".

## List Available Versions for a Specific Module

This is the primary endpoint for resolving module sources, returning the
available versions for a given fully-qualified module.

| Method | Path                                  | Produces                   |
| ------ | ------------------------------------- | -------------------------- |
| `GET`  | `:namespace/:name/:provider/versions`   | `application/json`         |

### Parameters

- `namespace` `(string: <required>)` - The user or organization the module is
  owned by. This is required and is specified as part of the URL path.

- `name` `(string: <required>)` - The name of the module.
  This is required and is specified as part of the URL path.

- `system` `(string: <required>)` - The name of the target system.
  This is required and is specified as part of the URL path.

### Sample Request

```text
$ curl 'https://registry.terraform.io/v1/modules/hashicorp/consul/aws/versions'
```

### Sample Response

The `modules` array in the response always includes the requested module as the
first element.

Other elements of this list are not currently used. Third-party implementations
should always use a single-element list for forward compatiblity with possible
future extensions to the protocol.

Each returned module has an array of available versions, which Terraform
matches against any version constraints given in configuration.

```json
{
   "modules": [
      {
         "versions": [
            {"version": "1.0.0"},
            {"version": "1.1.0"},
            {"version": "2.0.0"}
         ]
      }
   ]
}
```

Return `404 Not Found` to indicate that no module is available with the
requested namespace, name, and provider

## Download Source Code for a Specific Module Version

This endpoint downloads the specified version of a module for a single provider.

| Method | Path                                                   | Produces                   |
| ------ | ------------------------------------------------------ | -------------------------- |
| `GET`  | `:namespace/:name/:system/:version/download`           | `application/json`         |

### Parameters

- `namespace` `(string: <required>)` - The user the module is owned by.
  This is required and is specified as part of the URL path.

- `name` `(string: <required>)` - The name of the module.
  This is required and is specified as part of the URL path.

- `provider` `(string: <required>)` - The name of the target system.
  This is required and is specified as part of the URL path.

- `version` `(string: <required>)` - The version of the module.
  This is required and is specified as part of the URL path.

### Sample Request

```text
$ curl -i 'https://registry.terraform.io/v1/modules/hashicorp/consul/aws/0.0.1/download'
```

### Sample Response

```text
HTTP/1.1 204 No Content
Content-Length: 0
X-Terraform-Get: https://api.github.com/repos/hashicorp/terraform-aws-consul/tarball/v0.0.1//*?archive=tar.gz
```

A successful response has no body, and includes the location from which the
module version's source can be downloaded in the `X-Terraform-Get` header.
The value of this header accepts the same values as the `source` argument
in a `module` block in Terraform configuration, as described in
[Module Sources](https://www.terraform.io/docs/language/modules/sources.html),
except that it may not recursively refer to another module registry address.

The value of `X-Terraform-Get` may instead be a relative URL, indicated by
beginning with `/`, `./` or `../`, in which case it is resolved relative to
the full URL of the download endpoint to produce
[an HTTP URL module source](/docs/language/modules/sources.html#http-urls).
