---
layout: "docs"
page_title: "Command: login"
sidebar_current: "docs-commands-login"
description: |-
  The terraform login command can be used to automatically obtain and save an API token for Terraform Cloud, Terraform Enterprise, or any other host that offers Terraform services.
---

# Command: login

The `terraform login` command can be used to automatically obtain and save an
API token for Terraform Cloud, Terraform Enterprise, or any other host that offers Terraform services.

-> **Note:** This command is suitable only for use in interactive scenarios
where it is possible to launch a web browser on the same host where Terraform
is running. If you are running Terraform in an unattended automation scenario,
you can
[configure credentials manually in the CLI configuration](https://www.terraform.io/docs/commands/cli-config.html#credentials).

## Usage

Usage: `terraform login [hostname]`

If you don't provide an explicit hostname, Terraform will assume you want to
log in to Terraform Cloud at `app.terraform.io`.

[Private Terraform Enterprise](/docs/enterprise/index.html) does not currently
support `terraform login`, so to work with Private Terraform Enterprise you
must configure credentials manually as described in
[Remote Backend Configuration](https://www.terraform.io/docs/cloud/run/cli.html#remote-backend-configuration).

## Credentials Storage

By default, Terraform will obtain an API token and save it in plain text in a
local CLI configuration file called `credentials.tfrc.json`. When you run
`terraform login`, it will explain specifically where it intends to save
the API token and give you a chance to cancel if the current configuration is
not as desired.

If you don't wish to store your API token in the default location, you can
optionally configure a
[credentials helper program](cli-config.html#credentials-helpers) which knows
how to store and later retrieve credentials in some other system, such as
your organization's existing secrets management system.

---

## <a name="protocol-v1"></a>Server-side Login Protocol

~> **Note:** You don't need to read this section to _use_ `terraform login`.
The information below is for anyone intending to implement the server side
of `terraform login` in order to offer Terraform-native services in a
third-party system.

Terraform implements `terraform login` by performing an OAuth 2.0 authorization
request using configuration provided by the target host. You may wish to
implement this protocol if you are producing a third-party implementation of
any [Terraform-native services](/docs/internals/remote-service-discovery.html),
such as a Terraform module registry.

First, Terraform uses
[remote service discovery](/docs/internals/remote-service-discovery.html) to
find the OAuth configuration for the host. The host must support the service
name `login.v1` and define for it an object containing OAuth client
configuration values, like this:

```json
{
  "login.v1": {
    "client": "terraform-cli",
    "grant_types": ["authz_code"],
    "authz": "/oauth/authorization",
    "token": "/oauth/token",
    "ports": [10000, 10010],
  }
}
```

The properties within the discovery object are as follows:

* `client` (Required): The `client_id` value to use when making requests, as
  defined in [RFC 6749 section 2.2](https://tools.ietf.org/html/rfc6749#section-2.2).

  Because Terraform is a _public client_ (it is installed on end-user systems
  and thus cannot protect an OAuth client secret), the `client_id` is purely
  advisory and the server must not use it as a guarantee that an authorization
  request is truly coming from Terraform.

* `grant_types` (Optional): A JSON array of strings describing a set of OAuth
  2.0 grant types the server is able to support. A "grant type" selects a
  specific mechanism by which an OAuth server authenticates the request and
  issues an authorization token.

  Terraform CLI currently only supports a single grant type:

  * `authz_code`: [authorization code grant](https://tools.ietf.org/html/rfc6749#section-4.1).
    Both the `authz` and `token` properties are required when `authz_code` is
    present.

  Other grant types may be supported in future versions of Terraform CLI,
  and may impose different requirements on the `authz` and `token` properties.
  If not specified, `grant_types` defaults to `["authz_code"]`.

* `authz` (Required if needed for a given grant type): the server's
  [authorization endpoint](https://tools.ietf.org/html/rfc6749#section-3.1).
  If given as a relative URL, it is resolved from the location of the
  service discovery document.

* `token` (Required if needed for a given grant type): the server's
  [token endpoint](https://tools.ietf.org/html/rfc6749#section-3.2).
  If given as a relative URL, it is resolved from the location of the
  service discovery document.

* `ports` (Optional): A two-element JSON array giving an inclusive range of
  TCP ports that Terraform may use for the temporary HTTP server it will start
  to provide the [redirection endpoint](https://tools.ietf.org/html/rfc6749#section-3.1.2)
  for the first step of an authorization code grant. Terraform opens a TCP
  listen port on the loopback interface in order to receive the response from
  the server's authorization endpoint.

  If not specified, Terraform is free to select any TCP port greater than or
  equal to 1024.
  
  Terraform allows constraining this port range for interoperability with OAuth
  server implementations that require each `client_id` to be associated with
  a fixed set of valid redirection endpoint URLs. Configure such a server
  to expect a range of URLs of the form `http://localhost:10000/`
  with different consecutive port numbers, and then specify that port range
  using `ports`.

  We recommend allowing at least 10 distinct port numbers if possible, and
  assigning them to numbers greater than or equal to 10000, to minimize the
  risk that all of the possible ports will already be in use on a particular
  system.

When requesting an authorization code grant, Terraform CLI implements the
[Proof Key for Code Exchange](https://tools.ietf.org/html/rfc7636) extension in
order to protect against other applications on the system intercepting the
incoming request to the redirection endpoint. We strongly recommend that you
select an OAuth server implementation that also implements this extension and
verifies the code challenge sent to the token endpoint.

Terraform CLI does not support OAuth refresh tokens or token expiration. If your
server issues time-limited tokens, Terraform CLI will simply begin receiving
authorization errors once the token expires, after which the user can run
`terraform login` again to obtain a new token.

-> **Note:** As a special case, Terraform can use a
[Resource Owner Password Credentials Grant](https://tools.ietf.org/html/rfc6749#section-4.3)
only when interacting with `app.terraform.io` ([Terraform Cloud](/docs/cloud/)),
under the recommendation in the OAuth specification to use this grant type only
when the client and server are closely related. The `password` grant type is
not supported for any other hostname and will be ignored.
