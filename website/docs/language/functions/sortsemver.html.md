---
layout: "language"
page_title: "sortsemver - Functions - Configuration Language"
sidebar_current: "docs-funcs-collection-sortsemver"
description: |-
  The sortsemver function takes a version constraint string and a list of
  semantic version strings and returns the versions matching that constraint in
  precedence order.
---

# `sortsemver` Function

`sortsemver` takes a [version constraint string](/docs/language/expressions/version-constraints.html)
  and a list of semantic version strings and returns the versions matching that
  constraint in precedence order. A valid semantic version string is described
  by the v2.0.0 specification found at https://semver.org/. An empty version
  constraint string will successfully match all versions.

## Examples

```
> sortsemver("~> 1.2.0", ["1.0.0", "1.2.4", "1.4.0-5", "1.2.3"])
[
  "1.2.3",
  "1.2.4",
]
```
