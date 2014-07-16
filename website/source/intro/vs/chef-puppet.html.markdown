---
layout: "intro"
page_title: "Terraform vs. Chef, Puppet, etc."
sidebar_current: "vs-other-chef"
---

# Terraform vs. Chef, Puppet, etc.

It is not uncommon to find people using Chef, Puppet, and other configuration
management tools to build service discovery mechanisms. This is usually
done by querying global state to construct configuration files on each
node during a periodic convergence run.

Unfortunately, this approach has
a number of pitfalls. The configuration information is static,
and cannot update any more frequently than convergence runs. Generally this
is on the interval of many minutes or hours. Additionally, there is no
mechanism to incorporate the system state in the configuration. Nodes which
are unhealthy may receive traffic exacerbating issues further. Using this
approach also makes supporting multiple datacenters challenging as a central
group of servers must manage all datacenters.

Terraform is designed specifically as a service discovery tool. As such,
it is much more dynamic and responsive to the state of the cluster. Nodes
can register and deregister the services they provide, enabling dependent
applications and services to rapidly discover all providers. By using the
integrated health checking, Terraform can route traffic away from unhealthy
nodes, allowing systems and services to gracefully recover. Static configuration
that may be provided by configuration management tools can be moved into the
dynamic key/value store. This allows application configuration to be updated
without a slow convergence run. Lastly, because each datacenter runs indepedently,
supporting multiple datacenters is no different than a single datacenter.

That said, Terraform is not a replacement for configuration management tools.
These tools are still critical to setup applications and even to
configure Terraform itself. Static provisioning is best managed
by existing tools, while dynamic state and discovery is better managed by
Terraform. The separation of configuration management and cluster management
also has a number of advantageous side effects: Chef recipes and Puppet manifests
become simpler without global state, periodic runs are no longer required for service
or configuration changes, and the infrastructure can become immutable since config management
runs require no global state.
