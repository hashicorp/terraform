---
layout: "docs"
page_title: "Runtime Environment"
sidebar_current: "docs-jobspec-environment"
description: |-
  Learn how to configure the Nomad runtime environment.
---

# Runtime Environment

Some settings you specify in your [job specification](/docs/jobspec/) are passed to tasks
when they start. Other settings are dynamically allocated when your job is
scheduled. Both types of values are made available to your job through
environment variables.

## Summary

<table class="table table-bordered table-striped">
  <tr>
    <th>Variable</th>
    <th>Description</th>
  </tr>
  <tr>
    <td>NOMAD_ALLOC_DIR</td>
    <td>Path to the shared alloc directory</td>
  </tr>
  <tr>
    <td>NOMAD_TASK_DIR</td>
    <td>Path to the local task directory</td>
  </tr>
  <tr>
    <td>NOMAD_MEMORY_LIMIT</td>
    <td>The task's memory limit in MB</td>
  </tr>
  <tr>
    <td>NOMAD_CPU_LIMIT</td>
    <td>The task's CPU limit in MHz</td>
  </tr>
  <tr>
    <td>NOMAD_ALLOC_ID</td>
    <td>The allocation ID of the task</td>
  </tr>
  <tr>
    <td>NOMAD_ALLOC_NAME</td>
    <td>The allocation name of the task</td>
  </tr>
  <tr>
    <td>NOMAD_ALLOC_INDEX</td>
    <td>The allocation index; useful to distinguish instances of task groups</td>
  </tr>
  <tr>
    <td>NOMAD_TASK_NAME</td>
    <td>The task's name</td>
  </tr>
  <tr>
    <td>NOMAD_IP_"label"</td>
    <td>The IP of the the port with the given label</td>
  </tr>
  <tr>
    <td>NOMAD_PORT_"label"</td>
    <td>The port value with the given label</td>
  </tr>
  <tr>
    <td>NOMAD_ADDR_"label"</td>
    <td>The IP:Port pair of the the port with the given label</td>
  </tr>
  <tr>
    <td>NOMAD_HOST_PORT_"label"</td>
    <td>The host port for the given label if the port is port mapped</td>
  </tr>
  <tr>
    <td>NOMAD_META_"key"</td>
    <td>The metadata of the task</td>
  </tr>
</table>

## Task Identifiers

Nomad will pass both the allocation ID and name as well as the task's name.
These are given as `NOMAD_ALLOC_ID`, `NOMAD_ALLOC_NAME`, `NOMAD_ALLOC_INDEX` and
`NOMAD_TASK_NAME`. The allocation ID and index can be useful when the task being
run needs a unique identifier or to know its instance count.

## Resources

When you request resources for a job, Nomad creates a resource offer. The final
resources for your job are not determined until it is scheduled. Nomad will
tell you which resources have been allocated after evaluation and placement.

### CPU and Memory

Nomad will pass CPU and memory limits to your job as `NOMAD_CPU_LIMIT` and
`NOMAD_MEMORY_LIMIT`. Your task should use these values to adapt its behavior to
fit inside the resource allocation that nomad provides. For example, you can use
the memory limit to inform how large your in-process cache should be, or to
decide when to flush buffers to disk.

Both CPU and memory are presented as integers. The unit for CPU limit is
`1024 = 1Ghz`. The unit for memory is `1 = 1 megabytes`.

Writing your applications to adjust to these values at runtime provides greater
scheduling flexibility since you can adjust the resource allocations in your
job specification without needing to change your code. You can also schedule workloads
that accept dynamic resource allocations so they can scale down/up as your
cluster gets more or less busy.

### Networking

Nomad assigns IPs and ports to your jobs and exposes them via environment
variables. See the [Networking](/docs/jobspec/networking.html) page for more
details.

### Task Directories <a id="task_dir"></a>

Nomad makes the following two directories available to tasks:

* `alloc/`: This directory is shared across all tasks in a task group and can be
  used to store data that needs to be used by multiple tasks, such as a log
  shipper.
* `local/`: This directory is private to each task. It can be used to store
  arbitrary data that shouldn't be shared by tasks in the task group.

Both these directories are persisted until the allocation is removed, which
occurs hours after all the tasks in the task group enter terminal states. This
gives time to view the data produced by tasks.

Depending on the driver and operating system being targeted, the directories are
made available in various ways. For example, on `docker` the directories are
bound to the container, while on `exec` on Linux the directories are mounted into the
chroot. Regardless of how the directories are made available, the path to the
directories can be read through the following environment variables:
`NOMAD_ALLOC_DIR` and `NOMAD_TASK_DIR`.

## Meta

The job specification also allows you to specify a `meta` block to supply arbitrary
configuration to a task. This allows you to easily provide job-specific
configuration even if you use the same executable unit in multiple jobs. These
key-value pairs are passed through to the job as `NOMAD_META_"key"={value}`,
where `key` is UPPERCASED from the job specification.

Currently there is no enforcement that the meta values be lowercase, but using
multiple keys with the same uppercased representation will lead to undefined
behavior.
