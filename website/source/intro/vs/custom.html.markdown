---
layout: "intro"
page_title: "Terraform vs. Custom Solutions"
sidebar_current: "vs-other-custom"
---

# Terraform vs. Custom Solutions

As a code base grows, a monolithic app usually evolves into a Service Oriented Architecture (SOA).
A universal pain point for SOA is service discovery and configuration. In many
cases, this leads to organizations building home grown solutions.
It is an undisputed fact that distributed systems are hard; building one is error prone and time consuming.
Most systems cut corners by introducing single points of failure such
as a single Redis or RDBMS to maintain cluster state. These solutions may work in the short term,
but they are rarely fault tolerant or scalable. Besides these limitations,
they require time and resources to build and maintain.

Terraform provides the core set of features needed by a SOA out of the box. By using Terraform,
organizations can leverage open source work to reduce their time and resource commitment to
re-inventing the wheel and focus on their business applications.

Terraform is built on well-cited research, and is designed with the constraints of
distributed systems in mind. At every step, Terraform takes efforts to provide a robust
and scalable solution for organizations of any size.

