---
layout: "docs"
page_title: "Telemetry"
sidebar_current: "docs-agent-telemetry"
description: |-
  Learn about the telemetry data available in Nomad.
---

# Telemetry

The Nomad agent collects various runtime metrics about the performance of
different libraries and subsystems. These metrics are aggregated on a ten
second interval and are retained for one minute.

To view this data, you must send a signal to the Nomad process: on Unix,
this is `USR1` while on Windows it is `BREAK`. Once Nomad receives the signal,
it will dump the current telemetry information to the agent's `stderr`.

This telemetry information can be used for debugging or otherwise
getting a better view of what Nomad is doing.

Telemetry information can be streamed to both [statsite](https://github.com/armon/statsite)
as well as statsd based on providing the appropriate configuration options.

To configure the telemetry output please see the [agent
configuration](/docs/agent/config.html#telemetry_config).

Below is sample output of a telemetry dump:

```text
[2015-09-17 16:59:40 -0700 PDT][G] 'nomad.nomad.broker.total_blocked': 0.000
[2015-09-17 16:59:40 -0700 PDT][G] 'nomad.nomad.plan.queue_depth': 0.000
[2015-09-17 16:59:40 -0700 PDT][G] 'nomad.runtime.malloc_count': 7568.000
[2015-09-17 16:59:40 -0700 PDT][G] 'nomad.runtime.total_gc_runs': 8.000
[2015-09-17 16:59:40 -0700 PDT][G] 'nomad.nomad.broker.total_ready': 0.000
[2015-09-17 16:59:40 -0700 PDT][G] 'nomad.runtime.num_goroutines': 56.000
[2015-09-17 16:59:40 -0700 PDT][G] 'nomad.runtime.sys_bytes': 3999992.000
[2015-09-17 16:59:40 -0700 PDT][G] 'nomad.runtime.heap_objects': 4135.000
[2015-09-17 16:59:40 -0700 PDT][G] 'nomad.nomad.heartbeat.active': 1.000
[2015-09-17 16:59:40 -0700 PDT][G] 'nomad.nomad.broker.total_unacked': 0.000
[2015-09-17 16:59:40 -0700 PDT][G] 'nomad.nomad.broker.total_waiting': 0.000
[2015-09-17 16:59:40 -0700 PDT][G] 'nomad.runtime.alloc_bytes': 634056.000
[2015-09-17 16:59:40 -0700 PDT][G] 'nomad.runtime.free_count': 3433.000
[2015-09-17 16:59:40 -0700 PDT][G] 'nomad.runtime.total_gc_pause_ns': 6572135.000
[2015-09-17 16:59:40 -0700 PDT][C] 'nomad.memberlist.msg.alive': Count: 1 Sum: 1.000
[2015-09-17 16:59:40 -0700 PDT][C] 'nomad.serf.member.join': Count: 1 Sum: 1.000
[2015-09-17 16:59:40 -0700 PDT][C] 'nomad.raft.barrier': Count: 1 Sum: 1.000
[2015-09-17 16:59:40 -0700 PDT][C] 'nomad.raft.apply': Count: 1 Sum: 1.000
[2015-09-17 16:59:40 -0700 PDT][C] 'nomad.nomad.rpc.query': Count: 2 Sum: 2.000
[2015-09-17 16:59:40 -0700 PDT][S] 'nomad.serf.queue.Query': Count: 6 Sum: 0.000
[2015-09-17 16:59:40 -0700 PDT][S] 'nomad.nomad.fsm.register_node': Count: 1 Sum: 1.296
[2015-09-17 16:59:40 -0700 PDT][S] 'nomad.serf.queue.Intent': Count: 6 Sum: 0.000
[2015-09-17 16:59:40 -0700 PDT][S] 'nomad.runtime.gc_pause_ns': Count: 8 Min: 126492.000 Mean: 821516.875 Max: 3126670.000 Stddev: 1139250.294 Sum: 6572135.000
[2015-09-17 16:59:40 -0700 PDT][S] 'nomad.raft.leader.dispatchLog': Count: 3 Min: 0.007 Mean: 0.018 Max: 0.039 Stddev: 0.018 Sum: 0.054
[2015-09-17 16:59:40 -0700 PDT][S] 'nomad.nomad.leader.reconcileMember': Count: 1 Sum: 0.007
[2015-09-17 16:59:40 -0700 PDT][S] 'nomad.nomad.leader.reconcile': Count: 1 Sum: 0.025
[2015-09-17 16:59:40 -0700 PDT][S] 'nomad.raft.fsm.apply': Count: 1 Sum: 1.306
[2015-09-17 16:59:40 -0700 PDT][S] 'nomad.nomad.client.get_allocs': Count: 1 Sum: 0.110
[2015-09-17 16:59:40 -0700 PDT][S] 'nomad.nomad.worker.dequeue_eval': Count: 29 Min: 0.003 Mean: 363.426 Max: 503.377 Stddev: 228.126 Sum: 10539.354
[2015-09-17 16:59:40 -0700 PDT][S] 'nomad.serf.queue.Event': Count: 6 Sum: 0.000
[2015-09-17 16:59:40 -0700 PDT][S] 'nomad.raft.commitTime': Count: 3 Min: 0.013 Mean: 0.037 Max: 0.079 Stddev: 0.037 Sum: 0.110
[2015-09-17 16:59:40 -0700 PDT][S] 'nomad.nomad.leader.barrier': Count: 1 Sum: 0.071
[2015-09-17 16:59:40 -0700 PDT][S] 'nomad.nomad.client.register': Count: 1 Sum: 1.626
[2015-09-17 16:59:40 -0700 PDT][S] 'nomad.nomad.eval.dequeue': Count: 21 Min: 500.610 Mean: 501.753 Max: 503.361 Stddev: 1.030 Sum: 10536.813
[2015-09-17 16:59:40 -0700 PDT][S] 'nomad.memberlist.gossip': Count: 12 Min: 0.009 Mean: 0.017 Max: 0.025 Stddev: 0.005 Sum: 0.204
```

# Key Metrics

When telemetry is being streamed to statsite or statsd, `interval` is defined to
be their flush interval. Otherwise, the interval can be assumed to be 10 seconds
when retrieving metrics using the above described signals.

<table class="table table-bordered table-striped">
  <tr>
    <th>Metric</th>
    <th>Description</th>
    <th>Unit</th>
    <th>Type</th>
  </tr>
  <tr>
    <td>`nomad.runtime.num_goroutines`</td>
    <td>Number of goroutines and general load pressure indicator</td>
    <td># of goroutines</td>
    <td>Gauge</td>
  </tr>
  <tr>
    <td>`nomad.runtime.alloc_bytes`</td>
    <td>Memory utilization</td>
    <td># of bytes</td>
    <td>Gauge</td>
  </tr>
  <tr>
    <td>`nomad.runtime.heap_objects`</td>
    <td>Number of objects on the heap. General memory pressure indicator</td>
    <td># of heap objects</td>
    <td>Gauge</td>
  </tr>
  <tr>
    <td>`nomad.raft.apply`</td>
    <td>Number of Raft transactions</td>
    <td>Raft transactions / `interval`</td>
    <td>Counter</td>
  </tr>
  <tr>
    <td>`nomad.raft.replication.appendEntries`</td>
    <td>Raft transaction commit time</td>
    <td>ms / Raft Log Append</td>
    <td>Timer</td>
  </tr>
  <tr>
    <td>`nomad.raft.leader.lastContact`</td>
    <td>Time since last contact to leader. General indicator of Raft latency</td>
    <td>ms / Leader Contact</td>
    <td>Timer</td>
  </tr>
  <tr>
    <td>`nomad.broker.total_ready`</td>
    <td>Number of evaluations ready to be processed</td>
    <td># of evaluations</td>
    <td>Gauge</td>
  </tr>
  <tr>
    <td>`nomad.broker.total_unacked`</td>
    <td>Evaluations dispatched for processing but incomplete</td>
    <td># of evaluations</td>
    <td>Gauge</td>
  </tr>
  <tr>
    <td>`nomad.broker.total_blocked`</td>
    <td>
        Evaluations that are blocked til an existing evaluation for the same job
        completes
    </td>
    <td># of evaluations</td>
    <td>Gauge</td>
  </tr>
  <tr>
    <td>`nomad.plan.queue_depth`</td>
    <td>Number of scheduler Plans waiting to be evaluated</td>
    <td># of plans</td>
    <td>Gauge</td>
  </tr>
  <tr>
    <td>`nomad.plan.submit`</td>
    <td>
        Time to submit a scheduler Plan. Higher values cause lower scheduling
        throughput
    </td>
    <td>ms / Plan Submit</td>
    <td>Timer</td>
  </tr>
  <tr>
    <td>`nomad.plan.evaluate`</td>
    <td>
        Time to validate a scheduler Plan. Higher values cause lower scheduling
        throughput. Similar to `nomad.plan.submit` but does not include RPC time
        or time in the Plan Queue
    </td>
    <td>ms / Plan Evaluation</td>
    <td>Timer</td>
  </tr>
  <tr>
    <td>`nomad.worker.invoke_scheduler.<type>`</td>
    <td>Time to run the scheduler of the given type</td>
    <td>ms / Scheduler Run</td>
    <td>Timer</td>
  </tr>
  <tr>
    <td>`nomad.worker.wait_for_index`</td>
    <td>
        Time waiting for Raft log replication from leader. High delays result in
        lower scheduling throughput
    </td>
    <td>ms / Raft Index Wait</td>
    <td>Timer</td>
  </tr>
  <tr>
    <td>`nomad.heartbeat.active`</td>
    <td>
        Number of active heartbeat timers. Each timer represents a Nomad Client
        connection
    </td>
    <td># of heartbeat timers</td>
    <td>Gauge</td>
  </tr>
  <tr>
    <td>`nomad.heartbeat.invalidate`</td>
    <td>
        The length of time it takes to invalidate a Nomad Client due to failed
        heartbeats
    </td>
    <td>ms / Heartbeat Invalidation</td>
    <td>Timer</td>
  </tr>
  <tr>
    <td>`nomad.rpc.query`</td>
    <td>Number of RPC queries</td>
    <td>RPC Queries / `interval`</td>
    <td>Counter</td>
  </tr>
  <tr>
    <td>`nomad.rpc.request`</td>
    <td>Number of RPC requests being handled</td>
    <td>RPC Requests / `interval`</td>
    <td>Counter</td>
  </tr>
  <tr>
    <td>`nomad.rpc.request_error`</td>
    <td>Number of RPC requests being handled that result in an error</td>
    <td>RPC Errors / `interval`</td>
    <td>Counter</td>
  </tr>
</table>

# Client Metrics

The Nomad client emits metrics related to the resource usage of the allocations
and tasks running on it and the node itself. By default the collection interval
is 1 second but it can be changed by the changing the value of the
`collection_interval` key in the `telemetry` configuration block.

## Host Metrics

<table class="table table-bordered table-striped">
  <tr>
    <th>Metric</th>
    <th>Description</th>
    <th>Unit</th>
    <th>Type</th>
  </tr>
  <tr>
    <td>`nomad.client.host.memmory.<HostID>.total`</td>
    <td>Total amount of physical memory on the node</td>
    <td>Bytes</td>
    <td>Gauge</td>
  </tr>
  <tr>
    <td>`nomad.client.host.memmory.<HostID>.available`</td>
    <td>Total amount of memory available to processes which includes free and
    cached memory</td>
    <td>Bytes</td>
    <td>Gauge</td>
  </tr>
  <tr>
    <td>`nomad.client.host.memory.<HostID>.used`</td>
    <td>Amount of memory used by processes</td>
    <td>Bytes</td>
    <td>Gauge</td>
  </tr>
  <tr>
    <td>`nomad.client.host.memory.<HostID>.free`</td>
    <td>Amount of memory which is free</td>
    <td>Bytes</td>
    <td>Gauge</td>
  </tr>
  <tr>
    <td>`nomad.client.uptime.<HostID>`</td>
    <td>Uptime of the host running the Nomad client</td>
    <td>Seconds</td>
    <td>Gauge</td>
  </tr>
  <tr>
    <td>`nomad.client.host.cpu.<HostID>.<CPU-Core>.total`</td>
    <td>Total CPU utilization</td>
    <td>Percentage</td>
    <td>Gauge</td>
  </tr>
  <tr>
    <td>`nomad.client.host.cpu.<HostID>.<CPU-Core>.user`</td>
    <td>CPU utilization in the user space</td>
    <td>Percentage</td>
    <td>Gauge</td>
  </tr>
  <tr>
    <td>`nomad.client.host.cpu.<HostID>.<CPU-Core>.system`</td>
    <td>CPU utilization in the system space</td>
    <td>Percentage</td>
    <td>Gauge</td>
  </tr>
  <tr>
    <td>`nomad.client.host.cpu.<HostID>.<CPU-Core>.idle`</td>
    <td>Idle time spent by the CPU</td>
    <td>Percentage</td>
    <td>Gauge</td>
  </tr>
  <tr>
    <td>`nomad.client.host.disk.<HostID>.<Device-Name>.size`</td>
    <td>Total size of the device</td>
    <td>Bytes</td>
    <td>Gauge</td>
  </tr>
  <tr>
    <td>`nomad.client.host.disk.<HostID>.<Device-Name>.used`</td>
    <td>Amount of space which has been used</td>
    <td>Bytes</td>
    <td>Gauge</td>
  </tr>
  <tr>
    <td>`nomad.client.host.disk.<HostID>.<Device-Name>.available`</td>
    <td>Amount of space which is available</td>
    <td>Bytes</td>
    <td>Gauge</td>
  </tr>
  <tr>
    <td>`nomad.client.host.disk.<HostID>.<Device-Name>.used_percent`</td>
    <td>Percentage of disk space used</td>
    <td>Percentage</td>
    <td>Gauge</td>
  </tr>
  <tr>
    <td>`nomad.client.host.disk.<HostID>.<Device-Name>.inodes_percent`</td>
    <td>Disk space consumed by the inodes</td>
    <td>Percent</td>
    <td>Gauge</td>
  </tr>
</table>

## Allocation Metrics 

<table class="table table-bordered table-striped">
  <tr>
    <th>Metric</th>
    <th>Description</th>
    <th>Unit</th>
    <th>Type</th>
  </tr>
  <tr>
    <td>`nomad.client.allocs.<Job>.<TaskGroup>.<AllocID>.<Task>.memory.rss`</td>
    <td>Amount of RSS memory consumed by the task</td>
    <td>Bytes</td>
    <td>Gauge</td>
  </tr>
  <tr>
    <td>`nomad.client.allocs.<Job>.<TaskGroup>.<AllocID>.<Task>.memory.cache`</td>
    <td>Amount of memory cached by the task</td>
    <td>Bytes</td>
    <td>Gauge</td>
  </tr>
  <tr>
    <td>`nomad.client.allocs.<Job>.<TaskGroup>.<AllocID>.<Task>.memory.swap`</td>
    <td>Amount of memory swapped by the task</td>
    <td>Bytes</td>
    <td>Gauge</td>
  </tr>
  <tr>
    <td>`nomad.client.allocs.<Job>.<TaskGroup>.<AllocID>.<Task>.memory.max_usage`</td>
    <td>Maximum amount of memory ever used by the task</td>
    <td>Bytes</td>
    <td>Gauge</td>
  </tr>
  <tr>
    <td>`nomad.client.allocs.<Job>.<TaskGroup>.<AllocID>.<Task>.memory.kernel_usage`</td>
    <td>Amount of memory used by the kernel for this task</td>
    <td>Bytes</td>
    <td>Gauge</td>
  </tr>
  <tr>
    <td>`nomad.client.allocs.<Job>.<TaskGroup>.<AllocID>.<Task>.memory.kernel_max_usage`</td>
    <td>Maximum amount of memory ever used by the kernel for this task</td>
    <td>Bytes</td>
    <td>Gauge</td>
  </tr>
  <tr>
    <td>`nomad.client.allocs.<Job>.<TaskGroup>.<AllocID>.<Task>.cpu.total_percent`</td>
    <td>Total CPU resources consumed by the task across all cores</td>
    <td>Percentage</td>
    <td>Gauge</td>
  </tr>
  <tr>
    <td>`nomad.client.allocs.<Job>.<TaskGroup>.<AllocID>.<Task>.cpu.system`</td>
    <td>Total CPU resources consumed by the task in the system space</td>
    <td>Percentage</td>
    <td>Gauge</td>
  </tr>
  <tr>
    <td>`nomad.client.allocs.<Job>.TaskGroup>.<AllocID>.<Task>.cpu.user`</td>
    <td>Total CPU resources consumed by the task in the user space</td>
    <td>Percentage</td>
    <td>Gauge</td>
  </tr>
  <tr>
    <td>`nomad.client.allocs.<Job>.<TaskGroup>.<AllocID>.<Task>.cpu.throttled_time`</td>
    <td>Total time that the task was throttled</td>
    <td>Nanoseconds</td>
    <td>Gauge</td>
  </tr>
  <tr>
    <td>`nomad.client.allocs.<Job>.<TaskGroup>.<AllocID>.<Task>.cpu.total_ticks`</td>
    <td>CPU ticks consumed by the process in the last collection interval</td>
    <td>Integer</td>
    <td>Gauge</td>
  </tr>
</table>

# Metric Types

<table class="table table-bordered table-striped">
  <tr>
    <th>Type</th>
    <th>Description</th>
    <th>Quantiles</th>
  </tr>
  <tr>
    <td>Gauge</td>
    <td>
        Gauge types report an absolute number at the end of the aggregation
        interval
    </td>
    <td>false</td>
  </tr>
  <tr>
    <td>Counter</td>
    <td>
        Counts are incremented and flushed at the end of the aggregation
        interval and then are reset to zero
    </td>
    <td>true</td>
  </tr>
  <tr>
    <td>Timer</td>
    <td>
        Timers measure the time to complete a task and will include quantiles,
        means, standard deviation, etc per interval.
    </td>
    <td>true</td>
  </tr>
</table>
