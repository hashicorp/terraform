# Terraform Core RPC API

This directory contains package `rpcapi`, which is the main implementation code
for the Terraform Core RPC API.

What follows here is documentation aimed at those who are maintaining or
otherwise contributing to this code.
**This is not end-user-oriented documentation**; information on how to _use_
the RPC API as an external caller belongs elsewhere.

**NOTE WELL!** The RPC API is currently experimental, existing primarily as
a vehicle for the Terraform Stacks private preview. It is subject to arbitrary
breaking changes -- even in patch releases -- until the Terraform Stacks
features are considered stable.

## What is the RPC API?

The RPC API is an integration point for making use of some Terraform Core
functionality within external software.

It's primarily aimed at entirely separate programs using its gRPC server
configured using HashiCorp's `go-plugin` library, although it does also have an
internal access point for use by the parts of this codebase that are
architecturally part of Terraform CLI rather than Terraform Core, to help
reinforce that architectural boundary despite them both currently living in the
same codebase.

The relationship between this package's implementation of the RPC API and
external callers of that API is mechanically similar to the relationship
between the Terraform Plugin Framework/SDK and Terraform Core: it's a
client/server protocol using gRPC as the transport, over a local socket.

The protocol buffers definition for the API lives in
[`terraform1.proto`](./terraform1/terraform1.proto), which when included in
a tagged commit of this repository acts as the source of truth for a particular
version of the API.

## RPC API services

The RPC API exposes a few different services that each wrap different parts
of Terraform Core's functionality. These are broad thematic groupings, but
they are all part of the same API and in particular values returned by one
service are often accepted as input to another.

- `Setup`: This is a special service that's used only to prepare the other
  services for use by performing a negotiation handshake.
  
  Clients must always make exactly one call to `Setup.Handshake` before
  interacting with any other part of this API. That call acts as a capability
  negotiation which might therefore influence the behavior of other subequent
  calls as a measure of forward and backward compatibility.

- `Dependencies`: Deals with some cross-cutting concerns related to dependencies
  such as remote source packages (e.g. external modules) and providers.

- `Stacks`: Provides external access to the Terraform Stacks runtime, including
  planning and applying changes to the infrastructure described by a stack
  configuration.

## API Object Handles

To allow passing live objects between different services and different
functions within the same service, the RPC API uses _handles_, which are
`int64` values that each uniquely identify a live object of a particular
type.

Handles are typically (but not always) created by RPC functions whose names
start with the prefix `Open`, and are later closed by functions whose names
start with `Close`.

Handles persist between calls to the same RPC API process, but are automatically
discarded when that process shuts down. Depending on the handle type, this
automatic discarding may or may not be equivalent to explicitly closing the
handle, and so callers should typically explicitly close handles for objects
they no longer intend to use.

Objects represented by handles can sometimes depend on other objects. In that
case, it might be necessary to close one handle before closing another.

Internally, handles are represented as values of the `handle` generic type,
which is parameterized by the type of the underlying object the handle is
representing. This therefore allows a measure of type safety to help avoid
mistakes like using the wrong kind of handle when calling a function.

In the wire protocol the handle type information is erased, and so when
accepting handles from a client the service implementation must check that
the given handle is of the expected type.

Currently handles are unique across objects of all types, but that's an
implementation detail that clients are not allowed to rely on. If designing
a service which can accept handles of multiple different types, always design
it to accept each handle type as a separate request field, and never rely on
the system's internal state about what type each handle has, so that we can
give the best possible feedback to clients when they have their own bugs that
cause mixups between different handle types.

## Handshake Dynamic Initialization

In order to allow clients to dynamically negotiate capabilities at runtime,
the server implementation of this API uses an extra indirection over the
real service implementations that's implemented in the subdirectory
`dynrpcserver`.

The service implementations registered with the gRPC server are actually
instances of the wrapper stubs in that package. Initially those stubs are
all wrapping nothing at all, and so all calls to the service functions will
return errors.

During a `Setup.Handshake` call, the system finally instantiates the real
service implementations that are implemented within this package directory.
The exact details of what types are instantiated and how they are populated
can vary based on the negotiated capabilities, allowing some flexibility in
how we will handle requests based on those capabilities.

The `dynrpcserver` stubs are automatically generated by a `go:generate`
directive in that package based on the protocol buffers definitions. Therefore
each time we change the set of service functions or the request and response
types for those functions we must first run `make protobuf` to regenerate the
protocol buffers stubs, and then
`go generate ./internal/rpcapi/...` to update the `dynrpcserver` stubs to
match.

## API Entry Points

The main entry point is `rpcapi.CLICommandFactory`, which returns a factory
function intended for use with the `github.com/mitchellh/cli` module that
Terraform CLI uses to route execution into its various subcommands.

Terraform CLI's `package main` binds the subcommand `rpcapi` directly to the
factory returned by that function, thereby providing the smallest possible
amount of Terraform CLI execution before reaching the RPC API. This is
intentional to help reinforce that `rpcapi` is _an alternative to_ using
Terraform CLI, rather than part of Terraform CLI itself, despite the
unavoidable use of some of its early entry-point code to get up and running.

When Terraform CLI itself needs to access Terraform Core functionality that's
exposed by the RPC API, an alternative entry point is
`rpcapi.NewInternalClient`. This function returns an object which provides
access to gRPC clients just as would be used by an external caller accessing
the API when using `go-plugin`, but arranges for its requests to be routed
via local buffers in-process rather than using a socket.

The intent of this "internal client" is to reinforce the architectural boundary
between Terraform CLI and Terraform Core despite them living in the same
codebase. Commands that interact with the internal client could potentially
be factored out into separate codebases in future with only minimal
modification to use `go-plugin` to arrange access instead of using the
internal client.

At the time of writing this documentation there is plenty of surface area in
Terraform CLI that predates the RPC API which accesses Terraform Core
functionality which itself predates the RPC API, and therefore those calls
are made directly via normal function calls. It's fine to continue maintaining
those callers and callees until there's a strong reason to update them, but
most new functionality should be mediated through the RPC API.

In particular, the RPC API is the only public interface to the Terraform Stacks
runtime, and so any Terraform CLI code which is orchestrating the Stacks
runtime _must_ access it through the RPC API internal client, and must not
directly import anything under `./internal/stacks`.
