---
layout: "docs"
page_title: "Creating a Nomad Cluster"
sidebar_current: "docs-cluster-bootstrap"
description: |-
  Learn how to bootstrap a Nomad cluster.
---

# Creating a cluster

Nomad models infrastructure as regions and datacenters. Regions may contain
multiple datacenters. Servers are assigned to regions and manage all state for
the region and make scheduling decisions within that region. Clients are
registered to a single datacenter and region.

[![Regional Architecture](/assets/images/nomad-architecture-region.png)](/assets/images/nomad-architecture-region.png)

This page will explain how to bootstrap a production grade Nomad region, both
with and without Consul, and how to federate multiple regions together.

[![Global Architecture](/assets/images/nomad-architecture-global.png)](/assets/images/nomad-architecture-global.png)

Bootstrapping Nomad is made significantly easier when there already exists a
Consul cluster in place. Since Nomad's topology is slightly richer than Consul's
since it supports not only datacenters but also regions lets start with how
Consul should be deployed in relation to Nomad.

For more details on the architecture of Nomad and how it models infrastructure
see the [Architecture page](/docs/internals/architecture.html).

## Deploying Consul Clusters

A Nomad cluster gains the ability to bootstrap itself as well as provide service
and health check registration to applications when Consul is deployed along side
Nomad.

Consul models infrastructures as datacenters and multiple Consul datacenters can
be connected over the WAN so that clients can discover nodes in other
datacenters. Since Nomad regions can encapsulate many datacenters, we recommend
running a Consul cluster in every Nomad datacenter and connecting them over the
WAN. Please refer to the Consul guide for both
[bootstrapping](https://www.consul.io/docs/guides/bootstrapping.html) a single datacenter and 
[connecting multiple Consul clusters over the
WAN](https://www.consul.io/docs/guides/datacenters.html).


## Bootstrapping a Nomad cluster

Nomad supports merging multiple configuration files together on startup. This is
done to enable generating a base configuration that can be shared by Nomad
servers and clients. A suggested base configuration is:

```
# Name the region, if omitted, the default "global" region will be used.
region = "europe"

# Persist data to a location that will survive a machine reboot.
data_dir = "/opt/nomad/"

# Bind to all addresses so that the Nomad agent is available both on loopback
# and externally.
bind_addr = "0.0.0.0"

# Advertise an accessible IP address so the server is reachable by other servers
# and clients. The IPs can be materialized by Terraform or be replaced by an
# init script.
advertise {
    http = "${self.ipv4_address}:4646"
    rpc = "${self.ipv4_address}:4647"
    serf = "${self.ipv4_address}:4648"
}

# Ship metrics to monitor the health of the cluster and to see task resource
# usage.
telemetry {
    statsite_address = "${var.statsite}"
    disable_hostname = true
}

# Enable debug endpoints.
enable_debug = true
```

### With Consul

If a local Consul cluster is bootstrapped before Nomad, on startup Nomad
server's will register with Consul and discover other server's. With their set
of peers, they will automatically form quorum, respecting the `bootstrap_expect`
field. Thus to form a 3 server region, the below configuration can be used in
conjunction with the base config:

```
server {
    enabled = true
    bootstrap_expect = 3
}
```

And an equally simple configuration can be used for clients:

```
# Replace with the relevant datacenter.
datacenter = "dc1"

client {
    enabled = true
}
```

As you can see, the above configurations have no mention of the other server's to
join or any Consul configuration. That is because by default, the following is
merged with the configuration file:

```
consul {
    # The address to the Consul agent.
    address = "127.0.0.1:8500"

    # The service name to register the server and client with Consul.
    server_service_name = "nomad"
    client_service_name = "nomad-client"

    # Enables automatically registering the services.
    auto_advertise = true

    # Enabling the server and client to bootstrap using Consul.
    server_auto_join = true
    client_auto_join = true
}
```

Since the `consul` block is merged by default, bootstrapping a cluster becomes
as easy as running the following on each of the three servers:

```
$ nomad agent -config base.hcl -config server.hcl
```

And on every client in the cluster, the following should be run:

```
$ nomad agent -config base.hcl -config client.hcl
```

With the above configurations and commands the Nomad agents will automatically
register themselves with Consul and discover other Nomad servers. If the agent
is a server, it will join the quorum and if it is a client, it will register
itself and join the cluster.

Please refer to the [Consul documentation](/docs/agent/config.html#consul_options)
for the complete set of configuration options.

### Without Consul

When bootstrapping without Consul, Nomad servers and clients must be started
knowing the address of at least one Nomad server.

To join the Nomad server's we can either encode the address in the server
configs as such:

```
server {
    enabled = true
    bootstrap_expect = 3
    retry_join = ["<known-address>"]
}
```

Alternatively, the address can be supplied after the servers have all been started by
running the [`server-join` command](/docs/commands/server-join.html) on the servers
individual to cluster the servers. All servers can join just one other server,
and then rely on the gossip protocol to discover the rest.

```
nomad server-join <known-address>
```

On the client side, the addresses of the servers are expected to be specified
via the client configuration.

```
client {
    enabled = true
    servers = ["10.10.11.2:4648", "10.10.11.3:4648", "10.10.11.4:4648"]
}
```

If servers are added or removed from the cluster, the information will be
pushed to the client. This means, that only one server must be specified because
after initial contact, the full set of servers in the client's region will be
pushed to the client.

The same commmands can be used to start the servers and clients as shown in the
bootstrapping with Consul section.

### Federating a cluster

Nomad clusters across multiple regions can be federated allowing users to submit
jobs or interact with the HTTP API targeting any region, from any server.

Federating multiple Nomad clusters is as simple as joining servers. From any
server in one region, simply issue a join command to a server in the remote
region:

```
nomad server-join 10.10.11.8:4648
```

Servers across regions discover other servers in the cluster via the gossip
protocol and hence it enough to join one known server.

If the Consul clusters in the different Nomad regions are federated, and Consul
`server_auto_join` is enabled, then federation occurs automatically.

## Network Topology

### Nomad Servers

Nomad servers are expected to have sub 10 millisecond network latencies between
each other to ensure liveness and high throughput scheduling. Nomad servers
can be spread across multiple datacenters if they have low latency
connections between them to achieve high availability.

For example, on AWS every region comprises of multiple zones which have very low
latency links between them, so every zone can be modeled as a Nomad datacenter
and every Zone can have a single Nomad server which could be connected to form a
quorum and a region. 

Nomad servers uses Raft for state replication and Raft being highly consistent
needs a quorum of servers to function, therefore we recommend running an odd
number of Nomad servers in a region.  Usually running 3-5 servers in a region is
recommended. The cluster can withstand a failure of one server in a cluster of
three servers and two failures in a cluster of five servers. Adding more servers
to the quorum adds more time to replicate state and hence throughput decreases
so we don't recommend having more than seven servers in a region.

### Nomad Clients

Nomad clients do not have the same latency requirements as servers since they
are not participating in Raft. Thus clients can have 100+ millisecond latency to
their servers. This allows having a set of Nomad servers that service clients
that can be spread geographically over a continent or even the world in the case
of having a single "global" region and many datacenter.

## Production Considerations

### Nomad Servers

Depending on the number of jobs the cluster will be managing and the rate at
which jobs are submitted, the Nomad servers may need to be run on large machine
instances. We suggest having 8+ cores, 32 GB+ of memory, 80 GB+ of disk and
significant network bandwith. The core count and network recommendations are to
ensure high throughput as Nomad heavily relies on network communication and as
the Servers are managing all the nodes in the region and performing scheduling.
The memory and disk requirements are due to the fact that Nomad stores all state
in memory and will store two snapshots of this data onto disk. Thus disk should
be at least 2 times the memory available to the server when deploying a high
load cluster.

### Nomad Clients

Nomad clients support reserving resources on the node that should not be used by
Nomad. This should be used to target a specific resource utilization per node
and to reserve resources for applications running outside of Nomad's supervision
such as Consul and the operating system itself.

Please see the [`reservation` config](/docs/agent/config.html#reserved) for more detail.
