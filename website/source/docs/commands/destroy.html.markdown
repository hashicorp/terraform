---
layout: "docs"
page_title: "Command: destroy"
sidebar_current: "docs-commands-destroy"
---

# Command: destroy

The `terraform destroy` command is used to destroy the Terraform-managed
infrastructure.

## Usage

Usage: `terraform destroy [options] [dir]`

Infrastructure managed by Terraform will be destroyed. This will ask for
confirmation before destroying.

This command accepts all the flags that the
[apply command](/docs/commands/apply.html) accepts. If `-input=false` is
set, then the destroy confirmation will not be shown.
