---
layout: "docs"
page_title: "Command: fmt"
sidebar_current: "docs-commands-fmt"
description: |-
  The `terraform fmt` command is used to rewrite Terraform configuration files to a canonical format and style.
---

# Command: fmt

The `terraform fmt` command is used to rewrite Terraform configuration files
to a canonical format and style.

## Usage

Usage: `terraform fmt [options] [DIR]`

By default, `fmt` scans the current directory for configuration files. If
the `dir` argument is provided then it will scan that given directory
instead. If `dir` is a single dash (`-`) then `fmt` will read from standard
input (STDIN).

The command-line flags are all optional. The list of available flags are:

* `-list=true` - List files whose formatting differs (disabled if using STDIN)
* `-write=true` - Write result to source file instead of STDOUT (disabled if
    using STDIN)
* `-diff=false` - Display diffs of formatting changes
