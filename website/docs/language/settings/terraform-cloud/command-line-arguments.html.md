---
layout: "language"
page_title: "Command Line Arguments"
sidebar_current: "terraform-cloud-configuration"
description: "Command Line Arguments"
---

# Command Line Arguments

For configurations that include a `cloud` block to integrate with Terraform Cloud, commands that
make local modifications to Terraform state and then push them back up to the remote workspace
accept the following option to modify that behavior:

* `-ignore-remote-version` - Override checking that the local and remote
  Terraform versions agree, making an operation proceed even when there is
  a mismatch.

    Normally state-modification operations require using a local version of
    Terraform CLI which is compatible with the Terraform version selected
    for the remote workspace as part of its settings. This is to avoid the
    local operation creating a new state snapshot which the workspace's
    remote execution environment would then be unable to decode.

    Overriding this check can result in a Terraform Cloud workspace that is
    no longer able to complete remote operations, so we recommend against
    using this option unless absolutely necessary.

