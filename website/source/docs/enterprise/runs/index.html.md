---
layout: "enterprise"
page_title: "Runs - Terraform Enterprise"
sidebar_current: "docs-enterprise-runs"
description: |-
  A "run" in Atlas represents the logical grouping of two Terraform steps - a "plan" and an "apply".
---

# About Terraform Enterprise Runs

A "run" represents the logical grouping of two Terraform steps - a "plan" and an
"apply". The distinction between these two phases of a Terraform run are
documented below.

When a [new run is created](/docs/enterprise/runs/starting.html), Terraform
Enterprise automatically queues a Terraform plan. Because a plan does not change
the state of infrastructure, it is safe to execute a plan multiple times without
consequence. An apply executes the output of a plan and actively changes
infrastructure. To prevent race conditions, the platform will only execute one
plan/apply at a time (plans for validating GitHub Pull Requests are allowed to
happen concurrently, as they do not modify state). You can read more about
Terraform plans and applies below.

## Plan

During the plan phase of a run, the command `terraform plan` is executed.
Terraform performs a refresh and then determines what actions are necessary to
reach the desired state specified in the Terraform configuration files. A
successful plan outputs an executable file that is securely stored in Terraform
Enterprise and may be used in the subsequent apply.

Terraform plans do not change the state of infrastructure, so it is
safe to execute a plan multiple times. In fact, there are a number of components
that can trigger a Terraform plan. You can read more about this in the
[starting runs](/docs/enterprise/runs/starting.html) section.

## Apply

During the apply phase of a run, the command `terraform apply` is executed
with the executable result of the prior Terraform plan. This phase **can change
infrastructure** by applying the changes required to reach the desired state
specified in the Terraform configuration file.

While Terraform plans are safe to run multiple times, Terraform applies often
change active infrastructure. Because of this, the default behavior
is to require user confirmation as part of the
[Terraform run execution](/docs/enterprise/runs/how-runs-execute.html). Upon
user confirmation, the Terraform apply will be queued and executed. It is also
possible to configure
[automatic applies](/docs/enterprise/runs/automatic-applies.html), but this option is
disabled by default.

## Environment Locking

During run execution, the environment will lock to prevent other plans
and applies from executing simultaneously. When the run completes, the next
pending run, if any, will be started.

An administrator of the environment can also manually lock the environment, for
example during a maintenance period.

You can see the lock status of an environment, and lock/unlock the environment
by visiting that environment's settings page.

## Notifications

To receive alerts when user confirmation is needed or for any other phase of the
run process, you can
[enable run notifications](/docs/enterprise/runs/notifications.html) for your
organization or environment.
