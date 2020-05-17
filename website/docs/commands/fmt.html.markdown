---
layout: "docs"
page_title: "Command: format"
sidebar_current: "docs-commands-format"
description: |-
  The `terraform format` command is used to rewrite Terraform configuration files to a canonical format and style.
---

# Command: format

The `terraform format` command is used to rewrite Terraform configuration files
to a canonical format and style. This command applies a subset of
the [Terraform language style conventions](/docs/configuration/style.html),
along with other minor adjustments for readability.

Other Terraform commands that generate Terraform configuration will produce
configuration files that conform to the style imposed by `terraform format`, so
using this style in your own files will ensure consistency.

The canonical format may change in minor ways between Terraform versions, so
after upgrading Terraform we recommend to proactively run `terraform format`
on your modules along with any other changes you are making to adopt the new
version.

## Usage

Usage: `terraform format [options] [DIR]`

By default, `format` scans the current directory for configuration files. If
the `dir` argument is provided then it will scan that given directory
instead. If `dir` is a single dash (`-`) then `format` will read from standard
input (STDIN).

The command-line flags are all optional. The list of available flags are:

* `-list=false` - Don't list the files containing formatting inconsistencies.
* `-write=false` - Don't overwrite the input files. (This is implied by `-check` or when the input is STDIN.)
* `-diff` - Display diffs of formatting changes
* `-check` - Check if the input is formatted. Exit status will be 0 if
    all input is properly formatted and non-zero otherwise.
* `-recursive` - Also process files in subdirectories. By default, only the given directory (or current directory) is processed.
