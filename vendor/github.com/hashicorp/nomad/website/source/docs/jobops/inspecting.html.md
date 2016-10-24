---
layout: "docs"
page_title: "Operating a Job: Inspecting State"
sidebar_current: "docs-jobops-inspection"
description: |-
  Learn how to inspect a Nomad Job.
---

# Inspecting state

Once a job is submitted, the next step is to ensure it is running. This section
will assume we have submitted a job with the name _example_.

To get a high-level over view of our job we can use the [`nomad status`
command](/docs/commands/status.html). This command will display the list of
running allocations, as well as any recent placement failures. An example below
shows that the job has some allocations placed but did not have enough resources
to place all of the desired allocations. We run with `-evals` to see that there
is an outstanding evaluation for the job:

```
$ nomad status example
ID          = example
Name        = example
Type        = service
Priority    = 50
Datacenters = dc1
Status      = running
Periodic    = false

Evaluations
ID        Priority  Triggered By  Status    Placement Failures
5744eb15  50        job-register  blocked   N/A - In Progress
8e38e6cf  50        job-register  complete  true

Placement Failure
Task Group "cache":
  * Resources exhausted on 1 nodes
  * Dimension "cpu exhausted" exhausted on 1 nodes

Allocations
ID        Eval ID   Node ID   Task Group  Desired  Status   Created At
12681940  8e38e6cf  4beef22f  cache       run      running  08/08/16 21:03:19 CDT
395c5882  8e38e6cf  4beef22f  cache       run      running  08/08/16 21:03:19 CDT
4d7c6f84  8e38e6cf  4beef22f  cache       run      running  08/08/16 21:03:19 CDT
843b07b8  8e38e6cf  4beef22f  cache       run      running  08/08/16 21:03:19 CDT
a8bc6d3e  8e38e6cf  4beef22f  cache       run      running  08/08/16 21:03:19 CDT
b0beb907  8e38e6cf  4beef22f  cache       run      running  08/08/16 21:03:19 CDT
da21c1fd  8e38e6cf  4beef22f  cache       run      running  08/08/16 21:03:19 CDT
```

In the above example we see that the job has a "blocked" evaluation that is in
progress. When Nomad can not place all the desired allocations, it creates a
blocked evaluation that waits for more resources to become available. We can use
the [`eval-status` command](/docs/commands/eval-status.html) to examine any
evaluation in more detail. For the most part this should never be necessary but
can be useful to see why all of a job's allocations were not placed. For
example if we run it on the _example_ job, which had a placement failure
according to the above output, we see:

```
nomad eval-status 8e38e6cf
ID                 = 8e38e6cf
Status             = complete
Status Description = complete
Type               = service
TriggeredBy        = job-register
Job ID             = example
Priority           = 50
Placement Failures = true

Failed Placements
Task Group "cache" (failed to place 3 allocations):
  * Resources exhausted on 1 nodes
  * Dimension "cpu exhausted" exhausted on 1 nodes

Evaluation "5744eb15" waiting for additional capacity to place remainder
```

More interesting though is the [`alloc-status`
command](/docs/commands/alloc-status.html). This command gives us the most
recent events that occurred for a task, its resource usage, port allocations and
more:

```
nomad alloc-status 12
ID            = 12681940
Eval ID       = 8e38e6cf
Name          = example.cache[1]
Node ID       = 4beef22f
Job ID        = example
Client Status = running

Task "redis" is "running"
Task Resources
CPU    Memory           Disk     IOPS  Addresses
2/500  6.3 MiB/256 MiB  300 MiB  0     db: 127.0.0.1:57161

Recent Events:
Time                   Type        Description
06/28/16 15:46:42 UTC  Started     Task started by client
06/28/16 15:46:10 UTC  Restarting  Task restarting in 30.863215327s
06/28/16 15:46:10 UTC  Terminated  Exit Code: 137, Exit Message: "Docker container exited with non-zero exit code: 137"
06/28/16 15:37:46 UTC  Started     Task started by client
06/28/16 15:37:44 UTC  Received    Task received by client
```

In the above example we forced killed the Docker container so that we could see
in the event history that Nomad detected the failure and restarted the
allocation.

The `alloc-status` command is a good starting to point for debugging an
application that did not start. In this example task we are trying to start a
redis image using `redis:2.8` but the user has accidentally put a comma instead
of a period, typing `redis:2,8`.


When the job is run, it produces an allocation that fails. The `alloc-status`
command gives us the reason why:

```
nomad alloc-status c0f1
ID            = c0f1b34c
Eval ID       = 4df393cb
Name          = example.cache[0]
Node ID       = 13063955
Job ID        = example
Client Status = failed

Task "redis" is "dead"
Task Resources
CPU  Memory   Disk     IOPS  Addresses
500  256 MiB  300 MiB  0     db: 127.0.0.1:23285

Recent Events:
Time                   Type            Description
06/28/16 15:50:22 UTC  Not Restarting  Error was unrecoverable
06/28/16 15:50:22 UTC  Driver Failure  failed to create image: Failed to pull `redis:2,8`: API error (500): invalid tag format
06/28/16 15:50:22 UTC  Received        Task received by client
```

Not all failures are this easily debuggable. If the `alloc-status` command shows
many restarts occurring as in the example below, it is a good hint that the error
is occurring at the application level during start up. These failures can be
debugged by looking at logs which is covered in the [Nomad Job Logging
documentation](/docs/jobops/logs.html).

```
$ nomad alloc-status e6b6
ID            = e6b625a1
Eval ID       = 68b742e8
Name          = example.cache[0]
Node ID       = 83ef596c
Job ID        = example
Client Status = pending

Task "redis" is "pending"
Task Resources
CPU  Memory   Disk     IOPS  Addresses
500  256 MiB  300 MiB  0     db: 127.0.0.1:30153

Recent Events:
Time                   Type        Description
06/28/16 15:56:16 UTC  Restarting  Task restarting in 5.178426031s
06/28/16 15:56:16 UTC  Terminated  Exit Code: 1, Exit Message: "Docker container exited with non-zero exit code: 1"
06/28/16 15:56:16 UTC  Started     Task started by client
06/28/16 15:56:00 UTC  Restarting  Task restarting in 5.00123931s
06/28/16 15:56:00 UTC  Terminated  Exit Code: 1, Exit Message: "Docker container exited with non-zero exit code: 1"
06/28/16 15:55:59 UTC  Started     Task started by client
06/28/16 15:55:48 UTC  Received    Task received by client
```
