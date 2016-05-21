---
layout: "docs"
page_title: "Import"
sidebar_current: "docs-import"
description: |-
  Terraform is able to import existing infrastructure. This allows you take
  resources you've created by some other means and bring it under Terraform
  management.
---

# Import

Terraform is able to import existing infrastructure. This allows you take
resources you've created by some other means and bring it under Terraform
management.

This is a great way to slowly transition infrastructure to Terraform, or
to be able to be confident that you can use Terraform in the future if it
potentially doesn't support every feature you need today.

## Currently State Only

The current implementation of Terraform import can only import resources
into the [state](/docs/state). It does not generate configuration. A future
version of Terraform will also generate configuration.

Because of this, the behavior of importing resources into Terraform right now
is that after an import, if you run a `terraform plan`, Terraform views it
as an orphan (a resource with no configuration) and marks it for destruction.
After importing a resource you have to manually write configuration to match
the resource.

While this may seem tedious, it still gives Terraform users an avenue for
importing existing resources. A future version of Terraform will fully generate
configuration significantly simplifying this process.
