---
layout: "intro"
page_title: "Terraform vs. Serf"
sidebar_current: "vs-other-serf"
---

# Terraform vs. Serf

[Serf](http://www.serfdom.io) is a node discovery and orchestration tool and is the only
tool discussed so far that is built on an eventually consistent gossip model,
with no centralized servers. It provides a number of features, including group
membership, failure detection, event broadcasts and a query mechanism. However,
Serf does not provide any high-level features such as service discovery, health
checking or key/value storage. To clarify, the discovery feature of Serf is at a node
level, while Terraform provides a service and node level abstraction.

Terraform is a complete system providing all of those features. In fact, the internal
[gossip protocol](/docs/internals/gossip.html) used within Terraform, is powered by
the Serf library. Terraform leverages the membership and failure detection features,
and builds upon them.

The health checking provided by Serf is very low level, and only indicates if the
agent is alive. Terraform extends this to provide a rich health checking system,
that handles liveness, in addition to arbitrary host and service-level checks.
Health checks are integrated with a central catalog that operators can easily
query to gain insight into the cluster.

The membership provided by Serf is at a node level, while Terraform focuses
on the service level abstraction, with a single node to multiple service model.
This can be simulated in Serf using tags, but it is much more limited, and does
not provide useful query interfaces. Terraform also makes use of a strongly consistent
Catalog, while Serf is only eventually consistent.

In addition to the service level abstraction and improved health checking,
Terraform provides a key/value store and support for multiple datacenters.
Serf can run across the WAN but with degraded performance. Terraform makes use
of [multiple gossip pools](/docs/internals/architecture.html), so that
the performance of Serf over a LAN can be retained while still using it over
a WAN for linking together multiple datacenters.

Terraform is opinionated in its usage, while Serf is a more flexible and
general purpose tool. Terraform uses a CP architecture, favoring consistency over
availability. Serf is a AP system, and sacrifices consistency for availability.
This means Terraform cannot operate if the central servers cannot form a quorum,
while Serf will continue to function under almost all circumstances.

