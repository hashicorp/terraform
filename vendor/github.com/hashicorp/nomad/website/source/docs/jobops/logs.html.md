---
layout: "docs"
page_title: "Operating a Job: Accessing Logs"
sidebar_current: "docs-jobops-logs"
description: |-
  Learn how to operate a Nomad Job.
---

# Accessing Logs

Accessing applications logs is critical when debugging issues, performance
problems or even for verifying the application is starting correctly. To make
this as simple as possible, Nomad provides [log
rotation](/docs/jobspec/index.html#log_rotation) in the jobspec, provides a [CLI
command](/docs/commands/logs.html) and an [API](/docs/http/client-fs.html#logs)
for accessing application logs and data files.

To see this in action we can just run the example job which created using `nomad
init`:

```
$ nomad init
Example job file written to example.nomad
```

This job will start a redis instance in a Docker container. We can run it now:

```
$ nomad run example.nomad
==> Monitoring evaluation "7a3b78c0"
    Evaluation triggered by job "example"
    Allocation "c3c58508" created: node "b5320e2d", group "cache"
    Evaluation status changed: "pending" -> "complete"
==> Evaluation "7a3b78c0" finished with status "complete"
```

We can grab the allocation ID from above and use the [`nomad logs`
command](/docs/commands/logs.html) to access the applications logs. The `logs`
command supports both displaying the logs as well as following logs, blocking
for more output. 

Thus to access the `stdout` we can issue the below command:

```
$ nomad logs c3c58508 redis
                 _._
            _.-``__ ''-._
       _.-``    `.  `_.  ''-._           Redis 3.2.1 (00000000/0) 64 bit
   .-`` .-```.  ```\/    _.,_ ''-._
  (    '      ,       .-`  | `,    )     Running in standalone mode
  |`-._`-...-` __...-.``-._|'` _.-'|     Port: 6379
  |    `-._   `._    /     _.-'    |     PID: 1
   `-._    `-._  `-./  _.-'    _.-'
  |`-._`-._    `-.__.-'    _.-'_.-'|
  |    `-._`-._        _.-'_.-'    |           http://redis.io
   `-._    `-._`-.__.-'_.-'    _.-'
  |`-._`-._    `-.__.-'    _.-'_.-'|
  |    `-._`-._        _.-'_.-'    |
   `-._    `-._`-.__.-'_.-'    _.-'
       `-._    `-.__.-'    _.-'
           `-._        _.-'
               `-.__.-'

 1:M 28 Jun 19:49:30.504 # WARNING: The TCP backlog setting of 511 cannot be enforced because /proc/sys/net/core/somaxconn is set to the lower value of 128.
 1:M 28 Jun 19:49:30.505 # Server started, Redis version 3.2.1
 1:M 28 Jun 19:49:30.505 # WARNING overcommit_memory is set to 0! Background save may fail under low memory condition. To fix this issue add 'vm.overcommit_memory = 1' to /etc/sysctl.conf and then reboot or run the command 'sysctl vm.overcommit_memory=1' for this to take effect.
 1:M 28 Jun 19:49:30.505 # WARNING you have Transparent Huge Pages (THP) support enabled in your kernel. This will create latency and memory usage issues with Redis. To fix this issue run the command 'echo never > /sys/kernel/mm/transparent_hugepage/enabled' as root, and add it to your /etc/rc.local in order to retain the setting after a reboot. Redis must be restarted after THP is disabled.
 1:M 28 Jun 19:49:30.505 * The server is now ready to accept connections on port 6379
```

To display the `stderr` for the task we would run the following: 

```
$ nomad logs -stderr c3c58508 redis
```

While this works well for quickly accessing logs, we recommend running a
log-shipper for long term storage of logs. In many cases this will not be needed
and the above will suffice but for use cases in which log retention is needed
Nomad can accommodate.

Since we place application logs inside the `alloc/` directory, all tasks within
the same task group have access to each others logs. Thus we can have a task
group as follows:

```
group "my-group" {
    task "log-producer" {...}
    task "log-shipper" {...}
}
```

In the above example, the `log-producer` task is the application that should be
run and will be producing the logs we would like to ship and the `log-shipper`
reads these logs from the `alloc/logs/` directory and ships them to a long term
storage such as S3.
