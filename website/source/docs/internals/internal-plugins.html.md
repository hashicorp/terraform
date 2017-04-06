---
layout: "docs"
page_title: "Internal Plugins"
sidebar_current: "docs-internals-plugins"
description: |-
  Terraform includes many popular plugins compiled into the main binary.
---

# Internal Plugins

Terraform providers and provisioners are provided via plugins. Each plugin provides an implementation for a specific service, such as AWS, or provisioner, such as bash. Plugins are executed as a separate process and communicate with the main Terraform binary over an RPC interface.

# Upgrading From Versions Earlier Than 0.7

In versions of Terraform prior to 0.7, each plugin shipped as a separate binary. In versions of Terraform >= 0.7, all of the official plugins are shipped as a single binary. This saves a lot of disk space and makes downloads faster for you!

However, when you upgrade you will need to manually delete old plugins from disk. You can do this via something like this, depending on where you installed `terraform`:

	rm /usr/local/bin/terraform-*

If you don't do this you will see an error message like the following:

```text
[WARN] /usr/local/bin/terraform-provisioner-file overrides an internal plugin for file-provisioner.
  If you did not expect to see this message you will need to remove the old plugin.
  See https://www.terraform.io/docs/internals/plugins.html
Error configuring: 2 error(s) occurred:

* Unrecognized remote plugin message: 2|unix|/var/folders/pj/66q7ztvd17v_vgfg8c99gm1m0000gn/T/tf-plugin604337945

This usually means that the plugin is either invalid or simply
needs to be recompiled to support the latest protocol.
* Unrecognized remote plugin message: 2|unix|/var/folders/pj/66q7ztvd17v_vgfg8c99gm1m0000gn/T/tf-plugin647987867

This usually means that the plugin is either invalid or simply
needs to be recompiled to support the latest protocol.
```

## Why Does This Happen?

In previous versions of Terraform all of the plugins were included in a zip file. For example, when you upgraded from 0.6.12 to 0.6.15, the newer version of each plugin file would have replaced the older one on disk, and you would have ended up with the latest version of each plugin.

Going forward there is only one file in the distribution so you will need to perform a one-time cleanup when upgrading from Terraform < 0.7 to Terraform 0.7 or higher.

If you're curious about the low-level details, keep reading!

## Go Plugin Architecture

Terraform is written in the Go programming language. One of Go's interesting properties is that it produces statically-compiled binaries. This means that it does not need to find libraries on your computer to run, and in general only needs to be compatible with your operating system (to make system calls) and with your CPU architecture (so the assembly instructions match the CPU you're running on).

Another property of Go is that it does not support dynamic libraries. It _only_ supports static binaries. This is part of Go's overall design and is the reason why it produces statically-compiled binaries in the first place -- once you have a Go binary for your platform it should _Just Work_.

In other languages, plugins are built using dynamic libraries. Since this is not an option for us in Go we use a network RPC interface instead. This means that each plugin is an independent program, and instead of communicating via shared memory, the main process communicates with the plugin process over HTTP. When you start Terraform, it identifies the plugin you want to use, finds it on disk, runs the other binary, and does some handshaking to make sure they can talk to each other (the error you may see after upgrading is a handshake failure in the RPC code).

### Downsides

There is a significant downside to this approach. Statically compiled binaries are much larger than dynamically-linked binaries because they include everything they need to run. And because Terraform shares a lot of code with its plugins, there is a lot of binary data duplicated between each of these programs.

In Terraform 0.6.15 there were 42 programs in total, using around 750MB on disk. And it turns out that about 600MB of this is duplicate data! This uses up a lot of space on your hard drive and a lot of bandwidth on our CDN. Fortunately, there is a way to resolve this problem.

### Our Solution

In Terraform 0.7 we merged all of the programs into the same binary. We do this by using a special command `terraform internal-plugin` which allows us to invoke a plugin just by calling the same Terraform binary with extra arguments. In essence, Terraform now just calls itself in order to activate the special behavior in each plugin.

### Supporting our Community

> Why would you do this? Why not just eliminate the network RPC interface and simplify everything?

Terraform is an open source project with a large community, and while we maintain a wide range of plugins as part of the core distribution, we also want to make it easy for people anywhere to write and use their own plugins.

By using the network RPC interface, you can build and distribute a plugin for Terraform without having to rebuild Terraform itself. This makes it easy for you to build a Terraform plugin for your organization's internal use, for a proprietary API that you don't want to open source, or to prototype something before contributing it back to the main project.

In theory, because the plugin interface is HTTP, you could even develop a plugin using a completely different programming language! (Disclaimer, you would also have to re-implement the plugin API which is not a trivial amount of work.)

So to conclude, with the RPC interface _and_ internal plugins, we get the best of all of these features: Binaries that _Just Work_, savings from shared code, and extensibility through plugins. We hope you enjoy using these features in Terraform.
