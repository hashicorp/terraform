---
layout: "commands-env"
page_title: "Command: env switch"
sidebar_current: "docs-env-sub-switch"
description: |-
  The terraform env switch command is used to switch state environments.
---

# Command: env switch

The `terraform env switch` command is used to switch to a different
environment that is already created.

## Usage

Usage: `terraform env switch [NAME]`

This command will switch to another environment. The environment must
already be created.

## Example

```
$ terraform env list
  default
* development
  mitchellh-test

$ terraform env switch default
Switch to environment "default"!
```
