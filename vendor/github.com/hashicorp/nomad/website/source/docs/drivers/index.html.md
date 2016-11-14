---
layout: "docs"
page_title: "Task Drivers"
sidebar_current: "docs-drivers"
description: |-
  Task Drivers are used to integrate with the host OS to run tasks in Nomad.
---

# Task Drivers

Task drivers are used by Nomad clients to execute a task and provide resource
isolation. By having extensible task drivers, Nomad has the flexibility to
support a broad set of workloads across all major operating systems.

The list of supported task drivers is provided on the left of this page. 
Each task driver documents the configuration available in a 
[job specification](/docs/jobspec/index.html), the environments it can 
be used in, and the resource isolation mechanisms available.

Nomad strives to mask the details of running a task from users and instead
provides a clean abstraction. It is possible for the same task to be executed
with different isolation levels depending on the client running the task.
The goal is to use the strictest isolation available and gracefully degrade
protections where necessary.

