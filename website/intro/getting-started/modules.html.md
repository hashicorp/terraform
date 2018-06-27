---
layout: "intro"
page_title: "Modules"
sidebar_current: "gettingstarted-modules"
description: |-
  Up to this point, we've been configuring Terraform by editing Terraform configurations directly. As our infrastructure grows, this practice has a few key problems: a lack of organization, a lack of reusability, and difficulties in management for teams.
---

# Modules

Up to this point, we've been configuring Terraform by editing Terraform
configurations directly. As our infrastructure grows, this practice has a few
key problems: a lack of organization, a lack of reusability, and difficulties
in management for teams.

_Modules_ in Terraform are self-contained packages of Terraform configurations
that are managed as a group. Modules are used to create reusable components,
improve organization, and to treat pieces of infrastructure as a black box.

This section of the getting started will cover the basics of using modules.
Writing modules is covered in more detail in the
[modules documentation](/docs/modules/index.html).

~> **Warning!** The examples on this page are _**not** eligible_ for
[the AWS free tier](https://aws.amazon.com/free/). Do not try the examples
on this page unless you're willing to spend a small amount of money.

## Using Modules

If you have any instances running from prior steps in the getting
started guide, use `terraform destroy` to destroy them, and remove all
configuration files.

The [Terraform Registry](https://registry.terraform.io/) includes a directory
of ready-to-use modules for various common purposes, which can serve as
larger building-blocks for your infrastructure.

In this example, we're going to use
[the Consul Terraform module for AWS](https://registry.terraform.io/modules/hashicorp/consul/aws),
which will set up a complete [Consul](https://www.consul.io) cluster.
This and other modules can be found via the search feature on the Terraform
Registry site.

Create a configuration file with the following contents:

```hcl
provider "aws" {
  access_key = "AWS ACCESS KEY"
  secret_key = "AWS SECRET KEY"
  region     = "us-east-1"
}

module "consul" {
  source = "hashicorp/consul/aws"

  num_servers = "3"
}
```

The `module` block begins with the example given on the Terraform Registry
page for this module, telling Terraform to create and manage this module.
This is similar to a `resource` block: it has a name used within this
configuration -- in this case, `"consul"` -- and a set of input values
that are listed in
[the module's "Inputs" documentation](https://registry.terraform.io/modules/hashicorp/consul/aws?tab=inputs).

(Note that the `provider` block can be omitted in favor of environment
variables. See the [AWS Provider docs](/docs/providers/aws/index.html)
for details.  This module requires that your AWS account has a default VPC.)

The `source` attribute is the only mandatory argument for modules. It tells
Terraform where the module can be retrieved. Terraform automatically
downloads and manages modules for you.

In this case, the module is retrieved from the official Terraform Registry.
Terraform can also retrieve modules from a variety of sources, including
private module registries or directly from Git, Mercurial, HTTP, and local
files.

The other attributes shown are inputs to our module. This module supports many
additional inputs, but all are optional and have reasonable values for
experimentation.

After adding a new module to configuration, it is necessary to run (or re-run)
`terraform init` to obtain and install the new module's source code:

```
$ terraform init
# ...
```

By default, this command does not check for new module versions that may be
available, so it is safe to run multiple times. The `-upgrade` option will
additionally check for any newer versions of existing modules and providers
that may be available.

## Apply Changes

With the Consul module (and its dependencies) installed, we can now apply
these changes to create the resources described within.

If you run `terraform apply`, you will see a large list of all of the
resources encapsulated in the module. The output is similar to what we
saw when using resources directly, but the resource names now have
module paths prefixed to their names, like in the following example:

```
  + module.consul.module.consul_clients.aws_autoscaling_group.autoscaling_group
      id:                                        <computed>
      arn:                                       <computed>
      default_cooldown:                          <computed>
      desired_capacity:                          "6"
      force_delete:                              "false"
      health_check_grace_period:                 "300"
      health_check_type:                         "EC2"
      launch_configuration:                      "${aws_launch_configuration.launch_configuration.name}"
      max_size:                                  "6"
      metrics_granularity:                       "1Minute"
      min_size:                                  "6"
      name:                                      <computed>
      protect_from_scale_in:                     "false"
      tag.#:                                     "2"
      tag.2151078592.key:                        "consul-clients"
      tag.2151078592.propagate_at_launch:        "true"
      tag.2151078592.value:                      "consul-example"
      tag.462896764.key:                         "Name"
      tag.462896764.propagate_at_launch:         "true"
      tag.462896764.value:                       "consul-example-client"
      termination_policies.#:                    "1"
      termination_policies.0:                    "Default"
      vpc_zone_identifier.#:                     "6"
      vpc_zone_identifier.1880739334:            "subnet-5ce4282a"
      vpc_zone_identifier.3458061785:            "subnet-16600f73"
      vpc_zone_identifier.4176925006:            "subnet-485abd10"
      vpc_zone_identifier.4226228233:            "subnet-40a9b86b"
      vpc_zone_identifier.595613151:             "subnet-5131b95d"
      vpc_zone_identifier.765942872:             "subnet-595ae164"
      wait_for_capacity_timeout:                 "10m"
```

The `module.consul.module.consul_clients` prefix shown above indicates
not only that the resource is from the `module "consul"` block we wrote,
but in fact that this module has its own `module "consul_clients"` block
within it. Modules can be nested to decompose complex systems into
manageable components.

The full set of resources created by this module includes an autoscaling group,
security groups, IAM roles and other individual resources that all support
the Consul cluster that will be created.

Note that as we warned above, the resources created by this module are
not eligible for the AWS free tier and so proceeding further will have some
cost associated. To proceed with the creation of the Consul cluster, type
`yes` at the confirmation prompt.

```
# ...

module.consul.module.consul_clients.aws_security_group.lc_security_group: Creating...
  description:            "" => "Security group for the consul-example-client launch configuration"
  egress.#:               "" => "<computed>"
  ingress.#:              "" => "<computed>"
  name:                   "" => "<computed>"
  name_prefix:            "" => "consul-example-client"
  owner_id:               "" => "<computed>"
  revoke_rules_on_delete: "" => "false"
  vpc_id:                 "" => "vpc-22099946"

# ...

Apply complete! Resources: 34 added, 0 changed, 0 destroyed.
```

After several minutes and many log messages about all of the resources
being created, you'll have a three-server Consul cluster up and running.
Without needing any knowledge of how Consul works, how to install Consul,
or how to form a Consul cluster, you've created a working cluster in just
a few minutes.

## Module Outputs

Just as the module instance had input arguments such as `num_servers` above,
module can also produce _output_ values, similar to resource attributes.

[The module's outputs reference](https://registry.terraform.io/modules/hashicorp/consul/aws?tab=outputs)
describes all of the different values it produces. Overall, it exposes the
id of each of the resources it creates, as well as echoing back some of the
input values.

One of the supported outputs is called `asg_name_servers`, and its value
is the name of the auto-scaling group that was created to manage the Consul
servers.

To reference this, we'll just put it into our _own_ output value. This
value could actually be used anywhere: in another resource, to configure
another provider, etc.

Add the following to the end of the existing configuration file created
above:

```hcl
output "consul_server_asg_name" {
  value = "${module.consul.asg_name_servers}"
}
```

The syntax for referencing module outputs is `${module.NAME.OUTPUT}`, where
`NAME` is the module name given in the header of the `module` configuration
block and `OUTPUT` is the name of the output to reference.

If you run `terraform apply` again, Terraform will make no changes to
infrastructure, but you'll now see the "consul\_server\_asg\_name" output with
the name of the created auto-scaling group:

```
# ...

Apply complete! Resources: 0 added, 0 changed, 0 destroyed.

Outputs:

consul_server_asg_name = tf-asg-2017103123350991200000000a
```

If you look in the Auto-scaling Groups section of the EC2 console you should
find an autoscaling group of this name, and from there find the three
Consul servers it is running. (If you can't find it, make sure you're looking
in the right region!)

## Destroy

Just as with top-level resources, we can destroy the resources created by
the Consul module to avoid ongoing costs:

```
$ terraform destroy
# ...

Terraform will perform the following actions:

  - module.consul.module.consul_clients.aws_autoscaling_group.autoscaling_group

  - module.consul.module.consul_clients.aws_iam_instance_profile.instance_profile

  - module.consul.module.consul_clients.aws_iam_role.instance_role

# ...
```

As usual, Terraform describes all of the actions it will take. In this case,
it plans to destroy all of the resources that were created by the module.
Type `yes` to confirm and, after a few minutes and even more log output,
all of the resources should be destroyed:

```
Destroy complete! Resources: 34 destroyed.
```

With all of the resources destroyed, you can delete the configuration file
we created above. We will not make any further use of it, and so this avoids
the risk of accidentally re-creating the Consul cluster.

## Next

For more information on modules, the types of sources supported, how
to write modules, and more, read the in-depth
[module documentation](/docs/modules/index.html).

Next, we learn about [Terraform's remote collaboration features](/intro/getting-started/remote.html).
