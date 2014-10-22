---
layout: "docs"
page_title: "Resource Lifecycle"
sidebar_current: "docs-internals-lifecycle"
description: |-
  Resources have a strict lifecycle, and can be thought of as basic state machines. Understanding this lifecycle can help better understand how Terraform generates an execution plan, how it safely executes that plan, and what the resource provider is doing throughout all of this.
---

# Resource Lifecycle

Resources have a strict lifecycle, and can be thought of as basic
state machines. Understanding this lifecycle can help better understand
how Terraform generates an execution plan, how it safely executes that
plan, and what the resource provider is doing throughout all of this.

~> **Advanced Topic!** This page covers technical details
of Terraform. You don't need to understand these details to
effectively use Terraform. The details are documented here for
those who wish to learn about them without having to go
spelunking through the source code.

## Lifecycle

A resource roughly follows the steps below:

  1. `ValidateResource` is called to do a high-level structural
     validation of a resource's configuration. The configuration
     at this point is raw and the interpolations have not been processed.
     The value of any key is not guaranteed and is just meant to be
     a quick structural check.

  1. `Diff` is called with the current state and the configuration.
     The resource provider inspects this and returns a diff, outlining
     all the changes that need to occur to the resource. The diff includes
     details such as whether or not the resource is being destroyed, what
     attribute necessitates the destroy, old values and new values, whether
     a value is computed, etc. It is up to the resource provider to
     have this knowledge.

  1. `Apply` is called with the current state and the diff. Apply does
     not have access to the configuration. This is a safety mechanism
     that limits the possibility that a provider changes a diff on the
     fly. `Apply` must apply a diff as prescribed and do nothing else
     to remain true to the Terraform execution plan. Apply returns the
     new state of the resource (or nil if the resource was destroyed).

  1. If a resource was just created and did not exist before, and the
     apply succeeded without error, then the provisioners are executed
     in sequence. If any provisioner errors, the resource is marked as
     _tainted_, so that it will be destroyed on the next apply.

## Partial State and Error Handling

If an error happens at any stage in the lifecycle of a resource,
Terraform stores a partial state of the resource. This behavior is
critical for Terraform to ensure that you don't end up with any
_zombie_ resources: resources that were created by Terraform but
no longer managed by Terraform due to a loss of state.
