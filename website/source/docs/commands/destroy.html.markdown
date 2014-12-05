---
layout: "docs"
page_title: "Command: destroy"
sidebar_current: "docs-commands-destroy"
description: |-
  The `terraform destroy` command is used to destroy the Terraform-managed infrastructure.
---

# Command: destroy

The `terraform destroy` command is used to destroy the Terraform-managed
infrastructure.

## Usage

Usage: `terraform destroy [options] [dir]`

Infrastructure managed by Terraform will be destroyed. This will ask for
confirmation before destroying.

This command accepts all the flags that the
[apply command](/docs/commands/apply.html) accepts. If `-force` is
set, then the destroy confirmation will not be shown.
