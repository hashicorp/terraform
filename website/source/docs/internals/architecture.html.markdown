---
layout: "docs"
page_title: "Terraform Architecture"
sidebar_current: "docs-internals-architecture"
---

# Terraform Architecture

Terraform is a complex system that has many different moving parts. To help
users and developers of Terraform form a mental model of how it works, this
page documents the system architecture.

<div class="alert alert-block alert-warning">
<strong>Advanced Topic!</strong> This page covers technical details of
the internals of Terraform. You don't need to know these details to effectively
operate and use Terraform. These details are documented here for those who wish
to learn about them without having to go spelunking through the source code.
</div>

## Glossary

Before describing the architecture, we provide a glossary of terms to help
clarify what is being discussed:

* Agent - An agent is the long running daemon on every member of the Terraform cluster.
It is started by running `terraform agent`. The agent is able to run in either *client*,
or *server* mode. Since all nodes must be running an agent, it is simpler to refer to
the node as either being a client or server, but there are other instances of the agent. All
agents can run the DNS or HTTP interfaces, and are responsible for running checks and
keeping services in sync.

* Client - A client is an agent that forwards all RPCs to a server. The client is relatively
stateless. The only background activity a client performs is taking part of LAN gossip pool.
This has a minimal resource overhead and consumes only a small amount of network bandwidth.

* Server - An agent that is server mode. When in server mode, there is an expanded set
of responsibilities including participating in the Raft quorum, maintaining cluster state,
responding to RPC queries, WAN gossip to other datacenters, and forwarding queries to leaders
or remote datacenters.

* Datacenter - A datacenter seems obvious, but there are subtle details such as multiple
availability zones in EC2. We define a datacenter to be a networking environment that is
private, low latency, and high bandwidth. This excludes communication that would traverse
the public internet.

* Consensus - When used in our documentation we use consensus to mean agreement upon
the elected leader as well as agreement on the ordering of transactions. Since these
transactions are applied to a FSM, we implicitly include the consistency of a replicated
state machine. Consensus is described in more detail on [Wikipedia](http://en.wikipedia.org/wiki/Consensus_(computer_science)),
as well as our [implementation here](/docs/internals/consensus.html).

* Gossip - Terraform is built on top of [Serf](http://www.serfdom.io/), which provides a full
[gossip protocol](http://en.wikipedia.org/wiki/Gossip_protocol) that is used for multiple purposes.
Serf provides membership, failure detection, and event broadcast mechanisms. Our use of these
is described more in the [gossip documentation](/docs/internals/gossip.html). It is enough to know
gossip involves random node-to-node communication, primarily over UDP.

* LAN Gossip - This is used to mean that there is a gossip pool, containing nodes that
are all located on the same local area network or datacenter.

* WAN Gossip - This is used to mean that there is a gossip pool, containing servers that
are primary located in different datacenters and must communicate over the internet or
wide area network.

* RPC - RPC is short for a Remote Procedure Call. This is a request / response mechanism
allowing a client to make a request from a server.

## 10,000 foot view

From a 10,000 foot altitude the architecture of Terraform looks like this:

![Terraform Architecture](/images/terraform-arch.png)

Lets break down this image and describe each piece. First of all we can see
that there are two datacenters, one and two respectively. Terraform has first
class support for multiple datacenters and expects this to be the common case.

Within each datacenter we have a mixture of clients and servers. It is expected
that there be between three to five servers. This strikes a balance between
availability in the case of failure and performance, as consensus gets progressively
slower as more machines are added. However, there is no limit to the number of clients,
and they can easily scale into the thousands or tens of thousands.

All the nodes that are in a datacenter participate in a [gossip protocol](/docs/internals/gossip.html).
This means there is a gossip pool that contains all the nodes for a given datacenter. This serves
a few purposes: first, there is no need to configure clients with the addresses of servers,
discovery is done automatically. Second, the work of detecting node failures
is not placed on the servers but is distributed. This makes the failure detection much more
scalable than naive heartbeating schemes. Thirdly, it is used as a messaging layer to notify
when important events such as leader election take place.

The servers in each datacenter are all part of a single Raft peer set. This means that
they work together to elect a leader, which has extra duties. The leader is responsible for
processing all queries and transactions. Transactions must also be replicated to all peers
as part of the [consensus protocol](/docs/internals/consensus.html). Because of this requirement,
when a non-leader server receives an RPC request it forwards it to the cluster leader.

The server nodes also operate as part of a WAN gossip. This pool is different from the LAN pool,
as it is optimized for the higher latency of the internet, and is expected to only contain
other Terraform server nodes. The purpose of this pool is to allow datacenters to discover each
other in a low touch manner. Bringing a new datacenter online is as easy as joining the existing
WAN gossip. Because the servers are all operating in this pool, it also enables cross-datacenter requests.
When a server receives a request for a different datacenter, it forwards it to a random server
in the correct datacenter. That server may then forward to the local leader.

This results in a very low coupling between datacenters, but because of failure detection,
connection caching and multiplexing, cross-datacenter requests are relatively fast and reliable.

## Getting in depth

At this point we've covered the high level architecture of Terraform, but there are much
more details to each of the sub-systems. The [consensus protocol](/docs/internals/consensus.html) is
documented in detail, as is the [gossip protocol](/docs/internals/gossip.html). The [documentation](/docs/internals/security.html)
for the security model and protocols used are also available.

For other details, either terraformt the code, ask in IRC or reach out to the mailing list.

