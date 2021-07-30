---
layout: "docs"
page_title: "Forcing Re-creation of Resources (Tainting) - Terraform CLI"
description: "Commands that allow you to destroy and re-create resources manually."
---

# Forcing Re-creation of Resources (Tainting)

When a resource declaration is modified, Terraform usually attempts to update
the existing resource in place (although some changes can require destruction
and re-creation, usually due to upstream API limitations).

In some cases, you might want a resource to be destroyed and re-created even
when Terraform doesn't think it's necessary. This is usually for objects that
aren't fully described by their resource arguments due to side-effects that
happen during creation; for example, a virtual machine that configures itself
with `cloud-init` on startup might no longer meet your needs if the cloud-init
configuration changes.

- [The `terraform taint` command](/docs/cli/commands/taint.html) tells Terraform to
  destroy and re-create a particular resource during the next apply, regardless
  of whether its resource arguments would normally require that.

- [The `terraform untaint` command](/docs/cli/commands/untaint.html) undoes a
  previous taint, or can preserve a resource that was automatically tainted due
  to failed [provisioners](/docs/language/resources/provisioners/syntax.html).
