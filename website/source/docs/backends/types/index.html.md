---
layout: "backend-types"
page_title: "Backend: Supported Backend Types"
sidebar_current: "docs-backends-types-index"
description: |-
  Terraform can store the state remotely, making it easier to version and work with in a team.
---

# Backend Types

This section documents the various backend types supported by Terraform.
If you're not familiar with backends, please
[read the sections about backends](/docs/backends/index.html) first.

Backends may support differing levels of features in Terraform. We differentiate
these by calling a backend either **standard** or **enhanced**. All backends
must implement **standard** functionality. These are defined below:

  * **Standard**: State management, functionality covered in
    [State Storage & Locking](/docs/backends/state.html)

  * **Enhanced**: Everything in standard plus
    [remote operations](/docs/backends/operations.html).

The backends are separated in the left by standard and enhanced.
