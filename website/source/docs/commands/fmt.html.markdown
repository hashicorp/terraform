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

`fmt` scans the current directory for configuration files.

The command-line flags are all optional. The list of available flags are:

* `-list=true` - List files whose formatting differs
* `-write=true` - Write result to source file instead of STDOUT
* `-diff=false` - Display diffs instead of rewriting files
