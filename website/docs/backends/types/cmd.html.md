---
layout: "backend-types"
page_title: "Backend Type: cmd"
sidebar_current: "docs-backends-types-standard-cmd"
description: |-
  Terraform can delegate state storage and locking to an external command.
---

# cmd

**Kind: Standard (with optional locking)**

Delegates the storage and locking of state to an external command.
Uses files to pass the content of state and lock between terraform and the command.
Calls the external command with one of these subcommands: 'GET', 'PUT', 'DELETE', 'LOCK', 'UNLOCK'.
* GET: retrieve the state from storage and save its content to `state_transfer_file`
* PUT: read content of the state from `state_transfer_file` and save the state to storage
* DELETE: delete the state from storage
* LOCK (optional): read content of the lock from `lock_transfer_file` and create a lock with the content
* UNLOCK (optional): remove the lock

```hcl
terraform {
  backend "cmd" {
    base_command = "./backend.sh"
    state_transfer_file = "state_transfer"
    lock_transfer_file = "lock_transfer"
  }
}
```

## Example Referencing

```hcl
data "terraform_remote_state" "foo" {
  backend = "cmd"
  config = {
    base_command = "./backend.sh"
    state_transfer_file = "state_transfer"
    lock_transfer_file = "lock_transfer"
  }
}
```

## Configuration variables

The following configuration options / environment variables are supported:

 * `base_command` (Required) - Pass to the external command
 * `state_transfer_file` (Required) - Path to the intermediate file for state
 * `lock_transfer_file` (Optional) - Path to the intermediate file for lock

## Sample external command

[Sample implementation of external command](https://github.com/bzcnsh/tf_remote_state_cmd_samples)