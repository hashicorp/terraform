---
layout: "docs"
page_title: "Backends: State Storage and Locking"
sidebar_current: "docs-backends-state"
description: |-
  Backends are configured directly in Terraform files in the `terraform` section.
---

# State Storage and Locking

Backends are responsible for storing state and providing an API for
[state locking](/docs/state/locking.html). State locking is optional.

Despite the state being stored remotely, all Terraform commands such
as `terraform console`, the `terraform state` operations, `terraform taint`,
and more will continue to work as if the state was local.

## State Storage

Backends determine where state is stored. For example, the local (default)
backend stores state in a local JSON file on disk. The Consul backend stores
the state within Consul. Both of these backends happen to provide locking:
local via system APIs and Consul via locking APIs.

When using a non-local backend, Terraform will not persist the state anywhere
on disk except in the case of a non-recoverable error where writing the state
to the backend failed. This behavior is a major benefit for backends: if
sensitive values are in your state, using a remote backend allows you to use
Terraform without that state ever being persisted to disk.

In the case of an error persisting the state to the backend, Terraform will
write the state locally. This is to prevent data loss. If this happens the
end user must manually push the state to the remote backend once the error
is resolved.

## Manual State Pull/Push

You can still manually retrieve the state from the remote state using
the `terraform state pull` command. This will load your remote state and
output it to stdout. You can choose to save that to a file or perform any
other operations.

You can also manually write state with `terraform state push`. **This
is extremely dangerous and should be avoided if possible.** This will
overwrite the remote state. This can be used to do manual fixups if necessary.

When manually pushing state, Terraform will attempt to protect you from
some potentially dangerous situations:

  * **Differing lineage**: The "lineage" is a unique ID assigned to a state
    when it is created. If a lineage is different, then it means the states
    were created at different times and its very likely you're modifying a
    different state. Terraform will not allow this.

  * **Higher serial**: Every state has a monotonically increasing "serial"
    number. If the destination state has a higher serial, Terraform will
    not allow you to write it since it means that changes have occurred since
    the state you're attempting to write.

Both of these protections can be bypassed with the `-force` flag if you're
confident you're making the right decision. Even if using the `-force` flag,
we recommend making a backup of the state with `terraform state pull`
prior to forcing the overwrite.

## State Locking

Backends are responsible for supporting [state locking](/docs/state/locking.html)
if possible. Not all backend types support state locking. In the
[list of supported backend types](/docs/backends/types) we explicitly note
whether locking is supported.

For more information on state locking, view the
[page dedicated to state locking](/docs/state/locking.html).
