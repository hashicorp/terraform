---
layout: "enterprise"
page_title: "Glossary - Terraform Enterprise"
sidebar_current: "docs-enterprise-glossary"
description: |-
  Terminology for Terraform Enterprise.
---

# Glossary

Terraform Enterprise, and this documentation, covers a large set of terminology
adopted from tools, industry standards and the community. This glossary seeks to
define as many of those terms as possible to help increase understanding in
interfacing with the platform and reading documentation.

## Authentication Tokens

Authentication tokens are tokens used to authenticate with Terraform Enterprise
via APIs or through tools. Authentication tokens can be revoked, expired or
created under any user.

## ACL

ACL is an acronym for access control list. This defines access to a set of
resources. Access to an object in Terraform Enterprise limited to "read" for
certain users is an example of an ACL.

## Alert

An alert represents a health check status change on a Consul node that is sent
to Terraform Enterprise, and then recorded and distributed to various
notification methods.

## Application

An application is a set of code that represents an application that should be
deployed. Applications can be linked to builds to be made available in the
Packer environment.

## Apply

An apply is the second step of the two steps required for Terraform to make
changes to infrastructure. The apply is the process of communicating with
external APIs to make the changes.

## Artifact

An artifact is an abstract representation of something you wish to store and use
again that has undergone configuration, compilation or some other build process.
An artifact is typically an image created by Packer that is then deployed by
Terraform, or used locally with Vagrant.

## Box

Boxes are a Vagrant specific package format. Vagrant can install and uses images
in box format.

## Build

Builds are resources that represent Packer configurations. A build is a generic
name, sometimes called a "Build Configuration" when defined in the Terraform
Enterprise UI.

## Build Configuration

A build configuration are settings associated with a resource that creates
artifacts via builds. A build configuration is the name in `packer push -name
acemeinc/web`.

## Catalog

The box catalog is a publicly available index of Vagrant Boxes that can be
downloaded from Terraform Enterprise and used for development.

## Consul

[Consul](https://consul.io) is a HashiCorp tool for service discovery,
configuration, and orchestration. Consul enables rapid deployment,
configuration, monitoring and maintenance of service-oriented architectures.

## Datacenter

A datacenter represents a group of nodes in the same network or datacenter
within Consul.

## Environment

Environments show the real-time status of your infrastructure, any pending
changes, and its change history. Environments can be configured to use any or
all of these three components.

Environments are the namespace of your Terraform Enterprise managed
infrastructure. As an example, if you to have a production environment for a
company named Acme Inc., your environment may be named
`my-username/production`.

To read more about features provided under environments, read the
[Terraform](/docs/enterprise) sections.

## Environment Variables

Environment variables injected into the environment of Packer builds or
Terraform Runs (plans and applies).

## Flapping

Flapping is something entering and leaving a healthy state rapidly. It is
typically associated with a health checks that briefly report unhealthy status
before recovering.

## Health Check

Health checks trigger alerts by changing status on a Consul node. That status
change is seen by Terraform Enterprise, when connected, and an associated alert
is recorded and sent to any configured notification methods, like email.

## Infrastructure

An infrastructure is a stateful representation of a set of Consul datacenters.

## Operator

An operator is a person who is making changes to infrastructure or settings.

## Packer

[Packer](https://packer.io) is a tool for creating images for platforms such as
Amazon AWS, OpenStack, VMware, VirtualBox, Docker, and more — all from a single
source configuration.

## Packer Template

A Packer template is a JSON file that configure the various components of Packer
in order to create one or more machine images.

## Plan

A plan is the second step of the two steps required for Terraform to make
changes to infrastructure. The plan is the process of determining what changes
will be made to.

## Providers

Providers are often referenced when discussing Packer or Terraform. Terraform
providers manage resources in Terraform.
[Read more](https://terraform.io/docs/providers/index.html).

## Post-Processors

The post-processor section within a Packer template configures any
post-processing that will be done to images built by the builders. Examples of
post-processing would be compressing files, uploading artifacts, etc..

## Registry

Often referred to as the "Artifact Registry", the registry stores artifacts, be
it images or IDs for cloud provider images.

## Run

A run represents a two step Terraform plan and a subsequent apply.

## Service

A service in Consul represents an application or service, which could be active
on any number of nodes.

## Share

Shares are let you instantly share public access to your running Vagrant
environment (virtual machine).

## State

Terraform state is the state of your managed infrastructure from the last time
Terraform was run. By default this state is stored in a local file named
`terraform.tfstate`, but it can also be stored in Terraform Enterprise and is
then called "Remote state".

## Terraform

[Terraform](https://terraform.io) is a tool for safely and efficiently changing
infrastructure across providers.

## Terraform Configuration

Terraform configuration is the configuration files and any files that may be
used in provisioners like `remote-exec`.

## Terraform Variables

Variables in Terraform, uploaded with `terraform push` or set in the UI. These
differ from environment variables as they are a first class Terraform variable
used in interpolation.
