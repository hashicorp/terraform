---
layout: "docs"
page_title: "Command: fmt"
sidebar_current: "docs-commands-fmt"
description: |-
  The `terraform fmt` command is used to rewrite Terraform configuration files to a canonical format and style.
---

# Command: fmt

The `terraform fmt` command is used to rewrite Terraform configuration files
to a canonical format and style. This command applies a subset of
the [Terraform language style conventions](/docs/configuration/style.html),
along with other minor adjustments for readability.

Other Terraform commands that generate Terraform configuration will produce
configuration files that conform to the style imposed by `terraform fmt`, so
using this style in your own files will ensure consistency.

The canonical format may change in minor ways between Terraform versions, so
after upgrading Terraform we recommend to proactively run `terraform fmt`
on your modules along with any other changes you are making to adopt the new
version.

## Usage

Usage: `terraform fmt [options] [DIR]`

By default, `fmt` scans the current directory for configuration files. If
the `dir` argument is provided then it will scan that given directory
instead. If `dir` is a single dash (`-`) then `fmt` will read from standard
input (STDIN).

The command-line flags are all optional. The list of available flags are:

* `-list=true` - List files whose formatting differs (disabled if using STDIN)
* `-write=true` - Write result to source file instead of STDOUT (disabled if
    using STDIN or -check)
* `-diff=false` - Display diffs of formatting changes
* `-check=false` - Check if the input is formatted. Exit status will be 0 if
    all input is properly formatted and non-zero otherwise.
