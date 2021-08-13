---
layout: "intro"
page_title: "Use Cases"
sidebar_current: "use-cases"
description: |-
  Learn common use cases for Terraform including managing Heroku apps, self-service clusters, and multi-cloud deployments.
---

# Use Cases

This page lists a subset of use cases for [Terraform](/intro/index.html).

## Multi-Cloud Deployment

It's often attractive to spread infrastructure across multiple clouds to
increase fault-tolerance. By using only a single region or cloud provider,
fault tolerance is limited by the availability of that provider. Multi-cloud
deployment allows for more graceful recovery of the loss of a region or entire
provider.

Realizing multi-cloud deployments can be very challenging as many existing
tools for infrastructure management are cloud-specific. Terraform is
cloud-agnostic and allows a single configuration to be used to manage multiple
providers, and to even handle cross-cloud dependencies. This simplifies
management and orchestration, helping operators build large-scale multi-cloud
infrastructures.

> **Hands-on:** Try the [Deploy Federated Multi-Cloud Kubernetes Clusters](https://learn.hashicorp.com/tutorials/terraform/multicloud-kubernetes) tutorial on HashiCorp Learn.

## Heroku App Setup

Heroku is a popular PaaS for hosting web apps. Developers create an app, and then attach add-ons, such as a database, or email provider. One of the best features is the ability to elastically scale the number of dynos or workers. However, most non-trivial applications quickly need many add-ons and external services.

You can use Terraform to codify the setup required for a Heroku application, ensuring that all the required add-ons are available, but it can go even further: configuring DNSimple to set a CNAME, or setting up Cloudflare as a CDN for the app. Best of all, Terraform can do all of this in under 30 seconds without using a web interface.

## Multi-Tier Applications

A very common pattern is the N-tier architecture. The most common 2-tier architecture is
a pool of web servers that use a database tier. Additional tiers get added for API servers,
caching servers, routing meshes, etc. This pattern is used because the tiers can be scaled
independently and provide a separation of concerns.

Terraform is an ideal tool for building and managing these infrastructures. You can group resources in each tier together, and Terraform will automatically handle the dependencies between each tier. For example, Terraform will ensure the database tier is available before provisioning the web servers and that the load balancers are connected to the web nodes. You can then use Terraform to easily scale each tier by modifying the `count` configuration value. Because resource creation and provisioning is codified and automated, elastically scaling
with load becomes trivial.

## Self-Service Clusters

At a certain organizational size, it becomes very challenging for a centralized
operations team to manage a large and growing infrastructure. Instead it becomes
more attractive to make "self-serve" infrastructure, allowing product teams to
manage their own infrastructure using tooling provided by the central operations team.

You can use Terraform configuration to codify the knowledge of how to build and scale a service. You can then share these configurations throughout your organization, enabling customer teams to use Terraform to manage their services.

## Software Demos

Modern software is increasingly networked and distributed. Although tools like
[Vagrant](https://www.vagrantup.com/) exist to build virtualized environments
for demos, it is still very challenging to demo software on real infrastructure
which more closely matches production environments.

Software writers can provide a Terraform configuration to create, provision and
bootstrap a demo on cloud providers like AWS. This allows end users to easily demo the software on their own infrastructure, and even enables tweaking parameters like cluster size to more rigorously test tools at any scale.

## Disposable Environments

It is common practice to have both a production and staging or QA environment.
These environments are smaller clones of their production counterpart, but are
used to test new applications before releasing in production. As the production
environment grows larger and more complex, it becomes increasingly onerous to
maintain an up-to-date staging environment.

Using Terraform, the production environment can be codified and then shared with
staging, QA or dev. These configurations can be used to rapidly spin up new
environments to test in, and then be easily disposed of. Terraform can help tame
the difficulty of maintaining parallel environments, and makes it practical
to elastically create and destroy them.

## Software Defined Networking

Software Defined Networking (SDN) is becoming increasingly prevalent in the
datacenter, as it provides more control to operators and developers and
allows the network to better support the applications running on top. Most SDN
implementations have a control layer and infrastructure layer.

You can use Terraform to codify the configuration for software defined networks.
Terraform can then use this configuration to automatically set up and modify settings by interfacing with the control layer. This allows the configuration to be
versioned and changes to be automated. For example, you can [use Terraform to configure AWS VPC](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/vpc).

## Resource Schedulers

In large-scale infrastructures, static assignment of applications to machines
becomes increasingly challenging. To solve that problem, there are a number
of schedulers like Borg, Mesos, YARN, and Kubernetes. These can be used to
dynamically schedule Docker containers, Hadoop, Spark, and many other software
tools.

Terraform is not limited to physical providers like AWS. Resource schedulers
can be treated as a provider, enabling Terraform to request resources from them.
This allows Terraform to be used in layers: to setup the physical infrastructure
running the schedulers as well as provisioning onto the scheduled grid.
