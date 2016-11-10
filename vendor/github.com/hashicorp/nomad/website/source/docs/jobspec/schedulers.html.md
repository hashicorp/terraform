---
layout: "docs"
page_title: "Nomad Schedulers"
sidebar_current: "docs-jobspec-schedulers"
description: |-
  Learn about Nomad's various schedulers.
---

# Scheduler Types

Nomad has three scheduler types that can be used when creating your
[job](/docs/jobspec/): `service`, `batch` and `system`. Here we will describe
the differences between each of these schedulers.

## Service

The `service` scheduler is designed for scheduling long lived services that
should never go down. As such, the `service` scheduler ranks a large portion
of the nodes that meet the jobs constraints and selects the optimal node to
place a task group on. The `service` scheduler uses a best fit scoring algorithm
influenced by Google work on Borg. Ranking this larger set of candidate nodes
increases scheduling time but provides greater guarantees about the optimality
of a job placement, which given the service workload is highly desirable.

## Batch

Batch jobs are much less sensitive to short term performance fluctuations and
are short lived, finishing in a few minutes to a few days. Although the `batch`
scheduler is very similar to the `service` scheduler, it makes certain
optimizations for the batch workload. The main distinction is that after finding
the set of nodes that meet the jobs constraints it uses the power of two choices
described in Berkeley's Sparrow scheduler to limit the number of nodes that are
ranked.

## System

The `system` scheduler is used to register jobs that should be run on all
clients that meet the job's constraints. The `system` scheduler is also invoked
when clients join the cluster or transition into the ready state. This means
that all registered `system` jobs will be re-evaluated and their tasks will be
placed on the newly available nodes if the constraints are met.

This scheduler type is extremely useful for deploying and managing tasks that
should be present on every node in the cluster. Since these tasks are being
managed by Nomad, they can take advantage of job updating, rolling deploys,
service discovery and more.
