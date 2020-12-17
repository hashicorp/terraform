---
layout: "docs"
page_title: "Command: state delete"
sidebar_current: "docs-commands-state-sub-delete"
description: |-
  The terraform state delete command is used to delete a named state.
---

# Command: state delete

The `terraform state delete` command deletes one of the additional states that
you can optionally associate with your configuration.

## Usage

Usage: `terraform state delete [OPTIONS] NAME [DIR]`

If you are using [multiple states](/docs/state/multiple.html) with your
configuration then you can use `terraform state delete` to discard one of
the non-default states.

By default this command will refuse to delete a state that has at least one
resource tracked in it. You can use
[`terraform destroy`](../destroy.html) with that state selected to destroy the
objects associated with those resources.

Deleting a state without destroying all of the associated objects first would
cause Terraform to "forget" those objects and thus require you to delete them
manually outside of Terraform. If you want to delete a state without deleting
the associated objects first &mdash; and thus leaving them behind unmanaged
by Terraform &mdash; you can use the `-force` option to override the usual
error message in that case.

This command accepts the following options:

* `-force` - Delete the workspace even if its state is not empty.
* `-lock=false` - Disable the default behavior of locking the state before deleting it.
* `-lock-timeout=DURATION` - If the lock is already held, wait for the given duration in case the lock is released before returning an error.

## Example

```
$ terraform state delete example
Deleted state "example".
```
