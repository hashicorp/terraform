---
layout: "intro"
page_title: "Terraform vs. SmartStack"
sidebar_current: "vs-other-smartstack"
---

# Terraform vs. SmartStack

SmartStack is another tool which tackles the service discovery problem.
It has a rather unique architecture, and has 4 major components: ZooKeeper,
HAProxy, Synapse, and Nerve. The ZooKeeper servers are responsible for storing cluster
state in a consistent and fault tolerant manner. Each node in the SmartStack
cluster then runs both Nerves and Synapses. The Nerve is responsible for running
health checks against a service, and registering with the ZooKeeper servers.
Synapse queries ZooKeeper for service providers and dynamically configures
HAProxy. Finally, clients speak to HAProxy, which does health checking and
load balancing across service providers.

Terraform is a much simpler and more contained system, as it does not rely on any external
components. Terraform uses an integrated [gossip protocol](/docs/internals/gossip.html)
to track all nodes and perform server discovery. This means that server addresses
do not need to be hardcoded and updated fleet wide on changes, unlike SmartStack.

Service registration for both Terraform and Nerves can be done with a configuration file,
but Terraform also supports an API to dynamically change the services and checks that are in use.

For discovery, SmartStack clients must use HAProxy, requiring that Synapse be
configured with all desired endpoints in advance. Terraform clients instead
use the DNS or HTTP APIs without any configuration needed in advance. Terraform
also provides a "tag" abstraction, allowing services to provide metadata such
as versions, primary/secondary designations, or opaque labels that can be used for
filtering. Clients can then request only the service providers which have
matching tags.

The systems also differ in how they manage health checking.
Nerve's performs local health checks in a manner similar to Terraform agents.
However, Terraform maintains separate catalog and health systems, which allow
operators to see which nodes are in each service pool, as well as providing
insight into failing checks. Nerve simply deregisters nodes on failed checks,
providing limited operator insight. Synapse also configures HAProxy to perform
additional health checks. This causes all potential service clients to check for
liveness. With large fleets, this N-to-N style health checking may be prohibitively
expensive.

Terraform generally provides a much richer health checking system. Terraform supports
Nagios style plugins, enabling a vast catalog of checks to be used. It also
allows for service and host-level checks. There is also a "dead man's switch"
check that allows applications to easily integrate custom health checks. All of this
is also integrated into a Health and Catalog system with APIs enabling operator
to gain insight into the broader system.

In addition to the service discovery and health checking, Terraform also provides
an integrated key/value store for configuration and multi-datacenter support.
While it may be possible to configure SmartStack for multiple datacenters,
the central ZooKeeper cluster would be a serious impediment to a fault tolerant
deployment.

