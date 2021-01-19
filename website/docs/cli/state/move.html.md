---
layout: "docs"
page_title: "Moving Resources - Terraform CLI"
---

# Moving Resources

Terraform's state associates each real-world object with a configured resource
at a specific [resource address](/docs/cli/state/resource-addressing.html). This
is seamless when changing a resource's attributes, but Terraform will lose track
of a resource if you change its name, move it to a different module, or change
its provider.

Usually that's fine: Terraform will destroy the old resource, replace it with a
new one (using the new resource address), and update any resources that rely on
its attributes.

In cases where it's important to preserve an existing infrastructure object, you
can explicitly tell Terraform to associate it with a different configured
resource.

- [The `terraform state mv` command](/docs/cli/commands/state/mv.html) changes
  which resource address in your configuration is associated with a particular
  real-world object. Use this to preserve an object when renaming a resource, or
  when moving a resource into or out of a child module.

- [The `terraform state rm` command](/docs/cli/commands/state/rm.html) tells
  Terraform to stop managing a resource as part of the current working directory
  and workspace, _without_ destroying the corresponding real-world object. (You
  can later use `terraform import` to start managing that resource in a
  different workspace or a different Terraform configuration.)

- [The `terraform state replace-provider` command](/docs/cli/commands/state/replace-provider.html)
  transfers existing resources to a new provider without requiring them to be
  re-created.
