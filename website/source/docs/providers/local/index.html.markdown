---
layout: "local"
page_title: "Provider: Local"
sidebar_current: "docs-local-index"
description: |-
  The Local provider is used to manage local resources, such as files.
---

# Local Provider

The Local provider is used to manage local resources, such as files.

Use the navigation to the left to read about the available resources.

~> **Note** Terraform primarily deals with remote resources which are able
to outlive a single Terraform run, and so local resources can sometimes violate
its assumptions. The resources here are best used with care, since depending
on local state can make it hard to apply the same Terraform configuration on
many different local systems where the local resources may not be universally
available. See specific notes in each resource for more information.
