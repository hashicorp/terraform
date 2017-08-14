---
layout: "intro"
page_title: "Terraform vs. Custom Solutions"
sidebar_current: "vs-other-custom"
description: |-
  Most organizations start by manually managing infrastructure through simple scripts or web-based interfaces. As the infrastructure grows, any manual approach to management becomes both error-prone and tedious, and many organizations begin to home-roll tooling to help automate the mechanical processes involved.
---

# Terraform vs. Custom Solutions

Most organizations start by manually managing infrastructure through
simple scripts or web-based interfaces. As the infrastructure grows,
any manual approach to management becomes both error-prone and tedious,
and many organizations begin to home-roll tooling to help
automate the mechanical processes involved.

These tools require time and resources to build and maintain.
As tools of necessity, they represent the minimum viable
features needed by an organization, being built to handle only
the immediate needs. As a result, they are often hard
to extend and difficult to maintain. Because the tooling must be
updated in lockstep with any new features or infrastructure,
it becomes the limiting factor for how quickly the infrastructure
can evolve.

Terraform is designed to tackle these challenges. It provides a simple,
unified syntax, allowing almost any resource to be managed without
learning new tooling. By capturing all the resources required, the
dependencies between them can be resolved automatically so that operators
do not need to remember and reason about them. Removing the burden
of building the tool allows operators to focus on their infrastructure
and not the tooling.

Furthermore, Terraform is an open source tool. In addition to
HashiCorp, the community around Terraform helps to extend its features,
fix bugs and document new use cases. Terraform helps solve a problem
that exists in every organization and provides a standard that can
be adopted to avoid reinventing the wheel between and within organizations.
Its open source nature ensures it will be around in the long term.

