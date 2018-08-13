---
layout: "intro"
page_title: "Consul Example"
sidebar_current: "examples-consul"
description: |-
  Consul is a tool for service discovery, configuration and orchestration. The Key/Value store it provides is often used to store application configuration and information about the infrastructure necessary to process requests.
---

# Consul Example

[**Example Source Code**](https://github.com/terraform-providers/terraform-provider-consul/tree/master/examples/kv)

[Consul](https://www.consul.io) is a tool for service discovery, configuration
and orchestration. The Key/Value store it provides is often used to store
application configuration and information about the infrastructure necessary
to process requests.

Terraform provides a [Consul provider](/docs/providers/consul/index.html) which
can be used to interface with Consul from inside a Terraform configuration.

For our example, we use the [Consul demo cluster](https://demo.consul.io/)
to both read configuration and store information about a newly created EC2 instance.
The size of the EC2 instance will be determined by the `tf_test/size` key in Consul,
and will default to `m1.small` if that key does not exist. Once the instance is created
the `tf_test/id` and `tf_test/public_dns` keys will be set with the computed
values for the instance.

Before we run the example, use the [Web UI](https://demo.consul.io/ui/dc1/kv/)
to set the `tf_test/size` key to `t1.micro`. Once that is done,
copy the configuration into a configuration file (`consul.tf` works fine).
Either provide the AWS credentials as a default value in the configuration
or invoke `apply` with the appropriate variables set.

Once the `apply` has completed, we can see the keys in Consul by
visiting the [Web UI](https://demo.consul.io/ui/dc1/kv/). We can see
that the `tf_test/id` and `tf_test/public_dns` values have been
set.

You can now [tear down the infrastructure](/intro/getting-started/destroy.html)
Because we set the `delete` property of two of the Consul keys, Terraform
will clean up those keys on destroy. We can verify this by using
the Web UI.

This example has shown that Consul can be used with Terraform both to read
existing values and to store generated results.

Inputs like AMI name, security groups, Puppet roles, bootstrap scripts,
etc can all be loaded from Consul. This allows the specifics of an
infrastructure to be decoupled from its overall architecture. This enables
details to be changed without updating the Terraform configuration.

Outputs from Terraform can also be easily stored in Consul. One powerful
feature this enables is using Consul for inventory management. If an
application relies on ELB for routing, Terraform can update the application's
configuration directly by setting the ELB address into Consul. Any resource
attribute can be stored in Consul, allowing an operator to capture anything
useful.
