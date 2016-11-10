---
layout: "docs"
page_title: "Scheduling"
sidebar_current: "docs-internals-scheduling"
description: |-
  Learn about how scheduling works in Nomad.
---

# Scheduling

Scheduling is a core function of Nomad. It is the process of assigning tasks
from jobs to client machines. This process must respect the constraints as declared
in the job, and optimize for resource utilization. This page documents the details
of how scheduling works in Nomad to help both users and developers
build a mental model. The design is heavily inspired by Google's
work on both [Omega: flexible, scalable schedulers for large compute clusters](https://research.google.com/pubs/pub41684.html)
and [Large-scale cluster management at Google with Borg](https://research.google.com/pubs/pub43438.html).

~> **Advanced Topic!** This page covers technical details
of Nomad. You do not need to understand these details to
effectively use Nomad. The details are documented here for
those who wish to learn about them without having to go
spelunking through the source code.

# Scheduling in Nomad

[![Nomad Data Model](/assets/images/nomad-data-model.png)](/assets/images/nomad-data-model.png)

There are four primary "nouns" in Nomad; jobs, nodes, allocations, and evaluations.
Jobs are submitted by users and represent a _desired state_. A job is a declarative description
of tasks to run which are bounded by constraints and require resources. Tasks can be scheduled on 
nodes in the cluster running the Nomad client. The mapping of tasks in a job to clients is done
using allocations. An allocation is used to declare that a set of tasks in a job should be run
on a particular node. Scheduling is the process of determining the appropriate allocations and
is done as part of an evaluation.

An evaluation is created any time the external state, either desired or emergent, changes. The desired
state is based on jobs, meaning the desired state changes if a new job is submitted, an
existing job is updated, or a job is deregistered. The emergent state is based on the client
nodes, and so we must handle the failure of any clients in the system. These events trigger
the creation of a new evaluation, as Nomad must _evaluate_ the state of the world and reconcile
it with the desired state.

This diagram shows the flow of an evaluation through Nomad:

[![Nomad Evaluation Flow](/assets/images/nomad-evaluation-flow.png)](/assets/images/nomad-evaluation-flow.png)

The lifecycle of an evaluation begins with an event causing the evaluation to be
created. Evaluations are created in the `pending` state and are enqueued into the
evaluation broker. There is a single evaluation broker which runs on the leader server.
The evaluation broker is used to manage the queue of pending evaluations, provide priority ordering,
and ensure at least once delivery.

Nomad servers run scheduling workers, defaulting to one per CPU core, which are used to
process evaluations. The workers dequeue evaluations from the broker, and then invoke
the appropriate scheduler as specified by the job. Nomad ships with a `service` scheduler
that optimizes for long-lived services, a `batch` scheduler that is used for fast placement
of batch jobs, a `system` scheduler that is used to run jobs on every node,
and a `core` scheduler which is used for internal maintenance.
Nomad can be extended to support custom schedulers as well.

Schedulers are responsible for processing an evaluation and generating an allocation _plan_.
The plan is the set of allocations to evict, update, or create. The specific logic used to
generate a plan may vary by scheduler, but generally the scheduler needs to first reconcile
the desired state with the real state to determine what must be done. New allocations need
to be placed and existing allocations may need to be updated, migrated, or stopped.

Placing allocations is split into two distinct phases, feasibility
checking and ranking. In the first phase the scheduler finds nodes that are
feasible by filtering unhealthy nodes, those missing necessary drivers, and those
failing the specified constraints.

The second phase is ranking, where the scheduler scores feasible nodes to find the best fit.
Scoring is primarily based on bin packing, which is used to optimize the resource utilization
and density of applications, but is also augmented by affinity and anti-affinity rules.
Once the scheduler has ranked enough nodes, the highest ranking node is selected and
added to the allocation plan.

When planning is complete, the scheduler submits the plan to the leader which adds
the plan to the plan queue. The plan queue manages pending plans, provides priority
ordering, and allows Nomad to handle concurrency races. Multiple schedulers are running
in parallel without locking or reservations, making Nomad optimistically concurrent.
As a result, schedulers might overlap work on the same node and cause resource
over-subscription. The plan queue allows the leader node to protect against this and
do partial or complete rejections of a plan.

As the leader processes plans, it creates allocations when there is no conflict
and otherwise informs the scheduler of a failure in the plan result. The plan result
provides feedback to the scheduler, allowing it to terminate or explore alternate plans
if the previous plan was partially or completely rejected.

Once the scheduler has finished processing an evaluation, it updates the status of
the evaluation and acknowledges delivery with the evaluation broker. This completes
the lifecycle of an evaluation. Allocations that were created, modified or deleted
as a result will be picked up by client nodes and will begin execution.

