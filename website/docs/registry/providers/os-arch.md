---
layout: "registry"
page_title: "Recommended Provider Binary OS and Architecture - Terraform Registry"
sidebar_current: "docs-registry-provider-os-arch"
description: |-
  Recommended Provider Binary OS and Architecture
---

-> __Publishing Beta__<br>Welcome! Thanks for your interest participating in our Providers in the Registry beta! Paired with Terraform 0.13, our vision is to make it easier than ever to discover, distribute, and maintain your provider(s). We welcome any feedback you have throughout the process and encourage you to reach out if you have any questions or issues by emailing terraform-registry-beta@hashicorp.com.

## Recommended Provider Binary OS and Architecture

We recommend the following OS / architecture combinations for compiled binaries available in the registry (this list is already satisfied by our [recommended **.goreleaser.yml** configuration file](https://github.com/hashicorp/terraform-provider-scaffolding/blob/master/.goreleaser.yml)):

* Darwin / AMD64
* Linux / AMD64
* Linux / ARMv8 (sometimes referred to as AArch64 or ARM64)
* Linux / ARMv6
* Windows / AMD64

We also recommend shipping binaries for the following combinations, but we typically do not prioritize fixes for these:

* Linux / 386
* Windows / 386
* FreeBSD / 386
* FreeBSD / AMD64
