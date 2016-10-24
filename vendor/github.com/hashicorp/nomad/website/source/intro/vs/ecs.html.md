---
layout: "intro"
page_title: "Nomad vs. AWS ECS"
sidebar_current: "vs-other-ecs"
description: |-
  Comparison between Nomad and AWS ECS
---

# Nomad vs. AWS ECS

Amazon Web Services provides the EC2 Container Service (ECS), which is
a cluster manager. The ECS service is only available within AWS and
can only be used for Docker workloads. Amazon provides customers with
the agent that is installed on EC2 instances, but does not provide
the servers which are a hosted service of AWS.

There are a number of fundamental differences between Nomad and ECS.
Nomad is completely open source, including both the client and server
components. By contrast, only the agent code for ECS is open and
the servers are closed sourced and managed by Amazon.

As a side effect of the ECS servers being managed by AWS, it is not possible
to use ECS outside of AWS. Nomad is agnostic to the environment in which it is run,
supporting public and private clouds, as well as bare metal datacenters.
Clusters in Nomad can span multiple datacenters and regions, meaning
a single cluster could be managing machines on AWS, Azure, and GCE simultaneously.

The ECS service is specifically focused on containers and the Docker
engine, while Nomad is more general purpose. Nomad supports virtualized,
containerized, and standalone applications, including Docker. Nomad is
designed with extensible drivers and support will be extended to all
common drivers.

