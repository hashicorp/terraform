---
layout: "intro"
page_title: "Nomad vs. Mesos with Aurora, Marathon, etc"
sidebar_current: "vs-other-mesos"
description: |-
  Comparison between Nomad and Mesos with Aurora, Marathon, etc
---

# Nomad vs. Mesos with Aurora, Marathon

Mesos is a resource manager, which is used to pool together the
resources of a datacenter and exposes an API to integrate with
Frameworks that have scheduling and job management logic. Mesos
depends on ZooKeeper to provide both coordination and storage.

There are many different frameworks that integrate with Mesos;
popular general purpose ones include Aurora and Marathon.
These frameworks allow users to submit jobs and implement scheduling
logic. They depend on Mesos for resource management, and external
systems like ZooKeeper to provide coordination and storage.

Nomad is architecturally much simpler. Nomad is a single binary, both for clients
and servers, and requires no external services for coordination or storage.
Nomad combines features of both resource managers and schedulers into a single system.
This makes Nomad operationally simpler and enables more sophisticated
optimizations.

Nomad is designed to be a global state, optimistically concurrent scheduler.
Global state means schedulers get access to the entire state of the cluster when
making decisions enabling richer constraints, job priorities, resource preemption,
and faster placements. Optimistic concurrency allows Nomad to make scheduling
decisions in parallel increasing throughput, reducing latency, and increasing
the scale that can be supported.

Mesos does not support federation or multiple failure isolation regions.
Nomad supports multi-datacenter and multi-region configurations for failure
isolation and scalability.

