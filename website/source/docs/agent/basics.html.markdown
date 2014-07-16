---
layout: "docs"
page_title: "Agent"
sidebar_current: "docs-agent-running"
---

# Terraform Agent

The Terraform agent is the core process of Terraform. The agent maintains membership
information, registers services, runs checks, responds to queries
and more. The agent must run on every node that is part of a Terraform cluster.

Any Agent may run in one of two modes: client or server. A server
node takes on the additional responsibility of being part of the [consensus quorum](#).
These nodes take part in Raft, and provide strong consistency and availability in
the case of failure. The higher burden on the server nodes means that usually they
should be run on dedicated instances, as they are more resource intensive than a client
node. Client nodes make up the majority of the cluster, and they are very lightweight
as they maintain very little state and interface with the server nodes for most operations.

## Running an Agent

The agent is started with the `terraform agent` command. This command blocks,
running forever or until told to quit. The agent command takes a variety
of configuration options but the defaults are usually good enough. When
running `terraform agent`, you should see output similar to that below:

```
$ terraform agent -data-dir=/tmp/terraform
==> Starting Terraform agent...
==> Starting Terraform agent RPC...
==> Terraform agent running!
       Node name: 'Armons-MacBook-Air'
      Datacenter: 'dc1'
          Server: false (bootstrap: false)
     Client Addr: 127.0.0.1 (HTTP: 8500, DNS: 8600, RPC: 8400)
    Cluster Addr: 192.168.1.43 (LAN: 8301, WAN: 8302)

==> Log data will now stream in as it occurs:

    [INFO] serf: EventMemberJoin: Armons-MacBook-Air.local 192.168.1.43
...
```

There are several important components that `terraform agent` outputs:

* **Node name**: This is a unique name for the agent. By default this
  is the hostname of the machine, but you may customize it to whatever
  you'd like using the `-node` flag.

* **Datacenter**: This is the datacenter the agent is configured to run
 in. Terraform has first-class support for multiple datacenters, but to work efficiently
 each node must be configured to correctly report its datacenter. The `-dc` flag
 can be used to set the datacenter. For single-DC configurations, the agent
 will default to "dc1".

* **Server**: This shows if the agent is running in the server or client mode.
  Server nodes have the extra burden of participating in the consensus quorum,
  storing cluster state, and handling queries. Additionally, a server may be
  in "bootstrap" mode. The first server must be in this mode to allow additional
  servers to join the cluster. Multiple servers cannot be in bootstrap mode,
  otherwise the cluster state will be inconsistent.

* **Client Addr**: This is the address used for client interfaces to the agent.
  This includes the ports for the HTTP, DNS, and RPC interfaces. The RPC
  address is used for other `terraform` commands. Other Terraform commands such
  as `terraform members` connect to a running agent and use RPC to query and
  control the agent. By default, this binds only to localhost. If you
  change this address or port, you'll have to specify an `-rpc-addr` to commands
  such as `terraform members` so they know how to talk to the agent. This is also
  the address other applications can use over [RPC to control Terraform](/docs/agent/rpc.html).

* **Cluster Addr**: This is the address and ports used for communication between
  Terraform agents in a cluster. Every Terraform agent in a cluster does not have to
  use the same port, but this address **MUST** be reachable by all other nodes.

## Stopping an Agent

An agent can be stopped in two ways: gracefully or forcefully. To gracefully
halt an agent, send the process an interrupt signal, which is usually
`Ctrl-C` from a terminal. When gracefully exiting, the agent first notifies
the cluster it intends to leave the cluster. This way, other cluster members
notify the cluster that the node has _left_.

Alternatively, you can force kill the agent by sending it a kill signal.
When force killed, the agent ends immediately. The rest of the cluster will
eventually (usually within seconds) detect that the node has died and will
notify the cluster that the node has _failed_.

It is especially important that a server node be allowed to gracefully leave,
so that there will be a minimal impact on availability as the server leaves
the consensus quorum.

For client agents, the difference between a node _failing_ and a node _leaving_
may not be important for your use case. For example, for a web server and load
balancer setup, both result in the same action: remove the web node
from the load balancer pool. But for other situations, you may handle
each scenario differently.
