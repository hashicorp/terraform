---
layout: "docs"
page_title: "Plugin Basics"
sidebar_current: "docs-plugins-basics"
description: |-
  This page documents the basics of how the plugin system in Terraform works, and how to setup a basic development environment for plugin development if you're writing a Terraform plugin.
---

# Plugin Basics

~> **Advanced topic!** Plugin development is a highly advanced
topic in Terraform, and is not required knowledge for day-to-day usage.
If you don't plan on writing any plugins, this section of the documentation is
not necessary to read. For general use of Terraform, please see our
[Intro to Terraform](/intro/index.html) and [Getting
Started](/intro/getting-started/install.html) guides.

This page documents the basics of how the plugin system in Terraform
works, and how to setup a basic development environment for plugin development
if you're writing a Terraform plugin.

## How it Works

Terraform providers and provisioners are provided via plugins. Each plugin
exposes an implementation for a specific service, such as AWS, or provisioner,
such as bash. Plugins are executed as a separate process and communicate with
the main Terraform binary over an RPC interface.

More details are available in
_[Plugin Internals](/docs/internals/internal-plugins.html)_.

The code within the binaries must adhere to certain interfaces.
The network communication and RPC is handled automatically by higher-level
Terraform libraries. The exact interface to implement is documented
in its respective documentation section.

## Installing Plugins

The [provider plugins distributed by HashiCorp](/docs/providers/index.html) are
automatically installed by `terraform init`. Third-party plugins (both
providers and provisioners) can be manually installed into the user plugins
directory, located at `%APPDATA%\terraform.d\plugins` on Windows and
`~/.terraform.d/plugins` on other systems.

For more information, see:

- [Configuring Providers](/docs/configuration/providers.html)
- [Configuring Providers: Third-party Plugins](/docs/configuration/providers.html#third-party-plugins)

For developer-centric documentation, see:

- [How Terraform Works: Plugin Discovery](/docs/extend/how-terraform-works.html#discovery)

## Developing a Plugin

Developing a plugin is simple. The only knowledge necessary to write
a plugin is basic command-line skills and basic knowledge of the
[Go programming language](http://golang.org).

-> **Note:** A common pitfall is not properly setting up a
<code>$GOPATH</code>. This can lead to strange errors. You can read more about
this [here](https://golang.org/doc/code.html) to familiarize
yourself.

Create a new Go project somewhere in your `$GOPATH`. If you're a
GitHub user, we recommend creating the project in the directory
`$GOPATH/src/github.com/USERNAME/terraform-NAME`, where `USERNAME`
is your GitHub username and `NAME` is the name of the plugin you're
developing. This structure is what Go expects and simplifies things down
the road.

The `NAME` should either begin with `provider-` or `provisioner-`,
depending on what kind of plugin it will be. The repository name will,
by default, be the name of the binary produced by `go install` for
your plugin package.

With the package directory made, create a `main.go` file. This project will
be a binary so the package is "main":

```golang
package main

import (
	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(new(MyPlugin))
}
```

The name `MyPlugin` is a placeholder for the struct type that represents
your plugin's implementation. This must implement either
`terraform.ResourceProvider` or `terraform.ResourceProvisioner`, depending
on the plugin type.

To test your plugin, the easiest method is to copy your `terraform` binary
to `$GOPATH/bin` and ensure that this copy is the one being used for testing.
`terraform init` will search for plugins within the same directory as the
`terraform` binary, and `$GOPATH/bin` is the directory into which `go install`
will place the plugin executable.
