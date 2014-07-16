---
layout: "intro"
page_title: "Terraform vs. ZooKeeper, doozerd, etcd"
sidebar_current: "vs-other-zk"
---

# Terraform vs. ZooKeeper, doozerd, etcd

ZooKeeper, doozerd and etcd are all similar in their architecture.
All three have server nodes that require a quorum of nodes to operate (usually a simple majority).
They are strongly consistent, and expose various primitives that can be used
through client libraries within applications to build complex distributed systems.

Terraform works in a similar way within a single datacenter with only server nodes.
In each datacenter, Terraform servers require a quorum to operate
and provide strong consistency. However, Terraform has native support for multiple datacenters,
as well as a more complex gossip system that links server nodes and clients.

If any of these systems are used for pure key/value storage, then they all
roughly provide the same semantics. Reads are strongly consistent, and availability
is sacrificed for consistency in the face of a network partition. However, the differences
become more apparent when these systems are used for advanced cases.

The semantics provided by these systems are attractive for building
service discovery systems. ZooKeeper et al. provide only a primitive K/V store,
and require that application developers build their own system to provide service
discovery. Terraform provides an opinionated framework for service discovery, and
eliminates the guess work and development effort. Clients simply register services
and then perform discovery using a DNS or HTTP interface. Other systems
require a home-rolled solution.

A compelling service discovery framework must incorporate health checking and the
possibility of failures as well. It is not useful to know that Node A
provides the Foo service if that node has failed or the service crashed. Naive systems
make use of heartbeating, using periodic updates and TTLs. These schemes require work linear
to the number of nodes and place the demand on a fixed number of servers. Additionally, the
failure detection window is at least as long as the TTL. ZooKeeper provides ephemeral
nodes which are K/V entries that are removed when a client disconnects. These are more
sophisticated than a heartbeat system, but also have inherent scalability issues and add
client side complexity. All clients must maintain active connections to the ZooKeeper servers,
and perform keep-alives. Additionally, this requires "thick clients", which are difficult
to write and often result in difficult to debug issues.

Terraform uses a very different architecture for health checking. Instead of only
having server nodes, Terraform clients run on every node in the cluster.
These clients are part of a [gossip pool](/docs/internals/gossip.html), which
serves several functions including distributed health checking. The gossip protocol implements
an efficient failure detector that can scale to clusters of any size without concentrating
the work on any select group of servers. The clients also enable a much richer set of health checks to be run locally,
whereas ZooKeeper ephemeral nodes are a very primitive check of liveness. Clients can check that
a web server is returning 200 status codes, that memory utilization is not critical, there is sufficient
disk space, etc. The Terraform clients expose a simple HTTP interface and avoid exposing the complexity
of the system is to clients in the same way as ZooKeeper.

Terraform provides first class support for service discovery, health checking,
K/V storage, and multiple datacenters. To support anything more than simple K/V storage,
all these other systems require additional tools and libraries to be built on
top. By using client nodes, Terraform provides a simple API that only requires thin clients.
Additionally, the API can be avoided entirely by using configuration files and the
DNS interface to have a complete service discovery solution with no development at all.

