---
layout: "intro"
page_title: "Introduction"
sidebar_current: "what"
description: |-
  Welcome to the intro guide to Terraform! This guide is the best place to start with Terraform. We cover what Terraform is, what problems it can solve, how it compares to existing software, and contains a quick start for using Terraform.
---

# Introduction to Terraform

Welcome to the intro guide to Terraform! This guide is the best
place to start with Terraform. We cover what Terraform is, what
problems it can solve, how it compares to existing software,
and contains a quick start for using Terraform.

If you are already familiar with the basics of Terraform, the
[documentation](/docs/index.html) provides a better reference
guide for all available features as well as internals.

## What is Terraform?

Terraform is a tool for building, changing, and versioning infrastructure
safely and efficiently. Terraform can manage existing and popular service
providers as well as custom in-house solutions.

Configuration files describe to Terraform the components needed to
run a single application or your entire datacenter.
Terraform generates an execution plan describing
what it will do to reach the desired state, and then executes it to build the
described infrastructure. As the configuration changes, Terraform is able
to determine what changed and create incremental execution plans which
can be applied.

The infrastructure Terraform can manage includes
low-level components such as
compute instances, storage, and networking, as well as high-level
components such as DNS entries, SaaS features, etc.

Examples work best to showcase Terraform. Please see the
[use cases](/intro/use-cases.html).

The key features of Terraform are:

### Infrastructure as Code

Infrastructure is described using a high-level configuration syntax. This allows
a blueprint of your datacenter to be versioned and treated as you would any
other code. Additionally, infrastructure can be shared and re-used.

### Execution Plans

Terraform has a "planning" step where it generates an _execution plan_. The
execution plan shows what Terraform will do when you call apply. This lets you
avoid any surprises when Terraform manipulates infrastructure.

### Resource Graph

Terraform builds a graph of all your resources, and parallelizes the creation
and modification of any non-dependent resources. Because of this, Terraform
builds infrastructure as efficiently as possible, and operators get insight into
dependencies in their infrastructure.

### Change Automation

Complex changesets can be applied to your infrastructure with minimal human
interaction. With the previously mentioned execution plan and resource graph,
you know exactly what Terraform will change and in what order, avoiding many
possible human errors.

## Next Steps

See the page on [Terraform use cases](/intro/use-cases.html) to see the
multiple ways Terraform can be used. Then see
[how Terraform compares to other software](/intro/vs/index.html)
to see how it fits into your existing infrastructure. Finally, continue onwards with
the [getting started guide](/intro/getting-started/install.html) to use
Terraform to manage real infrastructure and to see how it works.
