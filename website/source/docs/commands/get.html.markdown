---
layout: "docs"
page_title: "Command: get"
sidebar_current: "docs-commands-get"
description: |-
  The `terraform get` command is used to download and update modules.
---

# Command: get

The `terraform get` command is used to download and update
[modules](/docs/modules/index.html).

## Usage

Usage: `terraform get [options] [dir]`

The modules are downloaded into a local `.terraform` folder. This
folder should not be committed to version control.

If a module is already downloaded and the `-update` flag is _not_ set,
Terraform will do nothing. As a result, it is safe (and fast) to run this
command multiple times.

The command-line flags are all optional. The list of available flags are:

* `-update` - If specified, modules that are already downloaded will be
   checked for updates and the updates will be downloaded if present.
