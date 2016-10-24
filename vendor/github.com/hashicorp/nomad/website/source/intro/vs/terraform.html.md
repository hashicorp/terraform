---
layout: "intro"
page_title: "Nomad vs. Terraform"
sidebar_current: "vs-other-terraform"
description: |-
  Comparison between Nomad and Terraform
---

# Nomad vs. Terraform

[Terraform](https://www.terraform.io) is a tool for building, changing, and versioning
infrastructure safely and efficiently. Configuration files describe to Terraform
the components needed to run a single application or your entire datacenter. Terraform
generates an execution plan describing what it will do to reach the desired state,
and then executes it to build the described infrastructure. As the configuration changes,
Terraform is able to determine what changed and create incremental execution plans which can be applied.

Nomad differs from Terraform in a number of key ways. Terraform is designed to support
any type of resource including low-level components such as compute instances, storage,
and networking, as well as high-level components such as DNS entries, SaaS features, etc.
Terraform knows how to create, provision, and manage the lifecycle of these resources.
Nomad runs on existing infrastructure and manages the lifecycle of applications running
on that infrastructure.

Another major distinction is that Terraform is an offline tool that runs to completion,
while Nomad is an online system with long lived servers. Nomad allows new jobs to
be submitted, existing jobs updated or deleted, and can handle node failures. This
requires operating continuously instead of in a single shot like Terraform.

For small infrastructures with only a handful of servers or applications, the complexity
of Nomad may not outweigh simply using Terraform to statically assign applications to
machines. At larger scales, Terraform should be used to provision capacity for Nomad,
and Nomad used to manage scheduling applications to machines dynamically.

