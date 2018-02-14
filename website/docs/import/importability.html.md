---
layout: "docs"
page_title: "Import: Resource Importability"
sidebar_current: "docs-import-importability"
description: |-
  Each resource in Terraform must implement some basic logic to become
  importable. As a result, not all Terraform resources are currently importable.
---

# Resource Importability

Each resource in Terraform must implement some basic logic to become
importable. As a result, not all Terraform resources are currently importable.
For those resources that support import, they are documented at the bottom of
each resource documentation page, under the Import heading. If you find a
resource that you want to import and Terraform reports that it is not
importable, please report an issue in the relevant provider repository.

Converting a resource to be importable is also relatively simple, so if
you're interested in contributing that functionality, the Terraform team
would be grateful.

To make a resource importable, please see the
[plugin documentation on writing a resource](/docs/plugins/provider.html).
