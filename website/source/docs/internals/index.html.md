---
layout: "docs"
page_title: "Internals"
sidebar_current: "docs-internals"
description: |-
  This section covers the internals of Terraform and explains how plans are generated, the lifecycle of a provider, etc. The goal of this section is to remove any notion of "magic" from Terraform. We want you to be able to trust and understand what Terraform is doing to function.
---

# Terraform Internals

This section covers the internals of Terraform and explains how
plans are generated, the lifecycle of a provider, etc. The goal
of this section is to remove any notion of "magic" from Terraform.
We want you to be able to trust and understand what Terraform is
doing to function.

-> **Note:** Knowledge of Terraform internals is not
required to use Terraform. If you aren't interested in the internals
of Terraform, you may safely skip this section.
