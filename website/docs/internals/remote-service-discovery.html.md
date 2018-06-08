---
layout: "docs"
page_title: "Internals: Remote Service Discovery"
sidebar_current: "docs-internals-remote-service-discovery"
description: |-
  Remote service discovery is a protocol used to locate Terraform-native
  services provided at a user-friendly hostname.
---

# Remote Service Discovery

Terraform implements much of its functionality in terms of remote services.
While in many cases these are generic third-party services that are useful
to many applications, some of these services are tailored specifically to
Terraform's needs. We call these _Terraform-native services_, and Terraform
interacts with them via the remote service discovery protocol described below.

## User-facing Hostname

Terraform-native services are provided, from a user's perspective, at a
user-facing "friendly hostname" which serves as the key for configuration and
for any authentication credentials required.

The discovery protocol's purpose is to map from a user-provided hostname to
the base URL of a particular service. Each host can provide different
combinations of services -- or no services at all! -- and so the discovery
protocol has a secondary purpose of allowing Terraform to identify _which_
services are valid for a given hostname.

For example, module source strings can include a module registry hostname
as their first segment, like `example.com/namespace/name/provider`, and
Terraform uses service discovery to determine whether `example.com` _has_
a module registry, and if so where its API is available.

A user-facing hostname is a fully-specified
[internationalized domain name](https://en.wikipedia.org/wiki/Internationalized_domain_name)
expressed in its Unicode form (the corresponding "punycode" form is not allowed)
which must be resolvable in DNS to an address that has an HTTPS server running
on port 443.

User-facing hostnames are normalized for internal comparison using the
standard Unicode [Nameprep](https://en.wikipedia.org/wiki/Nameprep) algorithm,
which includes converting all letters to lowercase, normalizing combining
diacritics to precomposed form where possible, and various other normalization
steps.

## Discovery Process

Given a hostname, discovery begins by forming an initial discovery URL
using that hostname with the `https:` scheme and the fixed path
`/.well-known/terraform.json`.

For example, given the hostname `example.com` the initial discovery URL
would be `https://example.com/.well-known/terraform.json`.

Terraform then sends a `GET` request to this discovery URL and expects a
JSON response. If the response does not have status 200, does not have a media
type of `application/json` or, if the body cannot be parsed as a JSON object,
then discovery fails and Terraform considers the host to not support _any_
Terraform-native services.

If the response is an HTTP redirect then Terraform repeats this step with the
new location as its discovery URL. Terraform is guaranteed to follow at least
one redirect, but nested redirects are not guaranteed nor recommended.

If the response is a valid JSON object then its keys are Terraform native
service identifiers, consisting of a service type name and a version string
separated by a period. For example, the service identifier for version 1 of
the module registry protocol is `modules.v1`.

The value of each object element is the base URL for the service in question.
This URL may be either absolute or relative, and if relative it is resolved
against the final discovery URL (_after_ following redirects).

The following is an example discovery document declaring support for
version 1 of the module registry protocol:

```json
{
  "modules.v1": "https://modules.example.com/v1/"
}
```

## Supported Services

At present, only one service identifier is in use:

* `modules.v1`: [module registry API version 1](/docs/registry/api.html)

## Authentication

If credentials for the given hostname are available in
[the CLI config](/docs/commands/cli-config.html) then they will be included
in the request for the discovery document.

The credentials may also be provided to endpoints declared in the discovery
document, depending on the requirements of the service in question.

## Non-standard Ports in User-facing Hostnames

It is strongly recommended to provide the discovery document for a hostname
on the standard HTTPS port 443. However, in development environments this is
not always possible or convenient, so Terraform allows a hostname to end
with a port specification consisting of a colon followed by one or more
decimal digits.

When a custom port number is present, the service on that port is expected to
implement HTTPS and respond to the same fixed discovery path.

For day-to-day use it is strongly recommended _not_ to rely on this mechanism
and to instead provide the discovery document on the standard port, since this
allows use of the most user-friendly hostname form.
