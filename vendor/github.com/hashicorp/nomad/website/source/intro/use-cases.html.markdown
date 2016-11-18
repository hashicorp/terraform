---
layout: "intro"
page_title: "Use Cases"
sidebar_current: "use-cases"
description: |-
  This page lists some concrete use cases for Nomad, but the possible use cases are much broader than what we cover.
---

# Use Cases

Before understanding use cases, it's useful to know [what Nomad is](/intro/index.html).
This page lists some concrete use cases for Nomad, but the possible use cases are
much broader than what we cover.

#### Microservices Platform

Microservices, or Service Oriented Architectures (SOA), are a design paradigm in which many
services with narrow scope, tight state encapsulation, and API driven interfaces interact together
to form a larger application. However, they add an operational challenge of managing hundreds
or thousands of services instead of a few large applications. Nomad provides a platform for
managing microservices, making it easier to adopt the paradigm.

#### Hybrid Cloud Deployments

Nomad is designed to handle multi-datacenter and multi-region deployments and is cloud agnostic.
This allows Nomad to schedule in private datacenters running bare metal, OpenStack, or VMware
alongside an AWS, Azure, or GCE cloud deployment. This makes it easier to migrate workloads
incrementally, or to utilize the cloud for bursting.

#### E-Commerce

A typical E-Commerce website has a few types of workloads. There are long-lived services
used for web serving. These include the load balancer, web frontends, API servers, and OLTP databases.
Batch processing using Hadoop or Spark may run periodically for business reporting, user targeting,
or generating product recommendations. Nomad allows all these workloads to share an underlying cluster,
increasing utilization, reducing cost, simplifying scaling and providing a clean abstraction
for developers.

