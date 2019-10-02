---
layout: "docs"
page_title: "Folder includes - Configuration Language"
sidebar_current: "docs-config-folders"
description: |-
  Folder includes allow Terraform configuration content to be split across multiple
  file system folders with no side effects.
---

# Folder includes

-> **Note:** This page is about Terraform 0.12 and later. Folder includes are
not supported in earlier versions.

A folder include specifies an additional file system folder to be incorporated
into a Terraform configuration. Unlike a [module](./modules.html), a folder
include does not represent a configuration boundary and does not change the
way that resources are addressed in interpolations.

-> **Note:** Path-based interpolations are relative to the module root, not
the included folder.

## Declaring a Folder Include

A folder include is declared using a `folder` block:

```hcl
folder {
  source = "./security-group-rules"
}
```

Each referenced folder is merged entirely with the referencing module. Folder
includes may be nested.

## When To Use Folder Includes

Modules allow for creating reusable configurations and for representing more
abstract resources. Folder includes in contrast provide a way for managing
large modules where it does not make sense to use modules.