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

~> **Warning!** The examples on this page are _**not** eligible_ for the AWS
[free-tier](https://aws.amazon.com/free/). Do not execute the examples on this
page unless you're willing to spend a small amount of money.

## Using Modules

If you have any instances running from prior steps in the getting
started guide, use `terraform destroy` to destroy them, and remove all
configuration files.

As an example, we're going to use the
[Consul Terraform module](https://github.com/hashicorp/consul/tree/master/terraform)
which will setup a complete [Consul](https://www.consul.io) cluster
for us.

Create a configuration file with the following contents:

```
provider "aws" {
	access_key = "AWS ACCESS KEY"
	secret_key = "AWS SECRET KEY"
	region = "AWS REGION"
}

module "consul" {
	source = "github.com/hashicorp/consul/terraform/aws"

	key_name = "AWS SSH KEY NAME"
	key_path = "PATH TO ABOVE PRIVATE KEY"
	region = "us-east-1"
	servers = "3"
}
```

(Note that the `provider` block can be omitted in favor of environment
variables. See the [AWS Provider docs](/docs/providers/aws/index.html)
for details.)

The `module` block tells Terraform to create and manage a module. It is
very similar to the `resource` block. It has a logical name -- in this
case "consul" -- and a set of configurations.

The `source` configuration is the only mandatory key for modules. It tells
Terraform where the module can be retrieved. Terraform automatically
downloads and manages modules for you. For our example, we're getting the
module directly from GitHub. Terraform can retrieve modules from a variety
of sources including Git, Mercurial, HTTP, and file paths.

The other configurations are parameters to our module. Please fill them
in with the proper values.

Prior to running any command such as `plan` with a configuration that
uses modules, you'll have to [get](/docs/commands/get.html) the modules.
This is done using the [get command](/docs/commands/get.html).

```
$ terraform get
...
```

This command will download the modules if they haven't been already.
By default, the command will not check for updates, so it is safe (and fast)
to run multiple times. You can use the `-u` flag to check and download
updates.

## Planning and Apply Modules

With the modules downloaded, we can now plan and apply it. If you run
`terraform plan`, you should see output similar to below:

```
$ terraform plan
...
+ module.consul.aws_instance.server.0
...
+ module.consul.aws_instance.server.1
...
+ module.consul.aws_instance.server.2
...
+ module.consul.aws_security_group.consul
...
Plan: 4 to add, 0 to change, 0 to destroy.
```

Conceptually, the module is treated like a black box. In the plan, however
Terraform shows each resource the module manages so you can see each detail
about what the plan will do. If you'd like compressed plan output, you can
specify the `-module-depth=` flag to get Terraform to output summaries by
module.

Next, run `terraform apply` to create the module. Note that as we warned above,
the resources this module creates are outside of the AWS free tier, so this
will have some cost associated with it.

```
$ terraform apply
...
Apply complete! Resources: 3 added, 0 changed, 0 destroyed.
```

After a few minutes, you'll have a three server Consul cluster up and
running! Without any knowledge of how Consul works, how to install Consul,
or how to configure Consul into a cluster, you've created a real cluster in
just minutes.

## Module Outputs

Just as we parameterized the module with configurations such as
`servers` above, modules can also output information (just like a resource).

You'll have to reference the module's code or documentation to know what
outputs it supports for now, but for this guide we'll just tell you that the
Consul module has an output named `server_address` that has the address of
one of the Consul servers that was setup.

To reference this, we'll just put it into our own output variable. But this
value could be used anywhere: in another resource, to configure another
provider, etc.

```
output "consul_address" {
	value = "${module.consul.server_address}"
}
```

The syntax for referencing module outputs should be very familiar. The
syntax is `${module.NAME.ATTRIBUTE}`. The `NAME` is the logical name
we assigned earlier, and the `ATTRIBUTE` is the output attribute.

If you run `terraform apply` again, Terraform should make no changes, but
you'll now see the "consul\_address" output with the address of our Consul
server.

## Next

For more information on modules, the types of sources supported, how
to write modules, and more, read the in depth
[module documentation](/docs/modules/index.html).

We've now concluded the getting started guide, however
there are a number of [next steps](/intro/getting-started/next-steps.html)
to get started with Terraform.
