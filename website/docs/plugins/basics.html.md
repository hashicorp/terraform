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
[Internal Docs](/docs/internals/internal-plugins.html).

The code within the binaries must adhere to certain interfaces.
The network communication and RPC is handled automatically by higher-level
Terraform libraries. The exact interface to implement is documented
in its respective documentation section.

## Installing a Plugin

To install a plugin, put the binary somewhere on your filesystem, then
configure Terraform to be able to find it. The configuration where plugins
are defined is `~/.terraformrc` for Unix-like systems and
`%APPDATA%/terraform.rc` for Windows.

An example that configures a new provider is shown below:

```hcl
providers {
  privatecloud = "/path/to/privatecloud"
}
```

The key `privatecloud` is the _prefix_ of the resources for that provider.
For example, if there is `privatecloud_instance` resource, then the above
configuration would work. The value is the name of the executable. This
can be a full path. If it isn't a full path, the executable will be looked
up on the `PATH`.

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

With the directory made, create a `main.go` file. This project will
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

And that's basically it! You'll have to change the argument given to
`plugin.Serve` to be your actual plugin, but that is the only change
you'll have to make. The argument should be a structure implementing
one of the plugin interfaces (depending on what sort of plugin
you're creating).

Terraform plugins must follow a very specific naming convention of
`terraform-TYPE-NAME`. For example, `terraform-provider-aws`, which
tells Terraform that the plugin is a provider that can be referenced
as "aws".
