---
layout: "http"
page_title: "HTTP API"
sidebar_current: "docs-http-overview"
description: |-
  Nomad has an HTTP API that can be used to programmatically use Nomad.
---

# HTTP API

The Nomad HTTP API is the primary interface to using Nomad, and is used
to query the current state of the system as well as to modify it.
The Nomad CLI makes use of the Go HTTP client and invokes the HTTP API.

All API routes are prefixed with `/v1/`. This documentation is only for the v1 API.

## Data Model and API Layout

There are four primary "nouns" in Nomad, these are jobs, nodes, allocations, and evaluations:

[![Nomad Data Model](/assets/images/nomad-data-model.png)](/assets/images/nomad-data-model.png)

Jobs are submitted by users and represent a _desired state_. A job is a declarative description
of tasks to run which are bounded by constraints and require resources. Nodes are the servers
in the clusters that tasks can be scheduled on. The mapping of tasks in a job to nodes is done
using allocations. An allocation is used to declare that a set of tasks in a job should be run
on a particular node. Scheduling is the process of determining the appropriate allocations and
is done as part of an evaluation.

The API is modeled closely on the underlying data model. Use the links to the left for
documentation about specific endpoints. There are also "Agent" APIs which interact with
a specific agent and not the broader cluster used for administration.

<a name="blocking-queries"></a>
## Blocking Queries

Certain endpoints support a feature called a "blocking query." A blocking query
is used to wait for a potential change using long polling.

Not all endpoints support blocking, but those that do are clearly designated in
the documentation. Any endpoint that supports blocking will set the HTTP header
`X-Nomad-Index`, a unique identifier representing the current state of the
requested resource. On subsequent requests for this resource, the client can set
the `index` query string parameter to the value of `X-Nomad-Index`, indicating
that the client wishes to wait for any changes subsequent to that index.

In addition to `index`, endpoints that support blocking will also honor a `wait`
parameter specifying a maximum duration for the blocking request. This is limited to
10 minutes. If not set, the wait time defaults to 5 minutes. This value can be specified
in the form of "10s" or "5m" (i.e., 10 seconds or 5 minutes, respectively).

A critical note is that the return of a blocking request is **no guarantee** of a change. It
is possible that the timeout was reached or that there was an idempotent write that does
not affect the result of the query.

## Consistency Modes

Most of the read query endpoints support multiple levels of consistency. Since no policy will
suit all clients' needs, these consistency modes allow the user to have the ultimate say in
how to balance the trade-offs inherent in a distributed system.

The two read modes are:

* default - If not specified, the default is strongly consistent in almost all cases. However,
  there is a small window in which a new leader may be elected during which the old leader may
  service stale values. The trade-off is fast reads but potentially stale values. The condition
  resulting in stale reads is hard to trigger, and most clients should not need to worry about
  this case.  Also, note that this race condition only applies to reads, not writes.

* stale - This mode allows any server to service the read regardless of whether
  it is the leader. This means reads can be arbitrarily stale; however, results are generally
  consistent to within 50 milliseconds of the leader. The trade-off is very fast and
  scalable reads with a higher likelihood of stale values. Since this mode allows reads without
  a leader, a cluster that is unavailable will still be able to respond to queries.

To switch these modes, use the `stale` query parameter on request.

To support bounding the acceptable staleness of data, responses provide the `X-Nomad-LastContact`
header containing the time in milliseconds that a server was last contacted by the leader node.
The `X-Nomad-KnownLeader` header also indicates if there is a known leader. These can be used
by clients to gauge the staleness of a result and take appropriate action.

## Cross-Region Requests

By default any request to the HTTP API is assumed to pertain to the region of the machine
servicing the request. A target region can be explicitly specified with the `region` query
parameter. The request will be transparently forwarded and serviced by a server in the
appropriate region.

## Compressed Responses

The HTTP API will gzip the response if the HTTP request denotes that the client accepts
gzip compression. This is achieved via the standard, `Accept-Encoding: gzip`

## Formatted JSON Output

By default, the output of all HTTP API requests is minimized JSON.  If the client passes `pretty`
on the query string, formatted JSON will be returned.
