---
layout: "external"
page_title: "Provider: External"
sidebar_current: "docs-external-index"
description: |-
  The external provider allows external scripts to be integrated with Terraform.
---

# External Provider

`external` is a special provider that exists to provide an interface
between Terraform and external programs.

Using this provider it is possible to write separate programs that can
participate in the Terraform workflow by implementing a specific protocol.

This provider is intended to be used for simple situations where you wish
to integrate Terraform with a system for which a first-class provider
doesn't exist. It is not as powerful as a first-class Terraform provider,
so users of this interface should carefully consider the implications
described on each of the child documentation pages (available from the
navigation bar) for each type of object this provider supports.

~> **Warning** Terraform Enterprise does not guarantee availability of any
particular language runtimes or external programs beyond standard shell
utilities, so it is not recommended to use this provider within configurations
that are applied within Terraform Enterprise.
