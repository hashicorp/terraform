---
layout: "commands-state"
page_title: "Command: state push"
sidebar_current: "docs-commands-state-sub-push"
description: |-
  The `terraform state push` command pushes items to the Terraform state.
---

# Command: state push

The `terraform state push` command is used to manually upload a local
state file to [remote state](/docs/state/remote.html). This command also
works with local state.

This command should rarely be used. It is meant only as a utility in case
manual intervention is necessary with the remote state.

## Usage

Usage: `terraform state push [options] PATH`

This command will push the state specified by PATH to the currently
configured [backend](/docs/backends).

If PATH is "-" then the state data to push is read from stdin. This data
is loaded completely into memory and verified prior to being written to
the destination state.

Terraform will perform a number of safety checks to prevent you from
making changes that appear to be unsafe:

  * **Differing lineage**: If the "lineage" value in the state differs,
    Terraform will not allow you to push the state. A differing lineage
    suggests that the states are completely different and you may lose
    data.

  * **Higher remote serial**: If the "serial" value in the destination state
    is higher than the state being pushed, Terraform will prevent the push.
    A higher serial suggests that data is in the destination state that isn't
    accounted for in the local state being pushed.

Both of these safety checks can be disabled with the `-force` flag.
**This is not recommended.** If you disable the safety checks and are
pushing state, the destination state will be overwritten.
