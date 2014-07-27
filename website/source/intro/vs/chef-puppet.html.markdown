---
layout: "intro"
page_title: "Terraform vs. Chef, Puppet, etc."
sidebar_current: "vs-other-chef"
---

# Terraform vs. Chef, Puppet, etc.

Configuration management tools install and manage tools on a machine
that already exists.  Terraform is not a configuration management tool,
and uses the existing tooling to do what they do best: bootstrap and initialize resources.

Using provisioners, Terraform allows any configuration management tool
to be used to setup a resource once it has been created. Terraform
focuses on the higher-level abstraction of the datacenter and associated
services, without sacrificing the ability to use configuration management
tools to do what they do best. It also embraces the same codification that
is responsible for the success of those tools, making entire infrastructure
deployments easy and reliable.

