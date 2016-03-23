# Go Plugin System over RPC

`go-plugin` is a Go (golang) plugin system over RPC. It is the plugin system
that has been in use by HashiCorp tooling for over 3 years. While initially
created for [Packer](https://www.packer.io), it has since been used by
[Terraform](https://www.terraform.io) and [Otto](https://www.ottoproject.io),
with plans to also use it for [Nomad](https://www.nomadproject.io) and
[Vault](https://www.vaultproject.io).

While the plugin system is over RPC, it is currently only designed to work
over a local [reliable] network. Plugins over a real network are not supported
and will lead to unexpected behavior.

This plugin system has been used on millions of machines across many different
projects and has proven to be battle hardened and ready for production use.

## Features

The HashiCorp plugin system supports a number of features:

**Plugins are Go interface implementations.** This makes writing and consuming
plugins feel very natural. To a plugin author: you just implement an
interface as if it were going to run in the same process. For a plugin user:
you just use and call functions on an interface as if it were in the same
process. This plugin system handles the communication in between.

**Complex arguments and return values are supported.** This library
provides APIs for handling complex arguments and return values such
as interfaces, `io.Reader/Writer`, etc. We do this by giving you a library
(`MuxBroker`) for creating new connections between the client/server to
serve additional interfaces or transfer raw data.

**Bidirectional communication.** Because the plugin system supports
complex arguments, the host process can send it interface implementations
and the plugin can call back into the host process.

**Built-in Logging.** Any plugins that use the `log` standard library
will have log data automatically sent to the host process. The host
process will mirror this output prefixed with the path to the plugin
binary. This makes debugging with plugins simple.

**Protocol Versioning.** A very basic "protocol version" is supported that
can be incremented to invalidate any previous plugins. This is useful when
interface signatures are changing, protocol level changes are necessary,
etc. When a protocol version is incompatible, a human friendly error
message is shown to the end user.

**Stdout/Stderr Syncing.** While plugins are subprocesses, they can continue
to use stdout/stderr as usual and the output will get mirrored back to
the host process. The host process can control what `io.Writer` these
streams go to to prevent this from happening.

**TTY Preservation.** Plugin subprocesses are connected to the identical
stdin file descriptor as the host process, allowing software that requires
a TTY to work. For example, a plugin can execute `ssh` and even though there
are multiple subprocesses and RPC happening, it will look and act perfectly
to the end user.

**Host upgrade while a plugin is running.** Plugins can be "reattached"
so that the host process can be upgraded while the plugin is still running.
This requires the host/plugin to know this is possible and daemonize
properly. `NewClient` takes a `ReattachConfig` to determine if and how to
reattach.

## Architecture

The HashiCorp plugin system works by launching subprocesses and communicating
over RPC (using standard `net/rpc`). A single connection is made between
any plugin and the host process, and we use a
[connection multiplexing](https://github.com/hashicorp/yamux)
library to multiplex any other connections on top.

This architecture has a number of benefits:

  * Plugins can't crash your host process: A panic in a plugin doesn't
    panic the plugin user.

  * Plugins are very easy to write: just write a Go application and `go build`.
    Theoretically you could also use another language as long as it can
    communicate the Go `net/rpc` protocol but this hasn't yet been tried.

  * Plugins are very easy to install: just put the binary in a location where
    the host will find it (depends on the host but this library also provides
    helpers), and the plugin host handles the rest.

  * Plugins can be relatively secure: The plugin only has access to the
    interfaces and args given to it, not to the entire memory space of the
    process. More security features are planned (see the coming soon section
    below).

## Usage

To use the plugin system, you must take the following steps. These are
high-level steps that must be done. Examples are available in the
`examples/` directory.

  1. Choose the interface(s) you want to expose for plugins.

  2. For each interface, implement an implementation of that interface
     that communicates over an `*rpc.Client` (from the standard `net/rpc`
     package) for every function call. Likewise, implement the RPC server
     struct this communicates to which is then communicating to a real,
     concrete implementation.

  3. Create a `Plugin` implementation that knows how to create the RPC
     client/server for a given plugin type.

  4. Plugin authors call `plugin.Serve` to serve a plugin from the
     `main` function.

  5. Plugin users use `plugin.Client` to launch a subprocess and request
     an interface implementation over RPC.

That's it! In practice, step 2 is the most tedious and time consuming step.
Even so, it isn't very difficult and you can see examples in the `examples/`
directory as well as throughout our various open source projects.

For complete API documentation, see [GoDoc](https://godoc.org/github.com/hashicorp/go-plugin).

## Roadmap

Our plugin system is constantly evolving. As we use the plugin system for
new projects or for new features in existing projects, we constantly find
improvements we can make.

At this point in time, the roadmap for the plugin system is:

**Cryptographically Secure Plugins.** We'll implement signing plugins
and loading signed plugins in order to allow Vault to make use of multi-process
in a secure way.

**Semantic Versioning.** Plugins will be able to implement a semantic version.
This plugin system will give host processes a system for constraining
versions. This is in addition to the protocol versioning already present
which is more for larger underlying changes.

**Plugin fetching.** We will integrate with [go-getter](https://github.com/hashicorp/go-getter)
to support automatic download + install of plugins. Paired with cryptographically
secure plugins (above), we can make this a safe operation for an amazing
user experience.

## What About Shared Libraries?

When we started using plugins (late 2012, early 2013), plugins over RPC
were the only option since Go didn't support dynamic library loading. Today,
Go still doesn't support dynamic library loading, but they do intend to.
Since 2012, our plugin system has stabilized from millions of users using it,
and has many benefits we've come to value greatly.

For example, we intend to use this plugin system in
[Vault](https://www.vaultproject.io), and dynamic library loading will
simply never be acceptable in Vault for security reasons. That is an extreme
example, but we believe our library system has more upsides than downsides
over dynamic library loading and since we've had it built and tested for years,
we'll likely continue to use it.

Shared libraries have one major advantage over our system which is much
higher performance. In real world scenarios across our various tools,
we've never required any more performance out of our plugin system and it
has seen very high throughput, so this isn't a concern for us at the moment.

