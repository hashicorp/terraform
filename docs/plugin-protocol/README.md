# Terraform Plugin Protocol

This directory contains documentation about the physical wire protocol that
Terraform Core uses to communicate with provider plugins.

Most providers are not written directly against this protocol. Instead, prefer
to use an SDK that implements this protocol and write the provider against
the SDK's API.

----

**If you want to write a plugin for Terraform, please refer to
[Extending Terraform](https://www.terraform.io/docs/extend/index.html) instead.**

This documentation is for those who are developing _Terraform SDKs_, rather
than those implementing plugins.

----

From Terraform v0.12.0 onwards, Terraform's plugin protocol is built on
[gRPC](https://grpc.io/). This directory contains `.proto` definitions of
different versions of Terraform's protocol.

Only `.proto` files published as part of Terraform release tags are actually
official protocol versions. If you are reading this directory on the `main`
branch or any other development branch then it may contain protocol definitions
that are not yet finalized and that may change before final release.

## RPC Plugin Model

Terraform plugins are normal executable programs that, when launched, expose
gRPC services on a server accessed via the loopback interface. Terraform Core
discovers and launches plugins, waits for a handshake to be printed on the
plugin's `stdout`, and then connects to the indicated port number as a
gRPC client.

For this reason, we commonly refer to Terraform Core itself as the plugin
"client" and the plugin program itself as the plugin "server". Both of these
processes run locally, with the server process appearing as a child process
of the client. Terraform Core controls the lifecycle of these server processes
and will terminate them when they are no longer required.

The startup and handshake protocol is not currently documented. We hope to
document it here or to link to external documentation on it in future.

## Versioning Strategy

The Plugin Protocol uses a versioning strategy that aims to allow gradual
enhancements to the protocol while retaining compatibility, but also to allow
more significant breaking changes from time to time while allowing old and
new plugins to be used together for some period.

The versioning strategy described below was introduced with protocol version
5.0 in Terraform v0.12. Prior versions of Terraform and prior protocol versions
do not follow this strategy.

The authoritative definition for each protocol version is in this directory
as a Protocol Buffers (protobuf) service definition. The files follow the
naming pattern `tfpluginX.Y.proto`, where X is the major version and Y
is the minor version.

### Major and minor versioning

The minor version increases for each change introducing optional new
functionality that can be ignored by implementations of prior versions. For
example, if a new field were added to an response message, it could be a minor
release as long as Terraform Core can provide some default behavior when that
field is not populated.

The major version increases for any significant change to the protocol where
compatibility is broken. However, Terraform Core and an SDK may both choose
to support multiple major versions at once: the plugin handshake includes a
negotiation step where client and server can work together to select a
mutually-supported major version.

The major version number is encoded into the protobuf package name: major
version 5 uses the package name `tfplugin5`, and one day major version 6
will switch to `tfplugin6`. This change of name allows a plugin server to
implement multiple major versions at once, by exporting multiple gRPC services.
Minor version differences rely instead on feature-detection mechanisms, so they
are not represented directly on the wire and exist primarily as a human
communication tool to help us easily talk about which software supports which
features.

## Version compatibility for Core, SDK, and Providers

A particular version of Terraform Core has both a minimum minor version it
requires and a maximum major version that it supports. A particular version of
Terraform Core may also be able to optionally use a newer minor version when
available, but fall back on older behavior when that functionality is not
available.

Likewise, each provider plugin release is compatible with a set of versions.
The compatible versions for a provider are a list of major and minor version
pairs, such as "4.0", "5.2", which indicates that the provider supports the
baseline features of major version 4 and supports major version 5 including
the enhancements from both minor versions 1 and 2. This provider would
therefore be compatible with a Terraform Core release that supports only
protocol version 5.0, since major version 5 is supported and the optional
5.1 and 5.2 enhancements will be ignored.

If Terraform Core and the plugin do not have at least one mutually-supported
major version, Terraform Core will return an error from `terraform init`
during plugin installation:

```
Provider "aws" v1.0.0 is not compatible with Terraform v0.12.0.

Provider version v2.0.0 is the earliest compatible version.
Select it with the following version constraint:

    version = "~> 2.0.0"
```

```
Provider "aws" v3.0.0 is not compatible with Terraform v0.12.0.
Provider version v2.34.0 is the latest compatible version. Select 
it with the following constraint:

    version = "~> 2.34.0"

Alternatively, upgrade to the latest version of Terraform for compatibility with newer provider releases.
```

The above messages are for plugins installed via `terraform init` from a
Terraform registry, where the registry API allows Terraform Core to recognize
the protocol compatibility for each provider release. For plugins that are
installed manually to a local plugin directory, Terraform Core has no way to
suggest specific versions to upgrade or downgrade to, and so the error message
is more generic:

```
The installed version of provider "example" is not compatible with Terraform v0.12.0.

This provider was loaded from:
     /usr/local/bin/terraform-provider-example_v0.1.0
```

## Adding/removing major version support in SDK and Providers

The set of supported major versions is decided by the SDK used by the plugin.
Over time, SDKs will add support for new major versions and phase out support
for older major versions.

In doing so, the SDK developer passes those capabilities and constraints on to
any provider using their SDK, and that will in turn affect the compatibility
of the plugin in ways that affect its semver-based version numbering:

- If an SDK upgrade adds support for a new provider protocol, that will usually
  be considered a new feature and thus warrant a new minor version.
- If an SDK upgrade removes support for an old provider protocol, that is
  always a breaking change and thus requires a major release of the provider.

For this reason, SDK developers must be clear in their release notes about
the addition and removal of support for major versions.

Terraform Core also makes an assumption about major version support when
it produces actionable error messages for users about incompatibilities:
a particular protocol major version is supported for a single consecutive
range of provider releases, with no "gaps".

## Using the protobuf specifications in an SDK

If you wish to build an SDK for Terraform plugins, an early step will be to
copy one or more `.proto` files from this directory into your own repository
(depending on which protocol versions you intend to support) and use the
`protoc` protocol buffers compiler (with gRPC extensions) to generate suitable
RPC stubs and types for your target language.

For example, if you happen to be targeting Python, you might generate the
stubs using a command like this:

```
protoc --python_out=. --grpc_python_out=. tfplugin5.1.proto
```

You can find out more about the tool usage for each target language in
[the gRPC Quick Start guides](https://grpc.io/docs/quickstart/).

The protobuf specification for a version is immutable after it has been
included in at least one Terraform release. Any changes will be documented in
a new `.proto` file establishing a new protocol version.

The protocol buffer compiler will produce some sort of library object appropriate
for the target language, which depending on the language might be called a
module, or a package, or something else. We recommend to include the protocol
major version in your module or package name so that you can potentially
support multiple versions concurrently in future. For example, if you are
targeting major version 5 you might call your package or module `tfplugin5`.

To upgrade to a newer minor protocol version, copy the new `.proto` file
from this directory into the same location as your previous version, delete
the previous version, and then run the protocol buffers compiler again
against the new `.proto` file. Because minor releases are backward-compatible,
you can simply update your previous stubs in-place rather than creating a
new set alongside.

To support a new _major_ protocol version, create a new package or module
and copy the relevant `.proto` file into it, creating a separate set of stubs
that can in principle allow your SDK to support both major versions at the
same time. We recommend supporting both the previous and current major versions
together for a while across a major version upgrade so that users can avoid
having to upgrade both Terraform Core and all of their providers at the same
time, but you can delete the previous major version stubs once you remove
support for that version.

**Note:** Some of the `.proto` files contain statements about being updated
in-place for minor versions. This reflects an earlier version management
strategy which is no longer followed. The current process is to create a
new file in this directory for each new minor version and consider all
previously-tagged definitions as immutable. The outdated comments in those
files are retained in order to keep the promise of immutability, even though
it is now incorrect.
