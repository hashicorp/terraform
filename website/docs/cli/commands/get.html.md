---
layout: "docs"
page_title: "Command: get"
sidebar_current: "docs-commands-get"
description: "The terraform get command downloads and updates modules."
---

# Command: get

The `terraform get` command is used to download and update
[modules](/docs/language/modules/develop/index.html) mentioned in the root module.

## Usage

Usage: `terraform get [options] PATH`

The modules are downloaded into a `.terraform` subdirectory of the current
working directory. Don't commit this directory to your version control
repository.

The `get` command supports the following option:

* `-update` - If specified, modules that are already downloaded will be
   checked for updates and the updates will be downloaded if present.

* `-no-color` - Disable text coloring in the output.
