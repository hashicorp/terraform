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

Information is gathered from two places: the config and the state. The config
is populated from the user's config file; the state is populated from the
statefile. But before the state gets populated from the statefile, the
statefile is refreshed, using information about the provider(s) to get a more
accurate picture of the world. It then gets passed through an optional
per-resource `StateFunc`, which allows resources to modify their representation
in the state. Finally, the config and state get passed through an optional
`DiffSuppressFunc`, which allows resources to decide whether a config value and
a state value should be considered equivalent. This results in our diff.

![Terraform Provider Lifecycle](docs/lifecycle-diagram.png)

Once Terraform generates that diff, it knows which resources need to be
created, updated, or destroyed. The `Create`, `Update`, and `Destroy` functions
for the resources are called as appropriate, and the `ResourceData` instance
they modify is persisted as the state when the call returns.
