---
layout: "docs"
page_title: "State: Locking"
sidebar_current: "docs-state-locking"
description: |-
  Terraform stores state which caches the known state of the world the last time Terraform ran.
---

# State Locking

If supported by your [backend](/docs/backends), Terraform will lock your
state for all operations that could write state. This prevents
others from acquiring the lock and potentially corrupting your state.

State locking happens automatically on all operations that could write
state. You won't see any message that it is happening. If state locking fails,
Terraform will not continue. You can disable state locking for most commands
with the `-lock` flag but it is not recommended.

If acquiring the lock is taking longer than expected, Terraform will output
a status message. If Terraform doesn't output a message, state locking is
still occurring if your backend supports it.

Not all [backends](/docs/backends) support locking. Please view the list
of [backend types](/docs/backends/types) for details on whether a backend
supports locking or not.

## Force Unlock

Terraform has a [force-unlock command](/docs/commands/force-unlock.html)
to manually unlock the state if unlocking failed.

**Be very careful with this command.** If you unlock the state when someone
else is holding the lock it could cause multiple writers. Force unlock should
only be used to unlock your own lock in the situation where automatic
unlocking failed.

To protect you, the `force-unlock` command requires a unique lock ID. Terraform
will output this lock ID if unlocking fails. This lock ID acts as a
[nonce](https://en.wikipedia.org/wiki/Cryptographic_nonce), ensuring
that locks and unlocks target the correct lock.
