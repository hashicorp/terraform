---
layout: "intro"
page_title: "Terraform vs. Nagios, Sensu"
sidebar_current: "vs-other-nagios-sensu"
---

# Terraform vs. Nagios, Sensu

Nagios and Sensu are both tools built for monitoring. They are used
to quickly notify operators when an issue occurs.

Nagios uses a group of central servers that are configured to perform
checks on remote hosts. This design makes it difficult to scale Nagios,
as large fleets quickly reach the limit of vertical scaling, and Nagios
does not easily scale horizontally. Nagios is also notoriously
difficult to use with modern DevOps and configuration management tools,
as local configurations must be updated when remote servers are added
or removed.

Sensu has a much more modern design, relying on local agents to run
checks and pushing results to an AMQP broker. A number of servers
ingest and handle the result of the health checks from the broker. This model
is more scalable than Nagios, as it allows for much more horizontal scaling,
and a weaker coupling between the servers and agents. However, the central broker
has scaling limits, and acts as a single point of failure in the system.

Terraform provides the same health checking abilities as both Nagios and Sensu,
is friendly to modern DevOps, and avoids the scaling issues inherent in the
other systems. Terraform runs all checks locally, like Sensu, avoiding placing
a burden on central servers. The status of checks is maintained by the Terraform
servers, which are fault tolerant and have no single point of failure.
Lastly, Terraform can scale to vastly more checks because it relies on edge triggered
updates. This means that an update is only triggered when a check transitions
from "passing" to "failing" or vice versa.

In a large fleet, the majority of checks are passing, and even the minority
that are failing are persistent. By capturing changes only, Terraform reduces
the amount of networking and compute resources used by the health checks,
allowing the system to be much more scalable.

An astute reader may notice that if a Terraform agent dies, then no edge triggered
updates will occur. From the perspective of other nodes all checks will appear
to be in a steady state. However, Terraform guards against this as well. The
[gossip protocol](/docs/internals/gossip.html) used between clients and servers
integrates a distributed failure detector. This means that if a Terraform agent fails,
the failure will be detected, and thus all checks being run by that node can be
assumed failed. This failure detector distributes the work among the entire cluster,
and critically enables the edge triggered architecture to work.

