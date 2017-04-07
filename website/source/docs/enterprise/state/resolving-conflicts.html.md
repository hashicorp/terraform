---
layout: "enterprise"
page_title: "Resolving Conflicts - State - Terraform Enterprise"
sidebar_current: "docs-enterprise-state-resolving"
description: |-
  Resolving conflicts with remote states.
---

# Resolving Conflicts in Remote States

Resolving state conflicts can be time consuming and error prone, so it's
important to approach it carefully.

There are several tools provided by Terraform Enterprise to help resolve
conflicts and fix remote state issues. First, you can navigate between state
versions in the changes view of your environment (after toggling on the remote
state checkbox) and view plain-text differences between versions.

This allows you to pinpoint where things may have gone wrong and make a educated
decision about resolving the conflict.

### Rolling Back to a Specific State Version

The rollback feature allows you to choose a new version to set as the "Head"
version of the state. Rolling back to a version means it will then return that
state upon request from a client. It will not increment the serial in the state,
but perform a hard rollback to the exact version of the state provided.

This allows you to reset the state to an older version, essentially forgetting
changes made in versions after that point.

To roll back to a specific version, navigate to it in the changes view and use
the rollback link. You'll need to confirm the version number to perform the
operation.

### Using Terraform Locally

Another way to resolve remote state conflicts is to merge and conflicted copies
locally by inspecting the raw state available in the path
`.terraform/terraform.tfstate`.

When making state changes, it's important to make backup copies in order to
avoid losing any data.

Any state that is pushed with a serial that is lower than the known serial when
the MD5 of the state does not match will be rejected.

The serial is embedded in the state file:

```json
{
  "version": 1,
  "serial": 555,
  "remote": {
    "type": "atlas",
    "config": {
      "name": "my-username/production"
    }
  }
}
```

Once a conflict has been resolved locally by editing the state file, the serial
can be incremented past the current version and pushed:

```shell
$ terraform remote push
```

This will upload the manually resolved state and set it as the head version.
