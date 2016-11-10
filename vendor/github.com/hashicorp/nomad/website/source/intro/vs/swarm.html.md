---
layout: "intro"
page_title: "Nomad vs. Docker Swarm"
sidebar_current: "vs-other-swarm"
description: |-
  Comparison between Nomad and Docker Swarm
---

# Nomad vs. Docker Swarm

Docker Swarm is the native clustering solution for Docker. It provides
an API compatible with the Docker Remote API, and allows containers to
be scheduled across many machines.

Nomad differs in many ways with Docker Swarm, most obviously Docker Swarm
can only be used to run Docker containers, while Nomad is more general purpose.
Nomad supports virtualized, containerized and standalone applications, including Docker.
Nomad is designed with extensible drivers and support will be extended to all
common drivers.

Docker Swarm provides API compatibility with their remote API, which focuses
on the container abstraction. Nomad uses a higher-level abstraction of jobs.
Jobs contain task groups, which are sets of tasks. This allows more complex
applications to be expressed and easily managed without reasoning about the
individual containers that compose the application.

The architectures also differ between Nomad and Docker Swarm.
Nomad does not depend on external systems for coordination or storage,
is distributed, highly available, and supports multi-datacenter
and multi-region configurations.

By contrast, Swarm is not distributed or highly available by default.
External systems must be used for coordination to support replication.
When replication is enabled, Swarm uses an active/standby model,
meaning the other servers cannot be used to make scheduling decisions.
Swarm also does not support multiple failure isolation regions or federation.

