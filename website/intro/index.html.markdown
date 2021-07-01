---
layout: "intro"
page_title: "Introduction"
sidebar_current: "what"
description: |-
  Learn what Terraform is, what problems it can solve, and how it compares to existing software.
---

# Introduction to Terraform

Terraform is an infrastructure as code (IaC) tool that allows you to build, change, and version infrastructure safely and efficiently. This includes low-level components such as compute instances, storage, and networking, as well as high-level components such as DNS entries, SaaS features, etc. Terraform can manage both existing service providers and custom in-house solutions.

Below, HashiCorp co-founder and CTO Armon Dadgar describes how Terraform can help solve common infrastructure challenges.

<iframe src="https://www.youtube.com/embed/h970ZBgKINg" frameborder="0" allowfullscreen="true"  width="560" height="315" ></iframe>



## Key Features

### Infrastructure as Code

You describe your infrastructure using Terraform's high-level [configuration language](/docs/language/index.html) in human-readable, declarative configuration files. This allows you to create a blueprint that you can version, share, and reuse.

### Execution Plans

Terraform generates an _execution plan_ describing what it will do and asks for your approval before making any infrastructure changes. This allows you to review changes before Terraform creates, updates, or destroys infrastructure.

### Resource Graph

Terraform builds a resource graph and creates or modifies non-dependent resources in parallel. This allows Terraform to build resources as efficiently as possible and gives you greater insight into your infrastructure.

### Change Automation

Terraform can apply complex changesets to your infrastructure with minimal human interaction. When you update configuration files, Terraform determines what changed and creates incremental execution plans that respect dependencies.



## Next Steps

- Learn about common [Terraform use cases](/intro/use-cases.html).
- Learn [how Terraform compares to and complements other tools](/intro/vs/index.html).
- Try the [Terraform: Get Started](https://learn.hashicorp.com/collections/terraform/aws-get-started) tutorials on HashiCorp Learn.
