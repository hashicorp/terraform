---
layout: "intro"
page_title: "Nomad vs. HTCondor"
sidebar_current: "vs-other-htcondor"
description: |-
  Comparison between Nomad and HTCondor
---

# Nomad vs. HTCondor

HTCondor is a batch queuing system that is traditionally deployed in
grid computing environments. These environments have a fixed set of
resources, and large batch jobs that consume the entire cluster or
large portions. HTCondor is used to manage queuing, dispatching and
execution of these workloads.

HTCondor is not designed for services or long lived applications.
Due to the batch nature of workloads on HTCondor, it does not prioritize
high availability and is operationally complex to setup. It does support
federation in the form of "flocking" allowing batch workloads to
be run on alternate clusters if they would otherwise be forced to wait.

Nomad is focused on both long-lived services and batch workloads, and
is designed to be a platform for running large scale applications instead
of just managing a queue of batch work. Nomad supports a broader range
of workloads, is designed for high availability, supports much
richer constraint enforcement and bin packing logic.

