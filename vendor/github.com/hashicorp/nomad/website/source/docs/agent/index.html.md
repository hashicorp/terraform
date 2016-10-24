---
layout: "docs"
page_title: "Nomad Agent"
sidebar_current: "docs-agent-basics"
description: |-
  The Nomad agent is a long running process which can be used either in a client or server mode.
---

# Nomad Agent

The Nomad agent is a long running process which runs on every machine that
is part of the Nomad cluster. The behavior of the agent depends on if it is
running in client or server mode. Clients are responsible for running tasks,
while servers are responsible for managing the cluster.

Client mode agents are relatively simple. They make use of fingerprinting
to determine the capabilities and resources of the host machine, as well as 
determining what [drivers](/docs/drivers/index.html) are available. Clients
register with servers to provide the node information, heartbeat to provide
liveness, and run any tasks assigned to them.

Servers take on the responsibility of being part of the 
[consensus protocol](/docs/internals/consensus.html) and [gossip protocol](/docs/internals/gossip.html).
The consensus protocol, powered by Raft, allows the servers to perform
leader election and state replication. The gossip protocol allows for simple
clustering of servers and multi-region federation. The higher burden on the
server nodes means that usually they should be run on dedicated instances --
they are more resource intensive than a client node.

Client nodes make up the majority of the cluster, and are very lightweight
as they interface with the server nodes and maintain very little state of their own.
Each cluster has usually 3 or 5 server mode agents and potentially thousands of clients.

## Running an Agent

The agent is started with the [`nomad agent` command](/docs/commands/agent.html). This
command blocks, running forever or until told to quit. The agent command takes a variety
of configuration options, but most have sane defaults.

When running `nomad agent`, you should see output similar to this:

```text
$ nomad agent -dev
==> Starting Nomad agent...
==> Nomad agent configuration:

                 Atlas: (Infrastructure: 'armon/test' Join: false)
                Client: true
             Log Level: DEBUG
                Region: global (DC: dc1)
                Server: true

==> Nomad agent started! Log data will stream in below:

    [INFO] serf: EventMemberJoin: Armons-MacBook-Air.local.global 127.0.0.1
    [INFO] nomad: starting 4 scheduling worker(s) for [service batch _core]
...
```

There are several important messages that `nomad agent` outputs:

* **Atlas**: This shows the [Atlas infrastructure](https://atlas.hashicorp.com)
  with which the node is registered, if any. It also indicates if auto-join is enabled.
  The Atlas infrastructure is set using [`-atlas`](/docs/agent/config.html#_atlas)
  and auto-join is enabled by setting [`-atlas-join`](/docs/agent/config.html#_atlas_join).

* **Client**: This indicates whether the agent has enabled client mode.
  Client nodes fingerprint their host environment, register with servers,
  and run tasks.

* **Log Level**: This indicates the configured log level. Only messages with
  an equal or higher severity will be logged. This can be tuned to increase
  verbosity for debugging, or reduced to avoid noisy logging.

* **Region**: This is the region and datacenter in which the agent is configured to run.
 Nomad has first-class support for multi-datacenter and multi-region configurations.
 The [`-region` and `-dc`](/docs/agent/config.html#_region) flag can be used to set
 the region and datacenter. The default is the `global` region in `dc1`.

* **Server**: This indicates whether the agent has enabled server mode.
  Server nodes have the extra burden of participating in the consensus protocol,
  storing cluster state, and making scheduling decisions.

## Stopping an Agent

An agent can be stopped in two ways: gracefully or forcefully. By default,
any signal to an agent (interrupt, terminate, kill) will cause the agent
to forcefully stop. Graceful termination can be configured by either
setting `leave_on_interrupt` or `leave_on_terminate` to respond to the
respective signals.

When gracefully exiting, clients will update their status to terminal on
the servers so that tasks can be migrated to healthy agents. Servers
will notify their intention to leave the cluster which allows them to
leave the [consensus quorum](/docs/internals/consensus.html).

It is especially important that a server node be allowed to leave gracefully
so that there will be a minimal impact on availability as the server leaves
the consensus quorum. If a server does not gracefully leave, and will not
return into service, the [`server-force-leave` command](/docs/commands/server-force-leave.html)
should be used to eject it from the consensus quorum.

## Lifecycle

Every agent in the Nomad cluster goes through a lifecycle. Understanding
this lifecycle is useful for building a mental model of an agent's interactions
with a cluster and how the cluster treats a node.

When a client agent is first started, it fingerprints the host machine to
identify its attributes, capabilities, and [task drivers](/docs/drivers/index.html).
These are reported to the servers during an initial registration. The addresses
of known servers are provided to the agent via configuration, potentially using
DNS for resolution. Using [Consul](https://www.consul.io) provides a way to avoid hard
coding addresses and resolving them on demand.

While a client is running, it is performing heartbeating with servers to
maintain liveness. If the hearbeats fail, the servers assume the client node
has failed, and stop assigning new tasks while migrating existing tasks.
It is impossible to distinguish between a network failure and an agent crash,
so both cases are handled the same. Once the network recovers or a crashed agent
restarts the node status will be updated and normal operation resumed.

To prevent an accumulation of nodes in a terminal state, Nomad does periodic
garbage collection of nodes. By default, if a node is in a failed or 'down'
state for over 24 hours it will be garbage collected from the system.

Servers are slightly more complex as they perform additional functions. They
participate in a [gossip protocol](/docs/internals/gossip.html) both to cluster
within a region and to support multi-region configurations. When a server is
first started, it does not know the address of other servers in the cluster.
To discover its peers, it must _join_ the cluster. This is done with the
[`server-join` command](/docs/commands/server-join.html) or by providing the
proper configuration on start. Once a node joins, this information is gossiped
to the entire cluster, meaning all nodes will eventually be aware of each other.

When a server _leaves_, it specifies its intent to do so, and the cluster marks that
node as having _left_. If the server has _left_, replication to it will stop and it
is removed as a member of the consensus quorum. If the server has _failed_, replication
will attempt to make progress to recover from a software or network failure.
