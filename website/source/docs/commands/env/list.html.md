---
layout: "commands-env"
page_title: "Command: env list"
sidebar_current: "docs-env-sub-list"
description: |-
  The terraform env list command is used to list all created state environments.
---

# Command: env list

The `terraform env list` command is used to list all created state environments.

## Usage

Usage: `terraform env list`

The command will list all created environments. The current environment
will have an asterisk (`*`) next to it.

## Example

```
$ terraform env list
  default
* development
  mitchellh-test
```
