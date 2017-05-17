---
layout: "commands-env"
page_title: "Command: env select"
sidebar_current: "docs-env-sub-select"
description: |-
  The terraform env select command is used to select state environments.
---

# Command: env select

The `terraform env select` command is used to select to a different
environment that is already created.

## Usage

Usage: `terraform env select [NAME]`

This command will select to another environment. The environment must
already be created.

## Example

```
$ terraform env list
  default
* development
  mitchellh-test

$ terraform env select default
Switched to environment "default"!
```
