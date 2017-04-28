---
layout: "docs"
page_title: "Terraform Lifecycle"
sidebar_current: "docs-internals-provider-guide-lifecycle"
description: |-
  Information about the lifecycle your provider fits into.
---

# Terraform Lifecycle

Terraform runs follow a predictable lifecycle:

1. Gather information
2. Detect diffs
3. Apply updates
4. Set state

Terraform's information gathering consists of a few steps:

1. The user's config file populates the config.
2. Terraform refreshes the statefile by reading every resource in its
   statefile.
3. The statefile populates the state.
4. An optional, per-resource
   [`schema.StateFunc`](schema.html#customizing-state) transforms each
   resource's representation in the state.
5. An optional, per-resource
   [`schema.DiffSuppressfunc`](schema.html#customizing-diffs) modifies which
   values are considered "changed".

This process produces a diff.

![Terraform Provider Lifecycle](docs/lifecycle-diagram.png)

Once Terraform generates that diff, it knows which resources need to be
created, updated, or destroyed. The `Create`, `Update`, and `Destroy` functions
for the resources are called as appropriate, and the `ResourceData` instance
they modify is persisted as the state when the call returns.
