---
layout: "intro"
page_title: "Introduction"
sidebar_current: "what"
description: |-
  Learn what Terraform is, what problems it can solve, and how it compares to existing software.
---

# Introduction to Terraform

## What is Terraform?

Terraform is an infrastructure as code (IaC) tool that allows you to build, change, and version infrastructure safely and efficiently. Terraform can manage both low-level components such as compute instances, storage, and networking, as well as high-level components such as DNS entries, SaaS features, etc.

Below, HashiCorp co-founder and CTO Armon Dadgar describes how Terraform can help solve common infrastructure challenges.

<iframe src="https://www.youtube.com/embed/h970ZBgKINg" frameborder="0" allowfullscreen="true"  width="560" height="315" ></iframe>



## Key Features

### Infrastructure as Code

Infrastructure is described using a high-level [configuration language](/docs/language/index.html) in human-readable, declarative configuration files. This allows you to create a blueprint that can be versioned, shared, and reused.

### Execution Plans

Terraform generates an _execution plan_ describing what it will do and asks for your approval before creating, updating, or destroying infrastructure. This allows you to review changes before they are applied.

### Resource Graph

Terraform builds a resource graph and parallelizes the creation and modification of any non-dependent resources. This allows Terraform to
build resources as efficiently as possible and gives operators greater insight into their infrastructure.

### Change Automation

Terraform can apply complex changesets to your infrastructure with minimal human interaction.



## Next Steps

- Learn about common [Terraform use cases](/intro/use-cases.html).
- Learn [how Terraform compares to and complements other tools](/intro/vs/index.html).
- Try the [Terraform: Get Started](https://learn.hashicorp.com/collections/terraform/aws-get-started) tutorials on HashiCorp Learn.
