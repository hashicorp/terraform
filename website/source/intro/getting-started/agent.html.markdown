---
layout: "intro"
page_title: "Run the Agent"
sidebar_current: "gettingstarted-agent"
---

# Run the Terraform Agent

After Terraform is installed, the agent must be run. The agent can either run
in a server or client mode. Each datacenter must have at least one server,
although 3 or 5 is recommended. A single server deployment is _**highly**_ discouraged
as data loss is inevitable in a failure scenario. [This guide](/docs/guides/bootstrapping.html)
covers bootstrapping a new datacenter. All other agents run in client mode, which
is a very lightweight process that registers services, runs health checks,
and forwards queries to servers. The agent must be run for every node that
will be part of the cluster.

## Starting the Agent

For simplicity, we'll run a single Terraform agent in server mode right now:

```
$ terraform agent -server -bootstrap -data-dir /tmp/consul
==> WARNING: Bootstrap mode enabled! Do not enable unless necessary
==> WARNING: It is highly recommended to set GOMAXPROCS higher than 1
==> Starting Terraform agent...
==> Starting Terraform agent RPC...
==> Terraform agent running!
       Node name: 'Armons-MacBook-Air'
      Datacenter: 'dc1'
          Server: true (bootstrap: true)
     Client Addr: 127.0.0.1 (HTTP: 8500, DNS: 8600, RPC: 8400)
    Cluster Addr: 10.1.10.38 (LAN: 8301, WAN: 8302)

==> Log data will now stream in as it occurs:

[INFO] serf: EventMemberJoin: Armons-MacBook-Air.local 10.1.10.38
[INFO] raft: Node at 10.1.10.38:8300 [Follower] entering Follower state
[INFO] terraform: adding server for datacenter: dc1, addr: 10.1.10.38:8300
[ERR] agent: failed to sync remote state: rpc error: No cluster leader
[WARN] raft: Heartbeat timeout reached, starting election
[INFO] raft: Node at 10.1.10.38:8300 [Candidate] entering Candidate state
[INFO] raft: Election won. Tally: 1
[INFO] raft: Node at 10.1.10.38:8300 [Leader] entering Leader state
[INFO] terraform: cluster leadership acquired
[INFO] terraform: New leader elected: Armons-MacBook-Air
[INFO] terraform: member 'Armons-MacBook-Air' joined, marking health alive
```

As you can see, the Terraform agent has started and has output some log
data. From the log data, you can see that our agent is running in server mode,
and has claimed leadership of the cluster. Additionally, the local member has
been marked as a healthy member of the cluster.

<div class="alert alert-block alert-warning">
<strong>Note for OS X Users:</strong> Terraform uses your hostname as the
default node name. If your hostname contains periods, DNS queries to
that node will not work with Terraform. To avoid this, explicitly set
the name of your node with the <code>-node</code> flag.
</div>

## Cluster Members

If you run `terraform members` in another terminal, you can see the members of
the Terraform cluster. You should only see one member (yourself). We'll cover
joining clusters in the next section.

```
$ terraform members
Armons-MacBook-Air  10.1.10.38:8301  alive  role=terraform,dc=dc1,vsn=1,vsn_min=1,vsn_max=1,port=8300,bootstrap=1
```

The output shows our own node, the address it is running on, its
health state, and some metadata associated with the node. Some important
metadata keys to recognize are the `role` and `dc` keys. These tell you
the service name and the datacenter that member is within. These can be
used to lookup nodes and services using the DNS interface, which is covered
shortly.

The output from the `members` command is generated based on the
[gossip protocol](/docs/internals/gossip.html) and is eventually consistent.
For a strongly consistent view of the world, use the
[HTTP API](/docs/agent/http.html), which forwards the request to the
Terraform servers:

```
$ curl localhost:8500/v1/catalog/nodes
[{"Node":"Armons-MacBook-Air","Address":"10.1.10.38"}]
```

In addition to the HTTP API, the
[DNS interface](/docs/agent/dns.html) can be used to query the node. Note
that you have to make sure to point your DNS lookups to the Terraform agent's
DNS server, which runs on port 8600 by default. The format of the DNS
entries (such as "Armons-MacBook-Air.node.terraform") will be covered later.

```
$ dig @127.0.0.1 -p 8600 Armons-MacBook-Air.node.terraform
...

;; QUESTION SECTION:
;Armons-MacBook-Air.node.terraform.	IN	A

;; ANSWER SECTION:
Armons-MacBook-Air.node.terraform.	0 IN	A	10.1.10.38
```

## Stopping the Agent

You can use `Ctrl-C` (the interrupt signal) to gracefully halt the agent.
After interrupting the agent, you should see it leave the cluster gracefully
and shut down.

By gracefully leaving, Terraform notifies other cluster members that the
node _left_. If you had forcibly killed the agent process, other members
of the cluster would have detected that the node _failed_. When a member leaves,
its services and checks are removed from the catalog. When a member fails,
its health is simply marked as critical, but is not removed from the catalog.
Terraform will automatically try to reconnect to _failed_ nodes, which allows it
to recover from certain network conditions, while _left_ nodes are no longer contacted.

Additionally, if an agent is operating as a server, a graceful leave is important
to avoid causing a potential availability outage affecting the [consensus protocol](/docs/internals/consensus.html).
See the [guides section](/docs/guides/index.html) to safely add and remove servers.

