---
layout: "intro"
page_title: "Introduction"
sidebar_current: "what"
description: |-
  This guide explains what Terraform is, what problems it can solve, and how it compares to existing software.
---

# Introduction to Terraform

## What is Terraform?

Terraform is a tool for building, changing, and versioning infrastructure
safely and efficiently. Terraform can manage both low-level components such as compute instances, storage, and networking, as well as high-level components such as DNS entries, SaaS features, etc.

Terraform configuration files describe the components needed to run a single application or your entire datacenter. Terraform reads these configuration files, generates an execution plan describing what it will do to reach the desired state, and then executes that plan to build the described infrastructure. As the configuration changes, Terraform can determine what changed and implement changes in the right order to respect dependencies.

Below, HashiCorp co-founder and CTO Armon Dadgar describes Terraform and its uses in more depth.

<iframe src="https://www.youtube.com/embed/h970ZBgKINg" frameborder="0" allowfullscreen="true"  width="560" height="315" ></iframe>



## Key Features

### Infrastructure as Code

Infrastructure is described using a high-level [configuration language](/docs/language/index.html). This allows a blueprint of your datacenter to be versioned and treated as you would any other code. Additionally, infrastructure can be shared and re-used.

### Execution Plans

Terraform generates an _execution plan_ and asks for your approval before creating or destroying infrastructure. This allows you to review changes  before they are applied.

### Resource Graph

Terraform builds a graph of all your resources and parallelizes the creation
and modification of any non-dependent resources. This allows Terraform to
build infrastructure as efficiently as possible and gives operators greater insight into their infrastructure.

### Change Automation

Apply complex changesets to your infrastructure with minimal human interaction.



## Next Steps

- Learn about common [Terraform use cases](/intro/use-cases.html).
- Learn [how Terraform compares to other infrastructure tools](/intro/vs/index.html).
- Try the [Terraform: Get Started](https://learn.hashicorp.com/collections/terraform/aws-get-started) tutorials on HashiCorp Learn to install Terraform and learn how to use Terraform to manage real infrastructure.
