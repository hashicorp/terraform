---
layout: "docs"
page_title: "Import"
sidebar_current: "docs-import"
description: "Terraform can import and manage existing infrastructure. This can help you transition your infrastructure to Terraform."
---

# Import

> **Hands-on:** Try the [Import Terraform Configuration](https://learn.hashicorp.com/tutorials/terraform/state-import?in=terraform/state&utm_source=WEBSITE&utm_medium=WEB_IO&utm_offer=ARTICLE_PAGE&utm_content=DOCS) tutorial on HashiCorp Learn.

Terraform is able to import existing infrastructure. This allows you take
resources you've created by some other means and bring it under Terraform
management.

This is a great way to slowly transition infrastructure to Terraform, or
to be able to be confident that you can use Terraform in the future if it
potentially doesn't support every feature you need today.

~> Warning: Terraform expects that each remote object it is managing will be
bound to only one resource address, which is normally guaranteed by Terraform
itself having created all objects. If you import existing objects into Terraform,
be careful to import each remote object to only one Terraform resource address.
If you import the same object multiple times, Terraform may exhibit unwanted
behavior. For more information on this assumption, see
[the State section](/docs/language/state/index.html).

## Currently State Only

The current implementation of Terraform import can only import resources
into the [state](/docs/language/state/index.html). It does not generate configuration. A future
version of Terraform will also generate configuration.

Because of this, prior to running `terraform import` it is necessary to write
manually a `resource` configuration block for the resource, to which the
imported object will be mapped.

While this may seem tedious, it still gives Terraform users an avenue for
importing existing resources.

## Remote Backends

When using Terraform import on the command line with a [remote
backend](/docs/language/settings/backends/remote.html), such as Terraform Cloud, the import
command runs locally, unlike commands such as apply, which run inside your
Terraform Cloud environment. Because of this, the import command will not have
access to information from the remote backend, such as workspace variables.

In order to use Terraform import with a remote state backend, you may need to
set local variables equivalent to the remote workspace variables.
