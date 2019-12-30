---
layout: "backend-types"
page_title: "Backend Type: nil"
sidebar_current: "docs-backends-types-standard-nil"
description: |-
  Terraform can be run with a backend that does not record state at all.
---

# nil

**Kind: Standard**

Discards state after every run. Advanced usage for local operations only (e.g.
using Terraform to scaffold template terraform files).

## Example Usage

```hcl
terraform {
  backend "nil" {}
}
```
