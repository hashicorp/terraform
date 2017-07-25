---
layout: "docs"
page_title: "Command: force-unlock"
sidebar_current: "docs-commands-force-unlock"
description: |-
  The `terraform force-unlock`  manually unlocks the Terraform state
---

# Command: force-unlock

Manually unlock the state for the defined configuration.

This will not modify your infrastructure. This command removes the lock on the
state for the current configuration. The behavior of this lock is dependent
on the backend being used. Local state files cannot be unlocked by another
process.

## Usage

Usage: terraform force-unlock LOCK_ID [DIR]

Manually unlock the state for the defined configuration.

This will not modify your infrastructure. This command removes the lock on the
state for the current configuration. The behavior of this lock is dependent
on the backend being used. Local state files cannot be unlocked by another
process.

Options:

*  `-force` -  Don't ask for input for unlock confirmation.
