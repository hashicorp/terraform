---
layout: "commands-env"
page_title: "Command: env delete"
sidebar_current: "docs-env-sub-delete"
description: |-
  The terraform env delete command is used to create a delete state environment.
---

# Command: env delete

The `terraform env delete` command is used to delete an existing environment.

## Usage

Usage: `terraform env delete [NAME]`

This command will delete the specified environment.

To delete an environment, it must already exist, it must be empty, and
it must not be your current environment. If the environment
is not empty, Terraform will not allow you to delete it without the
`-force` flag.

If you delete a non-empty state (via force), then resources may become
"dangling". These are resources that Terraform no longer manages since
a state doesn't point to them, but still physically exist. This is sometimes
preferred: you want Terraform to stop managing resources. Most of the time,
however, this is not intended so Terraform protects you from doing this.

The command-line flags are all optional. The list of available flags are:

* `-force` - Delete the state even if non-empty. Defaults to false.

## Example

```
$ terraform env delete example
Deleted environment "example"!
```
