---
layout: "docs"
page_title: "Operating a Job: Resource Utilization"
sidebar_current: "docs-jobops-resource-utilization"
description: |-
  Learn how to see resource utilization of a Nomad Job.
---

# Determining Resource Utilization

Understanding the resource utilization of your application is important for many
reasons and Nomad supports reporting detailed statistics in many of its drivers.
The main interface for seeing resource utilization is with the [`alloc-status`
command](/docs/commands/alloc-status.html) by specifying the `-stats` flag.

In the below example we are running `redis` and can see its resource utilization
below:

```
$ nomad alloc-status c3e0
ID            = c3e0e3e0
Eval ID       = 617e5e39
Name          = example.cache[0]
Node ID       = 39acd6e0
Job ID        = example
Client Status = running

Task "redis" is "running"
Task Resources
CPU       Memory          Disk     IOPS  Addresses
957/1000  30 MiB/256 MiB  300 MiB  0     db: 127.0.0.1:34907

Memory Stats
Cache   Max Usage  RSS     Swap
32 KiB  79 MiB     30 MiB  0 B

CPU Stats
Percent  Throttled Periods  Throttled Time
73.66%   0                  0

Recent Events:
Time                   Type      Description
06/28/16 16:43:50 UTC  Started   Task started by client
06/28/16 16:42:42 UTC  Received  Task received by client
```

Here we can see that we are near the limit of our configured CPU but we have
plenty of memory headroom. We can use this information to alter our job's
resources to better reflect is actually needs:

```
resource {
    cpu = 2000
    memory = 100
}
```

Adjusting resources is very important for a variety of reasons:

* Ensuring your application does not get OOM killed if it hits its memory limit.
* Ensuring the application performs well by ensuring it has some CPU allowance.
* Optimizing cluster density by reserving what you need and not over-allocating.

While single point in time resource usage measurements are useful, it is often
more useful to graph resource usage over time to better understand and estimate
resource usage. Nomad supports outputting resource data to statsite and statsd
and is the recommended way of monitoring resources. For more information about
outputting telemetry see the [Telemetry documentation](/docs/agent/telemetry.html).

For more advanced use cases, the resource usage data may also be accessed via
the client's HTTP API. See the documentation of the Client's
[Allocation HTTP API](/docs/http/client-allocation-stats.html)
